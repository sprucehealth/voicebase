package internal

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/twilio/twiml"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"

	"golang.org/x/net/context"
)

var (
	twilioEventsHandlers = map[excomms.TwilioEvent]twilioEventHandleFunc{
		excomms.TwilioEvent_PROCESS_INCOMING_CALL:        processIncomingCall,
		excomms.TwilioEvent_PROCESS_OUTGOING_CALL:        processOutgoingCall,
		excomms.TwilioEvent_PROVIDER_ENTERED_DIGITS:      providerEnteredDigits,
		excomms.TwilioEvent_PROVIDER_CALL_CONNECTED:      providerCallConnected,
		excomms.TwilioEvent_TWIML_REQUESTED_VOICEMAIL:    voicemailTWIML,
		excomms.TwilioEvent_INCOMING_SMS:                 processIncomingSMS,
		excomms.TwilioEvent_PROCESS_INCOMING_CALL_STATUS: processIncomingCallStatus,
		excomms.TwilioEvent_PROCESS_VOICEMAIL:            processVoicemail,
		excomms.TwilioEvent_PROCESS_OUTGOING_CALL_STATUS: processOutgoingCallStatus,
	}
	maxPhoneNumbers = 10
)

type twilioEventHandleFunc func(*excomms.TwilioParams, *excommsService) (string, error)

func processOutgoingCall(params *excomms.TwilioParams, es *excommsService) (string, error) {
	cr, err := es.dal.ValidCallRequest(params.From)
	if errors.Cause(err) == dal.ErrCallRequestNotFound {
		return "", errors.Trace(fmt.Errorf("No call request found for %s", params.CallSID))
	} else if err != nil {
		return "", errors.Trace(err)
	}

	if time.Now().After(cr.Expires) {
		return "", errors.Trace(fmt.Errorf("Call request has expired for call from %s to %s", params.From, cr.Destination))
	}

	// look up the practice phone number using the organizationID
	res, err := es.directory.LookupEntities(
		context.Background(),
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: cr.OrganizationID,
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

	if rowsAffected, err := es.dal.UpdateCallRequest(params.From, params.CallSID); err != nil {
		return "", errors.Trace(err)
	} else if rowsAffected != 1 {
		return "", errors.Trace(fmt.Errorf("Expected to update a single call request, instead updated %d call requests with call sid %s", rowsAffected, params.CallSID))
	}

	tw := twiml.NewResponse(
		&twiml.Dial{
			CallerID: practicePhoneNumber,
			Nouns: []interface{}{
				&twiml.Number{
					StatusCallbackEvent: twiml.SCRinging | twiml.SCAnswered | twiml.SCCompleted,
					StatusCallback:      fmt.Sprintf("%s/twilio/process_outgoing_call_status", es.apiURL),
					Text:                cr.Destination,
				},
			},
		})

	return tw.GenerateTwiML()
}

func processIncomingCall(params *excomms.TwilioParams, es *excommsService) (string, error) {
	golog.Infof("Incoming call %s to %s.", params.From, params.To)

	// lookup the entity for the destination of the incoming call
	res, err := es.directory.LookupEntitiesByContact(
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
			URL:  "/twilio/provider_call_connected",
			Text: p,
		})
	}

	tw := twiml.NewResponse(
		&twiml.Dial{
			CallerID:         params.To,
			TimeoutInSeconds: 30,
			Action:           "/twilio/process_incoming_call_status",
			Nouns:            numbers,
		},
	)

	return tw.GenerateTwiML()
}

