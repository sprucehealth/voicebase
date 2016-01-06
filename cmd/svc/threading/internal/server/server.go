package server

import (
	"encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type threadsServer struct {
	dal         dal.DAL
	sns         snsiface.SNSAPI
	snsTopicARN string
}

func NewThreadsServer(dal dal.DAL, sns snsiface.SNSAPI, snsTopicARN string) threading.ThreadsServer {
	return &threadsServer{dal: dal, sns: sns, snsTopicARN: snsTopicARN}
}

// CreateSavedQuery saves a query for later use
func (s *threadsServer) CreateSavedQuery(ctx context.Context, in *threading.CreateSavedQueryRequest) (*threading.CreateSavedQueryResponse, error) {
	if in.OrganizationID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "OrganizationID is required")
	}
	if in.EntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "EntityID is required")
	}

	sq := &models.SavedQuery{
		OrganizationID: in.OrganizationID,
		EntityID:       in.EntityID,
	}
	id, err := s.dal.CreateSavedQuery(ctx, sq)
	if err != nil {
		return nil, errors.Trace(err)
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

// CreateThread create a new thread with an initial message
func (s *threadsServer) CreateThread(ctx context.Context, in *threading.CreateThreadRequest) (out *threading.CreateThreadResponse, err error) {
	if golog.Default().L(golog.DEBUG) {
		defer func() {
			golog.Debugf("CreateThread REQUEST %+v\n", in)
			golog.Debugf("CreateThread RESPONSE %+v\n", out)
		}()
	}
	// TODO: return proper error responses for invalid request
	if in.OrganizationID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "OrganizationID is required")
	}
	if in.FromEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "FromEntityID is required")
	}
	if in.Source == nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Source is required")
	}
	if in.Text == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Text is required")
	}
	// TODO: validate any attachments
	var threadID models.ThreadID
	var item *models.ThreadItem
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		var err error
		threadID, err = dl.CreateThread(ctx, &models.Thread{
			OrganizationID:  in.OrganizationID,
			PrimaryEntityID: in.FromEntityID,
		})
		if err != nil {
			return errors.Trace(err)
		}
		// The creator of the thread automatically becomes a follower
		if err := dl.UpdateMember(ctx, threadID, in.FromEntityID, &dal.MemberUpdate{
			Following: ptr.Bool(true),
		}); err != nil {
			return errors.Trace(err)
		}
		req := &dal.PostMessageRequest{
			ThreadID:     threadID,
			FromEntityID: in.FromEntityID,
			Internal:     in.Internal,
			Text:         in.Text,
			// TODO: safer transform for Endpoint.Type
			Source: &models.Endpoint{
				Channel: models.Endpoint_Channel(models.Endpoint_Channel_value[in.Source.Channel.String()]),
				ID:      in.Source.ID,
			},
		}
		req.Attachments, err = transformAttachmentsFromRequest(in.Attachments)
		if err != nil {
			return errors.Trace(err)
		}
		for _, dc := range in.Destinations {
			req.Destinations = append(req.Destinations, &models.Endpoint{
				Channel: models.Endpoint_Channel(models.Endpoint_Channel_value[dc.Channel.String()]),
				ID:      dc.ID,
			})
		}
		item, err = dl.PostMessage(ctx, req)
		return errors.Trace(err)
	}); err != nil {
		return nil, errors.Trace(err)
	}
	it, err := transformThreadItemToResponse(item)
	if err != nil {
		return nil, errors.Trace(err)
	}
	s.publishMessage(ctx, in.OrganizationID, in.FromEntityID, threadID, it)
	return &threading.CreateThreadResponse{
		ThreadID:   threadID.String(),
		ThreadItem: it,
	}, nil
}

// DeleteMessage deletes a message from a thread
func (s *threadsServer) DeleteMessage(context.Context, *threading.DeleteMessageRequest) (*threading.DeleteMessageResponse, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "DeleteMessage not implemented")
}

// MarThreadAsRead marks all posts in a thread as read by an entity
func (s *threadsServer) MarkThreadAsRead(context.Context, *threading.MarkThreadAsReadRequest) (*threading.MarkThreadAsReadResponse, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "MarkThreadAsRead not implemented")
}

