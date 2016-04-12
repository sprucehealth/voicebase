package server

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/api"
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
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
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

type threadsServer struct {
	clk                clock.Clock
	dal                dal.DAL
	sns                snsiface.SNSAPI
	snsTopicARN        string
	notificationClient notification.Client
	directoryClient    directory.DirectoryClient
	settingsClient     settings.SettingsClient
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
		webDomain:          webDomain,
	}
}

// CreateSavedQuery saves a query for later use
func (s *threadsServer) CreateSavedQuery(ctx context.Context, in *threading.CreateSavedQueryRequest) (*threading.CreateSavedQueryResponse, error) {
	if in.OrganizationID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "OrganizationID is required")
	}
	if in.EntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "EntityID is required")
	}

	sq := &models.SavedQuery{
		OrganizationID: in.OrganizationID,
		EntityID:       in.EntityID,
	}
	id, err := s.dal.CreateSavedQuery(ctx, sq)
	if err != nil {
		return nil, internalError(err)
	}
	sq.ID = id
	sqr, err := transformSavedQueryToResponse(sq)
	if err != nil {
		return nil, internalError(err)
	}
	return &threading.CreateSavedQueryResponse{
		SavedQuery: sqr,
	}, nil
}

// CreateEmptyThread create a new thread with no messages
func (s *threadsServer) CreateEmptyThread(ctx context.Context, in *threading.CreateEmptyThreadRequest) (*threading.CreateEmptyThreadResponse, error) {
	switch in.Type {
	case threading.ThreadType_EXTERNAL, threading.ThreadType_TEAM:
	default:
		return nil, grpcErrorf(codes.InvalidArgument, fmt.Sprintf("Type '%s' not allowed for CreateEmptyThread", in.Type.String()))
	}
	if in.OrganizationID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "OrganizationID is required")
	}
	if in.PrimaryEntityID == "" && in.Type != threading.ThreadType_TEAM {
		return nil, grpcErrorf(codes.InvalidArgument, "PrimaryEntityID is required for non-team threads")
	}
	if in.Summary == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "Summary is required")
	}
	in.Summary = textutil.TruncateUTF8(in.Summary, maxSummaryLength)
	if in.Type == threading.ThreadType_TEAM && in.FromEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "FromEntityID is required for TEAM threads")
	}

	if in.Type == threading.ThreadType_EXTERNAL && in.SystemTitle == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "SystemTitle is required for EXTERNAL threads")
	}

	tt, err := transformThreadTypeFromRequest(in.Type)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid thread type")
	}

	var systemTitle string
	switch in.Type {
	case threading.ThreadType_TEAM:
		systemTitle, err = s.teamThreadSystemTitle(ctx, in.OrganizationID, in.MemberEntityIDs)
		if err != nil {
			return nil, internalError(err)
		}
	case threading.ThreadType_SETUP:
		systemTitle = setupThreadTitle
	case threading.ThreadType_EXTERNAL:
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
		})
		if err != nil {
			return errors.Trace(err)
		}
		if in.Type == threading.ThreadType_TEAM {
			// Make sure posted is a member
			in.MemberEntityIDs = append(in.MemberEntityIDs, in.FromEntityID)
			if err := dl.UpdateThreadMembers(ctx, threadID, in.MemberEntityIDs); err != nil {
				return errors.Trace(err)
			}
		} else if in.FromEntityID != "" {
			if err := dl.UpdateThreadEntity(ctx, threadID, in.FromEntityID, nil); err != nil {
				return errors.Trace(err)
			}
		}
		return nil
	}); err != nil {
		return nil, grpcErrorf(codes.Internal, errors.Trace(err).Error())
	}
	thread, err := s.dal.Thread(ctx, threadID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	th, err := transformThreadToResponse(thread, false)
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
	case threading.ThreadType_EXTERNAL, threading.ThreadType_TEAM:
	default:
		return nil, grpcErrorf(codes.InvalidArgument, fmt.Sprintf("Type '%s' not allowed for CreateThread", in.Type.String()))
	}
	if in.OrganizationID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "OrganizationID is required")
	}
	if in.FromEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "FromEntityID is required")
	}
	if in.Type == threading.ThreadType_EXTERNAL && in.SystemTitle == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "SystemTitle is required")
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

	var systemTitle string
	switch in.Type {
	case threading.ThreadType_TEAM:
		systemTitle, err = s.teamThreadSystemTitle(ctx, in.OrganizationID, in.MemberEntityIDs)
		if err != nil {
			return nil, internalError(err)
		}
	case threading.ThreadType_SETUP:
		systemTitle = setupThreadTitle
	case threading.ThreadType_EXTERNAL:
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
		})
		if err != nil {
			return errors.Trace(err)
		}
		if in.Type == threading.ThreadType_TEAM {
			// Make sure posted is a member
			in.MemberEntityIDs = append(in.MemberEntityIDs, in.FromEntityID)
			if err := dl.UpdateThreadMembers(ctx, threadID, in.MemberEntityIDs); err != nil {
				return errors.Trace(err)
			}
		} else {
			if err := dl.UpdateThreadEntity(ctx, threadID, in.FromEntityID, nil); err != nil {
				return errors.Trace(err)
			}
		}

		// Update unread reference status for anyone mentioned
		for _, r := range textRefs {
			if err := dl.UpdateThreadEntity(ctx, threadID, r.ID, &dal.ThreadEntityUpdate{
				LastReferenced: ptr.Time(s.clk.Now()),
			}); err != nil {
				return errors.Trace(err)
			}
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
		return errors.Trace(err)
	}); err != nil {
		return nil, internalError(err)
	}
	thread, err := s.dal.Thread(ctx, threadID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	th, err := transformThreadToResponse(thread, !in.Internal)
	if err != nil {
		return nil, errors.Trace(err)
	}
	it, err := transformThreadItemToResponse(item, thread.OrganizationID)
	if err != nil {
		return nil, internalError(err)
	}
	s.publishMessage(ctx, in.OrganizationID, in.FromEntityID, threadID, it, in.UUID)
	s.notifyMembersOfPublishMessage(ctx, thread.OrganizationID, models.EmptySavedQueryID(), thread, item.ID, in.FromEntityID)
	return &threading.CreateThreadResponse{
		ThreadID:   threadID.String(),
		ThreadItem: it,
		Thread:     th,
	}, nil
}

