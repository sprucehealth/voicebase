package server

import (
	"encoding/base64"
	"fmt"
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
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

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
}

// NewThreadsServer returns an initialized instance of threadsServer
func NewThreadsServer(
	clk clock.Clock,
	dal dal.DAL,
	sns snsiface.SNSAPI,
	snsTopicARN string,
	notificationClient notification.Client) threading.ThreadsServer {
	return &threadsServer{clk: clk, dal: dal, sns: sns, snsTopicARN: snsTopicARN, notificationClient: notificationClient}
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
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
	}
	sq.ID = id
	sqr, err := transformSavedQueryToResponse(sq)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
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
	if in.Title == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Title is required")
	}
	if in.Summary == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Summary is required")
	}
	if len(in.Summary) > maxSummaryLength {
		in.Summary = in.Summary[:maxSummaryLength]
	}
	if err := bml.Parsef(in.Title).Validate(); err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, fmt.Sprintf("Title is invalid format: %s", err.Error()))
	}
	var textRefs []*models.Reference
	in.Text, textRefs, err = parseRefsAndNormalize(in.Text)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, fmt.Sprintf("Text is invalid format: %s", errors.Cause(err).Error()))
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
			Title:        in.Title,
			// TODO: safer transform for Endpoint.Type
			Source: &models.Endpoint{
				Channel: models.Endpoint_Channel(models.Endpoint_Channel_value[in.Source.Channel.String()]),
				ID:      in.Source.ID,
			},
			TextRefs: textRefs,
			Summary:  in.Summary,
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
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
	}
	thread, err := s.dal.Thread(ctx, threadID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	th, err := transformThreadToResponse(thread, !in.Internal)
	if err != nil {
		return nil, errors.Trace(err)
	}
	it, err := transformThreadItemToResponse(item)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
	}
	s.publishMessage(ctx, in.OrganizationID, in.FromEntityID, threadID, it, in.UUID)
	return &threading.CreateThreadResponse{
		ThreadID:   threadID.String(),
		ThreadItem: it,
		Thread:     th,
	}, nil
}

// DeleteMessage deletes a message from a thread
func (s *threadsServer) DeleteMessage(context.Context, *threading.DeleteMessageRequest) (*threading.DeleteMessageResponse, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "DeleteMessage not implemented")
}

