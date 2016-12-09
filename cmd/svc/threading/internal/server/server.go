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
	"github.com/sprucehealth/backend/svc/events"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const setupThreadTitle = "Setup"

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
	publisher          events.Publisher
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
	publisher events.Publisher,
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
		paymentsClient:     paymentsClient,
		publisher:          publisher,
		webDomain:          webDomain,
	}
}

// CreateSavedQuery saves a query for later use
func (s *threadsServer) CreateSavedQuery(ctx context.Context, in *threading.CreateSavedQueryRequest) (*threading.CreateSavedQueryResponse, error) {
	if in.EntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "EntityID is required")
	}
	if in.Query == nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Query is required")
	}
	if in.ShortTitle == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Title is required")
	}
	// TODO: in order to be backwards compatible with clients that don't send the type for now assume the normal type. remove once updated clients are deployed.
	if in.Type == threading.SAVED_QUERY_TYPE_INVALID {
		in.Type = threading.SAVED_QUERY_TYPE_NORMAL
	}

	query, err := transformQueryFromRequest(in.Query)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Query is invalid: %s", err)
	}

	sq := &models.SavedQuery{
		EntityID:             in.EntityID,
		ShortTitle:           in.ShortTitle,
		LongTitle:            in.LongTitle,
		Description:          in.Description,
		Query:                query,
		Ordinal:              int(in.Ordinal),
		NotificationsEnabled: in.NotificationsEnabled,
		Hidden:               in.Hidden,
		Template:             in.Template,
	}
	switch in.Type {
	case threading.SAVED_QUERY_TYPE_NORMAL:
		sq.Type = models.SavedQueryTypeNormal
	case threading.SAVED_QUERY_TYPE_NOTIFICATIONS:
		sq.Type = models.SavedQueryTypeNotifications
	default:
		return nil, grpc.Errorf(codes.InvalidArgument, "Unknown saved query type %s", in.Type)
	}
	id, err := s.dal.CreateSavedQuery(ctx, sq)
	if err != nil {
		return nil, errors.Trace(err)
	}
	sq.ID = id
	if err := s.rebuildSavedQuery(ctx, sq); err != nil {
		golog.ContextLogger(ctx).Errorf("Failed to build new saved query %s: %s", sq.ID, err)
	}
	if sq.NotificationsEnabled {
		if err := s.notifyBadgeCountUpdate(ctx, []string{sq.EntityID}); err != nil {
			golog.ContextLogger(ctx).Errorf("Failed to notify entity %s of updated badge count: %s", sq.EntityID, err)
		}
	}
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
		return nil, grpc.Errorf(codes.InvalidArgument, fmt.Sprintf("Type '%s' not allowed for CreateEmptyThread", in.Type.String()))
	}
	if in.OrganizationID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "OrganizationID is required")
	}
	if in.PrimaryEntityID == "" && in.Type != threading.THREAD_TYPE_TEAM && in.Type != threading.THREAD_TYPE_SECURE_EXTERNAL {
		return nil, grpc.Errorf(codes.InvalidArgument, "PrimaryEntityID is required for non app only threads")
	}
	if t, ok := validateTags(in.Tags); !ok {
		return nil, grpc.Errorf(codes.InvalidArgument, "Tag %q is invalid", t)
	}
	if in.Summary == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Summary is required")
	}
	in.Summary = textutil.TruncateUTF8(in.Summary, maxSummaryLength)
	if in.Type == threading.THREAD_TYPE_TEAM && in.FromEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "FromEntityID is required for TEAM threads")
	}
	if id, ok := validateEntityIDs(in.MemberEntityIDs); !ok {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid entity ID %s in members list", id)
	}

	if (in.Type == threading.THREAD_TYPE_EXTERNAL || in.Type == threading.THREAD_TYPE_SECURE_EXTERNAL) && in.SystemTitle == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "SystemTitle is required for EXTERNAL and SECURE_EXTERNAL threads")
	}

	tt, err := transformThreadTypeFromRequest(in.Type)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid thread type '%s'", in.Type)
	}

	to, err := transformThreadOriginFromRequest(in.Origin)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid thread origin '%s'", in.Origin)
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
		if len(in.Tags) != 0 {
			if err := dl.AddThreadTags(ctx, in.OrganizationID, threadID, in.Tags); err != nil {
				return errors.Trace(err)
			}
		}
		if in.FromEntityID != "" {
			if err := dl.UpdateThreadEntity(ctx, threadID, in.FromEntityID, nil); err != nil {
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
	}
	if len(threads) == 0 {
		return nil, errors.Errorf("thread with id %q just created not found", threadID)
	}
	if _, err := s.updateSavedQueriesAddThread(ctx, threads[0], memberEntityIDs); err != nil {
		golog.ContextLogger(ctx).Errorf("Failed to updated saved query when adding thread: %s", threadID)
	}
	th, err := transformThreadToResponse(threads[0], false)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// Publish that we created a new thread
	s.publisher.PublishAsync(&threading.NewThreadEvent{ThreadID: th.ID})
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
		return nil, grpc.Errorf(codes.InvalidArgument, fmt.Sprintf("Type %q not allowed for CreateThread", in.Type.String()))
	}
	if in.OrganizationID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "OrganizationID is required")
	}
	if in.FromEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "FromEntityID is required")
	}
	if in.Message == nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Message is required")
	}
	if in.Type == threading.THREAD_TYPE_EXTERNAL && in.SystemTitle == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "SystemTitle is required")
	}
	if id, ok := validateEntityIDs(in.MemberEntityIDs); !ok {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid entity ID %q in members list", id)
	}
	if t, ok := validateTags(in.Tags); !ok {
		return nil, grpc.Errorf(codes.InvalidArgument, "Tag %q is invalid", t)
	}

	tt, err := transformThreadTypeFromRequest(in.Type)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid thread type %q", in.Type)
	}
	to, err := transformThreadOriginFromRequest(in.Origin)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid thread origin %q", in.Origin)
	}

	textRefs, err := processMessagePost(in.Message, false)
	if err != nil {
		return nil, err
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
		if len(in.Tags) != 0 {
			if err := dl.AddThreadTags(ctx, in.OrganizationID, threadID, in.Tags); err != nil {
				return errors.Trace(err)
			}
		}

		req := &dal.PostMessageRequest{
			ThreadID:     threadID,
			FromEntityID: in.FromEntityID,
			Internal:     in.Message.Internal,
			Text:         in.Message.Text,
			Title:        in.Message.Title,
			TextRefs:     textRefs,
			Summary:      in.Message.Summary,
		}
		if in.Message.Source != nil {
			req.Source, err = transformEndpointFromRequest(in.Message.Source)
			if err != nil {
				return errors.Trace(err)
			}
		}
		req.Attachments, err = transformAttachmentsFromRequest(in.Message.Attachments)
		if err != nil {
			return errors.Trace(err)
		}
		for _, dc := range in.Message.Destinations {
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
		return nil, errors.Errorf("thread %q just created not found", threadID)
	}
	thread := threads[0]
	updateResult, err := s.updateSavedQueriesAddThread(ctx, thread, memberEntityIDs)
	if err != nil {
		golog.ContextLogger(ctx).Errorf("Failed to updated saved query when adding thread: %s", threadID)
	}
	th, err := transformThreadToResponse(thread, !in.Message.Internal)
	if err != nil {
		return nil, errors.Trace(err)
	}
	it, err := transformThreadItemToResponse(item, thread.OrganizationID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	s.publishMessage(ctx, in.OrganizationID, in.FromEntityID, threadID, it, in.UUID)
	if !in.DontNotify && updateResult != nil {
		s.notifyMembersOfPublishMessage(ctx, thread.OrganizationID, models.EmptySavedQueryID(), thread, item, in.FromEntityID, updateResult.entityShouldBeNotified)
	}
	// Publish that we created a new thread
	s.publisher.PublishAsync(&threading.NewThreadEvent{ThreadID: threadID.String()})
	return &threading.CreateThreadResponse{
		ThreadID:   threadID.String(),
		ThreadItem: it,
		Thread:     th,
	}, nil
}

func (s *threadsServer) CreateLinkedThreads(ctx context.Context, in *threading.CreateLinkedThreadsRequest) (*threading.CreateLinkedThreadsResponse, error) {
	if in.Type != threading.THREAD_TYPE_SUPPORT {
		return nil, grpc.Errorf(codes.InvalidArgument, "Only threads of type SUPPORT are allowed for linked threads")
	}
	if in.Organization1ID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Organization1ID is required")
	}
	if in.Organization2ID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Organization2ID is required")
	}
	if in.PrimaryEntity1ID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "PrimaryEntity1ID is required")
	}
	if in.PrimaryEntity2ID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "PrimaryEntity2ID is required")
	}
	tt, err := transformThreadTypeFromRequest(in.Type)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid thread type '%s'", in.Type)
	}
	if in.Summary == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Summary is required")
	}
	in.Summary = textutil.TruncateUTF8(in.Summary, maxSummaryLength)
	if in.MessageTitle != "" {
		if _, err := bml.Parse(in.MessageTitle); err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, fmt.Sprintf("MessageTitle is invalid format: %s", err.Error()))
		}
	}
	var textRefs []*models.Reference
	in.Text, textRefs, err = parseRefsAndNormalize(in.Text)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, fmt.Sprintf("Text is invalid format: %s", errors.Cause(err).Error()))
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

	if _, err := s.updateSavedQueriesAddThread(ctx, threads[0], []string{in.Organization1ID}); err != nil {
		golog.ContextLogger(ctx).Errorf("Failed to updated saved query when adding thread: %s", threads[0].ID)
	}
	if _, err := s.updateSavedQueriesAddThread(ctx, threads[1], []string{in.Organization2ID}); err != nil {
		golog.ContextLogger(ctx).Errorf("Failed to updated saved query when adding thread: %s", threads[1].ID)
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
func (s *threadsServer) DeleteMessage(ctx context.Context, in *threading.DeleteMessageRequest) (*threading.DeleteMessageResponse, error) {
	if in.ActorEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "ActorEntityID is required")
	}
	if in.ThreadItemID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "ThreadItemID is required")
	}
	threadItemID, err := models.ParseThreadItemID(in.ThreadItemID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ThreadItemID '%s'", in.ThreadItemID)
	}
	var item *models.ThreadItem
	var deleted bool
	err = s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		item, deleted, err = dl.DeleteMessage(ctx, threadItemID)
		if err != nil {
			return errors.Trace(err)
		}
		if deleted {
			return errors.Trace(dl.CreateThreadItem(ctx, &models.ThreadItem{
				ThreadID:      item.ThreadID,
				ActorEntityID: in.ActorEntityID,
				Internal:      item.Internal,
				Data: &models.MessageDelete{
					ThreadItemID: threadItemID.String(),
				},
			}))
		}
		return nil
	})
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "Thread item %q not found", in.ThreadItemID)
	}
	if err != nil {
		return nil, errors.Trace(err)
	}
	if deleted {
		threads, err := s.dal.Threads(ctx, []models.ThreadID{item.ThreadID})
		if err != nil {
			golog.ContextLogger(ctx).Errorf("Failed to fetch thread %s: %s", item.ThreadID, err)
		} else if _, err := s.updateSavedQueriesForThread(ctx, threads[0]); err != nil {
			golog.ContextLogger(ctx).Errorf("Failed to updated saved query for thread %s: %s", item.ThreadID, err)
		}
	}
	return &threading.DeleteMessageResponse{}, nil
}

