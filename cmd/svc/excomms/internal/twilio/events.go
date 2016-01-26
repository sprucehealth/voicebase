package twilio

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/sns"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/twilio/twiml"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"golang.org/x/net/context"
)

var (
	twilioEventsHandlers = map[rawmsg.TwilioEvent]twilioEventHandleFunc{
		rawmsg.TwilioEvent_PROCESS_INCOMING_CALL:        processIncomingCall,
		rawmsg.TwilioEvent_PROCESS_OUTGOING_CALL:        processOutgoingCall,
		rawmsg.TwilioEvent_PROVIDER_ENTERED_DIGITS:      providerEnteredDigits,
		rawmsg.TwilioEvent_PROVIDER_CALL_CONNECTED:      providerCallConnected,
		rawmsg.TwilioEvent_TWIML_REQUESTED_VOICEMAIL:    voicemailTWIML,
		rawmsg.TwilioEvent_PROCESS_INCOMING_CALL_STATUS: processIncomingCallStatus,
		rawmsg.TwilioEvent_PROCESS_VOICEMAIL:            processVoicemail,
		rawmsg.TwilioEvent_PROCESS_OUTGOING_CALL_STATUS: processOutgoingCallStatus,
	}
	maxPhoneNumbers = 10
)

type eventsHandler struct {
	directory            directory.DirectoryClient
	dal                  dal.DAL
	sns                  snsiface.SNSAPI
	clock                clock.Clock
	apiURL               string
	externalMessageTopic string
	incomingRawMsgTopic  string
}

func NewEventHandler(directory directory.DirectoryClient, dal dal.DAL, sns snsiface.SNSAPI, clock clock.Clock, apiURL, externalMessageTopic, incomingRawMsgTopic string) EventHandler {
	return &eventsHandler{
		directory:            directory,
		dal:                  dal,
		clock:                clock,
		sns:                  sns,
		apiURL:               apiURL,
		externalMessageTopic: externalMessageTopic,
		incomingRawMsgTopic:  incomingRawMsgTopic,
	}
}

func (e *eventsHandler) Process(event rawmsg.TwilioEvent, params *rawmsg.TwilioParams) (string, error) {
	handler := twilioEventsHandlers[event]
	if handler == nil {
		return "", fmt.Errorf("unknown event: %s", event.String())
	}
	twiml, err := handler(params, e)
	if err != nil {
		return "", errors.Trace(err)
	}

	conc.Go(func() {
		if err := e.dal.LogCallEvent(&models.CallEvent{
			Data:        params,
			Type:        event.String(),
			Source:      params.From,
			Destination: params.To,
		}); err != nil {
			golog.Errorf("Unable to log event %s: %s", event.String(), err.Error())
		}
	})
	return twiml, nil
}

type EventHandler interface {
	Process(event rawmsg.TwilioEvent, params *rawmsg.TwilioParams) (string, error)
}

type twilioEventHandleFunc func(*rawmsg.TwilioParams, *eventsHandler) (string, error)

