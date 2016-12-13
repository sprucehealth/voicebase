package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/proxynumber"
	exsettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/twilio"
	"github.com/sprucehealth/backend/libs/urlutil"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/events"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type excommsService struct {
	twilio                   *twilio.Client
	twilioAccountSID         string
	twilioApplicationSID     string
	twilioSigningKeySID      string
	twilioSigningKey         string
	twilioVideoConfigSID     string
	dal                      dal.DAL
	apiURL                   string
	directory                directory.DirectoryClient
	threading                threading.ThreadsClient
	settings                 settings.SettingsClient
	sns                      snsiface.SNSAPI
	externalMessageTopic     string
	eventTopic               string
	clock                    clock.Clock
	emailClient              EmailClient
	spruceEmailDomain        string
	transactionalEmailClient EmailClient
	transactionalEmailDomain string
	idgen                    idGenerator
	proxyNumberManager       proxynumber.Manager
	signer                   *urlutil.Signer
	httpClient               httputil.Client
	notificationClient       notification.Client
	genIPCallIdentity        func() (string, error)
}

func NewService(
	twilioAccountSID, twilioAuthToken, twilioApplicationSID string,
	twilioSigningKeySID, twilioSigningKey, twilioVideoConfigSID string,
	dal dal.DAL,
	apiURL string,
	directory directory.DirectoryClient,
	threading threading.ThreadsClient,
	settings settings.SettingsClient,
	sns snsiface.SNSAPI,
	externalMessageTopic string,
	eventTopic string,
	clock clock.Clock,
	spruceEmailDomain string,
	emailClient EmailClient,
	transactionalEmailDomain string,
	transactionalEmailClient EmailClient,
	idgen idGenerator,
	proxyNumberManager proxynumber.Manager,
	signer *urlutil.Signer,
	notificationClient notification.Client,
) excomms.ExCommsServer {

	es := &excommsService{
		apiURL:                   apiURL,
		twilio:                   twilio.NewClient(twilioAccountSID, twilioAuthToken, nil),
		twilioAccountSID:         twilioAccountSID,
		twilioApplicationSID:     twilioApplicationSID,
		twilioSigningKeySID:      twilioSigningKeySID,
		twilioSigningKey:         twilioSigningKey,
		twilioVideoConfigSID:     twilioVideoConfigSID,
		dal:                      dal,
		directory:                directory,
		threading:                threading,
		settings:                 settings,
		sns:                      sns,
		externalMessageTopic:     externalMessageTopic,
		eventTopic:               eventTopic,
		clock:                    clock,
		spruceEmailDomain:        spruceEmailDomain,
		emailClient:              emailClient,
		transactionalEmailDomain: transactionalEmailDomain,
		transactionalEmailClient: transactionalEmailClient,
		idgen:              idgen,
		proxyNumberManager: proxyNumberManager,
		signer:             signer,
		httpClient:         http.DefaultClient,
		notificationClient: notificationClient,
		genIPCallIdentity:  generateIPCallIdentity,
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
		return nil, errors.Trace(err)
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
	// First make sure the number hasn't yet been provisioned based on UUID.
	// We'll check again on insert to avoid a race condition, but want to check
	// early to avoid purchasing a number if we don't have to.
	if in.UUID != "" {
		prov, err := e.dal.LookupProvisionedEndpointByUUID(in.UUID)
		if err == nil {
			// Just to be safe make sure it's provisioned for the same entity and of the expected type
			if in.ProvisionFor != prov.ProvisionedFor || prov.EndpointType != models.EndpointTypePhone {
				return nil, grpc.Errorf(codes.AlreadyExists, "A provisioned endpoint %+v found with the same UUID %q but differnt owner %q or type %q", prov, in.UUID, in.ProvisionFor, models.EndpointTypePhone)
			}
			// Sanity check that if provisioning by exact phone number then make sure it matches.
			if pn := in.GetPhoneNumber(); pn != "" && pn != prov.Endpoint {
				return nil, grpc.Errorf(codes.AlreadyExists, "A provisioned endpoint found with the same UUID %q but different number %s expected %s", in.UUID, prov.Endpoint, pn)
			}
			return &excomms.ProvisionPhoneNumberResponse{
				PhoneNumber: prov.Endpoint,
			}, nil
		}
		if errors.Cause(err) != dal.ErrProvisionedEndpointNotFound {
			return nil, errors.Wrapf(err, "failed to lookup provisioned endpoint by UUID %q", in.UUID)
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
		if e, ok := err.(*twilio.Exception); ok {
			switch e.Code {
			case twilio.ErrorCodeInvalidAreaCode:
				return nil, grpc.Errorf(codes.NotFound, e.Message)
			case twilio.ErrorCodeNoPhoneNumberInAreaCode:
				return nil, grpc.Errorf(codes.InvalidArgument, e.Message)
			}
		}
		return nil, errors.Wrapf(err, "provision_for=%s area_code=%s phone_number=%s",
			in.ProvisionFor, in.GetAreaCode(), in.GetPhoneNumber())
	}

	// record the fact that number has been purchased
	if err := e.dal.ProvisionEndpoint(&models.ProvisionedEndpoint{
		ProvisionedFor: in.ProvisionFor,
		Endpoint:       ipn.PhoneNumber,
		EndpointType:   models.EndpointTypePhone,
	}, in.UUID); err != nil {
		return nil, errors.Wrapf(err, "provision_for=%s endpoint=%s endpoint_type=%s uuid=%q",
			in.ProvisionFor, ipn.PhoneNumber, models.EndpointTypePhone, in.UUID)
	}

	events.Publish(e.sns, e.eventTopic, events.Service_EXCOMMS, &excomms.Event{
		Type: excomms.Event_PROVISIONED_ENDPOINT,
		Details: &excomms.Event_ProvisionedEndpoint{
			ProvisionedEndpoint: &excomms.ProvisionedEndpoint{
				ForEntityID:  in.ProvisionFor,
				EndpointType: excomms.EndpointType_PHONE,
				Endpoint:     ipn.PhoneNumber,
			},
		},
	})

	return &excomms.ProvisionPhoneNumberResponse{
		PhoneNumber: ipn.PhoneNumber,
	}, nil
}

func (e *excommsService) DeprovisionPhoneNumber(ctx context.Context, in *excomms.DeprovisionPhoneNumberRequest) (*excomms.DeprovisionPhoneNumberResponse, error) {
	if in.PhoneNumber == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "phone number to deprovision required")
	} else if len(in.Reason) > 254 {
		return nil, grpc.Errorf(codes.InvalidArgument, "reason cannot be longer than 254 characters")
	}

	// lookup the phone number via twilio
	list, _, err := e.twilio.IncomingPhoneNumber.List(twilio.ListPurchasedPhoneNumberParams{
		PhoneNumber: in.PhoneNumber,
	})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(list.IncomingPhoneNumbers) == 0 {
		// nothing to do if no phone number to deprovision found
		return &excomms.DeprovisionPhoneNumberResponse{}, nil
	} else if len(list.IncomingPhoneNumbers) != 1 {
		return nil, errors.Errorf("Expected 1 purchased phone number but got %d for %s", len(list.IncomingPhoneNumbers), in.PhoneNumber)
	}

	numberToDeprovision := list.IncomingPhoneNumbers[0]

	// go ahead and release the number from twilio
	_, err = e.twilio.IncomingPhoneNumber.Delete(numberToDeprovision.SID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// mark the number as deprovisioned
	rowsUpdated, err := e.dal.UpdateProvisionedEndpoint(in.PhoneNumber, models.EndpointTypePhone, &dal.ProvisionedEndpointUpdate{
		Deprovisioned:          ptr.Bool(true),
		DeprovisionedTimestamp: ptr.Time(e.clock.Now()),
		DeprovisionedReason:    &in.Reason,
	})
	if err != nil {
		return nil, errors.Trace(err)
	} else if rowsUpdated > 1 {
		return nil, errors.Errorf("Expected no more than 1 row to be updated but got %d rows updated when deprovisioning %s", rowsUpdated, in.PhoneNumber)
	}

	return &excomms.DeprovisionPhoneNumberResponse{}, nil
}

func (e *excommsService) DeprovisionEmail(ctx context.Context, in *excomms.DeprovisionEmailRequest) (*excomms.DeprovisionEmailResponse, error) {
	if in.Email == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "email required")
	}

	if err := e.dal.Transact(func(dl dal.DAL) error {
		rowsUpdated, err := dl.UpdateProvisionedEndpoint(in.Email, models.EndpointTypeEmail, &dal.ProvisionedEndpointUpdate{
			Deprovisioned:          ptr.Bool(true),
			DeprovisionedTimestamp: ptr.Time(e.clock.Now()),
			DeprovisionedReason:    &in.Reason,
		})
		if err != nil {
			return err
		} else if rowsUpdated > 1 {
			return errors.Errorf("Expected no more than 1 row to be updated but got %d rows updated when deprovisioning %s", rowsUpdated, in.Email)
		}
		return nil
	}); err != nil {
		return nil, errors.Trace(err)
	}

	return &excomms.DeprovisionEmailResponse{}, nil
}