func (s *threadsServer) CreateLinkedThreads(ctx context.Context, in *threading.CreateLinkedThreadsRequest) (*threading.CreateLinkedThreadsResponse, error) {
	if in.Type != threading.ThreadType_SUPPORT {
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
	thread1, err := s.dal.Thread(ctx, thread1ID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	th1, err := transformThreadToResponse(thread1, false)
	if err != nil {
		return nil, errors.Trace(err)
	}
	thread2, err := s.dal.Thread(ctx, thread2ID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	th2, err := transformThreadToResponse(thread2, false)
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
	thread, err := s.dal.Thread(ctx, threadID)
	if api.IsErrNotFound(err) {
		return &threading.DeleteThreadResponse{}, nil
	} else if err != nil {
		return nil, internalError(err)
	}

	if thread.PrimaryEntityID != "" {
		// Get the primary entity on the thread first and determine if we need to delete it if it's external
		resp, err := s.directoryClient.LookupEntities(ctx, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: thread.PrimaryEntityID,
			},
		})
		if err != nil && grpc.Code(err) != codes.NotFound {
			return nil, internalError(err)
		}

		if resp != nil &&
			len(resp.Entities) != 0 &&
			resp.Entities[0].Type == directory.EntityType_EXTERNAL &&
			resp.Entities[0].Status != directory.EntityStatus_DELETED {
			if _, err := s.directoryClient.DeleteEntity(ctx, &directory.DeleteEntityRequest{
				EntityID: resp.Entities[0].ID,
			}); err != nil {
				return nil, internalError(err)
			}
		}
	}
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		if err := s.dal.DeleteThread(ctx, threadID); err != nil {
			return errors.Trace(err)
		}
		return errors.Trace(s.dal.RecordThreadEvent(ctx, threadID, in.ActorEntityID, models.ThreadEventDelete))
	}); err != nil {
		return nil, internalError(err)
	}
	return &threading.DeleteThreadResponse{}, nil
}