func processOutgoingCall(params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

	// look for an active reservation on the proxy phone number
	ppnr, err := eh.dal.ActiveProxyPhoneNumberReservation(&dal.ProxyPhoneNumberReservationLookup{
		ProxyPhoneNumber: ptr.String(params.To),
	})
	if errors.Cause(err) == dal.ErrProxyPhoneNumberReservationNotFound {
		return "", errors.Trace(fmt.Errorf("No active reservation found for %s", params.To))
	} else if err != nil {
		return "", errors.Trace(err)
	}

	// look up the practice phone number using the organizationID
	res, err := eh.directory.LookupEntities(
		context.Background(),
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: ppnr.OrganizationID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 1,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_MEMBERS,
					directory.EntityInformation_MEMBERSHIPS,
					directory.EntityInformation_CONTACTS,
				},
			},
		})
	if err != nil {
		return "", errors.Trace(err)
	}

	if len(res.Entities) != 1 {
		return "", errors.Trace(fmt.Errorf("Expected 1 entity. Got %d", len(res.Entities)))
	}

	orgEntity := res.Entities[0]
	if orgEntity.Type != directory.EntityType_ORGANIZATION {
		return "", errors.Trace(fmt.Errorf("Expected entity to be of type %s but got type %s", directory.EntityType_ORGANIZATION.String(), orgEntity.Type.String()))
	}

	var practicePhoneNumber string
	for _, c := range orgEntity.Contacts {
		if c.Provisioned && c.ContactType == directory.ContactType_PHONE {
			practicePhoneNumber = c.Value
		}
	}
	if practicePhoneNumber == "" {
		return "", errors.Trace(fmt.Errorf("Unable to find practice phone number for org %s", orgEntity.ID))
	}

	// lookup phone number of external entity to call
	res, err = eh.directory.LookupEntities(
		context.Background(),
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: ppnr.DestinationEntityID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 1,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
				},
			},
		})
	if err != nil {
		return "", errors.Trace(err)
	} else if len(res.Entities) != 1 {
		return "", errors.Trace(fmt.Errorf("Expected 1 external entity but got %d", len(res.Entities)))
	}

	var destinationPhoneNumber string
	for _, c := range res.Entities[0].Contacts {
		if c.Provisioned {
			continue
		} else if c.ContactType != directory.ContactType_PHONE {
			continue
		}
		destinationPhoneNumber = c.Value
		break
	}

	if destinationPhoneNumber == "" {
		return "", errors.Trace(fmt.Errorf("Unable to find phone number to call for entity %s", ppnr.DestinationEntityID))
	}

	source, err := phone.ParseNumber(params.From)
	if err != nil {
		return "", errors.Trace(err)
	}
	destination, err := phone.ParseNumber(destinationPhoneNumber)
	if err != nil {
		return "", errors.Trace(err)
	}

	if err := eh.dal.CreateCallRequest(&models.CallRequest{
		Source:         source,
		Destination:    destination,
		Proxy:          ppnr.PhoneNumber,
		OrganizationID: ppnr.OrganizationID,
		CallSID:        params.CallSID,
		Requested:      eh.clock.Now(),
	}); err != nil {
		return "", errors.Trace(err)
	}

	var text string
	if res.Entities[0].Info != nil && res.Entities[0].Info.DisplayName != "" {
		text = "You will be connected to " + res.Entities[0].Info.DisplayName
	} else {
		formattedNumber, err := destination.Format(phone.National)
		if err != nil {
			golog.Errorf(err.Error())
			text = "You will be connected to the patient"
		} else {
			text = "You will be connected to " + formattedNumber
		}
	}

	tw := twiml.NewResponse(
		&twiml.Say{
			Text:  text,
			Voice: "alice",
		},
		&twiml.Dial{
			CallerID: practicePhoneNumber,
			Nouns: []interface{}{
				&twiml.Number{
					StatusCallbackEvent: twiml.SCRinging | twiml.SCAnswered | twiml.SCCompleted,
					StatusCallback:      fmt.Sprintf("%s/twilio/call/process_outgoing_call_status", eh.apiURL),
					Text:                destinationPhoneNumber,
				},
			},
		})

	return tw.GenerateTwiML()
}

