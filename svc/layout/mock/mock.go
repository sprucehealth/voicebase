package mock

import (
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/layout"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var _ layout.LayoutClient = &Client{}

type Client struct {
	*mock.Expector
}

func New(t testing.TB) *Client {
	return &Client{&mock.Expector{T: t}}
}

func (c *Client) ListVisitLayouts(ctx context.Context, in *layout.ListVisitLayoutsRequest, opts ...grpc.CallOption) (*layout.ListVisitLayoutsResponse, error) {
	rets := c.Record(in)

	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*layout.ListVisitLayoutsResponse), mock.SafeError(rets[1])
}
func (c *Client) ListVisitCategories(ctx context.Context, in *layout.ListVisitCategoriesRequest, opts ...grpc.CallOption) (*layout.ListVisitCategoriesResponse, error) {
	rets := c.Record(in)

	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*layout.ListVisitCategoriesResponse), mock.SafeError(rets[1])
}
func (c *Client) CreateVisitLayout(ctx context.Context, in *layout.CreateVisitLayoutRequest, opts ...grpc.CallOption) (*layout.CreateVisitLayoutResponse, error) {
	rets := c.Record(in)

	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*layout.CreateVisitLayoutResponse), mock.SafeError(rets[1])
}
func (c *Client) GetVisitCategory(ctx context.Context, in *layout.GetVisitCategoryRequest, opts ...grpc.CallOption) (*layout.GetVisitCategoryResponse, error) {
	rets := c.Record(in)

	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*layout.GetVisitCategoryResponse), mock.SafeError(rets[1])
}
func (c *Client) GetVisitLayout(ctx context.Context, in *layout.GetVisitLayoutRequest, opts ...grpc.CallOption) (*layout.GetVisitLayoutResponse, error) {
	rets := c.Record(in)

	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*layout.GetVisitLayoutResponse), mock.SafeError(rets[1])
}
func (c *Client) GetVisitLayoutByVersion(ctx context.Context, in *layout.GetVisitLayoutByVersionRequest, opts ...grpc.CallOption) (*layout.GetVisitLayoutByVersionResponse, error) {
	rets := c.Record(in)

	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*layout.GetVisitLayoutByVersionResponse), mock.SafeError(rets[1])
}
func (c *Client) GetVisitLayoutVersion(ctx context.Context, in *layout.GetVisitLayoutVersionRequest, opts ...grpc.CallOption) (*layout.GetVisitLayoutVersionResponse, error) {
	rets := c.Record(in)

	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*layout.GetVisitLayoutVersionResponse), mock.SafeError(rets[1])
}
func (c *Client) UpdateVisitLayout(ctx context.Context, in *layout.UpdateVisitLayoutRequest, opts ...grpc.CallOption) (*layout.UpdateVisitLayoutResponse, error) {
	rets := c.Record(in)

	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*layout.UpdateVisitLayoutResponse), mock.SafeError(rets[1])
}
func (c *Client) DeleteVisitLayout(ctx context.Context, in *layout.DeleteVisitLayoutRequest, opts ...grpc.CallOption) (*layout.DeleteVisitLayoutResponse, error) {
	rets := c.Record(in)

	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*layout.DeleteVisitLayoutResponse), mock.SafeError(rets[1])
}
func (c *Client) CreateVisitCategory(ctx context.Context, in *layout.CreateVisitCategoryRequest, opts ...grpc.CallOption) (*layout.CreateVisitCategoryResponse, error) {
	rets := c.Record(in)

	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*layout.CreateVisitCategoryResponse), mock.SafeError(rets[1])
}
func (c *Client) UpdateVisitCategory(ctx context.Context, in *layout.UpdateVisitCategoryRequest, opts ...grpc.CallOption) (*layout.UpdateVisitCategoryResponse, error) {
	rets := c.Record(in)

	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*layout.UpdateVisitCategoryResponse), mock.SafeError(rets[1])
}
func (c *Client) DeleteVisitCategory(ctx context.Context, in *layout.DeleteVisitCategoryRequest, opts ...grpc.CallOption) (*layout.DeleteVisitCategoryResponse, error) {
	rets := c.Record(in)

	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*layout.DeleteVisitCategoryResponse), mock.SafeError(rets[1])
}
