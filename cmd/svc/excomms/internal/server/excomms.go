package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/proxynumber"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/twilio"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type excommsService struct {
	twilio               *twilio.Client
	twilioApplicationSID string
	dal                  dal.DAL
	apiURL               string
	directory            directory.DirectoryClient
	sns                  snsiface.SNSAPI
	externalMessageTopic string
	clock                clock.Clock
	emailClient          EmailClient
	idgen                idGenerator
	proxyNumberManager   proxynumber.Manager
}

func NewService(
	twilioAccountSID, twilioAuthToken, twilioApplicationSID string,
	dal dal.DAL,
	apiURL string,
	directory directory.DirectoryClient,
	sns snsiface.SNSAPI,
	externalMessageTopic string,
	clock clock.Clock,
	emailClient EmailClient,
	idgen idGenerator,
	proxyNumberManager proxynumber.Manager) excomms.ExCommsServer {

	es := &excommsService{
		apiURL:               apiURL,
		twilio:               twilio.NewClient(twilioAccountSID, twilioAuthToken, nil),
		twilioApplicationSID: twilioApplicationSID,
		dal:                  dal,
		directory:            directory,
		sns:                  sns,
		externalMessageTopic: externalMessageTopic,
		clock:                clock,
		emailClient:          emailClient,
		idgen:                idgen,
		proxyNumberManager:   proxyNumberManager,
	}
	return es
}

// SearchAvailablephoneNumbers returns a list of available phone numbers based on the search criteria.
func (e *excommsService) SearchAvailablePhoneNumbers(ctx context.Context, in *excomms.SearchAvailablePhoneNumbersRequest) (*excomms.SearchAvailablePhoneNumbersResponse, error) {
	params := twilio.AvailablePhoneNumbersParams{
		AreaCode:                      in.AreaCode,
		ExcludeAllAddressRequired:     true,
		ExcludeLocalAddressRequired:   true,
		ExcludeForeignAddressRequired: true,
	}

	if containsCapability(in.Capabilities, excomms.PhoneNumberCapability_VOICE_ENABLED) {
		params.VoiceEnabled = true
	}
	if containsCapability(in.Capabilities, excomms.PhoneNumberCapability_SMS_ENABLED) {
		params.SMSEnabled = true
	}
	if containsCapability(in.Capabilities, excomms.PhoneNumberCapability_MMS_ENABLED) {
		params.MMSEnabled = true
	}

	phoneNumbers, _, err := e.twilio.AvailablePhoneNumbers.ListLocal(params)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	res := &excomms.SearchAvailablePhoneNumbersResponse{
		PhoneNumbers: make([]*excomms.AvailablePhoneNumber, len(phoneNumbers)),
	}

	for i, pn := range phoneNumbers {
		capabilities := make([]excomms.PhoneNumberCapability, 0, 3)
		if pn.Capabilities["voice"] {
			capabilities = append(capabilities, excomms.PhoneNumberCapability_VOICE_ENABLED)
		}
		if pn.Capabilities["sms"] {
			capabilities = append(capabilities, excomms.PhoneNumberCapability_SMS_ENABLED)
		}
		if pn.Capabilities["mms"] {
			capabilities = append(capabilities, excomms.PhoneNumberCapability_MMS_ENABLED)
		}

		res.PhoneNumbers[i] = &excomms.AvailablePhoneNumber{
			FriendlyName: pn.FriendlyName,
			PhoneNumber:  pn.PhoneNumber,
			Capabilities: capabilities,
		}
	}

	return res, nil
}

func containsCapability(capabilities []excomms.PhoneNumberCapability, capability excomms.PhoneNumberCapability) bool {
	for _, c := range capabilities {
		if c == capability {
			return true
		}
	}

	return false
}