// DeleteThread deletes a thread
func (s *threadsServer) DeleteThread(ctx context.Context, in *threading.DeleteThreadRequest) (*threading.DeleteThreadResponse, error) {
	if in.ActorEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "ActorEntityID is required")
	}
	if in.ThreadID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "ThreadID is required")
	}
	threadID, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ThreadID '%s'", in.ThreadID)
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
		entity, err := directory.SingleEntity(ctx, s.directoryClient, &directory.LookupEntitiesRequest{
			Key: &directory.LookupEntitiesRequest_EntityID{
				EntityID: thread.PrimaryEntityID,
			},
			RootTypes: []directory.EntityType{directory.EntityType_EXTERNAL, directory.EntityType_PATIENT},
			Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		})
		if err != nil && err != directory.ErrEntityNotFound {
			return nil, errors.Trace(err)
		}

		if entity != nil {
			if _, err := s.directoryClient.DeleteEntity(ctx, &directory.DeleteEntityRequest{
				EntityID: entity.ID,
			}); err != nil {
				return nil, errors.Trace(err)
			}
		}
	}
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		if err := dl.DeleteThread(ctx, threadID); err != nil {
			return errors.Trace(err)
		}
		if err := dl.RecordThreadEvent(ctx, threadID, in.ActorEntityID, models.ThreadEventDelete); err != nil {
			return errors.Trace(err)
		}
		return errors.Trace(dl.RemoveThreadFromAllSavedQueryIndexes(ctx, threadID))
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.DeleteThreadResponse{}, nil
}