// MarkThreadAsRead marks all posts in a thread as read by an entity
func (s *threadsServer) MarkThreadAsRead(ctx context.Context, in *threading.MarkThreadAsReadRequest) (*threading.MarkThreadAsReadResponse, error) {
	if in.ThreadID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "ThreadID is required")
	}
	threadID, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid ThreadID")
	}
	if in.EntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "EntityID is required")
	}
	readTime := s.clk.Now()
	if in.Timestamp != 0 {
		readTime = time.Unix(int64(in.Timestamp), 0)
	}

	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		threadEntities, err := dl.ThreadEntities(ctx, []models.ThreadID{threadID}, in.EntityID, dal.ForUpdate)
		if err != nil {
			return errors.Trace(err)
		}
		tid := threadID.String()
		lastViewed := time.Unix(0, 0)
		if len(threadEntities) == 1 && threadEntities[tid].LastViewed != nil {
			lastViewed = *threadEntities[tid].LastViewed
		}

		// Update our timestamp or create one if it isn't already there
		if err := dl.UpdateThreadEntity(ctx, threadID, in.EntityID, &dal.ThreadEntityUpdate{LastViewed: &readTime}); err != nil {
			return errors.Trace(err)
		}

		threadItemIDs, err := dl.ThreadItemIDsCreatedAfter(ctx, threadID, lastViewed)
		if err != nil {
			return errors.Trace(err)
		}

		tivds := make([]*models.ThreadItemViewDetails, len(threadItemIDs))
		for i, tiid := range threadItemIDs {
			tivds[i] = &models.ThreadItemViewDetails{
				ThreadItemID:  tiid,
				ActorEntityID: in.EntityID,
				ViewTime:      &readTime,
			}
		}
		return errors.Trace(dl.CreateThreadItemViewDetails(ctx, tivds))
	}); err != nil {
		return nil, internalError(err)
	}
	return &threading.MarkThreadAsReadResponse{}, nil
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
			return nil, grpcErrorf(codes.InvalidArgument, fmt.Sprintf("Title is invalid format: %s", err.Error()))
		}
	}
	var textRefs []*models.Reference
	in.Text, textRefs, err = parseRefsAndNormalize(in.Text)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, fmt.Sprintf("Text is invalid format: %s", errors.Cause(err).Error()))
	}

	thread, err := s.dal.Thread(ctx, threadID)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpcErrorf(codes.NotFound, "Thread not found")
	}
	prePostLastMessageTimestamp := thread.LastMessageTimestamp

	linkedThread, prependSender, err := s.dal.LinkedThread(ctx, threadID)
	if err != nil && errors.Cause(err) != dal.ErrNotFound {
		return nil, internalError(err)
	}

	var item *models.ThreadItem
	var linkedItem *models.ThreadItem
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		// TODO: validate any attachments
		attachments, err := transformAttachmentsFromRequest(in.Attachments)
		if err != nil {
			return internalError(err)
		}
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
			return internalError(err)
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
			return internalError(err)
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
			return internalError(err)
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
				return internalError(err)
			}
		}

		return nil
	}); err != nil {
		return nil, internalError(err)
	}
	thread, err = s.dal.Thread(ctx, threadID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	th, err := transformThreadToResponse(thread, !in.Internal)
	if err != nil {
		return nil, errors.Trace(err)
	}
	it, err := transformThreadItemToResponse(item, thread.OrganizationID)
	if err != nil {
		return nil, internalError(err)
	}
	s.publishMessage(ctx, thread.OrganizationID, thread.PrimaryEntityID, threadID, it, in.UUID)
	s.notifyMembersOfPublishMessage(ctx, thread.OrganizationID, models.EmptySavedQueryID(), thread, item.ID, in.FromEntityID)
	if linkedItem != nil {
		it2, err := transformThreadItemToResponse(linkedItem, linkedThread.OrganizationID)
		if err != nil {
			return nil, internalError(err)
		}
		s.publishMessage(ctx, linkedThread.OrganizationID, linkedThread.PrimaryEntityID, linkedThread.ID, it2, "")
		s.notifyMembersOfPublishMessage(ctx, linkedThread.OrganizationID, models.EmptySavedQueryID(), linkedThread, linkedItem.ID, linkedItem.ActorEntityID)
	}
	return &threading.PostMessageResponse{
		Item:   it,
		Thread: th,
	}, nil
}

