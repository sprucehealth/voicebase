package twilio

import (
	"fmt"
	"html"
	"time"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	analytics "github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/cleaner"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/proxynumber"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/sns"
	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"

	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/twilio/twiml"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/settings"
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
		rawmsg.TwilioEvent_PROCESS_SMS_STATUS:           processOutgoingSMSStatus,
		rawmsg.TwilioEvent_NO_OP:                        processNoOp,
	}
	maxPhoneNumbers = 10
)

type eventsHandler struct {
	directory            directory.DirectoryClient
	settings             settings.SettingsClient
	dal                  dal.DAL
	store                storage.DeterministicStore
	sns                  snsiface.SNSAPI
	clock                clock.Clock
	proxyNumberManager   proxynumber.Manager
	apiURL               string
	externalMessageTopic string
	incomingRawMsgTopic  string
	resourceCleanerTopic string
	segmentClient        *analytics.Client
}

func NewEventHandler(
	directory directory.DirectoryClient,
	settingsClient settings.SettingsClient,
	dal dal.DAL,
	sns snsiface.SNSAPI,
	clock clock.Clock,
	proxyNumberManager proxynumber.Manager,
	apiURL, externalMessageTopic, incomingRawMsgTopic, resourceCleanerTopic string,
	segmentClient *analytics.Client,
	store storage.DeterministicStore) EventHandler {
	return &eventsHandler{
		directory:            directory,
		settings:             settingsClient,
		dal:                  dal,
		clock:                clock,
		sns:                  sns,
		apiURL:               apiURL,
		externalMessageTopic: externalMessageTopic,
		incomingRawMsgTopic:  incomingRawMsgTopic,
		proxyNumberManager:   proxyNumberManager,
		resourceCleanerTopic: resourceCleanerTopic,
		segmentClient:        segmentClient,
		store:                store,
	}
}

func (e *eventsHandler) Process(ctx context.Context, event rawmsg.TwilioEvent, params *rawmsg.TwilioParams) (string, error) {
	handler := twilioEventsHandlers[event]
	if handler == nil {
		return "", fmt.Errorf("unknown event: %s", event.String())
	}
	twiml, err := handler(ctx, params, e)
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
	Process(ctx context.Context, event rawmsg.TwilioEvent, params *rawmsg.TwilioParams) (string, error)
}

type twilioEventHandleFunc func(context.Context, *rawmsg.TwilioParams, *eventsHandler) (string, error)

func processNoOp(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	return "", nil
}

func processOutgoingCall(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

	originatingPhoneNumber, err := phone.ParseNumber(params.From)
	if err != nil {
		return "", errors.Trace(err)
	}

	proxyPhoneNumber, err := phone.ParseNumber(params.To)
	if err != nil {
		return "", errors.Trace(err)
	}

	// look for an active reservation on the proxy phone number
	ppnr, err := eh.proxyNumberManager.ActiveReservation(originatingPhoneNumber, proxyPhoneNumber)
	if err != nil {
		golog.Errorf(err.Error())
		return twiml.NewResponse(
			&twiml.Say{
				Text:  "Outbound calls to patients should be initiated from within the Spruce app. Please hang up and call the patient you are trying to reach by tapping the phone icon within their conversation thread. Thank you!",
				Voice: "alice",
			}).GenerateTwiML()
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
			Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		})
	if err != nil {
		return "", errors.Trace(err)
	} else if len(res.Entities) != 1 {
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

	if err := eh.proxyNumberManager.CallStarted(originatingPhoneNumber, proxyPhoneNumber); err != nil {
		return "", errors.Trace(err)
	}

	if err := eh.dal.CreateCallRequest(&models.CallRequest{
		Source:         originatingPhoneNumber,
		Destination:    ppnr.DestinationPhoneNumber,
		Proxy:          proxyPhoneNumber,
		OrganizationID: ppnr.OrganizationID,
		CallSID:        params.CallSID,
		Requested:      eh.clock.Now(),
		CallerEntityID: ppnr.OwnerEntityID,
		CalleeEntityID: ppnr.DestinationEntityID,
	}); err != nil {
		return "", errors.Trace(err)
	}

	// lookup external entity for name
	res, err = eh.directory.LookupEntities(
		context.Background(),
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: ppnr.DestinationEntityID,
			},
			Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		})
	if err != nil {
		return "", errors.Trace(err)
	} else if len(res.Entities) != 1 {
		return "", errors.Trace(fmt.Errorf("Expected 1 entity. Got %d", len(res.Entities)))
	}

	var text string
	if res.Entities[0].Info != nil && res.Entities[0].Info.DisplayName != "" {
		text = "You will be connected to " + res.Entities[0].Info.DisplayName
	} else {
		formattedNumber, err := ppnr.DestinationPhoneNumber.Format(phone.National)
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
					Text:                ppnr.DestinationPhoneNumber.String(),
				},
			},
		})

	return tw.GenerateTwiML()
}