func (s *threadsServer) LinkedThread(ctx context.Context, in *threading.LinkedThreadRequest) (*threading.LinkedThreadResponse, error) {
	if in.ThreadID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "ThreadID is required")
	}
	threadID, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ThreadID '%s'", in.ThreadID)
	}
	thread, prependSender, err := s.dal.LinkedThread(ctx, threadID)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "Linked thread for %q not found", threadID)
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
		return nil, grpc.Errorf(codes.InvalidArgument, "ThreadWatermarks required")
	}

	if in.EntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "EntityID is required")
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
			return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ThreadID '%s'", w.ThreadID)
		}
		threadIDs[i] = threadID
	}
	threads, threadEntities, err := s.dal.ThreadsWithEntity(ctx, in.EntityID, threadIDs)
	if err != nil {
		return nil, errors.Trace(err)
	}

	watermarks := make(map[string]uint64, len(in.ThreadWatermarks))
	for _, w := range in.ThreadWatermarks {
		watermarks[w.ThreadID] = w.LastMessageTimestamp
	}

	sqs, err := s.dal.SavedQueries(ctx, in.EntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var addIndex, removeIndex []*dal.SavedQueryThread

	currentTime := s.clk.Now()
	for i, thread := range threads {
		threadEntity := threadEntities[i]
		watermarkTimestamp := watermarks[thread.ID.String()]

		readTime := currentTime
		// only use the last message timestamp if one is provided by the client or it is in the past but after the reference date of the product launch
		if watermarkTimestamp != 0 && watermarkTimestamp < uint64(currentTime.Unix()) && watermarkTimestamp > uint64(baymaxLaunchDate.Unix()) {
			readTime = time.Unix(int64(watermarkTimestamp), 0)
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

		// If unread matches expected state then don't do anything
		currentUnread := isUnread(thread, threadEntity, externalEntity)
		newUnread := isUnread(thread, &models.ThreadEntity{LastViewed: &readTime}, externalEntity)
		if currentUnread == newUnread {
			continue
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
			golog.ContextLogger(ctx).Errorf("Entity '%s' trying to mark as a read a thread '%s' it is not a member of", in.EntityID, thread.ID)
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

		// find the notifications saved query
		var nsq *models.SavedQuery
		for _, sq := range sqs {
			if sq.Type == models.SavedQueryTypeNotifications {
				nsq = sq
				break
			}
		}

		for _, sq := range sqs {
			if sq.Type == models.SavedQueryTypeNotifications {
				continue
			}

			if ok, err := threadMatchesQuery(sq.Query, thread, threadEntity, externalEntity); err != nil {
				golog.ContextLogger(ctx).Errorf("Error matching thread %s against saved query %s: %s", thread.ID, sq.ID, err)
			} else if ok {
				timestamp := thread.LastMessageTimestamp
				if externalEntity {
					timestamp = thread.LastExternalMessageTimestamp
				}
				unread := isUnread(thread, threadEntity, externalEntity)
				addIndex = append(addIndex, &dal.SavedQueryThread{
					SavedQueryID: sq.ID,
					ThreadID:     thread.ID,
					Unread:       unread,
					Timestamp:    timestamp,
				})
				if sq.NotificationsEnabled && nsq != nil {
					addIndex = append(addIndex, &dal.SavedQueryThread{
						SavedQueryID: nsq.ID,
						ThreadID:     thread.ID,
						Unread:       unread,
						Timestamp:    timestamp,
					})
				}
			} else {
				removeIndex = append(removeIndex, &dal.SavedQueryThread{SavedQueryID: sq.ID, ThreadID: thread.ID})
			}
		}
	}

	if err := s.dal.AddItemsToSavedQueryIndex(ctx, addIndex); err != nil {
		return nil, errors.Trace(err)
	}
	if err := s.dal.RemoveItemsFromSavedQueryIndex(ctx, removeIndex); err != nil {
		return nil, errors.Trace(err)
	}

	if err := s.notifyBadgeCountUpdate(ctx, []string{in.EntityID}); err != nil {
		golog.ContextLogger(ctx).Errorf("Failed to notify entity %s of updated badge count: %s", in.EntityID, err)
	}

	return &threading.MarkThreadsAsReadResponse{}, nil
}

// PostMessage posts a message into a specified thread
func (s *threadsServer) PostMessage(ctx context.Context, in *threading.PostMessageRequest) (*threading.PostMessageResponse, error) {
	// TODO: return proper error responses for invalid request
	if in.ThreadID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "ThreadID is required")
	}
	threadID, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ThreadID '%s'", in.ThreadID)
	}
	if in.FromEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "FromEntityID is required")
	}

	if in.Message == nil {
		in.Message = &threading.MessagePost{
			Summary:      in.DeprecatedSummary,
			Title:        in.DeprecatedTitle,
			Attachments:  in.DeprecatedAttachments,
			Source:       in.DeprecatedSource,
			Destinations: in.DeprecatedDestinations,
			Internal:     in.DeprecatedInternal,
		}
	}

	item, linkedItem, err := s.postMessage(ctx, s.dal, in.UUID, threadID, in.FromEntityID, in.DontNotify, in.Message)
	if err != nil {
		return nil, errors.Trace(err)
	}

	threads, err := s.dal.Threads(ctx, []models.ThreadID{threadID})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(threads) == 0 {
		return nil, errors.Errorf("Thread %q that was just posted to was not found", threadID)
	}
	thread := threads[0]
	updateResult, err := s.updateSavedQueriesForThread(ctx, thread)
	if err != nil {
		golog.ContextLogger(ctx).Errorf("Failed to updated saved query for thread %s: %s", thread.ID, err)
	}

	th, err := transformThreadToResponse(thread, !in.Message.Internal)
	if err != nil {
		return nil, errors.Trace(err)
	}
	it, err := transformThreadItemToResponse(item, thread.OrganizationID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	s.publishMessage(ctx, thread.OrganizationID, thread.PrimaryEntityID, threadID, it, in.UUID)
	if !in.DontNotify && updateResult != nil {
		s.notifyMembersOfPublishMessage(ctx, thread.OrganizationID, models.EmptySavedQueryID(), thread, item, in.FromEntityID, updateResult.entityShouldBeNotified)
	}

	if linkedItem != nil {
		// Requery the linked thread to get updated metadata (e.g. last message timestamp)
		linkedThreads, err := s.dal.Threads(ctx, []models.ThreadID{linkedItem.ThreadID})
		if err != nil {
			return nil, errors.Trace(err)
		} else if len(linkedThreads) == 0 {
			golog.ContextLogger(ctx).Errorf("Thread %q that was just posted to was not found", linkedItem.ThreadID)
		}
		linkedThread := linkedThreads[0]
		updateResult, err := s.updateSavedQueriesForThread(ctx, linkedThreads[0])
		if err != nil {
			golog.ContextLogger(ctx).Errorf("Failed to updated saved query for thread %s: %s", linkedThreads[0].ID, err)
		}
		if !in.DontNotify && updateResult != nil {
			s.notifyMembersOfPublishMessage(ctx, linkedThread.OrganizationID, models.EmptySavedQueryID(), linkedThread, linkedItem, linkedItem.ActorEntityID, updateResult.entityShouldBeNotified)
		}

		it2, err := transformThreadItemToResponse(linkedItem, linkedThread.OrganizationID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		s.publishMessage(ctx, linkedThread.OrganizationID, linkedThread.PrimaryEntityID, linkedThread.ID, it2, "")
	}

	return &threading.PostMessageResponse{
		Item:   it,
		Thread: th,
	}, nil
}

// PostMessages posts a series of messages into a specified thread
func (s *threadsServer) PostMessages(ctx context.Context, in *threading.PostMessagesRequest) (*threading.PostMessagesResponse, error) {
	if in.ThreadID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "ThreadID is required")
	}
	threadID, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ThreadID '%s'", in.ThreadID)
	}
	if in.FromEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "FromEntityID is required")
	}
	if len(in.Messages) == 0 {
		return nil, grpc.Errorf(codes.InvalidArgument, "At least 1 message required")
	}

	items := make([]*models.ThreadItem, len(in.Messages))
	var linkedItems []*models.ThreadItem
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		for i, message := range in.Messages {
			item, linkedItem, err := s.postMessage(ctx, dl, in.UUID, threadID, in.FromEntityID, in.DontNotify, message)
			if err != nil {
				return errors.Trace(err)
			}
			items[i] = item
			if linkedItem != nil {
				linkedItems = append(linkedItems, linkedItem)
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
		return nil, errors.Errorf("Thread %q that was just posted to was not found", threadID)
	}
	thread := threads[0]
	updateResult, err := s.updateSavedQueriesForThread(ctx, thread)
	if err != nil {
		golog.ContextLogger(ctx).Errorf("Failed to updated saved query for thread %s: %s", thread.ID, err)
	}

	// Use the type of last message posted to set the watermark
	th, err := transformThreadToResponse(thread, !in.Messages[len(in.Messages)-1].Internal)
	if err != nil {
		return nil, errors.Trace(err)
	}

	its := make([]*threading.ThreadItem, len(items))
	for i, item := range items {
		it, err := transformThreadItemToResponse(item, thread.OrganizationID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		s.publishMessage(ctx, thread.OrganizationID, thread.PrimaryEntityID, threadID, it, in.UUID)
		if !in.DontNotify && updateResult != nil {
			s.notifyMembersOfPublishMessage(ctx, thread.OrganizationID, models.EmptySavedQueryID(), thread, item, in.FromEntityID, updateResult.entityShouldBeNotified)
		}
		its[i] = it
	}

	var linkedThread *models.Thread
	var linkedThreadUpdateResult *savedQueryUpdateResult
	if len(linkedItems) != 0 {
		// Requery the linked thread to get updated metadata (e.g. last message timestamp)
		linkedThreads, err := s.dal.Threads(ctx, []models.ThreadID{linkedItems[0].ThreadID})
		if err != nil {
			return nil, errors.Trace(err)
		} else if len(linkedThreads) == 0 {
			golog.ContextLogger(ctx).Errorf("Thread %q that was just posted to was not found", linkedItems[0].ThreadID)
		}
		linkedThread = linkedThreads[0]
		linkedThreadUpdateResult, err = s.updateSavedQueriesForThread(ctx, linkedThread)
		if err != nil {
			golog.ContextLogger(ctx).Errorf("Failed to updated saved query for thread %s: %s", linkedThreads[0].ID, err)
		}
	}
	for _, litem := range linkedItems {
		if !in.DontNotify && updateResult != nil {
			s.notifyMembersOfPublishMessage(ctx, linkedThread.OrganizationID, models.EmptySavedQueryID(), linkedThread, litem, litem.ActorEntityID, linkedThreadUpdateResult.entityShouldBeNotified)
		}
		lit, err := transformThreadItemToResponse(litem, linkedThread.OrganizationID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		s.publishMessage(ctx, linkedThread.OrganizationID, linkedThread.PrimaryEntityID, linkedThread.ID, lit, "")
	}

	return &threading.PostMessagesResponse{
		Items:  its,
		Thread: th,
	}, nil
}

func (s *threadsServer) postMessage(
	ctx context.Context,
	dl dal.DAL,
	uuid string,
	threadID models.ThreadID,
	fromEntityID string,
	dontNotify bool,
	message *threading.MessagePost) (*models.ThreadItem, *models.ThreadItem, error) {
	threads, err := s.dal.Threads(ctx, []models.ThreadID{threadID})
	if err != nil {
		return nil, nil, errors.Trace(err)
	} else if len(threads) == 0 {
		return nil, nil, grpc.Errorf(codes.NotFound, "Thread %q not found", threadID)
	}
	thread := threads[0]

	prePostLastMessageTimestamp := thread.LastMessageTimestamp

	linkedThread, prependSender, err := dl.LinkedThread(ctx, thread.ID)
	if err != nil && errors.Cause(err) != dal.ErrNotFound {
		return nil, nil, errors.Trace(err)
	}

	var item *models.ThreadItem
	var linkedItem *models.ThreadItem

	req, err := createPostMessageRequest(ctx, threadID, fromEntityID, message)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	if err := claimAttachments(ctx, s.mediaClient, s.paymentsClient, threadID, req.Attachments); err != nil {
		return nil, nil, errors.Trace(err)
	}

	if err := dl.Transact(ctx, func(ctx context.Context, tdl dal.DAL) error {
		item, err = dl.PostMessage(ctx, req)
		if err != nil {
			return errors.Trace(err)
		}

		// Update unread reference status for anyone mentioned
		for _, r := range req.TextRefs {
			if err := tdl.UpdateThreadEntity(ctx, threadID, r.ID, &dal.ThreadEntityUpdate{
				LastReferenced: &item.Created,
			}); err != nil {
				return errors.Trace(err)
			}
		}

		// Lock our membership row while doing this since we might update it
		tes, err := tdl.ThreadEntities(ctx, []models.ThreadID{threadID}, fromEntityID, dal.ForUpdate)
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
			if !lastViewed.Before(prePostLastMessageTimestamp.Truncate(time.Second)) {
				teUpdate = &dal.ThreadEntityUpdate{
					LastViewed: &item.Created,
				}
			}
		}
		if err := tdl.UpdateThreadEntity(ctx, threadID, fromEntityID, teUpdate); err != nil {
			return errors.Trace(err)
		}

		// Also post in linked thread if there is one
		if linkedThread != nil && !message.Internal {
			// TODO: should use primary entity name here
			summary, err := models.SummaryFromText("Spruce: " + message.Text)
			if err != nil {
				return errors.Trace(err)
			}
			text := message.Text
			if prependSender {
				resp, err := s.directoryClient.LookupEntities(ctx, &directory.LookupEntitiesRequest{
					Key: &directory.LookupEntitiesRequest_EntityID{
						EntityID: fromEntityID,
					},
					RequestedInformation: &directory.RequestedInformation{
						Depth: 0,
					},
				})
				if err != nil {
					golog.ContextLogger(ctx).Errorf("Unable to lookup entity for id %s: %s", fromEntityID, err.Error())
				} else if len(resp.Entities) != 1 {
					golog.ContextLogger(ctx).Errorf("Expected 1 entity for id %s but got %d back", fromEntityID, len(resp.Entities))
				} else if resp.Entities[0].Type == directory.EntityType_INTERNAL {
					validBML, err := bml.BML{resp.Entities[0].Info.DisplayName}.Format()
					if err != nil {
						golog.ContextLogger(ctx).Errorf("Unable to escape the display name %s:%s", resp.Entities[0].Info.DisplayName, err.Error())
					} else {
						text = validBML + ": " + text
					}
				}
			}
			req := &dal.PostMessageRequest{
				ThreadID:     linkedThread.ID,
				FromEntityID: linkedThread.PrimaryEntityID,
				Text:         text,
				Title:        message.Title,
				TextRefs:     req.TextRefs,
				Summary:      summary,
				Attachments:  req.Attachments,
			}
			if message.Source != nil {
				req.Source, err = transformEndpointFromRequest(message.Source)
				if err != nil {
					return errors.Trace(err)
				}
			}
			linkedItem, err = tdl.PostMessage(ctx, req)
			if err != nil {
				return errors.Trace(err)
			}
		}

		return nil
	}); err != nil {
		return nil, nil, errors.Trace(err)
	}

	return item, linkedItem, nil
}

// QueryThreads queries the list of threads
func (s *threadsServer) QueryThreads(ctx context.Context, in *threading.QueryThreadsRequest) (*threading.QueryThreadsResponse, error) {
	if in.ViewerEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "ViewerEntityID required")
	}

	var sq *models.SavedQuery
	var query *models.Query
	switch in.Type {
	case threading.QUERY_THREADS_TYPE_ADHOC:
		if in.GetQuery() == nil {
			return nil, grpc.Errorf(codes.InvalidArgument, "Query required for ADHOC queries")
		}
		var err error
		query, err = transformQueryFromRequest(in.GetQuery())
		if err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, "Query is not valid: %s", err)
		}
	case threading.QUERY_THREADS_TYPE_SAVED:
		sqID, err := models.ParseSavedQueryID(in.GetSavedQueryID())
		if err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, "Saved query ID %s is not valid", in.GetSavedQueryID())
		}
		sq, err = s.dal.SavedQuery(ctx, sqID)
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpc.Errorf(codes.NotFound, "Saved query %q not found", sqID)
		} else if err != nil {
			return nil, errors.Trace(err)
		}
		if sq.EntityID != in.ViewerEntityID {
			return nil, grpc.Errorf(codes.InvalidArgument, "Saved query %s not owned by %s", sq.ID, in.ViewerEntityID)
		}
		query = sq.Query
	case threading.QUERY_THREADS_TYPE_ALL_FOR_VIEWER:
		query = &models.Query{}
	default:
		return nil, grpc.Errorf(codes.InvalidArgument, "Unknown query type %s", in.Type)
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
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid iterator: %s", e)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	res := &threading.QueryThreadsResponse{
		Edges:   make([]*threading.ThreadEdge, 0, len(tc.Edges)),
		HasMore: tc.HasMore,
	}

	log := golog.ContextLogger(ctx).Context("viewerEntityID", in.ViewerEntityID)
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
		if e.Thread.Deleted {
			if in.Type == threading.QUERY_THREADS_TYPE_SAVED {
				log.Errorf("Deleted thread %s returned for saved query %s for entity %s", e.Thread.ID, in.GetSavedQueryID(), in.ViewerEntityID)
			} else {
				log.Errorf("Deleted thread %s returned for query %s for entity %s", e.Thread.ID, query, in.ViewerEntityID)
			}
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
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid SavedQueryID '%s'", in.SavedQueryID)
	}
	query, err := s.dal.SavedQuery(ctx, id)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "Saved query %q not found", in.SavedQueryID)
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
			return nil, errors.Errorf("Failed to transform saved query: %s", err)
		}
		res.SavedQueries[i] = sq
	}
	return res, nil
}