// QueryThreads queries the list of threads in an organization
func (s *threadsServer) QueryThreads(ctx context.Context, in *threading.QueryThreadsRequest) (*threading.QueryThreadsResponse, error) {
	// TODO: ignoring query entirely for now and returning all threads in an org instead
	d := dal.FromStart
	if in.Iterator.Direction == threading.Iterator_FROM_END {
		d = dal.FromEnd
	}
	forExternal := false // TODO: set to true for EXTERNAL entities
	ir, err := s.dal.IterateThreads(ctx, in.OrganizationID, in.ViewerEntityID, forExternal, &dal.Iterator{
		StartCursor: in.Iterator.StartCursor,
		EndCursor:   in.Iterator.EndCursor,
		Direction:   d,
		Count:       int(in.Iterator.Count),
	})
	if e, ok := errors.Cause(err).(dal.ErrInvalidIterator); ok {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid iterator: "+string(e))
	} else if err != nil {
		return nil, internalError(err)
	}
	res := &threading.QueryThreadsResponse{
		Edges:   make([]*threading.ThreadEdge, len(ir.Edges)),
		HasMore: ir.HasMore,
	}

	// If a ViewerEntityID is provided, track the pointers to all our threads so we can fetch their read/unread status
	var ths []*threading.Thread
	if in.ViewerEntityID != "" {
		ths = make([]*threading.Thread, len(ir.Edges))
	}
	for i, e := range ir.Edges {
		th, err := transformThreadToResponse(e.Thread, forExternal)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if in.ViewerEntityID != "" {
			ths[i] = th
		}
		if in.ViewerEntityID != "" && th.MessageCount != 0 {
			te := e.ThreadEntity
			th.Unread = te == nil || te.LastViewed == nil || (th.LastMessageTimestamp > uint64(te.LastViewed.Unix()))
			th.UnreadReference = te != nil && te.LastReferenced != nil && (te.LastViewed == nil || te.LastReferenced.After(*te.LastViewed))
		}
		res.Edges[i] = &threading.ThreadEdge{
			Thread: th,
			Cursor: e.Cursor,
		}
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
		return nil, internalError(err)
	}
	sq, err := transformSavedQueryToResponse(query)
	if err != nil {
		return nil, internalError(err)
	}
	return &threading.SavedQueryResponse{
		SavedQuery: sq,
	}, nil
}