func processIncomingCall(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	golog.Infof("Incoming call %s to %s.", params.From, params.To)

	entity, err := determineEntityWithProvisionedEndpoint(eh, params.To, 2)
	if err != nil {
		return "", errors.Trace(err)
	}

	var forwardingList []string
	var organizationID string
	var providersInOrg []*directory.Entity
	switch entity.Type {
	case directory.EntityType_ORGANIZATION:
		organizationID = entity.ID

		forwardingList, err = getForwardingListForProvisionedPhoneNumber(ctx, params.To, organizationID, eh)
		if err != nil {
			return "", errors.Trace(err)
		}

		for _, member := range entity.Members {
			if member.Type == directory.EntityType_INTERNAL {
				providersInOrg = append(providersInOrg, member)
			}
		}

	case directory.EntityType_INTERNAL:
		for _, c := range entity.Contacts {
			if c.Provisioned {
				continue
			} else if c.ContactType != directory.ContactType_PHONE {
				continue
			}
			// assuming for now that we are to call the first non-provisioned
			// phone number mapped to the provider.
			forwardingList = []string{c.Value}
			break
		}

		for _, m := range entity.Memberships {
			if m.Type == directory.EntityType_ORGANIZATION {
				organizationID = m.ID
				break
			}
		}

		providersInOrg = []*directory.Entity{entity}

	default:
		return "", errors.Trace(fmt.Errorf("Unexpected entity type %s", entity.Type.String()))
	}

	if organizationID == "" {
		return "", errors.Trace(fmt.Errorf("Unable to find organization for provisioned number %s", params.To))
	}

	source, err := phone.ParseNumber(params.From)
	if err != nil {
		return "", errors.Trace(err)
	}
	destination, err := phone.ParseNumber(params.To)
	if err != nil {
		return "", errors.Trace(err)
	}

	if err := eh.dal.CreateIncomingCall(&models.IncomingCall{
		Source:         source,
		Destination:    destination,
		OrganizationID: organizationID,
		CallSID:        params.CallSID,
	}); err != nil {
		return "", errors.Trace(err)
	}

	// check if the send all calls to voicemail flag is turned on for the phone number
	// at the org level. If so, then direct the call to voicemail rather than ringing any number in the call list.
	sendAllCallsToVoicemailValue, err := settings.GetBooleanValue(ctx, eh.settings, &settings.GetValuesRequest{
		NodeID: organizationID,
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeySendCallsToVoicemail,
				Subkey: params.To,
			},
		},
	})
	if err != nil {
		return "", errors.Trace(fmt.Errorf("Unable to get the setting to direct all calls to voicemail for org %s: %s", organizationID, err.Error()))
	} else if sendAllCallsToVoicemailValue.Value {
		return voicemailTWIML(ctx, params, eh)
	}

	numbers := make([]interface{}, 0, maxPhoneNumbers)
	for _, p := range forwardingList {
		parsedPn, err := phone.Format(p, phone.E164)
		if err != nil {
			golog.Errorf("Unable to parse phone number %s: %s", p, err.Error())
			continue
		}
		if len(numbers) == maxPhoneNumbers {
			golog.Errorf("Org %s is currently configured to simultaneously call more than 10 numbers when that is the maximum that twilio supports.", organizationID)
			break
		}

		numbers = append(numbers, &twiml.Number{
			URL:  "/twilio/call/provider_call_connected",
			Text: parsedPn,
		})
	}

	// if there are no numbers in the forwarding list, then direct calls to voicemail
	if len(numbers) == 0 {
		return voicemailTWIML(ctx, params, eh)
	}

	// put the incoming call into the queue to be deleted once the call is complete.
	cleaner.Publish(eh.sns, eh.resourceCleanerTopic, &models.DeleteResourceRequest{
		Type:       models.DeleteResourceRequest_TWILIO_CALL,
		ResourceID: params.CallSID,
	})

	tw := twiml.NewResponse(
		&twiml.Pause{
			Length: uint(2),
		},
		&twiml.Dial{
			CallerID:         params.To,
			TimeoutInSeconds: 30,
			Action:           "/twilio/call/process_incoming_call_status",
			Nouns:            numbers,
		},
	)

	return tw.GenerateTwiML()
}