// SendMessage sends the message over an external channel as specified in the SendMessageRequest.
func (e *excommsService) SendMessage(ctx context.Context, in *excomms.SendMessageRequest) (*excomms.SendMessageResponse, error) {
	if in.UUID == "" {
		// TODO: would be nice to require the UUID but for now that's not possible since clients don't include it.. so generate a random one
		uuid, err := idgen.NewUUID()
		if err != nil {
			return nil, errors.Trace(err)
		}
		in.UUID = uuid
	}

	var msgType models.SentMessage_Type
	var destination string
	var mediaIDs []string
	switch msg := in.Message.(type) {
	case *excomms.SendMessageRequest_SMS:
		msgType = models.SentMessage_SMS
		destination = in.GetSMS().ToPhoneNumber
		mediaIDs = msg.SMS.MediaIDs
	case *excomms.SendMessageRequest_Email:
		msgType = models.SentMessage_EMAIL
		destination = in.GetEmail().ToEmailAddress
		mediaIDs = msg.Email.MediaIDs
	default:
		return nil, errors.Errorf("unknown message type %T", in.Message)
	}

	// don't send the message if it has already been sent
	sm, err := e.dal.LookupSentMessageByUUID(in.UUID, destination)
	if err != nil && errors.Cause(err) != dal.ErrSentMessageNotFound {
		return nil, errors.Trace(err)
	} else if sm != nil {
		// message already handled
		return &excomms.SendMessageResponse{}, nil
	}

	sentMessage := &models.SentMessage{
		UUID:        in.UUID,
		Type:        msgType,
		Destination: destination,
	}

	// Get our internal media information and size and externalize it
	resizedURLs := make([]string, len(mediaIDs))
	for i, mID := range mediaIDs {
		// default everything to a max size of 3264x3264
		mediaID, err := media.ParseMediaID(mID)
		if err != nil {
			errors.Trace(err)
		}

		signedURL, err := e.signer.SignedURL(fmt.Sprintf("/media/%s/thumbnail", mediaID), url.Values{
			"width":  []string{"3264"},
			"height": []string{"3264"}}, ptr.Time(e.clock.Now().Add(time.Minute*15)))
		if err != nil {
			errors.Trace(err)
		}
		resizedURLs[i] = signedURL
	}
	if len(resizedURLs) != 0 {
		golog.Debugf("Resized media URLs: %v", resizedURLs)
		parallel := conc.NewParallel()
		// Perform GET calls on our resized urls so that the resize doesn't take place on the subsequent HEAD calls
		for _, rURL := range resizedURLs {
			parallel.Go(func() error {
				// TODO: Maybe we want to specify a timeout here? Pretty sure our implementation wont hang forever though
				resp, err := e.httpClient.Head(rURL)
				if err != nil {
					return err
				}
				// Note: Not sure if we need to do this if we didn't read from it, but just to be safe
				resp.Body.Close()
				return nil
			})
		}
		if err := parallel.Wait(); err != nil {
			return nil, errors.Trace(err)
		}
	}

	switch inMsg := in.Message.(type) {
	case *excomms.SendMessageRequest_SMS:
		sms := inMsg.SMS
		msg, _, err := e.twilio.Messages.Send(sms.FromPhoneNumber, sms.ToPhoneNumber, twilio.MessageParams{
			ApplicationSid: e.twilioApplicationSID,
			Body:           sms.Text,
			MediaUrl:       resizedURLs,
		})
		if err != nil {
			if e, ok := err.(*twilio.Exception); ok {
				switch e.Code {
				case twilio.ErrorCodeInvalidToPhoneNumber:
					// drop the message since the phone number is invalid.
					// TODO: In the future we might want to indicate to the provider
					// that they entered an invalid phone number?
					return &excomms.SendMessageResponse{}, nil
				case twilio.ErrorCodeMessageLengthExceeded:
					return nil, grpc.Errorf(excomms.ErrorCodeMessageLengthExceeded, "message length can only be 1600 characters in length, message length was %d characters", len(sms.Text))
				case twilio.ErrorCodeNotMessageCapableFromPhoneNumber:
					return nil, grpc.Errorf(excomms.ErrorCodeSMSIncapableFromPhoneNumber, "from phone number %s does not have SMS capabilities", sms.FromPhoneNumber)
				case twilio.ErrorNoSMSSupportToNumber:
					return nil, grpc.Errorf(excomms.ErrorCodeMessageDeliveryFailed, "the `to` phone number %s is not reachable via sms or mms", sms.ToPhoneNumber)
				case twilio.ErrorBlackListRuleViolation:
					return nil, grpc.Errorf(excomms.ErrorCodeMessageDeliveryFailed, "the `to` phone number %s requested to STOP receiving messages from %s so no message can be delivered until subscriber responds with START.", sms.ToPhoneNumber, sms.FromPhoneNumber)
				}
			}
			return nil, errors.Trace(err)
		}
		sentMessage.Message = &models.SentMessage_SMSMsg{
			SMSMsg: &models.SMSMessage{
				FromPhoneNumber: sms.FromPhoneNumber,
				ToPhoneNumber:   sms.ToPhoneNumber,
				Text:            sms.Text,
				ID:              msg.Sid,
				DateCreated:     uint64(msg.DateCreated.Unix()),
				DateSent:        uint64(msg.DateSent.Unix()),
				MediaURLs:       resizedURLs,
			},
		}
	case *excomms.SendMessageRequest_Email:
		email := inMsg.Email

		// ensure that the domain of the sender matches
		// the domain configuration

		ec := e.emailClient
		domainToEnforce := e.spruceEmailDomain

		if email.Transactional {
			ec = e.transactionalEmailClient
			domainToEnforce = e.transactionalEmailDomain
		}

		domain, err := domainFromEmail(email.FromEmailAddress)
		if err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, "Cannot parse domain from email %s: %s", email.FromEmailAddress, err)
		} else if domain != domainToEnforce {
			return nil, grpc.Errorf(codes.InvalidArgument, "Sender (%s) does not match expected domain (%s)", email.FromEmailAddress, domainToEnforce)
		}

		id, err := e.idgen.NewID()
		if err != nil {
			return nil, errors.Trace(err)
		}

		subs := make([]*models.EmailMessage_Substitution, len(email.TemplateSubstitutions))
		for i, s := range email.TemplateSubstitutions {
			subs[i] = &models.EmailMessage_Substitution{Key: s.Key, Value: s.Value}
		}
		sentMessage.Message = &models.SentMessage_EmailMsg{
			EmailMsg: &models.EmailMessage{
				ID:                    strconv.FormatUint(id, 10),
				Subject:               email.Subject,
				Body:                  email.Body,
				FromName:              email.FromName,
				FromEmail:             email.FromEmailAddress,
				ToName:                email.ToName,
				ToEmail:               email.ToEmailAddress,
				MediaURLs:             resizedURLs,
				TemplateID:            email.TemplateID,
				TemplateSubstitutions: subs,
			},
		}
		sentMessage.ID = id

		if err := ec.SendMessage(sentMessage.GetEmailMsg()); err != nil {
			return nil, errors.Trace(err)
		}
	default:
		return nil, errors.Errorf("unknown message type %T", in.Message)
	}

	// persist the message that was sent for tracking purposes
	conc.Go(func() {
		if err := e.dal.CreateSentMessage(sentMessage); err != nil {
			golog.Warningf(err.Error())
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

	// ensure caller belongs to the organization
	sourceEntity, err := directory.SingleEntity(ctx, e.directory, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: in.CallerEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 1,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	})
	if err == directory.ErrEntityNotFound {
		return nil, grpc.Errorf(codes.NotFound, "caller %s not found", in.CallerEntityID)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	var orgEntity *directory.Entity
	for _, m := range sourceEntity.Memberships {
		if m.Type == directory.EntityType_ORGANIZATION && m.ID == in.OrganizationID {
			orgEntity = m
			break
		}
	}
	if orgEntity == nil {
		return nil, grpc.Errorf(codes.NotFound, "%s is not the phone number of a caller belonging to the organization.", in.FromPhoneNumber)
	}

	toPhoneNumber, err := phone.Format(in.ToPhoneNumber, phone.E164)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "phone number %s not formatted: %s", toPhoneNumber, err)
	}

	// validate callee
	var destinationEntity *directory.Entity
	lookupByContacRes, err := e.directory.LookupEntitiesByContact(
		ctx,
		&directory.LookupEntitiesByContactRequest{
			ContactValue: toPhoneNumber,
			RequestedInformation: &directory.RequestedInformation{
				Depth:             0,
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
			},
			Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes:  []directory.EntityType{directory.EntityType_EXTERNAL, directory.EntityType_PATIENT},
			ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
		})
	if grpc.Code(err) == codes.NotFound {
		return nil, grpc.Errorf(codes.NotFound, "callee %s not found", in.ToPhoneNumber)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	// find an external entity for the callee
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

	var originatingPhoneNumber phone.Number
	if in.FromPhoneNumber != "" {
		originatingPhoneNumber, err = phone.ParseNumber(in.FromPhoneNumber)
		if err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, "%s is not a valid phone number", in.FromPhoneNumber)
		}
	} else {
		currentOriginatingPhoneNumber, err := e.dal.CurrentOriginatingNumber(in.CallerEntityID, in.DeviceID)
		if err != nil {
			if errors.Cause(err) != dal.ErrOriginatingNumberNotFound {
				return nil, errors.Trace(err)
			}

			// use a number associated with the provider's account as the originating number
			for _, c := range sourceEntity.Contacts {
				if c.ContactType == directory.ContactType_PHONE && !c.Provisioned {
					originatingPhoneNumber, err = phone.ParseNumber(c.Value)
					if err != nil {
						return nil, errors.Errorf("phone number %q for entity is of invalid format: %s", c.Value, err)
					}
					break
				}
			}
		} else {
			originatingPhoneNumber = currentOriginatingPhoneNumber
		}
	}

	if originatingPhoneNumber.IsEmpty() {
		return nil, errors.Errorf("Unable to find a default phone number for entity %s from which to place the call", sourceEntity.ID)
	}

	// track originating phone number
	conc.Go(func() {
		if err := e.dal.SetCurrentOriginatingNumber(originatingPhoneNumber, in.CallerEntityID, in.DeviceID); err != nil {
			golog.Errorf(err.Error())
		}
	})

	destinationPhoneNumber, err := phone.ParseNumber(in.ToPhoneNumber)
	if err != nil {
		return nil, errors.Errorf("destination phone number %s is of invalid format: %s", in.ToPhoneNumber, err)
	}

	var provisionedPhoneNumberStr string
	val, err := settings.GetTextValue(ctx, e.settings, &settings.GetValuesRequest{
		NodeID: sourceEntity.ID,
		Keys: []*settings.ConfigKey{
			{
				Key: exsettings.ConfigKeyDefaultProvisionedPhoneNumber,
			},
		},
	})
	if err == nil {
		provisionedPhoneNumberStr = val.Value
	} else if errors.Cause(err) != settings.ErrValueNotFound {
		return nil, errors.Errorf("unable to get default number setting for entity %s: %s", sourceEntity.ID, err)
	}
	if provisionedPhoneNumberStr == "" {
		// Use first provisioned number for organization if entity doesn't have a default set
		for _, c := range orgEntity.Contacts {
			if c.Provisioned && c.ContactType == directory.ContactType_PHONE {
				provisionedPhoneNumberStr = c.Value
				break
			}
		}
	}
	provisionedPhoneNumber, err := phone.ParseNumber(provisionedPhoneNumberStr)
	if err != nil {
		return nil, errors.Errorf("failed to parse number %q for entity %s in org %s: %s", provisionedPhoneNumberStr, sourceEntity.ID, orgEntity.ID, err)
	}

	proxyPhoneNumber, err := e.proxyNumberManager.ReserveNumber(originatingPhoneNumber, destinationPhoneNumber, provisionedPhoneNumber, destinationEntity.ID, sourceEntity.ID, in.OrganizationID)
	if err != nil {
		return nil, errors.Trace(err)
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

	emailAddress := strings.ToLower(req.EmailAddress)

	// check if an email has been provisioned for this reason
	provisionedEndpoint, err := e.dal.LookupProvisionedEndpoint(req.ProvisionFor, models.EndpointTypeEmail)
	if err != nil {
		if errors.Cause(err) != dal.ErrProvisionedEndpointNotFound {
			return nil, errors.Trace(err)
		}
	} else if provisionedEndpoint.Endpoint == emailAddress {
		return &excomms.ProvisionEmailAddressResponse{
			EmailAddress: emailAddress,
		}, nil
	} else {
		return nil, grpc.Errorf(codes.AlreadyExists, "Different email address (%s) provisioned for %s", provisionedEndpoint.Endpoint, req.ProvisionFor)
	}

	// if not, provision it
	if err := e.dal.ProvisionEndpoint(&models.ProvisionedEndpoint{
		EndpointType:   models.EndpointTypeEmail,
		ProvisionedFor: req.ProvisionFor,
		Endpoint:       emailAddress,
	}, ""); err != nil {
		return nil, errors.Trace(err)
	}

	events.Publish(e.sns, e.eventTopic, events.Service_EXCOMMS, &excomms.Event{
		Type: excomms.Event_PROVISIONED_ENDPOINT,
		Details: &excomms.Event_ProvisionedEndpoint{
			ProvisionedEndpoint: &excomms.ProvisionedEndpoint{
				ForEntityID:  req.ProvisionFor,
				EndpointType: excomms.EndpointType_EMAIL,
				Endpoint:     emailAddress,
			},
		},
	})

	return &excomms.ProvisionEmailAddressResponse{
		EmailAddress: emailAddress,
	}, nil
}