// SavedQueries returns the list of saved queries for an org / entity pair
func (s *threadsServer) SavedQueries(ctx context.Context, in *threading.SavedQueriesRequest) (*threading.SavedQueriesResponse, error) {
	queries, err := s.dal.SavedQueries(ctx, in.EntityID)
	if err != nil {
		return nil, internalError(err)
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

	forExternal := false // TODO: set to true for EXTERNAL entities

	thread, err := s.dal.Thread(ctx, tid)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpcErrorf(codes.NotFound, "Thread not found")
	} else if err != nil {
		return nil, internalError(err)
	}
	th, err := transformThreadToResponse(thread, forExternal)
	if err != nil {
		return nil, internalError(err)
	}
	if in.ViewerEntityID != "" {
		golog.Debugf("Populating viewer information for (entity_id, thread) (%s,%s)", in.ViewerEntityID, th.ID)
		ts, err := s.hydrateThreadForViewer(ctx, []*threading.Thread{th}, in.ViewerEntityID)
		if err != nil {
			return nil, internalError(err)
		}
		if len(ts) == 0 {
			return nil, grpcErrorf(codes.NotFound, "Thread not found")
		}
		// TODO: for now can't require the viewer since the graphql service requests the thread to get the org ID before it can know the entity viewing
		// } else if th.Type == threading.ThreadType_TEAM {
		// 	// Require a viewer entity for private threads
		// 	return nil, grpcErrorf(codes.NotFound, "Thread not found")
	} else {
		golog.Debugf("No viewer entity information for thread %s", in.ThreadID)
	}
	return &threading.ThreadResponse{
		Thread: th,
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
		return nil, internalError(err)
	}

	th, err := s.dal.Thread(ctx, item.ThreadID)
	if api.IsErrNotFound(err) {
		return nil, grpcErrorf(codes.NotFound, "Thread %s not found", tid)
	} else if err != nil {
		return nil, internalError(err)
	}

	ti, err := transformThreadItemToResponse(item, th.OrganizationID)
	if err != nil {
		return nil, internalError(err)
	}
	return &threading.ThreadItemResponse{
		Item: ti,
	}, nil
}

// ThreadsForMember looks up a list of threads by entity membership
func (s *threadsServer) ThreadsForMember(ctx context.Context, in *threading.ThreadsForMemberRequest) (*threading.ThreadsForMemberResponse, error) {
	threads, err := s.dal.ThreadsForMember(ctx, in.EntityID, in.PrimaryOnly)
	if err != nil {
		return nil, internalError(err)
	}

	forExternal := false // TODO: set to true for EXTERNAL entities

	res := &threading.ThreadsForMemberResponse{
		Threads: make([]*threading.Thread, len(threads)),
	}
	for i, t := range threads {
		th, err := transformThreadToResponse(t, forExternal)
		if err != nil {
			return nil, internalError(err)
		}
		res.Threads[i] = th
	}
	res.Threads, err = s.hydrateThreadForViewer(ctx, res.Threads, in.EntityID)
	if err != nil {
		return nil, internalError(err)
	}
	return res, nil
}

// ThreadItems returns the items (messages or events) in a thread
func (s *threadsServer) ThreadItems(ctx context.Context, in *threading.ThreadItemsRequest) (*threading.ThreadItemsResponse, error) {
	tid, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid ThreadID")
	}

	th, err := s.dal.Thread(ctx, tid)
	if api.IsErrNotFound(err) {
		return nil, grpcErrorf(codes.NotFound, "Thread %s not found", tid)
	} else if err != nil {
		return nil, internalError(err)
	}

	forExternal := false // TODO: set to true for EXTERNAL entities

	d := dal.FromStart
	if in.Iterator.Direction == threading.Iterator_FROM_END {
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
		return nil, internalError(err)
	}
	res := &threading.ThreadItemsResponse{
		Edges:   make([]*threading.ThreadItemEdge, len(ir.Edges)),
		HasMore: ir.HasMore,
	}
	for i, e := range ir.Edges {
		it, err := transformThreadItemToResponse(e.Item, th.OrganizationID)
		if err != nil {
			return nil, internalError(err)
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
		return nil, internalError(err)
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
		return nil, internalError(err)
	}
	res := &threading.ThreadMembersResponse{
		Members: make([]*threading.Member, 0, len(tes)),
	}
	for _, te := range tes {
		if te.Member {
			res.Members = append(res.Members, &threading.Member{
				EntityID: te.EntityID,
			})
		}
	}
	return res, nil
}