// MarkThreadAsRead marks all posts in a thread as read by an entity
func (s *threadsServer) MarkThreadAsRead(ctx context.Context, in *threading.MarkThreadAsReadRequest) (out *threading.MarkThreadAsReadResponse, err error) {
	if golog.Default().L(golog.DEBUG) {
		golog.Debugf("MarkThreadAsRead REQUEST %+v\n", in)
		defer func() {
			golog.Debugf("MarkThreadAsRead RESPONSE %+v, %+v\n", out, err)
		}()
	}

	if in.ThreadID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "ThreadID is required")
	}
	threadID, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ThreadID")
	}
	if in.EntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "EntityID is required")
	}
	readTime := s.clk.Now()
	if in.Timestamp != 0 {
		readTime = time.Unix(int64(in.Timestamp), 0)
	}

	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		forUpdate := true
		threadMembers, err := dl.ThreadMemberships(ctx, []models.ThreadID{threadID}, in.EntityID, forUpdate)
		if err != nil {
			return errors.Trace(err)
		} else if len(threadMembers) != 1 {
			return errors.Trace(errors.New("Expected to find only 1 record when getting membership for thread viewer"))
		}
		threadMember := threadMembers[0]

		if threadMember.LastViewed == nil {
			threadMember.LastViewed = ptr.Time(time.Unix(0, 0))
		}

		threadItemIDs, err := dl.ThreadItemIDsCreatedAfter(ctx, threadID, *threadMember.LastViewed)
		if err != nil {
			return errors.Trace(err)
		}

		tivds := make([]*models.ThreadItemViewDetails, len(threadItemIDs))
		for i, tiid := range threadItemIDs {
			tivds[i] = &models.ThreadItemViewDetails{
				ThreadItemID:  tiid,
				ActorEntityID: in.EntityID,
				ViewTime:      ptr.Time(readTime),
			}
		}
		if err := dl.CreateThreadItemViewDetails(ctx, tivds); err != nil {
			return errors.Trace(err)
		}

		return errors.Trace(dl.UpdateMember(ctx, threadID, in.EntityID, &dal.MemberUpdate{LastViewed: ptr.Time(readTime)}))
	}); err != nil {
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
	}
	return &threading.MarkThreadAsReadResponse{}, nil
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
	if in.Title == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Title is required")
	}
	if in.Summary == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Summary is required")
	}
	if len(in.Summary) > maxSummaryLength {
		in.Summary = in.Summary[:maxSummaryLength]
	}
	if err := bml.Parsef(in.Title).Validate(); err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, fmt.Sprintf("Title is invalid format: %s", err.Error()))
	}
	var textRefs []*models.Reference
	in.Text, textRefs, err = parseRefsAndNormalize(in.Text)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, fmt.Sprintf("Text is invalid format: %s", errors.Cause(err).Error()))
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
			Title:        in.Title,
			// TODO: safer transform for Endpoint.Channel
			Source: &models.Endpoint{
				Channel: models.Endpoint_Channel(models.Endpoint_Channel_value[in.Source.Channel.String()]),
				ID:      in.Source.ID,
			},
			TextRefs: textRefs,
			Summary:  in.Summary,
		}
		req.Attachments, err = transformAttachmentsFromRequest(in.Attachments)
		if err != nil {
			return grpc.Errorf(codes.Internal, errors.Trace(err).Error())
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
			return grpc.Errorf(codes.Internal, errors.Trace(err).Error())
		}
		// The poster is recorded as a member if necessary but does not become a follower
		if err := dl.UpdateMember(ctx, threadID, in.FromEntityID, nil); err != nil {
			return grpc.Errorf(codes.Internal, errors.Trace(err).Error())
		}
		return nil
	}); err != nil {
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
	}
	thread, err = s.dal.Thread(ctx, threadID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	th, err := transformThreadToResponse(thread, !in.Internal)
	if err != nil {
		return nil, errors.Trace(err)
	}
	it, err := transformThreadItemToResponse(item)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
	}
	s.publishMessage(ctx, thread.OrganizationID, thread.PrimaryEntityID, threadID, it, in.UUID)
	s.notifyMembersOfPublishMessage(ctx, thread.OrganizationID, threadID, in.FromEntityID)
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
	ir, err := s.dal.IterateThreads(ctx, in.OrganizationID, forExternal, &dal.Iterator{
		StartCursor: in.Iterator.StartCursor,
		EndCursor:   in.Iterator.EndCursor,
		Direction:   d,
		Count:       int(in.Iterator.Count),
	})
	if e, ok := errors.Cause(err).(dal.ErrInvalidIterator); ok {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid iterator: "+string(e))
	} else if err != nil {
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
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
		res.Edges[i] = &threading.ThreadEdge{
			Thread: th,
			Cursor: e.Cursor,
		}
	}
	if in.ViewerEntityID != "" {
		if err := s.populateReadStatus(ctx, ths, in.ViewerEntityID); err != nil {
			return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
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
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
	}
	sq, err := transformSavedQueryToResponse(query)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
	}
	return &threading.SavedQueryResponse{
		SavedQuery: sq,
	}, nil
}

// SavedQueries returns the list of saved queries for an org / entity pair
func (s *threadsServer) SavedQueries(ctx context.Context, in *threading.SavedQueriesRequest) (*threading.SavedQueriesResponse, error) {
	queries, err := s.dal.SavedQueries(ctx, in.EntityID)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
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

	forExternal := false // TODO: set to true for EXTERNAL entities

	thread, err := s.dal.Thread(ctx, tid)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "Thread not found")
	} else if err != nil {
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
	}
	th, err := transformThreadToResponse(thread, forExternal)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
	}
	if in.ViewerEntityID != "" {
		if err := s.populateReadStatus(ctx, []*threading.Thread{th}, in.ViewerEntityID); err != nil {
			return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
		}
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
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
	}
	ti, err := transformThreadItemToResponse(item)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
	}
	return &threading.ThreadItemResponse{
		Item: ti,
	}, nil
}