func (s *threadsServer) DeleteSavedQueries(ctx context.Context, in *threading.DeleteSavedQueriesRequest) (*threading.DeleteSavedQueriesResponse, error) {
	ids := make([]models.SavedQueryID, len(in.SavedQueryIDs))
	var err error
	for i, id := range in.SavedQueryIDs {
		ids[i], err = models.ParseSavedQueryID(id)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	if err := s.dal.DeleteSavedQueries(ctx, ids); err != nil {
		return nil, errors.Trace(err)
	}

	return &threading.DeleteSavedQueriesResponse{}, nil
}

func (s *threadsServer) SavedQueryTemplates(ctx context.Context, in *threading.SavedQueryTemplatesRequest) (*threading.SavedQueryTemplatesResponse, error) {
	queries, err := s.dal.SavedQueryTemplates(ctx, in.EntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(queries) == 0 {
		queries = models.DefaultSavedQueries
	}
	res := &threading.SavedQueryTemplatesResponse{
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
	tid, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ThreadID '%s'", in.ThreadID)
	}

	forExternal, err := s.forExternalViewer(ctx, in.ViewerEntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	threads, err := s.dal.Threads(ctx, []models.ThreadID{tid})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(threads) == 0 {
		return nil, grpc.Errorf(codes.NotFound, "Thread %q not found", tid)
	}
	thread := threads[0]

	th, err := transformThreadToResponse(thread, forExternal)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if in.ViewerEntityID != "" {
		ts, err := s.hydrateThreadForViewer(ctx, []*threading.Thread{th}, in.ViewerEntityID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if len(ts) == 0 {
			return nil, grpc.Errorf(codes.NotFound, "Thread %q not found", th.ID)
		}
		// TODO: for now can't require the viewer since the graphql service requests the thread to get the org ID before it can know the entity viewing
		// } else if th.Type == threading.THREAD_TYPE_TEAM {
		// 	// Require a viewer entity for private threads
		// 	return nil, grpc.Errorf(codes.NotFound, "Thread not found")
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

	threadsInResponse := make([]*threading.Thread, len(threads))
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
			return nil, grpc.Errorf(codes.NotFound, "Thread not found")
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
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ItemID '%s'", in.ItemID)
	}

	item, err := s.dal.ThreadItem(ctx, tid)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "Thread item %q not found", tid)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	threads, err := s.dal.Threads(ctx, []models.ThreadID{item.ThreadID})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(threads) == 0 {
		return nil, grpc.Errorf(codes.NotFound, "Thread %q not found", tid)
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
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ThreadID '%s'", in.ThreadID)
	}

	threads, err := s.dal.Threads(ctx, []models.ThreadID{tid})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(threads) == 0 {
		return nil, grpc.Errorf(codes.NotFound, "Thread %q not found", tid)
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
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid iterator: "+string(e))
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
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ItemID '%s'", in.ItemID)
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
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ThreadID '%s'", in.ThreadID)
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

// UpdateMessage updates the content of a message
func (s *threadsServer) UpdateMessage(ctx context.Context, in *threading.UpdateMessageRequest) (*threading.UpdateMessageResponse, error) {
	if in.ActorEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "ActorEntityID is required")
	}
	if in.ThreadItemID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "ThreadItemID is required")
	}
	if in.Message == nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Message is required")
	}
	threadItemID, err := models.ParseThreadItemID(in.ThreadItemID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ThreadItemID %q", in.ThreadItemID)
	}
	item, err := s.dal.ThreadItem(ctx, threadItemID)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "Thread item %q not found", threadItemID)
	}
	if err != nil {
		return nil, errors.Trace(err)
	}
	if item.Deleted {
		return nil, grpc.Errorf(codes.InvalidArgument, "Cannot update deleted message %q", threadItemID)
	}
	if _, ok := item.Data.(*models.Message); !ok {
		return nil, grpc.Errorf(codes.InvalidArgument, "Cannot update non-message item %q", threadItemID)
	}

	req, err := createPostMessageRequest(ctx, item.ThreadID, in.ActorEntityID, in.Message)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if err := claimAttachments(ctx, s.mediaClient, s.paymentsClient, item.ThreadID, req.Attachments); err != nil {
		return nil, errors.Trace(err)
	}

	err = s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		if err := dl.UpdateMessage(ctx, item.ThreadID, item.ID, req); err != nil {
			return errors.Trace(err)
		}
		item, err := dl.ThreadItem(ctx, item.ID)
		if err != nil {
			return errors.Trace(err)
		}
		return errors.Trace(dl.CreateThreadItem(ctx, &models.ThreadItem{
			ThreadID:      item.ThreadID,
			ActorEntityID: in.ActorEntityID,
			Internal:      item.Internal,
			Data: &models.MessageUpdate{
				ThreadItemID: threadItemID.String(),
				Message:      item.Data.(*models.Message),
			},
		}))
	})
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "Thread item %q not found", in.ThreadItemID)
	}
	if err != nil {
		return nil, errors.Trace(err)
	}
	threads, err := s.dal.Threads(ctx, []models.ThreadID{item.ThreadID})
	if err != nil {
		golog.ContextLogger(ctx).Errorf("Failed to fetch thread %s: %s", item.ThreadID, err)
	} else if _, err := s.updateSavedQueriesForThread(ctx, threads[0]); err != nil {
		golog.ContextLogger(ctx).Errorf("Failed to updated saved query for thread %s: %s", item.ThreadID, err)
	}
	return &threading.UpdateMessageResponse{}, nil
}