// UpdateSavedQuery updated a saved query
func (s *threadsServer) UpdateSavedQuery(ctx context.Context, in *threading.UpdateSavedQueryRequest) (*threading.UpdateSavedQueryResponse, error) {
	return nil, grpcErrorf(codes.Unimplemented, "UpdateSavedQuery not implemented")
}

// UpdateThread update thread members and info
func (s *threadsServer) UpdateThread(ctx context.Context, in *threading.UpdateThreadRequest) (*threading.UpdateThreadResponse, error) {
	tid, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid ThreadItemID")
	}

	thread, err := s.dal.Thread(ctx, tid)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpcErrorf(codes.NotFound, "Thread not found")
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	// can only update system title for an external thread
	switch thread.Type {
	case models.ThreadTypeTeam:
	default:
		if len(in.RemoveMemberEntityIDs) > 0 || len(in.AddMemberEntityIDs) > 0 || in.SystemTitle == "" {
			return nil, grpcErrorf(codes.PermissionDenied, "Can only update system title for non team threads")
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
		entities, err := s.dal.EntitiesForThread(ctx, tid)
		if err != nil {
			return nil, internalError(err)
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
			return nil, internalError(err)
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
		if len(in.AddMemberEntityIDs) != 0 || len(in.RemoveMemberEntityIDs) != 0 {
			if err := dl.UpdateThreadMembers(ctx, tid, memberIDs); err != nil {
				return errors.Trace(err)
			}
			update.SystemTitle = &systemTitle
		}
		if update.UserTitle != nil || update.SystemTitle != nil {
			if err := dl.UpdateThread(ctx, tid, update); err != nil {
				return errors.Trace(err)
			}
		}
		return nil
	}); err != nil {
		return nil, internalError(err)
	}
	thread, err = s.dal.Thread(ctx, tid)
	if err != nil {
		return nil, internalError(err)
	}
	th, err := transformThreadToResponse(thread, false)
	if err != nil {
		return nil, internalError(err)
	}
	return &threading.UpdateThreadResponse{
		Thread: th,
	}, nil
}