// ProvisionPhoneNumber provisions the phone number provided for the requester.
func (e *excommsService) ProvisionPhoneNumber(ctx context.Context, in *excomms.ProvisionPhoneNumberRequest) (*excomms.ProvisionPhoneNumberResponse, error) {

	// check if a phone number has already been provisioned for this purpose
	ppn, err := e.dal.LookupProvisionedEndpoint(in.ProvisionFor, models.EndpointTypePhone)
	if errors.Cause(err) != dal.ErrProvisionedEndpointNotFound && err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	// if there exists a provisioned phone number,
	// return the number if it belongs to the requester
	// else return error
	if ppn != nil {
		if in.GetPhoneNumber() != "" {
			if in.GetPhoneNumber() == ppn.Endpoint {
				return &excomms.ProvisionPhoneNumberResponse{
					PhoneNumber: ppn.Endpoint,
				}, nil
			} else {
				return nil, grpc.Errorf(codes.AlreadyExists, "a different number has already been provisioned. Provision For: %s, number provisioned: %s", in.ProvisionFor, ppn.Endpoint)
			}
		} else if in.GetAreaCode() != "" {
			if strings.HasPrefix(ppn.Endpoint[2:], in.GetAreaCode()) {
				return &excomms.ProvisionPhoneNumberResponse{
					PhoneNumber: ppn.Endpoint,
				}, nil
			} else {
				return nil, grpc.Errorf(codes.AlreadyExists, "a different number has already been provisioned. Provision For: %s, number provisioned: %s", in.ProvisionFor, ppn.Endpoint)
			}
		}
	}

	// Setup all purchased numbers to route incoming calls and call statuses to the
	// URLs setup in the specified twilio application.
	ipn, res, err := e.twilio.IncomingPhoneNumber.PurchaseLocal(twilio.PurchasePhoneNumberParams{
		AreaCode:            in.GetAreaCode(),
		PhoneNumber:         in.GetPhoneNumber(),
		VoiceApplicationSID: e.twilioApplicationSID,
		SMSApplicationSID:   e.twilioApplicationSID,
	})
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusBadRequest {
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	} else if res.StatusCode == http.StatusNotFound {
		return nil, grpc.Errorf(codes.NotFound, err.Error())
	}

	// record the fact that number has been purchased
	if err := e.dal.ProvisionEndpoint(&models.ProvisionedEndpoint{
		ProvisionedFor: in.ProvisionFor,
		Endpoint:       ipn.PhoneNumber,
		EndpointType:   models.EndpointTypePhone,
	}); err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	return &excomms.ProvisionPhoneNumberResponse{
		PhoneNumber: ipn.PhoneNumber,
	}, nil
}

// SendMessage sends the message over an external channel as specified in the SendMessageRequest.
func (e *excommsService) SendMessage(ctx context.Context, in *excomms.SendMessageRequest) (*excomms.SendMessageResponse, error) {

	var msgType models.SentMessage_Type
	var destination string
	switch in.Channel {
	case excomms.ChannelType_SMS:
		msgType = models.SentMessage_SMS
		destination = in.GetSMS().ToPhoneNumber
	case excomms.ChannelType_EMAIL:
		msgType = models.SentMessage_EMAIL
		destination = in.GetEmail().ToEmailAddress
	}

	// don't send the message if it has already been sent
	if in.UUID != "" {
		sm, err := e.dal.LookupSentMessageByUUID(in.UUID, destination)
		if err != nil && errors.Cause(err) != dal.ErrSentMessageNotFound {
			return nil, grpc.Errorf(codes.Internal, err.Error())
		} else if sm != nil {
			// message already handled
			return &excomms.SendMessageResponse{}, nil
		}
	}

	sentMessage := &models.SentMessage{
		UUID:        in.UUID,
		Type:        msgType,
		Destination: destination,
	}

	switch in.Channel {
	case excomms.ChannelType_VOICE:
		return nil, grpc.Errorf(codes.Unimplemented, "not implemented")
	case excomms.ChannelType_SMS:
		msg, _, err := e.twilio.Messages.SendSMS(in.GetSMS().FromPhoneNumber, in.GetSMS().ToPhoneNumber, in.GetSMS().Text)
		if err != nil {
			return nil, grpc.Errorf(codes.Internal, err.Error())
		}
		sentMessage.Message = &models.SentMessage_SMSMsg{
			SMSMsg: &models.SMSMessage{
				FromPhoneNumber: in.GetSMS().FromPhoneNumber,
				ToPhoneNumber:   in.GetSMS().ToPhoneNumber,
				Text:            in.GetSMS().Text,
				ID:              msg.Sid,
				DateCreated:     uint64(msg.DateCreated.Unix()),
				DateSent:        uint64(msg.DateSent.Unix()),
			},
		}
	case excomms.ChannelType_EMAIL:
		id, err := e.idgen.NewID()
		if err != nil {
			return nil, grpc.Errorf(codes.Internal, err.Error())
		}
		sentMessage.Message = &models.SentMessage_EmailMsg{
			EmailMsg: &models.EmailMessage{
				ID:        strconv.FormatInt(int64(id), 10),
				Subject:   in.GetEmail().Subject,
				Body:      in.GetEmail().Body,
				FromName:  in.GetEmail().FromName,
				FromEmail: in.GetEmail().FromEmailAddress,
				ToName:    in.GetEmail().ToName,
				ToEmail:   in.GetEmail().ToEmailAddress,
			},
		}
		sentMessage.ID = id

		if err := e.emailClient.SendMessage(sentMessage.GetEmailMsg()); err != nil {
			return nil, grpc.Errorf(codes.Internal, err.Error())
		}
	}

	// persist the message that was sent for tracking purposes
	conc.Go(func() {
		if err := e.dal.CreateSentMessage(sentMessage); err != nil {
			golog.Errorf(err.Error())
		}
	})

	return &excomms.SendMessageResponse{}, nil
}