func (e *excommsService) BlockNumber(ctx context.Context, in *excomms.BlockNumberRequest) (*excomms.BlockNumberResponse, error) {
	provisionedPhoneNumber, err := phone.ParseNumber(in.ProvisionedPhoneNumber)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "invalid phone number %s: %s", in.ProvisionedPhoneNumber, err)
	}

	blockedPhoneNumber, err := phone.ParseNumber(in.Number)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "invalid phone number %q: %s", in.Number, err)
	}

	// ensure that the number is provisioned by the org listed in the request
	pe, err := e.dal.LookupProvisionedEndpoint(in.OrgID, models.EndpointTypePhone)
	if errors.Cause(err) == dal.ErrProvisionedEndpointNotFound {
		return nil, grpc.Errorf(codes.NotFound, "provisioned phone number %s not found", in.ProvisionedPhoneNumber)
	} else if err != nil {
		return nil, errors.Trace(err)
	} else if pe.Endpoint != provisionedPhoneNumber.String() {
		return nil, grpc.Errorf(codes.InvalidArgument, "phone number %s not owned by %s", in.ProvisionedPhoneNumber, in.OrgID)
	}

	if err := e.dal.InsertBlockedNumber(ctx, blockedPhoneNumber, provisionedPhoneNumber); err != nil {
		return nil, errors.Trace(err)
	}

	blockedNumbers, err := e.dal.LookupBlockedNumbers(ctx, provisionedPhoneNumber)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &excomms.BlockNumberResponse{
		Numbers: blockedNumbers.ToStringSlice(),
	}, nil
}

