package twilio

import (
	"encoding/base64"

	"context"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"google.golang.org/grpc"
)

type mockDirectoryService_Twilio struct {
	directory.DirectoryClient
	entities     map[string][]*directory.Entity
	entitiesList []*directory.Entity
}

func (m *mockDirectoryService_Twilio) LookupEntities(ctx context.Context, in *directory.LookupEntitiesRequest, opts ...grpc.CallOption) (*directory.LookupEntitiesResponse, error) {
	return &directory.LookupEntitiesResponse{
		Entities: m.entities[in.GetEntityID()],
	}, nil
}

func (m *mockDirectoryService_Twilio) LookupEntitiesByContact(ctx context.Context, in *directory.LookupEntitiesByContactRequest, opts ...grpc.CallOption) (*directory.LookupEntitiesByContactResponse, error) {
	return &directory.LookupEntitiesByContactResponse{
		Entities: m.entitiesList,
	}, nil
}

type mockDAL_Twilio struct {
	*mock.Expector
	dal.DAL
	cr   *models.CallRequest
	ppnr *models.ProxyPhoneNumberReservation
}

func (m *mockDAL_Twilio) StoreIncomingRawMessage(msg *rawmsg.Incoming) (uint64, error) {
	m.Record(msg)
	return 0, nil
}

func (m *mockDAL_Twilio) LookupCallRequest(fromPhoneNumber string) (*models.CallRequest, error) {
	m.Record(fromPhoneNumber)
	return m.cr, nil
}
func (m *mockDAL_Twilio) CreateCallRequest(cr *models.CallRequest) error {
	m.Record(cr)
	return nil
}

type mockSNS_Twilio struct {
	snsiface.SNSAPI
	published []*sns.PublishInput
}

func (m *mockSNS_Twilio) Publish(input *sns.PublishInput) (*sns.PublishOutput, error) {
	m.published = append(m.published, input)
	return nil, nil
}

func parsePublishedExternalMessage(message string) (*excomms.PublishedExternalMessage, error) {
	data, err := base64.StdEncoding.DecodeString(message)
	if err != nil {
		return nil, err
	}

	var pem excomms.PublishedExternalMessage
	if err := pem.Unmarshal(data); err != nil {
		return nil, err
	}

	return &pem, nil
}