// InitiatePhoneCall initiates a phone call as defined in the InitiatePhoneCallRequest.
func (e *excommsService) InitiatePhoneCall(ctx context.Context, in *excomms.InitiatePhoneCallRequest) (*excomms.InitiatePhoneCallResponse, error) {
	if in.CallInitiationType == excomms.InitiatePhoneCallRequest_CONNECT_PARTIES {
		return nil, grpc.Errorf(codes.Unimplemented, "not implemented")
	} else if in.OrganizationID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "missing organization id")
	}

	// ensure organization exists
	lookupEntitiesRes, err := e.directory.LookupEntities(
		ctx,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: in.OrganizationID,
			},
		})
	if grpc.Code(err) == codes.NotFound {
		return nil, grpc.Errorf(codes.NotFound, "organization with id %s not found", in.OrganizationID)
	} else if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	} else if len(lookupEntitiesRes.Entities) != 1 {
		return nil, grpc.Errorf(codes.Internal, "organization with id %s not found", "Expected 1 org entity buy got back %d", len(lookupEntitiesRes.Entities))
	}

	// ensure caller belongs to the organization
	var sourceEntity *directory.Entity
	lookupEntitiesRes, err = e.directory.LookupEntities(
		ctx,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: in.CallerEntityID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
					directory.EntityInformation_MEMBERSHIPS,
				},
			},
		})
	if grpc.Code(err) == codes.NotFound {
		return nil, grpc.Errorf(codes.NotFound, "caller %s not found", in.CallerEntityID)
	} else if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	for _, entity := range lookupEntitiesRes.Entities {
		if sourceEntity != nil {
			break
		}
		for _, m := range entity.Memberships {
			if m.Type == directory.EntityType_ORGANIZATION && m.ID == in.OrganizationID {
				sourceEntity = entity
				break
			}
		}
	}
	if sourceEntity == nil {
		return nil, grpc.Errorf(codes.NotFound, "%s is not the phone number of a caller belonging to the organization.", in.FromPhoneNumber)
	}

	// validate callee
	var destinationEntity *directory.Entity
	lookupByContacRes, err := e.directory.LookupEntitiesByContact(
		ctx,
		&directory.LookupEntitiesByContactRequest{
			ContactValue: in.ToPhoneNumber,
			RequestedInformation: &directory.RequestedInformation{
				Depth:             1,
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
			}})
	if grpc.Code(err) == codes.NotFound {
		return nil, grpc.Errorf(codes.NotFound, "callee %s not found", in.ToPhoneNumber)
	} else if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	// find an external entity for the callee
	for _, entity := range lookupByContacRes.Entities {
		if destinationEntity != nil {
			break
		}
		if entity.Type != directory.EntityType_EXTERNAL {
			continue
		}
		for _, m := range entity.Memberships {
			if m.Type == directory.EntityType_ORGANIZATION && m.ID == in.OrganizationID {
				destinationEntity = entity
			}
		}
	}
	if destinationEntity == nil {
		return nil, grpc.Errorf(codes.NotFound, "%s is not the phone number of a callee belonging to the organization.", in.ToPhoneNumber)
	}

	var originatingPhoneNumber phone.Number
	if in.FromPhoneNumber != "" {
		originatingPhoneNumber, err = phone.ParseNumber(in.FromPhoneNumber)
		if err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, "%s is not a valid phone number", in.FromPhoneNumber)
		}
	} else {
		currentOriginatingPhoneNumber, err := e.dal.CurrentOriginatingNumber(in.CallerEntityID)
		if err != nil {
			if errors.Cause(err) != dal.ErrOriginatingNumberNotFound {
				return nil, grpc.Errorf(codes.Internal, err.Error())
			}

			// use a number associated with the provider's account as the originating number
			for _, c := range sourceEntity.Contacts {
				if c.ContactType == directory.ContactType_PHONE && !c.Provisioned {
					originatingPhoneNumber, err = phone.ParseNumber(c.Value)
					if err != nil {
						return nil, grpc.Errorf(codes.Internal, "phone number %s for entity is of invalid format: %s", c.Value, err.Error())
					}
					break
				}
			}
		} else {
			originatingPhoneNumber = currentOriginatingPhoneNumber
		}
	}

	if originatingPhoneNumber.IsEmpty() {
		return nil, grpc.Errorf(codes.Internal, "Unable to find a default phone number for entity %s from which to place the call", sourceEntity.ID)
	}

	// track originating phone number
	conc.Go(func() {
		if err := e.dal.SetCurrentOriginatingNumber(originatingPhoneNumber, in.CallerEntityID); err != nil {
			golog.Errorf(err.Error())
		}
	})

	destinationPhoneNumber, err := phone.ParseNumber(in.ToPhoneNumber)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "destination phone number %s is of invalid format: %s", in.ToPhoneNumber, err.Error())
	}

	proxyPhoneNumber, err := e.proxyNumberManager.ReserveNumber(originatingPhoneNumber, destinationPhoneNumber, destinationEntity.ID, sourceEntity.ID, in.OrganizationID)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}
	return &excomms.InitiatePhoneCallResponse{
		ProxyPhoneNumber:       proxyPhoneNumber.String(),
		OriginatingPhoneNumber: originatingPhoneNumber.String(),
	}, nil
}