func processIncomingCall(params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	golog.Infof("Incoming call %s to %s.", params.From, params.To)

	// lookup the entity for the destination of the incoming call
	res, err := eh.directory.LookupEntitiesByContact(
		context.Background(),
		&directory.LookupEntitiesByContactRequest{
			ContactValue: params.To,
			RequestedInformation: &directory.RequestedInformation{
				Depth: 2,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_MEMBERS,
					directory.EntityInformation_MEMBERSHIPS,
					directory.EntityInformation_CONTACTS,
				},
			},
		})
	if err != nil {
		return "", errors.Trace(err)
	}

	// we should get back a single entity at this point given that there should be a 1:1 mapping between a provisioned number
	// and an entity
	if len(res.Entities) != 1 {
		return "", errors.Trace(fmt.Errorf("Expected 1 entity for provisioned number, got back %d", len(res.Entities)))
	}

	golog.Debugf("response %+v", res.Entities)

	var phoneNumbers []string
	var organizationID string
	switch res.Entities[0].Type {
	case directory.EntityType_ORGANIZATION:
		organizationID = res.Entities[0].ID
		phoneNumbers = make([]string, 0, len(res.Entities[0].Contacts))
		for _, c := range res.Entities[0].Contacts {
			if c.Provisioned {
				continue
			} else if c.ContactType != directory.ContactType_PHONE {
				continue
			}

			phoneNumbers = append(phoneNumbers, c.Value)
		}
	case directory.EntityType_INTERNAL:
		for _, c := range res.Entities[0].Contacts {
			if c.Provisioned {
				continue
			} else if c.ContactType != directory.ContactType_PHONE {
				continue
			}
			// assuming for now that we are to call the first non-provisioned
			// phone number mapped to the provider.
			phoneNumbers = append(phoneNumbers, c.Value)
			break
		}

		for _, m := range res.Entities[0].Memberships {
			if m.Type == directory.EntityType_ORGANIZATION {
				organizationID = m.ID
				break
			}
		}
	default:
		return "", errors.Trace(fmt.Errorf("Unexpected entity type %s", res.Entities[0].Type.String()))
	}

	if len(phoneNumbers) == 0 {
		return "", errors.Trace(fmt.Errorf("Unable to find provider for provisioned number %s", params.To))
	} else if organizationID == "" {
		return "", errors.Trace(fmt.Errorf("Unable to find organization for provisioned number %s", params.To))
	}

	numbers := make([]interface{}, 0, maxPhoneNumbers)
	for _, p := range phoneNumbers {
		if len(numbers) == maxPhoneNumbers {
			golog.Errorf("Org %s is currently configured to simultaneously call more than 10 numbers when that is the maximum that twilio supports.", organizationID)
			break
		}
		numbers = append(numbers, &twiml.Number{
			URL:  "/twilio/call/provider_call_connected",
			Text: p,
		})
	}

	tw := twiml.NewResponse(
		&twiml.Dial{
			CallerID:         params.To,
			TimeoutInSeconds: 30,
			Action:           "/twilio/call/process_incoming_call_status",
			Nouns:            numbers,
		},
	)

	return tw.GenerateTwiML()
}

func providerCallConnected(params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	golog.Infof("Call connected for provider at %s.", params.To)

	tw := twiml.NewResponse(
		&twiml.Gather{
			Action:           "/twilio/call/provider_entered_digits",
			Method:           "POST",
			TimeoutInSeconds: 5,
			NumDigits:        1,
			Verbs: []interface{}{
				&twiml.Say{
					Voice: "woman",
					Text:  "You have an incoming call. Press 1 to answer.",
				},
			},
		},
		// In the event that no key is entered, we hang up the
		// dialed call to then direct the caller to voicemail.
		&twiml.Hangup{},
	)

	return tw.GenerateTwiML()
}

func providerEnteredDigits(params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	golog.Infof("Provider entered digits %s at %s.", params.Digits, params.To)

	if params.Digits == "1" {
		// accept the call if the provider entered the right digit
		// by generating an empty response.
		tw := twiml.NewResponse()
		return tw.GenerateTwiML()
	}

	// hangup they key on the provider side if any key other than 1 is pressed.
	tw := twiml.NewResponse(&twiml.Hangup{})
	return tw.GenerateTwiML()
}

func voicemailTWIML(params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

	// TODO: Configurable voice mail or default mehsage based on user configuration.
	tw := &twiml.Response{
		Verbs: []interface{}{
			&twiml.Play{
				Text: "http://dev-twilio.s3.amazonaws.com/kunal_clinic_voicemail.mp3",
			},
			&twiml.Record{
				Action:           "/twilio/call/process_voicemail",
				PlayBeep:         true,
				TimeoutInSeconds: 60,
			},
		},
	}

	return tw.GenerateTwiML()
}

