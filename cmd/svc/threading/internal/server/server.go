package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/textutil"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const setupThreadTitle = "Setup"

// go vet doesn't like that the first argument to grpcErrorf is not a string so alias the function with a different name :(
var grpcErrorf = grpc.Errorf

// maxSummaryLength sets the maximum length for the message summary. This must
// match what the underlying DAL supports so if updating here make sure the DAL
// is updated as well (e.g. db schema).
const maxSummaryLength = 1024

var (
	// baymaxLaunchDate represents the approximate day when the service was launched
	// so that we can use the timestamp as a reference point for when we started
	// receiving messages on the service.
	baymaxLaunchDate = time.Date(2016, 02, 25, 00, 00, 00, 00, time.UTC)
)

type threadsServer struct {
	clk                clock.Clock
	dal                dal.DAL
	sns                snsiface.SNSAPI
	snsTopicARN        string
	notificationClient notification.Client
	directoryClient    directory.DirectoryClient
	settingsClient     settings.SettingsClient
	mediaClient        media.MediaClient
	paymentsClient     payments.PaymentsClient
	webDomain          string
}

// NewThreadsServer returns an initialized instance of threadsServer
func NewThreadsServer(
	clk clock.Clock,
	dal dal.DAL,
	sns snsiface.SNSAPI,
	snsTopicARN string,
	notificationClient notification.Client,
	directoryClient directory.DirectoryClient,
	settingsClient settings.SettingsClient,
	mediaClient media.MediaClient,
	paymentsClient payments.PaymentsClient,
	webDomain string,
) threading.ThreadsServer {
	if clk == nil {
		clk = clock.New()
	}
	return &threadsServer{
		clk:                clk,
		dal:                dal,
		sns:                sns,
		snsTopicARN:        snsTopicARN,
		notificationClient: notificationClient,
		directoryClient:    directoryClient,
		settingsClient:     settingsClient,
		mediaClient:        mediaClient,
		webDomain:          webDomain,
		paymentsClient:     paymentsClient,
	}
}

// CreateSavedQuery saves a query for later use
func (s *threadsServer) CreateSavedQuery(ctx context.Context, in *threading.CreateSavedQueryRequest) (*threading.CreateSavedQueryResponse, error) {
	if in.EntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "EntityID is required")
	}
	if in.Query == nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Query is required")
	}
	if in.Title == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "Title is required")
	}

	query, err := transformQueryFromRequest(in.Query)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Query is invalid: %s", err)
	}

	sq := &models.SavedQuery{
		EntityID: in.EntityID,
		Title:    in.Title,
		Query:    query,
		Ordinal:  int(in.Ordinal),
	}
	id, err := s.dal.CreateSavedQuery(ctx, sq)
	if err != nil {
		return nil, errors.Trace(err)
	}
	sq.ID = id
	if err := s.rebuildSavedQuery(ctx, sq); err != nil {
		golog.Errorf("Failed to build new saved query %s: %s", sq.ID, err)
	}
	sq.ID = id
	sqr, err := transformSavedQueryToResponse(sq)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.CreateSavedQueryResponse{
		SavedQuery: sqr,
	}, nil
}

// CreateEmptyThread create a new thread with no messages
func (s *threadsServer) CreateEmptyThread(ctx context.Context, in *threading.CreateEmptyThreadRequest) (*threading.CreateEmptyThreadResponse, error) {
	switch in.Type {
	case threading.THREAD_TYPE_EXTERNAL, threading.THREAD_TYPE_SECURE_EXTERNAL, threading.THREAD_TYPE_TEAM:
	default:
		return nil, grpcErrorf(codes.InvalidArgument, fmt.Sprintf("Type '%s' not allowed for CreateEmptyThread", in.Type.String()))
	}
	if in.OrganizationID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "OrganizationID is required")
	}
	if in.PrimaryEntityID == "" && in.Type != threading.THREAD_TYPE_TEAM && in.Type != threading.THREAD_TYPE_SECURE_EXTERNAL {
		return nil, grpcErrorf(codes.InvalidArgument, "PrimaryEntityID is required for non app only threads")
	}
	if in.Summary == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "Summary is required")
	}
	in.Summary = textutil.TruncateUTF8(in.Summary, maxSummaryLength)
	if in.Type == threading.THREAD_TYPE_TEAM && in.FromEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "FromEntityID is required for TEAM threads")
	}
	if id, ok := validateEntityIDs(in.MemberEntityIDs); !ok {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid entity ID %s in members list", id)
	}

	if (in.Type == threading.THREAD_TYPE_EXTERNAL || in.Type == threading.THREAD_TYPE_SECURE_EXTERNAL) && in.SystemTitle == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "SystemTitle is required for EXTERNAL and SECURE_EXTERNAL threads")
	}

	tt, err := transformThreadTypeFromRequest(in.Type)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid thread type")
	}

	to, err := transformThreadOriginFromRequest(in.Origin)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid thread origin")
	}

	memberEntityIDs, err := memberEntityIDsForNewThread(in.Type, in.OrganizationID, in.FromEntityID, in.MemberEntityIDs)
	if err != nil {
		return nil, err
	}

	var systemTitle string
	switch in.Type {
	case threading.THREAD_TYPE_TEAM:
		systemTitle, err = s.teamThreadSystemTitle(ctx, in.OrganizationID, memberEntityIDs)
		if err != nil {
			return nil, errors.Trace(err)
		}
	case threading.THREAD_TYPE_SETUP:
		systemTitle = setupThreadTitle
	case threading.THREAD_TYPE_EXTERNAL, threading.THREAD_TYPE_SECURE_EXTERNAL:
		systemTitle = in.SystemTitle
	}

	var threadID models.ThreadID
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		threadID, err = dl.CreateThread(ctx, &models.Thread{
			OrganizationID:     in.OrganizationID,
			PrimaryEntityID:    in.PrimaryEntityID,
			LastMessageSummary: in.Summary,
			Type:               tt,
			SystemTitle:        systemTitle,
			UserTitle:          in.UserTitle,
			Origin:             to,
		})
		if err != nil {
			return errors.Trace(err)
		}
		if err := dl.AddThreadMembers(ctx, threadID, memberEntityIDs); err != nil {
			return errors.Trace(err)
		}
		if in.FromEntityID != "" {
			if err := dl.UpdateThreadEntity(ctx, threadID, in.FromEntityID, nil); err != nil {
				return errors.Trace(err)
			}
		}
		return nil
	}); err != nil {
		return nil, grpcErrorf(codes.Internal, errors.Trace(err).Error())
	}
	threads, err := s.dal.Threads(ctx, []models.ThreadID{threadID})
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(threads) == 0 {
		return nil, errors.Trace(fmt.Errorf("thread with id %s just created not found", threadID))
	}
	if err := s.updateSavedQueriesAddThread(ctx, threads[0], memberEntityIDs); err != nil {
		golog.Errorf("Failed to updated saved query when adding thread: %s", threadID)
	}
	th, err := transformThreadToResponse(threads[0], false)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.CreateEmptyThreadResponse{
		Thread: th,
	}, nil
}