func providerCallConnected(params *excomms.TwilioParams, es *excommsService) (string, error) {
	golog.Infof("Call connected for provider at %s.", params.To)

	tw := twiml.NewResponse(
		&twiml.Gather{
			Action:           "/twilio/provider_entered_digits",
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

func providerEnteredDigits(params *excomms.TwilioParams, es *excommsService) (string, error) {
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

func voicemailTWIML(params *excomms.TwilioParams, es *excommsService) (string, error) {

	// TODO: Configurable voice mail or default message based on user configuration.
	tw := &twiml.Response{
		Verbs: []interface{}{
			&twiml.Play{
				Text: "http://dev-twilio.s3.amazonaws.com/kunal_clinic_voicemail.mp3",
			},
			&twiml.Record{
				Action:           "/twilio/process_voicemail",
				PlayBeep:         true,
				TimeoutInSeconds: 60,
			},
		},
	}

	return tw.GenerateTwiML()
}

func processIncomingCallStatus(params *excomms.TwilioParams, es *excommsService) (string, error) {
	switch params.DialCallStatus {
	case excomms.TwilioParams_ANSWERED, excomms.TwilioParams_COMPLETED:
		publishToSNSTopic(es.sns, es.externalMessageTopic, &excomms.PublishedExternalMessage{
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
		})
	case excomms.TwilioParams_CALL_STATUS_UNDEFINED:
	default:
		return voicemailTWIML(params, es)
	}

	return "", nil
}

func processOutgoingCallStatus(params *excomms.TwilioParams, es *excommsService) (string, error) {

	// NOTE: We use the callSID of the parent call to process the status of the outgoing
	// call placed as the outgoing call is dialed out via a separate call leg.
	// This is under the assumption that the outgoing call from provider to external
	// entity was placed via the Dial verb.
	if params.ParentCallSID == "" {
		golog.Debugf("Nothing to do because params.ParentCallSID is empty")
		// nothing to do
		return "", nil
	}

	cr, err := es.dal.LookupCallRequest(params.ParentCallSID)
	if errors.Cause(err) == dal.ErrCallRequestNotFound {
		return "", errors.Trace(fmt.Errorf("No call request found for call sid %s", params.ParentCallSID))
	} else if err != nil {
		return "", errors.Trace(err)
	}

	var cet *excomms.PublishedExternalMessage_CallEventItem
	switch params.CallStatus {
	case excomms.TwilioParams_RINGING:
		cet = &excomms.PublishedExternalMessage_CallEventItem{
			CallEventItem: &excomms.CallEventItem{
				Type:              excomms.CallEventItem_OUTGOING_PLACED,
				DurationInSeconds: params.CallDuration,
			},
		}
	case excomms.TwilioParams_ANSWERED, excomms.TwilioParams_COMPLETED:
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

	publishToSNSTopic(es.sns, es.externalMessageTopic, &excomms.PublishedExternalMessage{
		FromChannelID: cr.Source,
		ToChannelID:   cr.Destination,
		Direction:     excomms.PublishedExternalMessage_OUTBOUND,
		Timestamp:     uint64(time.Now().Unix()),
		Type:          excomms.PublishedExternalMessage_CALL_EVENT,
		Item:          cet,
	})

	return "", nil
}

func processVoicemail(params *excomms.TwilioParams, es *excommsService) (string, error) {

	publishToSNSTopic(es.sns, es.externalMessageTopic, &excomms.PublishedExternalMessage{
		FromChannelID: params.From,
		ToChannelID:   params.To,
		Timestamp:     uint64(time.Now().Unix()),
		Type:          excomms.PublishedExternalMessage_CALL_EVENT,
		Direction:     excomms.PublishedExternalMessage_INBOUND,
		Item: &excomms.PublishedExternalMessage_CallEventItem{
			CallEventItem: &excomms.CallEventItem{
				Type:              excomms.CallEventItem_INCOMING_LEFT_VOICEMAIL,
				DurationInSeconds: params.RecordingDuration,
				// TODO: Until we start downloading and serving recordings from S3,
				// appending .mp3 here so that the client gets the recordingURL in mp3 format.
				URL: params.RecordingURL + ".mp3",
			},
		},
	})

	return "", nil
}

func processIncomingSMS(params *excomms.TwilioParams, es *excommsService) (string, error) {

	smsItem := &excomms.PublishedExternalMessage_SMSItem{
		SMSItem: &excomms.SMSItem{
			Text:        params.Body,
			Attachments: make([]*excomms.MediaAttachment, params.NumMedia),
		},
	}

	for i, m := range params.MediaItems {
		smsItem.SMSItem.Attachments[i] = &excomms.MediaAttachment{
			URL:         m.MediaURL,
			ContentType: m.ContentType,
		}
	}

	publishToSNSTopic(es.sns, es.externalMessageTopic, &excomms.PublishedExternalMessage{
		FromChannelID: params.From,
		ToChannelID:   params.To,
		Timestamp:     uint64(time.Now().Unix()),
		Direction:     excomms.PublishedExternalMessage_INBOUND,
		Type:          excomms.PublishedExternalMessage_SMS,
		Item:          smsItem,
	})

	// empty response indicates twilio not to send a response to the incoming SMS.
	tw := twiml.Response{}

	return tw.GenerateTwiML()
}