func processIncomingCallStatus(params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	switch params.DialCallStatus {
	case rawmsg.TwilioParams_ANSWERED, rawmsg.TwilioParams_COMPLETED:
		conc.Go(func() {
			if err := sns.Publish(eh.sns, eh.externalMessageTopic, &excomms.PublishedExternalMessage{
				FromChannelID: params.From,
				ToChannelID:   params.To,
				Timestamp:     uint64(time.Now().Unix()),
				Direction:     excomms.PublishedExternalMessage_INBOUND,
				Type:          excomms.PublishedExternalMessage_CALL_EVENT,
				Item: &excomms.PublishedExternalMessage_CallEventItem{
					CallEventItem: &excomms.CallEventItem{
						Type:              excomms.CallEventItem_INCOMING_ANSWERED,
						DurationInSeconds: params.CallDuration,
					},
				},
			}); err != nil {
				golog.Errorf(err.Error())
			}
		})

	case rawmsg.TwilioParams_CALL_STATUS_UNDEFINED:
	default:
		return voicemailTWIML(params, eh)
	}

	return "", nil
}

func processOutgoingCallStatus(params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

	// NOTE: We use the callSID of the parent call to process the status of the outgoing
	// call placed as the outgoing call is dialed out via a separate call leg.
	// This is under the assumption that the outgoing call from provider to external
	// entity was placed via the Dial verb.
	if params.ParentCallSID == "" {
		golog.Debugf("Nothing to do because params.ParentCallSID is empty")
		// nothing to do
		return "", nil
	}

	cr, err := eh.dal.LookupCallRequest(params.ParentCallSID)
	if errors.Cause(err) == dal.ErrCallRequestNotFound {
		return "", errors.Trace(fmt.Errorf("No call requeht found for call sid %s", params.ParentCallSID))
	} else if err != nil {
		return "", errors.Trace(err)
	}

	var cet *excomms.PublishedExternalMessage_CallEventItem
	switch params.CallStatus {
	case rawmsg.TwilioParams_RINGING:
		cet = &excomms.PublishedExternalMessage_CallEventItem{
			CallEventItem: &excomms.CallEventItem{
				Type:              excomms.CallEventItem_OUTGOING_PLACED,
				DurationInSeconds: params.CallDuration,
			},
		}
	case rawmsg.TwilioParams_ANSWERED, rawmsg.TwilioParams_COMPLETED:
		cet = &excomms.PublishedExternalMessage_CallEventItem{
			CallEventItem: &excomms.CallEventItem{
				Type:              excomms.CallEventItem_OUTGOING_ANSWERED,
				DurationInSeconds: params.CallDuration,
			},
		}
	default:
		// nothing to do
		golog.Debugf("Ignoring call status %s", params.CallStatus.String())
		return "", nil
	}

	conc.Go(func() {
		if err := sns.Publish(eh.sns, eh.externalMessageTopic, &excomms.PublishedExternalMessage{
			FromChannelID: cr.Source.String(),
			ToChannelID:   cr.Destination.String(),
			Direction:     excomms.PublishedExternalMessage_OUTBOUND,
			Timestamp:     uint64(time.Now().Unix()),
			Type:          excomms.PublishedExternalMessage_CALL_EVENT,
			Item:          cet,
		}); err != nil {
			golog.Errorf(err.Error())
		}
	})

	return "", nil
}

func processVoicemail(params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

	rawMessageID, err := eh.dal.StoreIncomingRawMessage(&rawmsg.Incoming{
		Type: rawmsg.Incoming_TWILIO_VOICEMAIL,
		Message: &rawmsg.Incoming_Twilio{
			Twilio: params,
		},
	})
	if err != nil {
		return "", errors.Trace(err)
	}

	conc.Go(func() {
		if err := sns.Publish(eh.sns, eh.incomingRawMsgTopic, &sns.IncomingRawMessageNotification{
			ID: rawMessageID,
		}); err != nil {
			golog.Errorf(err.Error())
		}
	})

	return "", nil
}