// PostMessage posts a message into a specified thread
func (s *threadsServer) PostMessage(ctx context.Context, in *threading.PostMessageRequest) (out *threading.PostMessageResponse, err error) {
	if golog.Default().L(golog.DEBUG) {
		defer func() {
			golog.Debugf("PostMessage REQUEST %+v\n", in)
			golog.Debugf("PostMessage RESPONSE %+v\n", out)
		}()
	}

	// TODO: return proper error responses for invalid request
	if in.ThreadID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "ThreadID is required")
	}
	threadID, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ThreadID")
	}
	if in.FromEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "FromEntityID is required")
	}
	if in.Source == nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Source is required")
	}
	if in.Text == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Text is required")
	}

	thread, err := s.dal.Thread(ctx, threadID)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "Thread not found")
	}

	var item *models.ThreadItem
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		// TODO: validate any attachments
		req := &dal.PostMessageRequest{
			ThreadID:     threadID,
			FromEntityID: in.FromEntityID,
			Internal:     in.Internal,
			Text:         in.Text,
			// TODO: safer transform for Endpoint.Channel
			Source: &models.Endpoint{
				Channel: models.Endpoint_Channel(models.Endpoint_Channel_value[in.Source.Channel.String()]),
				ID:      in.Source.ID,
			},
		}
		req.Attachments, err = transformAttachmentsFromRequest(in.Attachments)
		if err != nil {
			return errors.Trace(err)
		}
		for _, dc := range in.Destinations {
			req.Destinations = append(req.Destinations, &models.Endpoint{
				Channel: models.Endpoint_Channel(models.Endpoint_Channel_value[dc.Channel.String()]),
				ID:      dc.ID,
			})
		}
		var err error
		item, err = s.dal.PostMessage(ctx, req)
		if err != nil {
			return errors.Trace(err)
		}
		// The poster is recorded as a member if necessary but does not become a follower
		if err := dl.UpdateMember(ctx, threadID, in.FromEntityID, nil); err != nil {
			return errors.Trace(err)
		}
		return nil
	}); err != nil {
		return nil, errors.Trace(err)
	}

	it, err := transformThreadItemToResponse(item)
	if err != nil {
		return nil, errors.Trace(err)
	}
	s.publishMessage(ctx, thread.OrganizationID, thread.PrimaryEntityID, threadID, it)
	return &threading.PostMessageResponse{
		Item: it,
	}, nil
}

// QueryThreads queries the list of threads in an organization
func (s *threadsServer) QueryThreads(ctx context.Context, in *threading.QueryThreadsRequest) (*threading.QueryThreadsResponse, error) {
	// TODO: ignoring query entirely for now and returning all threads in an org instead
	d := dal.FromStart
	if in.Iterator.Direction == threading.Iterator_FROM_END {
		d = dal.FromEnd
	}
	ir, err := s.dal.IterateThreads(ctx, in.OrganizationID, &dal.Iterator{
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
	res := &threading.QueryThreadsResponse{
		Edges:   make([]*threading.ThreadEdge, len(ir.Edges)),
		HasMore: ir.HasMore,
	}
	for i, e := range ir.Edges {
		res.Edges[i] = &threading.ThreadEdge{
			Thread: &threading.Thread{
				ID:              e.Thread.ID.String(),
				OrganizationID:  e.Thread.OrganizationID,
				PrimaryEntityID: e.Thread.PrimaryEntityID,
			},
			Cursor: e.Cursor,
		}
	}
	return res, nil
}

// SavedQuery returns a single saved query by ID
func (s *threadsServer) SavedQuery(ctx context.Context, in *threading.SavedQueryRequest) (*threading.SavedQueryResponse, error) {
	id, err := models.ParseSavedQueryID(in.SavedQueryID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid SavedQueryID")
	}
	query, err := s.dal.SavedQuery(ctx, id)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "Saved query not found")
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
	tid, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ThreadID")
	}

	thread, err := s.dal.Thread(ctx, tid)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "Thread not found")
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	th, err := transformThreadToResponse(thread)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.ThreadResponse{
		Thread: th,
	}, nil
}

// ThreadItem looks up and returns a single thread item by ID
func (s *threadsServer) ThreadItem(ctx context.Context, in *threading.ThreadItemRequest) (*threading.ThreadItemResponse, error) {
	tid, err := models.ParseThreadItemID(in.ItemID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ItemID")
	}

	item, err := s.dal.ThreadItem(ctx, tid)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "Thread item not found")
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	ti, err := transformThreadItemToResponse(item)
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
	res := &threading.ThreadsForMemberResponse{
		Threads: make([]*threading.Thread, len(threads)),
	}
	for i, t := range threads {
		th, err := transformThreadToResponse(t)
		if err != nil {
			return nil, errors.Trace(err)
		}
		res.Threads[i] = th
	}
	return res, nil
}

// ThreadItems returns the items (messages or events) in a thread
func (s *threadsServer) ThreadItems(ctx context.Context, in *threading.ThreadItemsRequest) (*threading.ThreadItemsResponse, error) {
	tid, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ThreadID")
	}

	d := dal.FromStart
	if in.Iterator.Direction == threading.Iterator_FROM_END {
		d = dal.FromEnd
	}
	ir, err := s.dal.IterateThreadItems(ctx, tid, &dal.Iterator{
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
		it, err := transformThreadItemToResponse(e.Item)
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

// ThreadMembers returns the members of a thread
func (s *threadsServer) ThreadMembers(ctx context.Context, in *threading.ThreadMembersRequest) (*threading.ThreadMembersResponse, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "ThreadMembers not implemented")
}

// UpdateSavedQuery updated a saved query
func (s *threadsServer) UpdateSavedQuery(ctx context.Context, in *threading.UpdateSavedQueryRequest) (*threading.UpdateSavedQueryResponse, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "UpdateSavedQuery not implemented")
}

// UpdateThreadMembership updates the membership status of an entity on a thread
func (s *threadsServer) UpdateThreadMembership(ctx context.Context, in *threading.UpdateThreadMembershipRequest) (*threading.UpdateThreadMembershipResponse, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "UpdateThreadMembership not implemented")
}

func (s *threadsServer) publishMessage(ctx context.Context, orgID, primaryEntityID string, threadID models.ThreadID, item *threading.ThreadItem) {
	go func() {
		pit := &threading.PublishedThreadItem{
			OrganizationID:  orgID,
			ThreadID:        threadID.String(),
			PrimaryEntityID: primaryEntityID,
			Item:            item,
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
	}()
}