func (e *excommsService) UnblockNumber(ctx context.Context, in *excomms.UnblockNumberRequest) (*excomms.UnblockNumberResponse, error) {
	provisionedPhoneNumber, err := phone.ParseNumber(in.ProvisionedPhoneNumber)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "invalid phone number %s: %s", in.ProvisionedPhoneNumber, err)
	}

	blockedPhoneNumber, err := phone.ParseNumber(in.Number)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "invalid phone number %q: %s", in.Number, err)
	}

	if err := e.dal.DeleteBlockedNumber(ctx, blockedPhoneNumber, provisionedPhoneNumber); err != nil {
		return nil, errors.Trace(err)
	}

	blockedNumbers, err := e.dal.LookupBlockedNumbers(ctx, provisionedPhoneNumber)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &excomms.UnblockNumberResponse{
		Numbers: blockedNumbers.ToStringSlice(),
	}, nil
}
func (e *excommsService) ListBlockedNumbers(ctx context.Context, in *excomms.ListBlockedNumbersRequest) (*excomms.ListBlockedNumbersResponse, error) {
	provisionedPhoneNumber, err := phone.ParseNumber(in.ProvisionedPhoneNumber)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "invalid phone number %s: %s", in.ProvisionedPhoneNumber, err)
	}

	blockedNumbers, err := e.dal.LookupBlockedNumbers(ctx, provisionedPhoneNumber)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &excomms.ListBlockedNumbersResponse{
		Numbers: blockedNumbers.ToStringSlice(),
	}, nil

}