func (s *threadsServer) hydrateThreadForViewer(ctx context.Context, ts []*threading.Thread, viewerEntityID string) ([]*threading.Thread, error) {
	tIDs := make([]models.ThreadID, 0, len(ts))
	for _, t := range ts {
		if t.MessageCount > 0 || t.Type == threading.ThreadType_TEAM {
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
		if t.Type != threading.ThreadType_TEAM || (te != nil && te.Member) {
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
			return "", errors.Trace(fmt.Errorf("Entity %s is not a member of org %s", e.ID, orgID))
		}
		names[i] = e.Info.DisplayName
	}
	return strings.Join(names, ", "), nil
}

const newMessageNotificationKey = "new_message" // This is used for both collapse and dedupe

func (s *threadsServer) notifyMembersOfPublishMessage(ctx context.Context, orgID string, savedQueryID models.SavedQueryID, thread *models.Thread, messageID models.ThreadItemID, publishingEntityID string) {
	if s.notificationClient == nil || s.directoryClient == nil {
		golog.Debugf("Member notification aborted because either notification client or directory client is not configured")
		return
	}
	if orgID == "" || thread == nil || !messageID.IsValid {
		golog.Errorf("Invalid message information for notification: %v, %v, %v, %v", orgID, savedQueryID, thread, messageID)
		return
	}
	golog.Debugf("Notifying members of org %s of activity on thread %s by entity %s", orgID, thread.ID, publishingEntityID)
	conc.Go(func() {
		ctx = context.Background()

		threadEntities, err := s.dal.EntitiesForThread(ctx, thread.ID)
		if err != nil {
			golog.Errorf("Failed to get entities for thread %s: %s", thread.ID, err)
			return
		}

		// Figure out who should receive notifications
		var entities []*directory.Entity
		if thread.Type == models.ThreadTypeTeam {
			entIDs := make([]string, 0, len(threadEntities))
			for _, te := range threadEntities {
				if te.Member {
					entIDs = append(entIDs, te.EntityID)
				}
			}
			if len(entIDs) != 0 {
				resp, err := s.directoryClient.LookupEntities(ctx, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
						BatchEntityID: &directory.IDList{
							IDs: entIDs,
						},
					},
					RequestedInformation: &directory.RequestedInformation{
						Depth:             0,
						EntityInformation: []directory.EntityInformation{},
					},
				})
				if err != nil {
					golog.Errorf("Failed to fetch entities to notify about thread %s: %s", thread.ID, err)
					return
				}
				entities = resp.Entities
			}
		} else {
			// TODO: for now treating all other types the same which is the old behavior
			// Lookup all members of the org this thread belongs to and notify them of the new message unless they published it
			resp, err := s.directoryClient.LookupEntities(ctx, &directory.LookupEntitiesRequest{
				LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
				LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
					EntityID: orgID,
				},
				RequestedInformation: &directory.RequestedInformation{
					Depth:             0,
					EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS},
				},
			})
			if err != nil {
				golog.Errorf("Failed to fetch org members of %s to notify about thread %s: %s", orgID, thread.ID, err)
				return
			}
			if len(resp.Entities) != 1 {
				golog.Errorf("Expected to find 1 org for ID %s but found %d", orgID, len(resp.Entities))
				return
			}
			org := resp.Entities[0]
			for _, m := range org.Members {
				if m.Type == directory.EntityType_INTERNAL && m.ID != publishingEntityID {
					entities = append(entities, m)
				}
			}
		}

		if len(entities) == 0 {
			return
		}

		teMap := make(map[string]*models.ThreadEntity, len(threadEntities))
		for _, te := range threadEntities {
			teMap[te.EntityID] = te
		}

		notificationText := s.getNotificationText(ctx, thread, messageID)

		// Track the messages we want to send and how many unread threads there were
		messages := make(map[string]string)

		// Get the unread and notification information
		if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
			// Update the memberships for everyone who needs to be notified
			// Note: It takes human interaction for this update state to trigger so shouldn't be too often.
			now := s.clk.Now()
			for _, ent := range entities {
				if ent.ID == publishingEntityID {
					continue
				}

				te := teMap[ent.ID]

				if s.isAlertAllMessagesEnabled(ctx, ent.ID) {
					messages[ent.ID] = notificationText
				} else if te == nil || te.LastUnreadNotify == nil || (te.LastViewed != nil && te.LastViewed.After(*te.LastUnreadNotify)) {
					// Only send a notification if no notification has been sent or the person has viewed the thread since the last notification
					if err := dl.UpdateThreadEntity(ctx, thread.ID, ent.ID, &dal.ThreadEntityUpdate{
						LastUnreadNotify: &now,
					}); err != nil {
						return errors.Trace(err)
					}
					messages[ent.ID] = notificationText
				}
			}
			return nil
		}); err != nil {
			golog.Errorf("Encountered error while calculating and updating unread and notify status: %s", err)
			return
		}

		// Note: We always send the unread push to all interested entities.
		//   This is because clients rely on the push to update state.
		//   An empty ShortMessage for an entity indicated that a notification
		//   should be sent silently or flagged as such.
		//   Notifications with ShortMEssages for the entity will be displayed to the user
		if err := s.notificationClient.SendNotification(&notification.Notification{
			// UnreadCounts:     unreadCounts, TODO: currently don't support counts
			ShortMessages:    messages,
			OrganizationID:   orgID,
			SavedQueryID:     savedQueryID.String(),
			ThreadID:         thread.ID.String(),
			MessageID:        messageID.String(),
			EntitiesToNotify: directory.EntityIDs(entities),
			// Note: Parameterizing with these may not be the best. The notification infterface needs to be
			//   rethought, but going with this for now
			DedupeKey:   newMessageNotificationKey,
			CollapseKey: newMessageNotificationKey,
		}); err != nil {
			golog.Errorf("Failed to notify members: %s", err)
		}
	})
}