func providerCallConnected(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	golog.Infof("Call connected for provider at %s.", params.To)

	// lookup the parent call information to be able to announce the name of the patient if we have it
	// use the parentCallSID as that identifies the originating call, given that this particular call leg to the provider
	// stems from that call.
	incomingCall, err := eh.dal.LookupIncomingCall(params.ParentCallSID)
	if err != nil {
		return "", errors.Trace(err)
	}

	externalEntityName, err := determineExternalEntityName(ctx, incomingCall.Source, incomingCall.OrganizationID, eh)
	if err != nil {
		golog.Errorf("Unable to determine external entity name based on call sid %s. Error: %s", params.ParentCallSID, err.Error())
	}

	// if no name is found, then use the phone number itself.
	if externalEntityName == "" {
		externalEntityName, err = incomingCall.Source.Format(phone.National)
		if err != nil {
			golog.Errorf("Unable to format number %s. Error: %s", incomingCall.Source.String(), err.Error())
		}
	}

	tw := twiml.NewResponse(
		&twiml.Gather{
			Action:           "/twilio/call/provider_entered_digits",
			Method:           "POST",
			TimeoutInSeconds: 5,
			NumDigits:        1,
			Verbs: []interface{}{
				&twiml.Say{
					Voice: "alice",
					Text:  fmt.Sprintf("You have an incoming call from %s", externalEntityName),
				},
				&twiml.Say{
					Voice: "alice",
					Text:  "Press 1 to answer.",
				},
			},
		},
		// In the event that no key is entered, we hang up the
		// dialed call to then direct the caller to voicemail.
		&twiml.Hangup{},
	)

	return tw.GenerateTwiML()
}

func providerEnteredDigits(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	golog.Infof("Provider entered digits %s at %s.", params.Digits, params.To)

	if params.Digits == "1" {
		// accept the call if the provider entered the right digit
		// by generating an empty response.
		tw := twiml.NewResponse()
		return tw.GenerateTwiML()
	}

	// repeate message if any key other than one pressed.
	return providerCallConnected(ctx, params, eh)
}