// UpdateSavedQuery updated a saved query
func (s *threadsServer) UpdateSavedQuery(ctx context.Context, in *threading.UpdateSavedQueryRequest) (*threading.UpdateSavedQueryResponse, error) {
	id, err := models.ParseSavedQueryID(in.SavedQueryID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid SavedQueryID '%s'", in.SavedQueryID)
	}
	rebuild := in.ForceRebuild
	update := &dal.SavedQueryUpdate{}
	if in.Title != "" {
		update.Title = &in.Title
	}
	if in.Ordinal > 0 {
		update.Ordinal = ptr.Int(int(in.Ordinal))
	}
	// Unfortunate artifact of go defaults and no grpc/protobuf ptr support
	if in.NotificationsEnabled != threading.NOTIFICATIONS_ENABLED_UPDATE_NONE {
		switch in.NotificationsEnabled {
		case threading.NOTIFICATIONS_ENABLED_UPDATE_TRUE:
			update.NotificationsEnabled = ptr.Bool(true)
		case threading.NOTIFICATIONS_ENABLED_UPDATE_FALSE:
			update.NotificationsEnabled = ptr.Bool(false)
		default:
			return nil, grpc.Errorf(codes.InvalidArgument, "Invalid NotificationsEnabled value %s", in.NotificationsEnabled)
		}
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
	sq, err := s.dal.SavedQuery(ctx, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if rebuild {
		if err := s.rebuildSavedQuery(ctx, sq); err != nil {
			return nil, errors.Trace(err)
		}
	}
	if rebuild || update.NotificationsEnabled != nil {
		if err := s.dal.RebuildNotificationsSavedQuery(ctx, sq.EntityID); err != nil {
			golog.ContextLogger(ctx).Errorf("Failed to update notifications saved query for entity %s: %s", sq.EntityID, err)
		} else if err := s.notifyBadgeCountUpdate(ctx, []string{sq.EntityID}); err != nil {
			golog.ContextLogger(ctx).Errorf("Failed to notify entity %s of updated badge count: %s", sq.EntityID, err)
		}
	}

	sqResp, err := transformSavedQueryToResponse(sq)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.UpdateSavedQueryResponse{
		Query: sqResp,
	}, nil
}

func (s *threadsServer) Tags(ctx context.Context, in *threading.TagsRequest) (*threading.TagsResponse, error) {
	if in.OrganizationID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "OrganizationID required")
	}
	tags, err := s.dal.TagsForOrg(ctx, in.OrganizationID, in.Prefix)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.TagsResponse{Tags: transformTagsToResponse(tags)}, nil
}