func (e *excommsService) ProvisionEmailAddress(ctx context.Context, req *excomms.ProvisionEmailAddressRequest) (*excomms.ProvisionEmailAddressResponse, error) {

	// validate email
	if !validate.Email(req.EmailAddress) {
		return nil, grpc.Errorf(codes.InvalidArgument, "%s is an invalid email address", req.EmailAddress)
	}

	// check if an email has been provisioned for this reason
	provisionedEndpoint, err := e.dal.LookupProvisionedEndpoint(req.ProvisionFor, models.EndpointTypeEmail)
	if err != nil {
		if errors.Cause(err) != dal.ErrProvisionedEndpointNotFound {
			return nil, grpc.Errorf(codes.Internal, err.Error())
		}
	} else if provisionedEndpoint.Endpoint == req.EmailAddress {
		return &excomms.ProvisionEmailAddressResponse{
			EmailAddress: req.EmailAddress,
		}, nil
	} else {
		return nil, grpc.Errorf(codes.AlreadyExists, "Different email address (%s) provisioned for %s", provisionedEndpoint.Endpoint, req.ProvisionFor)
	}

	// if not, provision it
	if err := e.dal.ProvisionEndpoint(&models.ProvisionedEndpoint{
		EndpointType:   models.EndpointTypeEmail,
		ProvisionedFor: req.ProvisionFor,
		Endpoint:       req.EmailAddress,
	}); err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	return &excomms.ProvisionEmailAddressResponse{
		EmailAddress: req.EmailAddress,
	}, nil
}
