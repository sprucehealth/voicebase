package mock

import (
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/care"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var _ care.CareClient = &Client{}

type Client struct {
	*mock.Expector
}

func New(t *testing.T) *Client {
	return &Client{
		&mock.Expector{T: t},
	}
}

func (c *Client) CreateVisit(ctx context.Context, in *care.CreateVisitRequest, opts ...grpc.CallOption) (*care.CreateVisitResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*care.CreateVisitResponse), mock.SafeError(rets[1])
}
func (c *Client) GetVisit(ctx context.Context, in *care.GetVisitRequest, opts ...grpc.CallOption) (*care.GetVisitResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*care.GetVisitResponse), mock.SafeError(rets[1])
}