// CreateThread create a new thread with an initial message
func (s *threadsServer) CreateThread(ctx context.Context, in *threading.CreateThreadRequest) (*threading.CreateThreadResponse, error) {
	// TODO: return proper error responses for invalid request
	switch in.Type {
	case threading.THREAD_TYPE_EXTERNAL, threading.THREAD_TYPE_TEAM:
	default:
		return nil, grpcErrorf(codes.InvalidArgument, fmt.Sprintf("Type '%s' not allowed for CreateThread", in.Type.String()))
	}
	if in.OrganizationID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "OrganizationID is required")
	}
	if in.FromEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "FromEntityID is required")
	}
	if in.Type == threading.THREAD_TYPE_EXTERNAL && in.SystemTitle == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "SystemTitle is required")
	}
	if id, ok := validateEntityIDs(in.MemberEntityIDs); !ok {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid entity ID %s in members list", id)
	}

	tt, err := transformThreadTypeFromRequest(in.Type)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid thread type")
	}
	to, err := transformThreadOriginFromRequest(in.Origin)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid thread origin")
	}
	if in.Summary == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "Summary is required")
	}
	in.Summary = textutil.TruncateUTF8(in.Summary, maxSummaryLength)
	if in.MessageTitle != "" {
		if _, err := bml.Parse(in.MessageTitle); err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, fmt.Sprintf("MessageTitle is invalid format: %s", err.Error()))
		}
	}
	var textRefs []*models.Reference
	in.Text, textRefs, err = parseRefsAndNormalize(in.Text)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, fmt.Sprintf("Text is invalid format: %s", errors.Cause(err).Error()))
	}

	memberEntityIDs, err := memberEntityIDsForNewThread(in.Type, in.OrganizationID, in.FromEntityID, in.MemberEntityIDs)
	if err != nil {
		return nil, err
	}

	var systemTitle string
	switch in.Type {
	case threading.THREAD_TYPE_TEAM:
		systemTitle, err = s.teamThreadSystemTitle(ctx, in.OrganizationID, memberEntityIDs)
		if err != nil {
			return nil, errors.Trace(err)
		}
	case threading.THREAD_TYPE_SETUP:
		systemTitle = setupThreadTitle
	case threading.THREAD_TYPE_EXTERNAL:
		systemTitle = in.SystemTitle
	}

	// TODO: validate any attachments
	var threadID models.ThreadID
	var item *models.ThreadItem
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		var err error
		threadID, err = dl.CreateThread(ctx, &models.Thread{
			OrganizationID:  in.OrganizationID,
			PrimaryEntityID: in.FromEntityID,
			Type:            tt,
			SystemTitle:     systemTitle,
			UserTitle:       in.UserTitle,
			Origin:          to,
		})
		if err != nil {
			return errors.Trace(err)
		}
		if err := dl.AddThreadMembers(ctx, threadID, memberEntityIDs); err != nil {
			return errors.Trace(err)
		}
		if err := dl.UpdateThreadEntity(ctx, threadID, in.FromEntityID, nil); err != nil {
			return errors.Trace(err)
		}

		req := &dal.PostMessageRequest{
			ThreadID:     threadID,
			FromEntityID: in.FromEntityID,
			Internal:     in.Internal,
			Text:         in.Text,
			Title:        in.MessageTitle,
			TextRefs:     textRefs,
			Summary:      in.Summary,
		}
		if in.Source != nil {
			req.Source, err = transformEndpointFromRequest(in.Source)
			if err != nil {
				return errors.Trace(err)
			}
		}
		req.Attachments, err = transformAttachmentsFromRequest(in.Attachments)
		if err != nil {
			return errors.Trace(err)
		}
		for _, dc := range in.Destinations {
			d, err := transformEndpointFromRequest(dc)
			if err != nil {
				return errors.Trace(err)
			}
			req.Destinations = append(req.Destinations, d)
		}
		item, err = dl.PostMessage(ctx, req)
		if err != nil {
			return errors.Trace(err)
		}
		// Update unread reference status for anyone mentioned
		for _, r := range textRefs {
			if err := dl.UpdateThreadEntity(ctx, threadID, r.ID, &dal.ThreadEntityUpdate{
				LastReferenced: &item.Created,
			}); err != nil {
				return errors.Trace(err)
			}
		}
		return nil
	}); err != nil {
		return nil, errors.Trace(err)
	}
	threads, err := s.dal.Threads(ctx, []models.ThreadID{threadID})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(threads) == 0 {
		return nil, errors.Errorf("thread %s just created not found", threadID)
	}
	thread := threads[0]
	if err := s.updateSavedQueriesAddThread(ctx, thread, memberEntityIDs); err != nil {
		golog.Errorf("Failed to updated saved query when adding thread: %s", threadID)
	}
	th, err := transformThreadToResponse(thread, !in.Internal)
	if err != nil {
		return nil, errors.Trace(err)
	}
	it, err := transformThreadItemToResponse(item, thread.OrganizationID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	s.publishMessage(ctx, in.OrganizationID, in.FromEntityID, threadID, it, in.UUID)
	s.notifyMembersOfPublishMessage(ctx, thread.OrganizationID, models.EmptySavedQueryID(), thread, item, in.FromEntityID)
	return &threading.CreateThreadResponse{
		ThreadID:   threadID.String(),
		ThreadItem: it,
		Thread:     th,
	}, nil
}

