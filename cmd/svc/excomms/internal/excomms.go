package internal

import (
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/twilio"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
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
}

func NewService(
	twilioAccountSID, twilioAuthToken, twilioApplicationSID string,
	dal dal.DAL,
	apiURL string,
	directory directory.DirectoryClient,
	sns snsiface.SNSAPI,
	externalMessageTopic string,
	clock clock.Clock) excomms.ExCommsServer {

	es := &excommsService{
		apiURL:               apiURL,
		twilio:               twilio.NewClient(twilioAccountSID, twilioAuthToken, nil),
		twilioApplicationSID: twilioApplicationSID,
		dal:                  dal,
		directory:            directory,
		sns:                  sns,
		externalMessageTopic: externalMessageTopic,
		clock:                clock,
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
	ppn, err := e.dal.LookupProvisionedPhoneNumber(&dal.ProvisionedNumberLookup{
		ProvisionedFor: ptr.String(in.ProvisionFor),
	})
	if errors.Cause(err) != dal.ErrProvisionedNumberNotFound && err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	// if there exists a provisioned phone number,
	// return the number if it belongs to the requester
	// else return error
	if ppn != nil {
		if in.GetPhoneNumber() != "" {
			if in.GetPhoneNumber() == ppn.PhoneNumber.String() {
				return &excomms.ProvisionPhoneNumberResponse{
					PhoneNumber: ppn.PhoneNumber.String(),
				}, nil
			} else {
				return nil, grpc.Errorf(codes.AlreadyExists, "a different number has already been provisioned. Provision For: %s, number provisioned: %s", in.ProvisionFor, ppn.PhoneNumber)
			}
		} else if in.GetAreaCode() != "" {
			if strings.HasPrefix(ppn.PhoneNumber.String()[2:], in.GetAreaCode()) {
				return &excomms.ProvisionPhoneNumberResponse{
					PhoneNumber: ppn.PhoneNumber.String(),
				}, nil
			} else {
				return nil, grpc.Errorf(codes.AlreadyExists, "a different number has already been provisioned. Provision For: %s, number provisioned: %s", in.ProvisionFor, ppn.PhoneNumber)
			}
		}
	}

	// Setup all purchased numbers to route incoming calls and call statuses to the
	// URLs setup in the specified twilio application.
	ipn, _, err := e.twilio.IncomingPhoneNumber.PurchaseLocal(twilio.PurchasePhoneNumberParams{
		AreaCode:            in.GetAreaCode(),
		PhoneNumber:         in.GetPhoneNumber(),
		VoiceApplicationSID: e.twilioApplicationSID,
		SMSApplicationSID:   e.twilioApplicationSID,
	})
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	pn, err := phone.ParseNumber(ipn.PhoneNumber)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	// record the fact that number has been purchased
	if err := e.dal.ProvisionPhoneNumber(&models.ProvisionedPhoneNumber{
		ProvisionedFor: in.ProvisionFor,
		PhoneNumber:    pn,
	}); err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	return &excomms.ProvisionPhoneNumberResponse{
		PhoneNumber: ipn.PhoneNumber,
	}, nil
}

// SendMessage sends the message over an external channel as specified in the SendMessageRequest.
func (e *excommsService) SendMessage(ctx context.Context, in *excomms.SendMessageRequest) (*excomms.SendMessageResponse, error) {
	if in.Channel == excomms.ChannelType_Voice {
		return nil, grpc.Errorf(codes.Unimplemented, "not implemented")
	}

	_, _, err := e.twilio.Messages.SendSMS(in.FromChannelID, in.ToChannelID, in.Text)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	return &excomms.SendMessageResponse{}, nil
}

// TODO: Move these values to config such that they are easily changeable.
var (
	// phoneReservationDuration represents the duration of time for which
	// a proxy phone number reservation to dial a particular number lasts.
	phoneReservationDuration = 5 * time.Minute

	// phoneReservationDurationGrace represents the grace period after the expiration
	// where the proxy phone number is not reserved for another phone call.
	phoneReservationDurationGrace = 5 * time.Minute
)

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
	lookupByContacRes, err := e.directory.LookupEntitiesByContact(
		ctx,
		&directory.LookupEntitiesByContactRequest{
			ContactValue: in.FromPhoneNumber,
			RequestedInformation: &directory.RequestedInformation{
				Depth:             1,
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
			},
		})
	if grpc.Code(err) == codes.NotFound {
		return nil, grpc.Errorf(codes.NotFound, "caller %s not found", in.FromPhoneNumber)
	} else if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	for _, entity := range lookupByContacRes.Entities {
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
	lookupByContacRes, err = e.directory.LookupEntitiesByContact(
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

	golog.Debugf("Destination lookup response: %#v", lookupByContacRes)
	for _, entity := range lookupByContacRes.Entities {
		if destinationEntity != nil {
			break
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

	var proxyPhoneNumber phone.Number
	if err := e.dal.Transact(func(dl dal.DAL) error {

		// check if an active reservation already exists for the caller/callee pair, and if
		// so, extend the reservation and return the same number rather than reserving a new number
		ppnr, err := dl.ActiveProxyPhoneNumberReservation(&dal.ProxyPhoneNumberReservationLookup{
			DestinationEntityID: ptr.String(destinationEntity.ID),
		})
		if err != nil && errors.Cause(err) != dal.ErrProxyPhoneNumberReservationNotFound {
			return errors.Trace(err)
		} else if ppnr != nil && ppnr.OwnerEntityID == sourceEntity.ID {

			expiration := e.clock.Now().Add(phoneReservationDuration)

			// extend the existing reservation rather than creating a new one and return
			if rowsAffected, err := dl.UpdateActiveProxyPhoneNumberReservation(ppnr.PhoneNumber, &dal.ProxyPhoneNumberReservationUpdate{
				Expires: ptr.Time(expiration),
			}); err != nil {
				return errors.Trace(err)
			} else if rowsAffected != 1 {
				return errors.Trace(fmt.Errorf("Expected 1 row to be updated, instead %d rows were updated for proxyPhoneNumber %s", rowsAffected, ppnr.PhoneNumber))
			}

			if rowsAffected, err := dl.UpdateProxyPhoneNumber(ppnr.PhoneNumber, &dal.ProxyPhoneNumberUpdate{
				Expires: ptr.Time(expiration),
			}); err != nil {
				return errors.Trace(err)
			} else if rowsAffected != 1 {
				return errors.Trace(fmt.Errorf("Expected 1 row to be updated, instead %d rows were updated for proxyPhoneNumber %s", rowsAffected, ppnr.PhoneNumber))
			}

			proxyPhoneNumber = ppnr.PhoneNumber
			return nil
		}

		// if no active reservation exists, then lets go ahead and reserve a new number
		ppns, err := dl.ProxyPhoneNumbers(dal.PPOUnexpiredOnly)
		if err != nil {
			return errors.Trace(err)
		}

		for _, ppn := range ppns {
			if ppn.Expires != nil && ppn.Expires.Add(phoneReservationDurationGrace).Before(e.clock.Now()) {
				proxyPhoneNumber = ppn.PhoneNumber
				break
			} else if ppn.Expires == nil {
				proxyPhoneNumber = ppn.PhoneNumber
				break
			}
		}

		if proxyPhoneNumber == "" {
			return errors.Trace(errors.New("Unable to find free phone number to reserve"))
		}

		expiration := e.clock.Now().Add(phoneReservationDuration)

		if err := dl.CreateProxyPhoneNumberReservation(&models.ProxyPhoneNumberReservation{
			PhoneNumber:         proxyPhoneNumber,
			DestinationEntityID: destinationEntity.ID,
			OwnerEntityID:       sourceEntity.ID,
			OrganizationID:      in.OrganizationID,
			Expires:             expiration,
		}); err != nil {
			return errors.Trace(err)
		}

		if rowsAffected, err := dl.UpdateProxyPhoneNumber(proxyPhoneNumber, &dal.ProxyPhoneNumberUpdate{
			Expires: ptr.Time(expiration),
		}); err != nil {
			return errors.Trace(err)
		} else if rowsAffected != 1 {
			return errors.Trace(fmt.Errorf("Expected 1 row to be updated, instead %d rows were updated for proxyPhoneNumber %s", rowsAffected, proxyPhoneNumber))
		}

		return nil
	}); err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}

	return &excomms.InitiatePhoneCallResponse{
		PhoneNumber: proxyPhoneNumber.String(),
	}, nil
}

func (e *excommsService) ProcessTwilioEvent(ctx context.Context, req *excomms.ProcessTwilioEventRequest) (*excomms.ProcessTwilioEventResponse, error) {
	res := &excomms.ProcessTwilioEventResponse{}
	handler := twilioEventsHandlers[req.Event]
	if handler == nil {
		return nil, grpc.Errorf(codes.NotFound, "unknown event: %s", req.Event.String())
	}
	twiml, err := handler(req.Params, e)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, err.Error())
	}
	res.Twiml = twiml

	conc.Go(func() {
		if err := e.dal.LogEvent(&models.Event{
			Data:        req.Params,
			Type:        req.Event.String(),
			Source:      req.Params.From,
			Destination: req.Params.To,
		}); err != nil {
			golog.Errorf("Unable to log event %s: %s", req.Event.String(), err.Error())
		}
	})
	return res, nil
}