// ThreadsForMember looks up a list of threads by entity membership
func (s *threadsServer) ThreadsForMember(ctx context.Context, in *threading.ThreadsForMemberRequest) (*threading.ThreadsForMemberResponse, error) {
	threads, err := s.dal.ThreadsForMember(ctx, in.EntityID, in.PrimaryOnly)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
	}

	forExternal := false // TODO: set to true for EXTERNAL entities

	res := &threading.ThreadsForMemberResponse{
		Threads: make([]*threading.Thread, len(threads)),
	}
	for i, t := range threads {
		th, err := transformThreadToResponse(t, forExternal)
		if err != nil {
			return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
		}
		res.Threads[i] = th
	}
	if err := s.populateReadStatus(ctx, res.Threads, in.EntityID); err != nil {
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
	}
	return res, nil
}

// ThreadItems returns the items (messages or events) in a thread
func (s *threadsServer) ThreadItems(ctx context.Context, in *threading.ThreadItemsRequest) (*threading.ThreadItemsResponse, error) {
	tid, err := models.ParseThreadID(in.ThreadID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ThreadID")
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
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid iterator: "+string(e))
	} else if err != nil {
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
	}
	res := &threading.ThreadItemsResponse{
		Edges:   make([]*threading.ThreadItemEdge, len(ir.Edges)),
		HasMore: ir.HasMore,
	}
	for i, e := range ir.Edges {
		it, err := transformThreadItemToResponse(e.Item)
		if err != nil {
			return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
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
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid ThreadItemID")
	}

	tivds, err := s.dal.ThreadItemViewDetails(ctx, tiid)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, errors.Trace(err).Error())
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

func (s *threadsServer) populateReadStatus(ctx context.Context, ts []*threading.Thread, viewerEntityID string) error {
	tIDs := make([]models.ThreadID, len(ts))
	for i, t := range ts {
		id, err := models.ParseThreadID(t.ID)
		if err != nil {
			return errors.Trace(err)
		}
		tIDs[i] = id
	}

	forUpdate := false
	tms, err := s.dal.ThreadMemberships(ctx, tIDs, viewerEntityID, forUpdate)
	if err != nil {
		return errors.Trace(err)
	}

	threadLastViewedMap := make(map[string]*time.Time)
	for _, tm := range tms {
		threadLastViewedMap[tm.ThreadID.String()] = tm.LastViewed
	}

	for _, t := range ts {
		lastViewed := threadLastViewedMap[t.ID]
		if lastViewed != nil {
			t.Unread = (t.LastMessageTimestamp > uint64(lastViewed.Unix()))
		}
	}
	return nil
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

func (s *threadsServer) notifyMembersOfPublishMessage(ctx context.Context, orgID string, threadID models.ThreadID, publishingEntityID string) {
	if s.notificationClient == nil {
		return
	}
	conc.Go(func() {
		members, err := s.dal.ThreadMembers(ctx, threadID)
		if err != nil {
			golog.Errorf("Failed to notify members: %s", err)
			return
		}
		var memberEntityIDs []string
		for _, m := range members {
			if m.EntityID == publishingEntityID {
				continue
			}
			memberEntityIDs = append(memberEntityIDs, m.EntityID)
		}
		if err := s.notificationClient.SendNotification(&notification.Notification{
			ShortMessage:     "A new message is available",
			ThreadID:         threadID.String(),
			OrganizationID:   orgID,
			EntitiesToNotify: memberEntityIDs,
		}); err != nil {
			golog.Errorf("Failed to notify members: %s", err)
		}
	})
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