func (s *threadsServer) CreateLinkedThreads(ctx context.Context, in *threading.CreateLinkedThreadsRequest) (*threading.CreateLinkedThreadsResponse, error) {
	if in.Type != threading.THREAD_TYPE_SUPPORT {
		return nil, grpcErrorf(codes.InvalidArgument, "Only threads of type SUPPORT are allowed for linked threads")
	}
	if in.Organization1ID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "Organization1ID is required")
	}
	if in.Organization2ID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "Organization2ID is required")
	}
	if in.PrimaryEntity1ID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "PrimaryEntity1ID is required")
	}
	if in.PrimaryEntity2ID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "PrimaryEntity2ID is required")
	}
	tt, err := transformThreadTypeFromRequest(in.Type)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid thread type")
	}
	if in.Summary == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "Summary is required")
	}
	in.Summary = textutil.TruncateUTF8(in.Summary, maxSummaryLength)
	if in.MessageTitle != "" {
		if _, err := bml.Parse(in.MessageTitle); err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, fmt.Sprintf("MessageTitle is invalid format: %s", err.Error()))
		}
	}
	var textRefs []*models.Reference
	in.Text, textRefs, err = parseRefsAndNormalize(in.Text)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, fmt.Sprintf("Text is invalid format: %s", errors.Cause(err).Error()))
	}

	var thread1ID, thread2ID models.ThreadID
	err = s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		var err error
		thread1ID, err = dl.CreateThread(ctx, &models.Thread{
			OrganizationID:     in.Organization1ID,
			PrimaryEntityID:    in.PrimaryEntity1ID,
			LastMessageSummary: in.Summary,
			Type:               tt,
			SystemTitle:        in.SystemTitle1,
		})
		if err != nil {
			return errors.Trace(err)
		}
		thread2ID, err = dl.CreateThread(ctx, &models.Thread{
			OrganizationID:     in.Organization2ID,
			PrimaryEntityID:    in.PrimaryEntity2ID,
			LastMessageSummary: in.Summary,
			Type:               tt,
			SystemTitle:        in.SystemTitle2,
		})
		if err != nil {
			return errors.Trace(err)
		}
		if err := dl.AddThreadMembers(ctx, thread1ID, []string{in.Organization1ID}); err != nil {
			return errors.Trace(err)
		}
		if err := dl.AddThreadMembers(ctx, thread2ID, []string{in.Organization2ID}); err != nil {
			return errors.Trace(err)
		}
		if err := dl.CreateThreadLink(ctx, &dal.ThreadLink{
			ThreadID:      thread1ID,
			PrependSender: in.PrependSenderThread1,
		}, &dal.ThreadLink{
			ThreadID:      thread2ID,
			PrependSender: in.PrependSenderThread2,
		}); err != nil {
			return errors.Trace(err)
		}
		if in.Text != "" {
			_, err = dl.PostMessage(ctx, &dal.PostMessageRequest{
				ThreadID:     thread1ID,
				FromEntityID: in.PrimaryEntity1ID,
				Internal:     false,
				Text:         in.Text,
				Title:        in.MessageTitle,
				TextRefs:     textRefs,
				Summary:      in.Summary,
			})
			if err != nil {
				return errors.Trace(err)
			}
			_, err = dl.PostMessage(ctx, &dal.PostMessageRequest{
				ThreadID:     thread2ID,
				FromEntityID: in.PrimaryEntity2ID,
				Internal:     false,
				Text:         in.Text,
				Title:        in.MessageTitle,
				TextRefs:     textRefs,
				Summary:      in.Summary,
			})
			if err != nil {
				return errors.Trace(err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	threads, err := s.dal.Threads(ctx, []models.ThreadID{thread1ID, thread2ID})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(threads) != 2 {
		return nil, errors.Errorf("expected 2 threads but got %d", len(threads))
	}
	// Order the threads as expected
	if threads[0].ID != thread1ID {
		threads[0], threads[1] = threads[1], threads[0]
	}

	if err := s.updateSavedQueriesAddThread(ctx, threads[0], []string{in.Organization1ID}); err != nil {
		golog.Errorf("Failed to updated saved query when adding thread: %s", threads[0].ID)
	}
	if err := s.updateSavedQueriesAddThread(ctx, threads[1], []string{in.Organization2ID}); err != nil {
		golog.Errorf("Failed to updated saved query when adding thread: %s", threads[1].ID)
	}

	th1, err := transformThreadToResponse(threads[0], false)
	if err != nil {
		return nil, errors.Trace(err)
	}
	th2, err := transformThreadToResponse(threads[1], false)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.CreateLinkedThreadsResponse{
		Thread1: th1,
		Thread2: th2,
	}, nil
}

// DeleteMessage deletes a message from a thread
func (s *threadsServer) DeleteMessage(context.Context, *threading.DeleteMessageRequest) (*threading.DeleteMessageResponse, error) {
	return nil, grpcErrorf(codes.Unimplemented, "DeleteMessage not implemented")
}

// DeleteThread deletes a thread
func (s *threadsServer) DeleteThread(ctx context.Context, in *threading.DeleteThreadRequest) (*threading.DeleteThreadResponse, error) {
	if in.ActorEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "ActorEntityID is required")
	}
	if in.ThreadID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "ThreadID is required")
	}
	threadID, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid ThreadID")
	}

	// If we can't find the thread then just return success
	threads, err := s.dal.Threads(ctx, []models.ThreadID{threadID})
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(threads) == 0 {
		return &threading.DeleteThreadResponse{}, nil
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	thread := threads[0]
	if thread.PrimaryEntityID != "" {
		// Get the primary entity on the thread first and determine if we need to delete it if it's external
		resp, err := s.directoryClient.LookupEntities(ctx, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: thread.PrimaryEntityID,
			},
		})
		if err != nil && grpc.Code(err) != codes.NotFound {
			return nil, errors.Trace(err)
		}

		if resp != nil &&
			len(resp.Entities) != 0 &&
			resp.Entities[0].Type == directory.EntityType_EXTERNAL &&
			resp.Entities[0].Status != directory.EntityStatus_DELETED {
			if _, err := s.directoryClient.DeleteEntity(ctx, &directory.DeleteEntityRequest{
				EntityID: resp.Entities[0].ID,
			}); err != nil {
				return nil, errors.Trace(err)
			}
		}
	}
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		if err := s.dal.DeleteThread(ctx, threadID); err != nil {
			return errors.Trace(err)
		}
		return errors.Trace(s.dal.RecordThreadEvent(ctx, threadID, in.ActorEntityID, models.ThreadEventDelete))
	}); err != nil {
		return nil, errors.Trace(err)
	}
	if err := s.updateSavedQueriesRemoveThread(ctx, thread.ID); err != nil {
		golog.Errorf("Failed to remove thread %s from saved queries: %s", thread.ID, err)
	}
	return &threading.DeleteThreadResponse{}, nil
}

func (s *threadsServer) LinkedThread(ctx context.Context, in *threading.LinkedThreadRequest) (*threading.LinkedThreadResponse, error) {
	if in.ThreadID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "ThreadID is required")
	}
	threadID, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid ThreadID")
	}
	thread, prependSender, err := s.dal.LinkedThread(ctx, threadID)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpcErrorf(codes.NotFound, "Linked thread not found")
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	th, err := transformThreadToResponse(thread, false)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.LinkedThreadResponse{
		Thread:        th,
		PrependSender: prependSender,
	}, nil
}

