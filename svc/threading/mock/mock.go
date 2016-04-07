package mock

import (
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Compile time check to make sure the mock conforms to the interface
var _ threading.ThreadsClient = &Client{}

// Client is a mock for the threading service client.
type Client struct {
	*mock.Expector
}

// New returns an initialized MockClient.
func New(t *testing.T) *Client {
	return &Client{&mock.Expector{T: t}}
}

// CreateSavedQuery saves a query for later use
func (c *Client) CreateSavedQuery(ctx context.Context, in *threading.CreateSavedQueryRequest, opts ...grpc.CallOption) (*threading.CreateSavedQueryResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.CreateSavedQueryResponse), mock.SafeError(rets[1])
}

// CreateEmptyThread create a new thread with no messages
func (c *Client) CreateEmptyThread(ctx context.Context, in *threading.CreateEmptyThreadRequest, opts ...grpc.CallOption) (*threading.CreateEmptyThreadResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.CreateEmptyThreadResponse), mock.SafeError(rets[1])
}

// CreateOnboardingThread create a new onboarding thread
func (c *Client) CreateOnboardingThread(ctx context.Context, in *threading.CreateOnboardingThreadRequest, opts ...grpc.CallOption) (*threading.CreateOnboardingThreadResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.CreateOnboardingThreadResponse), mock.SafeError(rets[1])
}

// CreateThread create a new thread with an initial message
func (c *Client) CreateThread(ctx context.Context, in *threading.CreateThreadRequest, opts ...grpc.CallOption) (*threading.CreateThreadResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.CreateThreadResponse), mock.SafeError(rets[1])
}

// CreateLinkedThreads is a mock
func (c *Client) CreateLinkedThreads(ctx context.Context, in *threading.CreateLinkedThreadsRequest, opts ...grpc.CallOption) (*threading.CreateLinkedThreadsResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.CreateLinkedThreadsResponse), mock.SafeError(rets[1])
}

// DeleteMessage deletes a message from a thread
func (c *Client) DeleteMessage(ctx context.Context, in *threading.DeleteMessageRequest, opts ...grpc.CallOption) (*threading.DeleteMessageResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.DeleteMessageResponse), mock.SafeError(rets[1])
}

// DeleteThread deletes a message from a thread
func (c *Client) DeleteThread(ctx context.Context, in *threading.DeleteThreadRequest, opts ...grpc.CallOption) (*threading.DeleteThreadResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.DeleteThreadResponse), mock.SafeError(rets[1])
}

// LinkedThread returns the linked thread of one exists
func (c *Client) LinkedThread(ctx context.Context, in *threading.LinkedThreadRequest, opts ...grpc.CallOption) (*threading.LinkedThreadResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.LinkedThreadResponse), mock.SafeError(rets[1])
}

// MarkThreadAsRead marks all posts in a thread as read by an entity
func (c *Client) MarkThreadAsRead(ctx context.Context, in *threading.MarkThreadAsReadRequest, opts ...grpc.CallOption) (*threading.MarkThreadAsReadResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.MarkThreadAsReadResponse), mock.SafeError(rets[1])
}

// OnboardingThreadEvent updated the setup thread due to an event
func (c *Client) OnboardingThreadEvent(ctx context.Context, in *threading.OnboardingThreadEventRequest, opts ...grpc.CallOption) (*threading.OnboardingThreadEventResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.OnboardingThreadEventResponse), mock.SafeError(rets[1])
}

// PostMessage posts a message into a specified thread
func (c *Client) PostMessage(ctx context.Context, in *threading.PostMessageRequest, opts ...grpc.CallOption) (*threading.PostMessageResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.PostMessageResponse), mock.SafeError(rets[1])
}

// QueryThreads queries the list of threads in an organization
func (c *Client) QueryThreads(ctx context.Context, in *threading.QueryThreadsRequest, opts ...grpc.CallOption) (*threading.QueryThreadsResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.QueryThreadsResponse), mock.SafeError(rets[1])
}

// SavedQuery returns a single saved query by ID
func (c *Client) SavedQuery(ctx context.Context, in *threading.SavedQueryRequest, opts ...grpc.CallOption) (*threading.SavedQueryResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.SavedQueryResponse), mock.SafeError(rets[1])
}

// SavedQueries returns the list of saved queries for an org / entity pair
func (c *Client) SavedQueries(ctx context.Context, in *threading.SavedQueriesRequest, opts ...grpc.CallOption) (*threading.SavedQueriesResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.SavedQueriesResponse), mock.SafeError(rets[1])
}

// Thread lookups and returns a single thread by ID
func (c *Client) Thread(ctx context.Context, in *threading.ThreadRequest, opts ...grpc.CallOption) (*threading.ThreadResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.ThreadResponse), mock.SafeError(rets[1])
}

// ThreadsForMember looks up a list of threads by entity membership
func (c *Client) ThreadsForMember(ctx context.Context, in *threading.ThreadsForMemberRequest, opts ...grpc.CallOption) (*threading.ThreadsForMemberResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.ThreadsForMemberResponse), mock.SafeError(rets[1])
}

// ThreadItem returns a single thread item
func (c *Client) ThreadItem(ctx context.Context, in *threading.ThreadItemRequest, opts ...grpc.CallOption) (*threading.ThreadItemResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.ThreadItemResponse), mock.SafeError(rets[1])
}

// ThreadItems returns the items (messages or events) in a thread
func (c *Client) ThreadItems(ctx context.Context, in *threading.ThreadItemsRequest, opts ...grpc.CallOption) (*threading.ThreadItemsResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.ThreadItemsResponse), mock.SafeError(rets[1])
}

// ThreadItemViewDetails returns the view details for a thread item
func (c *Client) ThreadItemViewDetails(ctx context.Context, in *threading.ThreadItemViewDetailsRequest, opts ...grpc.CallOption) (*threading.ThreadItemViewDetailsResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.ThreadItemViewDetailsResponse), mock.SafeError(rets[1])
}

// ThreadMembers returns the members of a thread
func (c *Client) ThreadMembers(ctx context.Context, in *threading.ThreadMembersRequest, opts ...grpc.CallOption) (*threading.ThreadMembersResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.ThreadMembersResponse), mock.SafeError(rets[1])
}

// UpdateSavedQuery updated a saved query
func (c *Client) UpdateSavedQuery(ctx context.Context, in *threading.UpdateSavedQueryRequest, opts ...grpc.CallOption) (*threading.UpdateSavedQueryResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.UpdateSavedQueryResponse), mock.SafeError(rets[1])
}

// UpdateThread is a mock
func (c *Client) UpdateThread(ctx context.Context, in *threading.UpdateThreadRequest, opts ...grpc.CallOption) (*threading.UpdateThreadResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.UpdateThreadResponse), mock.SafeError(rets[1])
}