func voicemailTWIML(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

	entity, err := determineEntityWithProvisionedEndpoint(eh, params.To, 1)
	if err != nil {
		return "", errors.Trace(err)
	}

	var orgName string
	var orgID string
	switch entity.Type {
	case directory.EntityType_ORGANIZATION:
		orgName = entity.Info.DisplayName
		orgID = entity.ID
	case directory.EntityType_INTERNAL:
		for _, m := range entity.Memberships {
			if m.Type != directory.EntityType_ORGANIZATION {
				continue
			}

			// find the organization that has this number listed as the provisioned number
			for _, c := range m.Contacts {
				if c.Provisioned && c.Value == params.To {
					orgName = m.Info.DisplayName
					orgID = m.ID
					break
				}
			}
		}
	}

	// check whether to use custom voicemail or not
	var customVoicemailURL string
	singleSelectValue, err := settings.GetSingleSelectValue(ctx, eh.settings, &settings.GetValuesRequest{
		NodeID: orgID,
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeyVoicemailOption,
				Subkey: params.To,
			},
		},
	})
	if err != nil {
		golog.Errorf("Unable to read setting for voicemail option for orgID %s phone number %s: %s", orgID, params.To, err.Error())
	}

	if singleSelectValue.GetItem().ID == excommsSettings.VoicemailOptionCustom {
		if url := singleSelectValue.GetItem().FreeTextResponse; url == "" {
			golog.Errorf("URL for custom voicemail not specified for orgID %s when custom voicemail selected", orgID)
		}
		customVoicemailMediaID := singleSelectValue.GetItem().FreeTextResponse
		customVoicemailURL, err = eh.store.ExpiringURL(customVoicemailMediaID, time.Hour)
		if err != nil {
			golog.Errorf("Unable to generate expiring url for %s:%s", customVoicemailMediaID, customVoicemailURL)
		}
		golog.Debugf("CustomVoicemail URL is %s", customVoicemailURL)
	}

	// check whether or not to transcribe voicemail
	var transcribeVoicemail bool
	booleanValue, err := settings.GetBooleanValue(ctx, eh.settings, &settings.GetValuesRequest{
		NodeID: orgID,
		Keys: []*settings.ConfigKey{
			{
				Key: excommsSettings.ConfigKeyTranscribeVoicemail,
			},
		},
	})
	if err != nil {
		golog.Errorf("Unable to get transcribe voicemail setting for orgID %s", orgID)
	}
	transcribeVoicemail = booleanValue.Value

	var action, transcribeCallback, transcriptionInfoInVoicemailMessage string
	if transcribeVoicemail {
		transcribeCallback = "/twilio/call/process_voicemail"
		action = "/twilio/call/no_op"
		transcriptionInfoInVoicemailMessage = " Speak slowly and clearly as your message will be transcribed."
	} else {
		action = "/twilio/call/process_voicemail"
		transcribeCallback = "/twilio/call/no_op"
	}

	var voicemailMessage string
	if orgName != "" {
		voicemailMessage = fmt.Sprintf("You have reached %s. Please leave a message after the tone.%s", orgName, transcriptionInfoInVoicemailMessage)
	} else {
		voicemailMessage = fmt.Sprintf("Please leave a message after the tone.%s", transcriptionInfoInVoicemailMessage)
	}

	var firstVerb interface{}
	if len(customVoicemailURL) > 0 {
		firstVerb = &twiml.Play{
			Text: html.EscapeString(customVoicemailURL),
		}
	} else {
		firstVerb = &twiml.Say{
			Voice: "alice",
			Text:  voicemailMessage,
		}
	}

	tw := &twiml.Response{
		Verbs: []interface{}{
			firstVerb,
			&twiml.Record{
				Action:             action,
				PlayBeep:           true,
				TranscribeCallback: transcribeCallback,
				TimeoutInSeconds:   60,
				// Note: manually setting the maxLength so that a voicemail longer than 2 minutes can be recorded
				// even if that long of a voicemail cannot be transcribed.
				MaxLength: 3600,
			},
		},
	}

	return tw.GenerateTwiML()
}

func processIncomingCallStatus(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	switch params.DialCallStatus {
	case rawmsg.TwilioParams_ANSWERED, rawmsg.TwilioParams_COMPLETED:
		conc.Go(func() {
			if err := sns.Publish(eh.sns, eh.externalMessageTopic, &excomms.PublishedExternalMessage{
				FromChannelID: params.From,
				ToChannelID:   params.To,
				Timestamp:     uint64(time.Now().Unix()),
				Direction:     excomms.PublishedExternalMessage_INBOUND,
				Type:          excomms.PublishedExternalMessage_INCOMING_CALL_EVENT,
				Item: &excomms.PublishedExternalMessage_Incoming{
					Incoming: &excomms.IncomingCallEventItem{
						Type:              excomms.IncomingCallEventItem_ANSWERED,
						DurationInSeconds: params.CallDuration,
					},
				},
			}); err != nil {
				golog.Errorf(err.Error())
			}
		})
		trackInboundCall(eh, params.CallSID, "answered")

	case rawmsg.TwilioParams_CALL_STATUS_UNDEFINED:
	default:
		return voicemailTWIML(ctx, params, eh)
	}

	// delete the dialed call
	cleaner.Publish(eh.sns, eh.resourceCleanerTopic, &models.DeleteResourceRequest{
		Type:       models.DeleteResourceRequest_TWILIO_CALL,
		ResourceID: params.DialCallSID,
	})

	return "", nil
}