// MarkThreadsAsRead marks all posts in a thread as read by an entity
func (s *threadsServer) MarkThreadsAsRead(ctx context.Context, in *threading.MarkThreadsAsReadRequest) (*threading.MarkThreadsAsReadResponse, error) {
	if len(in.ThreadWatermarks) == 0 {
		return nil, grpcErrorf(codes.InvalidArgument, "ThreadWatermarks required")
	}

	if in.EntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "EntityID is required")
	}

	// Make a map of all the entities memberships and itself to be used for validating memberships to the threads
	entities, err := s.entityAndMemberships(ctx, in.EntityID, []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT})
	if err != nil {
		return nil, errors.Trace(err)
	}
	externalEntity := true
	memberships := make(map[string]struct{}, len(entities))
	for _, e := range entities {
		memberships[e.ID] = struct{}{}
		if e.ID == in.EntityID {
			externalEntity = isExternalEntity(e)
		}
	}

	// Fetch all the threads in bulk
	threadIDs := make([]models.ThreadID, len(in.ThreadWatermarks))
	for i, w := range in.ThreadWatermarks {
		threadID, err := models.ParseThreadID(w.ThreadID)
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "Invalid ThreadID")
		}
		threadIDs[i] = threadID
	}
	threads, threadEntities, err := s.dal.ThreadsWithEntity(ctx, in.EntityID, threadIDs)
	if err != nil {
		return nil, errors.Trace(err)
	}

	sqs, err := s.dal.SavedQueries(ctx, in.EntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var addIndex, removeIndex []*dal.SavedQueryThread

	currentTime := s.clk.Now()
	for i, watermark := range in.ThreadWatermarks {
		thread := threads[i]
		threadEntity := threadEntities[i]

		readTime := currentTime
		// only use the last message timestamp if one is provided by the client or it is in the past but after the reference date of the product launch
		if watermark.LastMessageTimestamp != 0 && watermark.LastMessageTimestamp < uint64(readTime.Unix()) && watermark.LastMessageTimestamp > uint64(baymaxLaunchDate.Unix()) {
			readTime = time.Unix(int64(watermark.LastMessageTimestamp), 0)
		}

		// If unread matches expected state then don't do anything
		currentUnread := isUnread(thread, threadEntity, externalEntity)
		newUnread := isUnread(thread, &models.ThreadEntity{LastViewed: &readTime}, externalEntity)
		if currentUnread == newUnread {
			continue
		}

		// If the new read time is the last as the last message timestamp to the second then
		// actually use the last message timestamp to avoid any issues with time resolution.
		// Also, avoid storing a future time to make sure we see future posts.
		lastTimestamp := thread.LastMessageTimestamp
		if externalEntity {
			lastTimestamp = thread.LastExternalMessageTimestamp
		}
		if !readTime.Before(lastTimestamp.Truncate(time.Second)) {
			readTime = lastTimestamp
		}

		// Fetch members for thread to make sure the provided entity is a member
		members, err := s.membersForThread(ctx, thread.ID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		isMember := false
		for _, m := range members {
			if _, ok := memberships[m.EntityID]; ok {
				isMember = true
				break
			}
		}
		if !isMember {
			golog.Errorf("Entity '%s' trying to mark as a read a thread '%s' it is not a member of", in.EntityID, thread.ID)
			continue
		}

		// If seen then create read receipts, otherwise just update the watermark
		if in.Seen {
			if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
				threadEntities, err := dl.ThreadEntities(ctx, []models.ThreadID{thread.ID}, in.EntityID)
				if err != nil {
					return errors.Trace(err)
				}
				lastViewed := time.Unix(0, 0)
				if te := threadEntities[thread.ID.String()]; te != nil && te.LastViewed != nil {
					lastViewed = *te.LastViewed
				}

				// Update our timestamp or create one if it isn't already there
				if err := dl.UpdateThreadEntity(ctx, thread.ID, in.EntityID, &dal.ThreadEntityUpdate{LastViewed: &readTime}); err != nil {
					return errors.Trace(err)
				}

				threadItemIDs, err := dl.ThreadItemIDsCreatedAfter(ctx, thread.ID, lastViewed)
				if err != nil {
					return errors.Trace(err)
				}

				tivds := make([]*models.ThreadItemViewDetails, len(threadItemIDs))
				for i, tiid := range threadItemIDs {
					tivds[i] = &models.ThreadItemViewDetails{
						ThreadItemID:  tiid,
						ActorEntityID: in.EntityID,
						ViewTime:      &currentTime,
					}
				}
				return errors.Trace(dl.CreateThreadItemViewDetails(ctx, tivds))
			}); err != nil {
				return nil, errors.Trace(err)
			}
		} else {
			if err := s.dal.UpdateThreadEntity(ctx, thread.ID, in.EntityID, &dal.ThreadEntityUpdate{LastViewed: &readTime}); err != nil {
				return nil, errors.Trace(err)
			}
		}

		if threadEntity == nil {
			threadEntity = &models.ThreadEntity{EntityID: in.EntityID, ThreadID: thread.ID}
		}
		threadEntity.LastViewed = &readTime
		for _, sq := range sqs {
			if ok, err := threadMatchesQuery(sq.Query, thread, threadEntity, externalEntity); err != nil {
				golog.Errorf("Error matching thread %s against saved query %s: %s", thread.ID, sq.ID, err)
			} else if ok {
				timestamp := thread.LastMessageTimestamp
				if externalEntity {
					timestamp = thread.LastExternalMessageTimestamp
				}
				addIndex = append(addIndex, &dal.SavedQueryThread{
					SavedQueryID: sq.ID,
					ThreadID:     thread.ID,
					Unread:       isUnread(thread, threadEntity, externalEntity),
					Timestamp:    timestamp,
				})
			} else {
				removeIndex = append(removeIndex, &dal.SavedQueryThread{SavedQueryID: sq.ID, ThreadID: thread.ID})
			}
		}
	}

	p := conc.NewParallel()
	p.Go(func() error {
		return errors.Trace(s.dal.AddItemsToSavedQueryIndex(ctx, addIndex))
	})
	p.Go(func() error {
		return errors.Trace(s.dal.RemoveItemsFromSavedQueryIndex(ctx, removeIndex))
	})
	if err := p.Wait(); err != nil {
		return nil, errors.Trace(err)
	}

	return &threading.MarkThreadsAsReadResponse{}, nil
}

