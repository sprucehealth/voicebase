package mock

import (
	"testing"

	"context"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/care"
	"google.golang.org/grpc"
)

var _ care.CareClient = &Client{}

type Client struct {
	*mock.Expector
}

func New(t testing.TB) *Client {
	return &Client{
		&mock.Expector{T: t},
	}
}

func (c *Client) CarePlan(ctx context.Context, in *care.CarePlanRequest, opts ...grpc.CallOption) (*care.CarePlanResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*care.CarePlanResponse), mock.SafeError(rets[1])
}

func (c *Client) CreateCarePlan(ctx context.Context, in *care.CreateCarePlanRequest, opts ...grpc.CallOption) (*care.CreateCarePlanResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*care.CreateCarePlanResponse), mock.SafeError(rets[1])
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

func (c *Client) CreateVisitAnswers(ctx context.Context, in *care.CreateVisitAnswersRequest, opts ...grpc.CallOption) (*care.CreateVisitAnswersResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*care.CreateVisitAnswersResponse), mock.SafeError(rets[1])
}

func (c *Client) GetAnswersForVisit(ctx context.Context, in *care.GetAnswersForVisitRequest, opts ...grpc.CallOption) (*care.GetAnswersForVisitResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*care.GetAnswersForVisitResponse), mock.SafeError(rets[1])
}

func (c *Client) SearchMedications(ctx context.Context, in *care.SearchMedicationsRequest, opts ...grpc.CallOption) (*care.SearchMedicationsResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*care.SearchMedicationsResponse), mock.SafeError(rets[1])
}

func (c *Client) SearchAllergyMedications(ctx context.Context, in *care.SearchAllergyMedicationsRequest, opts ...grpc.CallOption) (*care.SearchAllergyMedicationsResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*care.SearchAllergyMedicationsResponse), mock.SafeError(rets[1])
}

func (c *Client) SearchSelfReportedMedications(ctx context.Context, in *care.SearchSelfReportedMedicationsRequest, opts ...grpc.CallOption) (*care.SearchSelfReportedMedicationsResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*care.SearchSelfReportedMedicationsResponse), mock.SafeError(rets[1])
}

func (c *Client) SubmitCarePlan(ctx context.Context, in *care.SubmitCarePlanRequest, opts ...grpc.CallOption) (*care.SubmitCarePlanResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*care.SubmitCarePlanResponse), mock.SafeError(rets[1])
}

func (c *Client) UpdateCarePlan(ctx context.Context, in *care.UpdateCarePlanRequest, opts ...grpc.CallOption) (*care.UpdateCarePlanResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*care.UpdateCarePlanResponse), mock.SafeError(rets[1])
}

func (c *Client) SubmitVisit(ctx context.Context, in *care.SubmitVisitRequest, opts ...grpc.CallOption) (*care.SubmitVisitResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*care.SubmitVisitResponse), mock.SafeError(rets[1])
}

func (c *Client) TriageVisit(ctx context.Context, in *care.TriageVisitRequest, opts ...grpc.CallOption) (*care.TriageVisitResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*care.TriageVisitResponse), mock.SafeError(rets[1])
}