func (s *threadsServer) getNotificationText(ctx context.Context, thread *models.Thread, messageID models.ThreadItemID) string {
	notificationText := "You have a new message"
	isClearText := s.isClearTextMessageNotificationsEnabled(ctx, thread.OrganizationID) || thread.Type == models.ThreadTypeSupport
	if isClearText {
		message, err := s.dal.ThreadItem(ctx, messageID)
		if err != nil {
			golog.Errorf("Encountered error when attempting to get message for clear text notification for message id %s", messageID)
			return notificationText
		}
		if message.Type == models.ItemTypeMessage {
			msg, ok := message.Data.(*models.Message)
			if !ok {
				golog.Errorf("Failed to convert thread item data to message for clear text notification for item id %s", messageID)
				return notificationText
			}
			bmlText, err := bml.Parse(msg.Text)
			if err != nil {
				golog.Errorf("Failed to convert thread item data to message for clear text notification for item id %s: %s", messageID, err)
				return notificationText
			}
			plainText, err := bmlText.PlainText()
			if err != nil {
				golog.Errorf("Failed to convert thread item data to message for clear text notification for item id %s: %s", messageID, err)
				return notificationText
			}
			notificationText = plainText
			if thread.Type != models.ThreadTypeSupport {
				if thread.UserTitle != "" {
					notificationText = thread.UserTitle + ": " + notificationText
				} else {
					notificationText = thread.SystemTitle + ": " + notificationText
				}
			}
			if len(notificationText) > 256 {
				notificationText = textutil.TruncateUTF8(notificationText, 253) + "..."
			}
			return notificationText
		}
	}
	return notificationText
}

func (s *threadsServer) isAlertAllMessagesEnabled(ctx context.Context, entityID string) bool {

	booleanValue, err := settings.GetBooleanValue(ctx, s.settingsClient, &settings.GetValuesRequest{
		Keys:   []*settings.ConfigKey{{Key: threading.AlertAllMessages}},
		NodeID: entityID,
	})
	if err != nil {
		golog.Errorf("Encountered an error when getting AlertAllMessages for entity %s: %s", entityID, err)
		return true
	}

	return booleanValue.Value
}

func (s *threadsServer) isClearTextMessageNotificationsEnabled(ctx context.Context, organizationID string) bool {

	booleanValue, err := settings.GetBooleanValue(ctx, s.settingsClient, &settings.GetValuesRequest{
		Keys:   []*settings.ConfigKey{{Key: threading.ClearTextMessageNotifications}},
		NodeID: organizationID,
	})
	if err != nil {
		golog.Errorf("Encountered an error when getting ClearTextMessageNotifications for org %s: %s", organizationID, err)
		return false
	}

	return booleanValue.Value
}

func parseRefsAndNormalize(s string) (string, []*models.Reference, error) {
	if s == "" {
		return "", nil, nil
	}
	b, err := bml.Parse(s)
	if err != nil {
		return "", nil, errors.Trace(err)
	}
	var refs []*models.Reference
	for _, e := range b {
		if r, ok := e.(*bml.Ref); ok {
			switch r.Type {
			case bml.EntityRef:
				refs = append(refs, &models.Reference{
					ID:   r.ID,
					Type: models.Reference_ENTITY,
				})
			default:
				return "", nil, errors.Trace(fmt.Errorf("unknown reference type %s", r.Type))
			}
		}
	}
	s, err = b.Format()
	if err != nil {
		return "", nil, errors.Trace(err)
	}
	return s, refs, nil
}

func internalError(err error) error {
	golog.LogDepthf(-1, golog.ERR, err.Error())
	return grpcErrorf(codes.Internal, errors.Trace(err).Error())
}