// PostMessage posts a message into a specified thread
func (s *threadsServer) PostMessage(ctx context.Context, in *threading.PostMessageRequest) (*threading.PostMessageResponse, error) {
	// TODO: return proper error responses for invalid request
	if in.ThreadID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "ThreadID is required")
	}
	threadID, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid ThreadID")
	}
	if in.FromEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "FromEntityID is required")
	}
	if in.Summary == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "Summary is required")
	}
	in.Summary = textutil.TruncateUTF8(in.Summary, maxSummaryLength)
	if in.Title != "" {
		if _, err := bml.Parse(in.Title); err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "Title is invalid format: %s", err.Error())
		}
	}
	var textRefs []*models.Reference
	in.Text, textRefs, err = parseRefsAndNormalize(in.Text)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Text is invalid format: %s", errors.Cause(err).Error())
	}

	threads, err := s.dal.Threads(ctx, []models.ThreadID{threadID})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(threads) == 0 {
		return nil, grpcErrorf(codes.NotFound, "Thread %s not found", threadID)
	}
	thread := threads[0]
	prePostLastMessageTimestamp := thread.LastMessageTimestamp

	linkedThread, prependSender, err := s.dal.LinkedThread(ctx, threadID)
	if err != nil && errors.Cause(err) != dal.ErrNotFound {
		return nil, errors.Trace(err)
	}

	var item *models.ThreadItem
	var linkedItem *models.ThreadItem

	// TODO: validate any attachments
	attachments, err := transformAttachmentsFromRequest(in.Attachments)
	if err != nil {
		return nil, errors.Trace(err)
	}
	mediaIDs := mediaIDsFromAttachments(attachments)
	if len(mediaIDs) > 0 {
		// Before posting the actual message, map all the attached media to the thread
		// Failure scenarios:
		// 1. This call succeeds and the post fails. The media is now mapped to the thread which should still allow a repost.
		// 2. This call fails. The media is still mapped to the caller
		_, err = s.mediaClient.ClaimMedia(ctx, &media.ClaimMediaRequest{
			MediaIDs:  mediaIDs,
			OwnerType: media.MediaOwnerType_THREAD,
			OwnerID:   threadID.String(),
		})
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	for _, pID := range paymentsIDsFromAttachments(attachments) {
		// This call should be idempotent as long as the payment request is just being submitted
		if _, err := s.paymentsClient.SubmitPayment(ctx, &payments.SubmitPaymentRequest{
			PaymentID: pID,
			ThreadID:  threadID.String(),
		}); err != nil {
			return nil, errors.Trace(err)
		}
	}
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		req := &dal.PostMessageRequest{
			ThreadID:     threadID,
			FromEntityID: in.FromEntityID,
			Internal:     in.Internal,
			Text:         in.Text,
			Title:        in.Title,
			TextRefs:     textRefs,
			Summary:      in.Summary,
			Attachments:  attachments,
		}
		if in.Source != nil {
			req.Source, err = transformEndpointFromRequest(in.Source)
			if err != nil {
				return errors.Trace(err)
			}
		}
		for _, dc := range in.Destinations {
			d, err := transformEndpointFromRequest(dc)
			if err != nil {
				return errors.Trace(err)
			}
			req.Destinations = append(req.Destinations, d)
		}
		item, err = dl.PostMessage(ctx, req)
		if err != nil {
			return errors.Trace(err)
		}

		now := s.clk.Now()

		// Update unread reference status for anyone mentioned
		for _, r := range textRefs {
			if err := dl.UpdateThreadEntity(ctx, threadID, r.ID, &dal.ThreadEntityUpdate{
				LastReferenced: &now,
			}); err != nil {
				return errors.Trace(err)
			}
		}

		// Lock our membership row while doing this since we might update it
		tes, err := dl.ThreadEntities(ctx, []models.ThreadID{threadID}, in.FromEntityID, dal.ForUpdate)
		if err != nil {
			return errors.Trace(err)
		}

		var teUpdate *dal.ThreadEntityUpdate
		if len(tes) > 0 {
			// Update the last read timestamp on the membership if all other messages have been read
			lastViewed := tes[threadID.String()].LastViewed
			if lastViewed == nil {
				lastViewed = &thread.Created
			}
			if lastViewed.Unix() >= prePostLastMessageTimestamp.Unix() {
				teUpdate = &dal.ThreadEntityUpdate{
					LastViewed: &now,
				}
			}
		}
		if err := dl.UpdateThreadEntity(ctx, threadID, in.FromEntityID, teUpdate); err != nil {
			return errors.Trace(err)
		}

		// Also post in linked thread if there is one
		if linkedThread != nil && !in.Internal {
			// TODO: should use primary entity name here
			summary, err := models.SummaryFromText("Spruce: " + in.Text)
			if err != nil {
				return errors.Trace(err)
			}
			text := in.Text
			if prependSender {
				resp, err := s.directoryClient.LookupEntities(ctx, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: in.FromEntityID,
					},
					RequestedInformation: &directory.RequestedInformation{
						Depth: 0,
					},
				})
				if err != nil {
					golog.Errorf("Unable to lookup entity for id %s: %s", in.FromEntityID, err.Error())
				} else if len(resp.Entities) != 1 {
					golog.Errorf("Expected 1 entity for id %s but got %d back", in.FromEntityID, len(resp.Entities))
				} else if resp.Entities[0].Type == directory.EntityType_INTERNAL {
					validBML, err := bml.BML{resp.Entities[0].Info.DisplayName}.Format()
					if err != nil {
						golog.Errorf("Unable to escape the display name %s:%s", resp.Entities[0].Info.DisplayName, err.Error())
					} else {
						text = validBML + ": " + text
					}
				}
			}
			req := &dal.PostMessageRequest{
				ThreadID:     linkedThread.ID,
				FromEntityID: linkedThread.PrimaryEntityID,
				Text:         text,
				Title:        in.Title,
				TextRefs:     textRefs,
				Summary:      summary,
				Attachments:  attachments,
			}
			if in.Source != nil {
				req.Source, err = transformEndpointFromRequest(in.Source)
				if err != nil {
					return errors.Trace(err)
				}
			}
			linkedItem, err = dl.PostMessage(ctx, req)
			if err != nil {
				return errors.Trace(err)
			}
		}

		return nil
	}); err != nil {
		return nil, errors.Trace(err)
	}

	threads, err = s.dal.Threads(ctx, []models.ThreadID{threadID})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(threads) == 0 {
		return nil, errors.Errorf("Thread %s that was just created was not found", threadID)
	}
	thread = threads[0]
	if err := s.updateSavedQueriesForThread(ctx, thread); err != nil {
		golog.Errorf("Failed to updated saved query for thread %s: %s", thread.ID, err)
	}

	th, err := transformThreadToResponse(thread, !in.Internal)
	if err != nil {
		return nil, errors.Trace(err)
	}
	it, err := transformThreadItemToResponse(item, thread.OrganizationID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	s.publishMessage(ctx, thread.OrganizationID, thread.PrimaryEntityID, threadID, it, in.UUID)
	if !in.DontNotify {
		s.notifyMembersOfPublishMessage(ctx, thread.OrganizationID, models.EmptySavedQueryID(), thread, item, in.FromEntityID)
	}

	if linkedItem != nil {
		it2, err := transformThreadItemToResponse(linkedItem, linkedThread.OrganizationID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		s.publishMessage(ctx, linkedThread.OrganizationID, linkedThread.PrimaryEntityID, linkedThread.ID, it2, "")
		if !in.DontNotify {
			s.notifyMembersOfPublishMessage(ctx, linkedThread.OrganizationID, models.EmptySavedQueryID(), linkedThread, linkedItem, linkedItem.ActorEntityID)
		}
	}
	return &threading.PostMessageResponse{
		Item:   it,
		Thread: th,
	}, nil
}

// QueryThreads queries the list of threads
func (s *threadsServer) QueryThreads(ctx context.Context, in *threading.QueryThreadsRequest) (*threading.QueryThreadsResponse, error) {
	if in.ViewerEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "ViewerEntityID required")
	}

	var sq *models.SavedQuery
	var query *models.Query
	switch in.Type {
	case threading.QUERY_THREADS_TYPE_ADHOC:
		if in.GetQuery() == nil {
			return nil, grpcErrorf(codes.InvalidArgument, "Query quired for ADHOC queries")
		}
		var err error
		query, err = transformQueryFromRequest(in.GetQuery())
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "Query is not valid: %s", err)
		}
	case threading.QUERY_THREADS_TYPE_SAVED:
		sqID, err := models.ParseSavedQueryID(in.GetSavedQueryID())
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "Saved query ID %s is not valid", in.GetSavedQueryID())
		}
		sq, err = s.dal.SavedQuery(ctx, sqID)
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "Saved query %s not found", sqID)
		} else if err != nil {
			return nil, errors.Trace(err)
		}
		if sq.EntityID != in.ViewerEntityID {
			return nil, grpcErrorf(codes.InvalidArgument, "Saved query %s not owned by %s", sq.ID, in.ViewerEntityID)
		}
		query = sq.Query
	case threading.QUERY_THREADS_TYPE_ALL_FOR_VIEWER:
		query = &models.Query{}
	default:
		return nil, grpcErrorf(codes.InvalidArgument, "Unknown query type %s", in.Type)
	}

	d := dal.FromStart
	if in.Iterator.Direction == threading.ITERATOR_DIRECTION_FROM_END {
		d = dal.FromEnd
	}

	memberEntities, err := s.entityAndMemberships(ctx, in.ViewerEntityID, []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT})
	if err != nil {
		return nil, errors.Trace(err)
	}
	forExternal := true
	var selfEntityID string
	memberEntityIDs := make([]string, len(memberEntities))
	for i, e := range memberEntities {
		memberEntityIDs[i] = e.ID
		if e.ID == in.ViewerEntityID {
			forExternal = isExternalEntity(e)
			selfEntityID = e.ID
		}
	}
	// For external (patient) entities only use the entity itself
	if forExternal {
		memberEntityIDs = []string{selfEntityID}
	}

	it := &dal.Iterator{
		StartCursor: in.Iterator.StartCursor,
		EndCursor:   in.Iterator.EndCursor,
		Direction:   d,
		Count:       int(in.Iterator.Count),
	}
	var tc *dal.ThreadConnection
	if sq == nil {
		tc, err = s.dal.IterateThreads(ctx, query, memberEntityIDs, in.ViewerEntityID, forExternal, it)
	} else {
		tc, err = s.dal.IterateThreadsInSavedQuery(ctx, sq.ID, in.ViewerEntityID, it)
	}
	if e, ok := errors.Cause(err).(dal.ErrInvalidIterator); ok {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid iterator: %s", e)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	res := &threading.QueryThreadsResponse{
		Edges:   make([]*threading.ThreadEdge, 0, len(tc.Edges)),
		HasMore: tc.HasMore,
	}

	log := golog.Context("viewerEntityID", in.ViewerEntityID)
	if sq != nil {
		log = log.Context("savedQueryID", sq.ID)
	}
	for _, e := range tc.Edges {
		// Sanity check as either the saved query could be out of date or the adhoc sql query doesn't match our internal version
		if ok, err := threadMatchesQuery(query, e.Thread, e.ThreadEntity, forExternal); err != nil {
			log.Errorf("Failed to match thread %s against query %s: %s", e.Thread.ID, query.String(), err)
		} else if !ok {
			log.Errorf("Thread query %s returned non-matching thread %s", query.String(), e.Thread.ID)
			continue
		}

		th, err := transformThreadToResponse(e.Thread, forExternal)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if in.ViewerEntityID != "" && th.MessageCount != 0 {
			te := e.ThreadEntity
			th.Unread = isUnread(e.Thread, te, forExternal)
			th.UnreadReference = hasUnreadReference(te)
		}
		res.Edges = append(res.Edges, &threading.ThreadEdge{
			Thread: th,
			Cursor: e.Cursor,
		})
	}

	if sq != nil {
		res.Total = uint32(sq.Total)
		res.TotalType = threading.VALUE_TYPE_EXACT
	} else {
		res.Total = uint32(len(tc.Edges))
		res.TotalType = threading.VALUE_TYPE_UNKNOWN
	}
	return res, nil
}