func processOutgoingCallStatus(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

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

	var cet *excomms.PublishedExternalMessage_Outgoing
	switch params.CallStatus {
	case rawmsg.TwilioParams_RINGING:
		cet = &excomms.PublishedExternalMessage_Outgoing{
			Outgoing: &excomms.OutgoingCallEventItem{
				Type:              excomms.OutgoingCallEventItem_PLACED,
				DurationInSeconds: params.CallDuration,
				CallerEntityID:    cr.CallerEntityID,
				CalleeEntityID:    cr.CalleeEntityID,
			},
		}
	case rawmsg.TwilioParams_ANSWERED, rawmsg.TwilioParams_COMPLETED:
		cet = &excomms.PublishedExternalMessage_Outgoing{
			Outgoing: &excomms.OutgoingCallEventItem{
				Type:              excomms.OutgoingCallEventItem_ANSWERED,
				DurationInSeconds: params.CallDuration,
				CallerEntityID:    cr.CallerEntityID,
				CalleeEntityID:    cr.CalleeEntityID,
			},
		}
		if err := eh.proxyNumberManager.CallEnded(cr.Source, cr.Proxy); err != nil {
			return "", errors.Trace(err)
		}

		cleaner.Publish(eh.sns, eh.resourceCleanerTopic, &models.DeleteResourceRequest{
			Type:       models.DeleteResourceRequest_TWILIO_CALL,
			ResourceID: params.CallSID,
		})

		cleaner.Publish(eh.sns, eh.resourceCleanerTopic, &models.DeleteResourceRequest{
			Type:       models.DeleteResourceRequest_TWILIO_CALL,
			ResourceID: params.ParentCallSID,
		})

		trackOutboundCall(eh, cr.CallerEntityID, cr.OrganizationID, cr.Destination.String(), params.CallDuration)
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
			Type:          excomms.PublishedExternalMessage_OUTGOING_CALL_EVENT,
			Item:          cet,
		}); err != nil {
			golog.Errorf(err.Error())
		}
	})

	return "", nil
}

func processVoicemail(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

	rawMessageID, err := eh.dal.StoreIncomingRawMessage(&rawmsg.Incoming{
		Type: rawmsg.Incoming_TWILIO_VOICEMAIL,
		Message: &rawmsg.Incoming_Twilio{
			Twilio: params,
		},
	})
	if err != nil {
		return "", errors.Trace(err)
	}

	trackInboundCall(eh, params.CallSID, "voicemail")

	conc.Go(func() {
		if err := sns.Publish(eh.sns, eh.incomingRawMsgTopic, &sns.IncomingRawMessageNotification{
			ID: rawMessageID,
		}); err != nil {
			golog.Errorf(err.Error())
		}
	})

	return "", nil
}

func processOutgoingSMSStatus(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	switch params.MessageStatus {
	case rawmsg.TwilioParams_MSG_STATUS_DELIVERED:
		cleaner.Publish(eh.sns, eh.resourceCleanerTopic, &models.DeleteResourceRequest{
			Type:       models.DeleteResourceRequest_TWILIO_SMS,
			ResourceID: params.MessageSID,
		})
	case rawmsg.TwilioParams_MSG_STATUS_FAILED:
		// for now if message sending failed lets log error so that we
		// can investigate why a message that passed validation failed to send
		golog.Errorf("Failed to send message %s", params.MessageSID)
	}
	return "", nil
}