// UpdateThread update thread members and info
func (s *threadsServer) UpdateThread(ctx context.Context, in *threading.UpdateThreadRequest) (*threading.UpdateThreadResponse, error) {
	if in.ActorEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "ActorEntityID required")
	}
	if id, ok := validateEntityIDs(in.AddMemberEntityIDs); !ok {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid entity ID %q when adding members", id)
	}
	if id, ok := validateEntityIDs(in.RemoveMemberEntityIDs); !ok {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid entity ID %q when removing members", id)
	}
	if id, ok := validateEntityIDs(in.AddFollowerEntityIDs); !ok {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid entity ID %q when adding followers", id)
	}
	if id, ok := validateEntityIDs(in.RemoveFollowerEntityIDs); !ok {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid entity ID %q when removing followers", id)
	}
	if tag, ok := validateTags(in.AddTags); !ok {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid tag %q when adding tags", tag)
	}
	if tag, ok := validateTags(in.RemoveTags); !ok {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid tag %q when removing tags", tag)
	}

	tid, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ThreadID '%s'", in.ThreadID)
	}

	threads, err := s.dal.Threads(ctx, []models.ThreadID{tid})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(threads) == 0 {
		return nil, grpc.Errorf(codes.NotFound, "Thread %q not found", tid)
	}
	thread := threads[0]

	// TODO: for now assume the thread exists only in a single org
	orgID := thread.OrganizationID

	// Verify authorization by checking actor is part of the memberslist
	// The acting entity can be the organization itself in which case it's allowed to modify any thread in the organization.
	if in.ActorEntityID != thread.OrganizationID {
		actorEntities, err := s.entityAndMemberships(ctx, in.ActorEntityID,
			[]directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_ORGANIZATION})
		if err != nil {
			return nil, errors.Trace(err)
		}
		if len(actorEntities) == 0 {
			return nil, grpc.Errorf(codes.InvalidArgument, "No entities found for actor")
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
			return nil, grpc.Errorf(codes.PermissionDenied, "Entity is not a member of thread %s", thread.ID)
		}
	}

	// can only update system title for an external thread
	switch thread.Type {
	case models.ThreadTypeTeam:
		if in.SystemTitle != "" {
			return nil, grpc.Errorf(codes.PermissionDenied, "Can only update system title for non-team thread")
		}
	default:
		if len(in.RemoveMemberEntityIDs) > 0 || len(in.AddMemberEntityIDs) > 0 {
			return nil, grpc.Errorf(codes.PermissionDenied, "Can only update members for a team thread")
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
		if len(in.AddTags) != 0 {
			if err := dl.AddThreadTags(ctx, orgID, thread.ID, in.AddTags); err != nil {
				return errors.Trace(err)
			}
		}
		if len(in.RemoveTags) != 0 {
			if err := dl.RemoveThreadTags(ctx, orgID, thread.ID, in.RemoveTags); err != nil {
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
	if _, err := s.updateSavedQueriesForThread(ctx, thread); err != nil {
		golog.Errorf("Failed to updated saved query for thread %s: %s", thread.ID, err)
	}

	// Notify all possible people affected about their new badge count
	conc.Go(func() {
		ctx := context.Background()
		members, err := s.membersForThread(ctx, thread.ID)
		if err != nil {
			golog.Errorf(err.Error())
			return
		}
		entityIDs := make([]string, 0, len(members)+len(in.RemoveMemberEntityIDs))
		for _, m := range members {
			entityIDs = append(entityIDs, m.EntityID)
		}
		for _, id := range in.RemoveMemberEntityIDs {
			entityIDs = append(entityIDs, id)
		}
		if err := s.notifyBadgeCountUpdate(ctx, entityIDs); err != nil {
			golog.Fatalf("Failed to notify entities of updated badge count: %s", err)
		}
	})

	th, err := transformThreadToResponse(thread, false)
	if err != nil {
		return nil, errors.Trace(err)
	}
	ts, err := s.hydrateThreadForViewer(ctx, []*threading.Thread{th}, in.ActorEntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(ts) == 0 {
		th = nil
	} else {
		th = ts[0]
	}
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
		Key: &directory.LookupEntitiesRequest_BatchEntityID{
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
			return "", errors.Errorf("Entity %q is not a member of org %q", e.ID, orgID)
		}
		names[i] = e.Info.DisplayName
	}
	return strings.Join(names, ", "), nil
}

func (s *threadsServer) forExternalViewer(ctx context.Context, viewerEntityID string) (bool, error) {
	forExternal := true
	if viewerEntityID != "" {
		ent, err := directory.SingleEntity(ctx, s.directoryClient, &directory.LookupEntitiesRequest{
			Key: &directory.LookupEntitiesRequest_EntityID{
				EntityID: viewerEntityID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
			},
		})
		if grpc.Code(err) == codes.NotFound {
			return false, grpc.Errorf(codes.NotFound, "Viewing entity %q not found", viewerEntityID)
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