// SavedQuery returns a single saved query by ID
func (s *threadsServer) SavedQuery(ctx context.Context, in *threading.SavedQueryRequest) (*threading.SavedQueryResponse, error) {
	id, err := models.ParseSavedQueryID(in.SavedQueryID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid SavedQueryID")
	}
	query, err := s.dal.SavedQuery(ctx, id)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpcErrorf(codes.NotFound, "Saved query not found")
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	sq, err := transformSavedQueryToResponse(query)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.SavedQueryResponse{
		SavedQuery: sq,
	}, nil
}

// SavedQueries returns the list of saved queries for an org / entity pair
func (s *threadsServer) SavedQueries(ctx context.Context, in *threading.SavedQueriesRequest) (*threading.SavedQueriesResponse, error) {
	queries, err := s.dal.SavedQueries(ctx, in.EntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res := &threading.SavedQueriesResponse{
		SavedQueries: make([]*threading.SavedQuery, len(queries)),
	}
	for i, q := range queries {
		sq, err := transformSavedQueryToResponse(q)
		if err != nil {
			return nil, errors.Trace(fmt.Errorf("Failed to transform saved query: %s", err))
		}
		res.SavedQueries[i] = sq
	}
	return res, nil
}

// Thread looks up and returns a single thread by ID
func (s *threadsServer) Thread(ctx context.Context, in *threading.ThreadRequest) (*threading.ThreadResponse, error) {
	golog.Debugf("Querying for thread %s", in.ThreadID)
	tid, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid ThreadID")
	}

	forExternal, err := s.forExternalViewer(ctx, in.ViewerEntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	threads, err := s.dal.Threads(ctx, []models.ThreadID{tid})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(threads) == 0 {
		return nil, grpcErrorf(codes.NotFound, "Thread not found")
	}
	thread := threads[0]

	th, err := transformThreadToResponse(thread, forExternal)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if in.ViewerEntityID != "" {
		golog.Debugf("Populating viewer information for (entity_id, thread) (%s,%s)", in.ViewerEntityID, th.ID)
		ts, err := s.hydrateThreadForViewer(ctx, []*threading.Thread{th}, in.ViewerEntityID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if len(ts) == 0 {
			return nil, grpcErrorf(codes.NotFound, "Thread not found")
		}
		// TODO: for now can't require the viewer since the graphql service requests the thread to get the org ID before it can know the entity viewing
		// } else if th.Type == threading.THREAD_TYPE_TEAM {
		// 	// Require a viewer entity for private threads
		// 	return nil, grpcErrorf(codes.NotFound, "Thread not found")
	} else {
		golog.Debugf("No viewer entity information for thread %s", in.ThreadID)
	}
	return &threading.ThreadResponse{
		Thread: th,
	}, nil
}

func (s *threadsServer) Threads(ctx context.Context, in *threading.ThreadsRequest) (*threading.ThreadsResponse, error) {
	forExternal, err := s.forExternalViewer(ctx, in.ViewerEntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	threadIDs := make([]models.ThreadID, len(in.ThreadIDs))
	for i, threadID := range in.ThreadIDs {
		threadIDs[i], err = models.ParseThreadID(threadID)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	threads, err := s.dal.Threads(ctx, threadIDs)
	if err != nil {
		return nil, errors.Trace(err)
	}

	threadsInResponse := make([]*threading.Thread, len(threadIDs))
	for i, thread := range threads {
		th, err := transformThreadToResponse(thread, forExternal)
		if err != nil {
			return nil, errors.Trace(err)
		}
		threadsInResponse[i] = th
	}

	if in.ViewerEntityID != "" {
		ts, err := s.hydrateThreadForViewer(ctx, threadsInResponse, in.ViewerEntityID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if len(ts) == 0 {
			return nil, grpcErrorf(codes.NotFound, "Thread not found")
		}
		threadsInResponse = ts
	}

	return &threading.ThreadsResponse{
		Threads: threadsInResponse,
	}, nil
}

// ThreadItem looks up and returns a single thread item by ID
func (s *threadsServer) ThreadItem(ctx context.Context, in *threading.ThreadItemRequest) (*threading.ThreadItemResponse, error) {
	tid, err := models.ParseThreadItemID(in.ItemID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid ItemID")
	}

	item, err := s.dal.ThreadItem(ctx, tid)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpcErrorf(codes.NotFound, "Thread item not found")
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	threads, err := s.dal.Threads(ctx, []models.ThreadID{item.ThreadID})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(threads) == 0 {
		return nil, grpcErrorf(codes.NotFound, "Thread %s not found", tid)
	}
	th := threads[0]

	ti, err := transformThreadItemToResponse(item, th.OrganizationID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.ThreadItemResponse{
		Item: ti,
	}, nil
}

// ThreadsForMember looks up a list of threads by entity membership
func (s *threadsServer) ThreadsForMember(ctx context.Context, in *threading.ThreadsForMemberRequest) (*threading.ThreadsForMemberResponse, error) {
	threads, err := s.dal.ThreadsForMember(ctx, in.EntityID, in.PrimaryOnly)
	if err != nil {
		return nil, errors.Trace(err)
	}

	forExternal, err := s.forExternalViewer(ctx, in.EntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	res := &threading.ThreadsForMemberResponse{
		Threads: make([]*threading.Thread, len(threads)),
	}
	for i, t := range threads {
		th, err := transformThreadToResponse(t, forExternal)
		if err != nil {
			return nil, errors.Trace(err)
		}
		res.Threads[i] = th
	}
	res.Threads, err = s.hydrateThreadForViewer(ctx, res.Threads, in.EntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return res, nil
}

// ThreadItems returns the items (messages or events) in a thread
func (s *threadsServer) ThreadItems(ctx context.Context, in *threading.ThreadItemsRequest) (*threading.ThreadItemsResponse, error) {
	tid, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid ThreadID")
	}

	threads, err := s.dal.Threads(ctx, []models.ThreadID{tid})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(threads) == 0 {
		return nil, grpcErrorf(codes.NotFound, "Thread %s not found", tid)
	}
	th := threads[0]

	forExternal, err := s.forExternalViewer(ctx, in.ViewerEntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	d := dal.FromStart
	if in.Iterator.Direction == threading.ITERATOR_DIRECTION_FROM_END {
		d = dal.FromEnd
	}
	ir, err := s.dal.IterateThreadItems(ctx, tid, forExternal, &dal.Iterator{
		StartCursor: in.Iterator.StartCursor,
		EndCursor:   in.Iterator.EndCursor,
		Direction:   d,
		Count:       int(in.Iterator.Count),
	})
	if e, ok := errors.Cause(err).(dal.ErrInvalidIterator); ok {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid iterator: "+string(e))
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	res := &threading.ThreadItemsResponse{
		Edges:   make([]*threading.ThreadItemEdge, len(ir.Edges)),
		HasMore: ir.HasMore,
	}
	for i, e := range ir.Edges {
		it, err := transformThreadItemToResponse(e.Item, th.OrganizationID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		res.Edges[i] = &threading.ThreadItemEdge{
			Item:   it,
			Cursor: e.Cursor,
		}
	}
	return res, nil
}

// ThreadItemViewDetails returns the view details of a thread
func (s *threadsServer) ThreadItemViewDetails(ctx context.Context, in *threading.ThreadItemViewDetailsRequest) (*threading.ThreadItemViewDetailsResponse, error) {
	tiid, err := models.ParseThreadItemID(in.ItemID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid ThreadItemID")
	}

	tivds, err := s.dal.ThreadItemViewDetails(ctx, tiid)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ptivds := make([]*threading.ThreadItemViewDetails, len(tivds))
	for i, tivd := range tivds {
		ptivds[i] = &threading.ThreadItemViewDetails{
			ThreadItemID: tivd.ThreadItemID.String(),
			EntityID:     tivd.ActorEntityID,
			ViewTime:     uint64(tivd.ViewTime.Unix()),
		}
	}

	return &threading.ThreadItemViewDetailsResponse{
		ItemViewDetails: ptivds,
	}, nil
}

// ThreadMembers returns the members of a thread
func (s *threadsServer) ThreadMembers(ctx context.Context, in *threading.ThreadMembersRequest) (*threading.ThreadMembersResponse, error) {
	tid, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid ThreadItemID")
	}
	tes, err := s.dal.EntitiesForThread(ctx, tid)
	if err != nil {
		return nil, errors.Trace(err)
	}
	res := &threading.ThreadMembersResponse{
		Members:           make([]*threading.Member, 0, len(tes)),
		FollowerEntityIDs: make([]string, 0, len(tes)),
	}
	for _, te := range tes {
		if te.Member {
			res.Members = append(res.Members, &threading.Member{EntityID: te.EntityID})
		}
		if te.Following {
			res.FollowerEntityIDs = append(res.FollowerEntityIDs, te.EntityID)
		}
	}
	return res, nil
}

// UpdateSavedQuery updated a saved query
func (s *threadsServer) UpdateSavedQuery(ctx context.Context, in *threading.UpdateSavedQueryRequest) (*threading.UpdateSavedQueryResponse, error) {
	id, err := models.ParseSavedQueryID(in.SavedQueryID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid SavedQueryID")
	}
	rebuild := in.ForceRebuild
	update := &dal.SavedQueryUpdate{}
	if in.Title != "" {
		update.Title = &in.Title
	}
	if in.Ordinal > 0 {
		update.Ordinal = ptr.Int(int(in.Ordinal))
	}
	if in.Query != nil {
		update.Query, err = transformQueryFromRequest(in.Query)
		if err != nil {
			return nil, errors.Trace(err)
		}
		rebuild = true
	}
	if err := s.dal.UpdateSavedQuery(ctx, id, update); err != nil {
		return nil, errors.Trace(err)
	}
	if rebuild {
		sq, err := s.dal.SavedQuery(ctx, id)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if err := s.rebuildSavedQuery(ctx, sq); err != nil {
			return nil, errors.Trace(err)
		}
	}
	return &threading.UpdateSavedQueryResponse{}, nil
}

// UpdateThread update thread members and info
func (s *threadsServer) UpdateThread(ctx context.Context, in *threading.UpdateThreadRequest) (*threading.UpdateThreadResponse, error) {
	if in.ActorEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "ActorEntityID required")
	}
	if id, ok := validateEntityIDs(in.AddMemberEntityIDs); !ok {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid entity ID %s when adding members", id)
	}
	if id, ok := validateEntityIDs(in.RemoveMemberEntityIDs); !ok {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid entity ID %s when removing members", id)
	}
	if id, ok := validateEntityIDs(in.AddFollowerEntityIDs); !ok {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid entity ID %s when adding followers", id)
	}
	if id, ok := validateEntityIDs(in.RemoveFollowerEntityIDs); !ok {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid entity ID %s when removing followers", id)
	}

	tid, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid ThreadID")
	}

	threads, err := s.dal.Threads(ctx, []models.ThreadID{tid})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(threads) == 0 {
		return nil, grpcErrorf(codes.NotFound, "Thread not found")
	}
	thread := threads[0]

	// Verify authorization by checking actor is part of the memberslist
	// The acting entity can be the organization itself in which case it's allowed to modify any thread in the organization.
	if in.ActorEntityID != thread.OrganizationID {
		actorEntities, err := s.entityAndMemberships(ctx, in.ActorEntityID,
			[]directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_ORGANIZATION})
		if err != nil {
			return nil, errors.Trace(err)
		}
		if len(actorEntities) == 0 {
			return nil, grpcErrorf(codes.InvalidArgument, "No entities found for actor")
		}
		actorEntityIDs := make(map[string]struct{}, len(actorEntities))
		for _, e := range actorEntities {
			actorEntityIDs[e.ID] = struct{}{}
		}
		members, err := s.membersForThread(ctx, thread.ID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		isMember := false
		for _, m := range members {
			if _, ok := actorEntityIDs[m.EntityID]; ok {
				isMember = true
				break
			}
		}
		if !isMember {
			return nil, grpcErrorf(codes.PermissionDenied, "Entity is not a member of thread %s", thread.ID)
		}
	}

	// can only update system title for an external thread
	switch thread.Type {
	case models.ThreadTypeTeam:
		if in.SystemTitle != "" {
			return nil, grpcErrorf(codes.PermissionDenied, "Can only update system title for non-team thread")
		}
	default:
		if len(in.RemoveMemberEntityIDs) > 0 || len(in.AddMemberEntityIDs) > 0 {
			return nil, grpcErrorf(codes.PermissionDenied, "Can only update members for a team thread")
		}
	}

	var systemTitle string
	var memberIDs []string
	if len(in.RemoveMemberEntityIDs) != 0 || len(in.AddMemberEntityIDs) != 0 {
		// Prefetch what we assume will be the members list, we'll reconcile any issues of concurrent update inside the
		// transactions when the thread is locked.
		var removeMap map[string]struct{}
		if len(in.RemoveMemberEntityIDs) != 0 {
			removeMap = make(map[string]struct{}, len(in.RemoveMemberEntityIDs))
			for _, id := range in.RemoveMemberEntityIDs {
				removeMap[id] = struct{}{}
			}
		}
		entities, err := s.dal.EntitiesForThread(ctx, thread.ID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		memberIDs = make([]string, 0, len(entities))
		for _, e := range entities {
			if _, remove := removeMap[e.EntityID]; e.Member && !remove {
				memberIDs = append(memberIDs, e.EntityID)
			}
		}
		memberIDs = append(memberIDs, in.AddMemberEntityIDs...)
		systemTitle, err = s.teamThreadSystemTitle(ctx, thread.OrganizationID, memberIDs)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		update := &dal.ThreadUpdate{}
		if in.UserTitle != "" {
			update.UserTitle = &in.UserTitle
		}
		if in.SystemTitle != "" {
			update.SystemTitle = &in.SystemTitle
		}
		if len(in.AddMemberEntityIDs) != 0 {
			if err := dl.AddThreadMembers(ctx, thread.ID, in.AddMemberEntityIDs); err != nil {
				return errors.Trace(err)
			}
			update.SystemTitle = &systemTitle
		}
		if len(in.RemoveMemberEntityIDs) != 0 {
			if err := dl.RemoveThreadMembers(ctx, thread.ID, in.RemoveMemberEntityIDs); err != nil {
				return errors.Trace(err)
			}
			update.SystemTitle = &systemTitle
		}
		if len(in.AddFollowerEntityIDs) != 0 {
			if err := dl.AddThreadFollowers(ctx, thread.ID, in.AddFollowerEntityIDs); err != nil {
				return errors.Trace(err)
			}
		}
		if len(in.RemoveFollowerEntityIDs) != 0 {
			if err := dl.RemoveThreadFollowers(ctx, thread.ID, in.RemoveFollowerEntityIDs); err != nil {
				return errors.Trace(err)
			}
		}
		if update.UserTitle != nil || update.SystemTitle != nil {
			if err := dl.UpdateThread(ctx, thread.ID, update); err != nil {
				return errors.Trace(err)
			}
		}
		return nil
	}); err != nil {
		return nil, errors.Trace(err)
	}

	threads, err = s.dal.Threads(ctx, []models.ThreadID{tid})
	if err != nil {
		return nil, errors.Trace(err)
	}
	thread = threads[0]
	if err := s.updateSavedQueriesForThread(ctx, thread); err != nil {
		golog.Errorf("Failed to updated saved query for thread %s: %s", thread.ID, err)
	}
	th, err := transformThreadToResponse(thread, false)
	if err != nil {
		return nil, errors.Trace(err)
	}
	ts, err := s.hydrateThreadForViewer(ctx, []*threading.Thread{th}, in.ActorEntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	th = ts[0]
	return &threading.UpdateThreadResponse{
		Thread: th,
	}, nil
}

func (s *threadsServer) hydrateThreadForViewer(ctx context.Context, ts []*threading.Thread, viewerEntityID string) ([]*threading.Thread, error) {
	tIDs := make([]models.ThreadID, 0, len(ts))
	for _, t := range ts {
		if t.MessageCount > 0 || t.Type == threading.THREAD_TYPE_TEAM {
			id, err := models.ParseThreadID(t.ID)
			if err != nil {
				return nil, errors.Trace(err)
			}
			tIDs = append(tIDs, id)
		}
	}
	if len(tIDs) == 0 {
		golog.Debugf("No threadIDs populated..returning original list")
		return ts, nil
	}

	tes, err := s.dal.ThreadEntities(ctx, tIDs, viewerEntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ts2 := make([]*threading.Thread, 0, len(ts))
	for _, t := range ts {
		te := tes[t.ID]
		if t.MessageCount > 0 {
			t.Unread = te == nil || te.LastViewed == nil || (t.LastMessageTimestamp > uint64(te.LastViewed.Unix()))
		}

		// Filter out threads which the viewer doesn't have access to
		golog.Debugf("Thread type %s", t.Type)
		if t.Type != threading.THREAD_TYPE_TEAM || (te != nil && te.Member) {
			golog.Debugf("Appending thread %s to hydrated list", t.ID)
			ts2 = append(ts2, t)
		}
	}
	return ts2, nil
}

func (s *threadsServer) publishMessage(ctx context.Context, orgID, primaryEntityID string, threadID models.ThreadID, item *threading.ThreadItem, uuid string) {
	if s.sns == nil {
		return
	}
	conc.Go(func() {
		pit := &threading.PublishedThreadItem{
			OrganizationID:  orgID,
			ThreadID:        threadID.String(),
			PrimaryEntityID: primaryEntityID,
			Item:            item,
			UUID:            uuid,
		}
		data, err := pit.Marshal()
		if err != nil {
			golog.Errorf("Failed to marshal PublishedThreadItem: %s", err)
			return
		}
		msg := base64.StdEncoding.EncodeToString(data)
		if _, err := s.sns.Publish(&sns.PublishInput{
			Message:  &msg,
			TopicArn: &s.snsTopicARN,
		}); err != nil {
			golog.Errorf("Failed to publish SNS: %s", err)
		}
	})
}

// teamThreadSystemTitle generates a system title for a thread and verifies that all members are part of the expected organization
func (s *threadsServer) teamThreadSystemTitle(ctx context.Context, orgID string, memberEntityIDs []string) (string, error) {
	if len(memberEntityIDs) == 0 {
		return "", nil
	}

	res, err := s.directoryClient.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
			BatchEntityID: &directory.IDList{
				IDs: memberEntityIDs,
			},
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 1,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
	})
	if err != nil {
		return "", errors.Trace(err)
	}
	names := make([]string, len(res.Entities))
	for i, e := range res.Entities {
		var found bool
		for _, m := range e.Memberships {
			if m.ID == orgID {
				found = true
				break
			}
		}
		if !found {
			return "", errors.Errorf("Entity %s is not a member of org %s", e.ID, orgID)
		}
		names[i] = e.Info.DisplayName
	}
	return strings.Join(names, ", "), nil
}

func (s *threadsServer) forExternalViewer(ctx context.Context, viewerEntityID string) (bool, error) {
	forExternal := true
	if viewerEntityID != "" {
		ent, err := directory.SingleEntity(ctx, s.directoryClient, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: viewerEntityID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
			},
		})
		if grpc.Code(err) == codes.NotFound {
			return false, grpcErrorf(codes.NotFound, "Viewing entity %s not found", viewerEntityID)
		} else if err != nil {
			return false, errors.Trace(err)
		}
		forExternal = isExternalEntity(ent)
	}
	return forExternal, nil
}

func (s *threadsServer) membersForThread(ctx context.Context, threadID models.ThreadID) ([]*models.ThreadEntity, error) {
	tes, err := s.dal.EntitiesForThread(ctx, threadID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	members := make([]*models.ThreadEntity, 0, len(tes))
	for _, te := range tes {
		if te.Member {
			members = append(members, te)
		}
	}
	return members, nil
}
