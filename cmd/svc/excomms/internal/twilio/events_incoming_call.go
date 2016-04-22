package twilio

import (
	"fmt"
	"html"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/cleaner"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/sns"
	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/twilio/twiml"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/settings"
	"golang.org/x/net/context"
)

// STEP: Determine what to do with incoming call (answering service triage or call forwarding list?)

func processIncomingCall(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

	entity, err := determineEntityWithProvisionedEndpoint(eh, params.To, 2)
	if err != nil {
		return "", errors.Trace(err)
	} else if entity.Type != directory.EntityType_ORGANIZATION {
		return "", errors.Trace(fmt.Errorf("expected entity %s of type %s but got %s", entity.ID, directory.EntityType_ORGANIZATION, entity.Type))
	}
	organizationID := entity.ID

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

	return callForwardingList(ctx, organizationID, entity, params, eh)

}

// STEP: If call forwarding list, then send all calls to voicemail or simultaneously call numbers in list?

func callForwardingList(ctx context.Context, organizationID string, entity *directory.Entity, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
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

	forwardingList, err := getForwardingListForProvisionedPhoneNumber(ctx, params.To, organizationID, eh)
	if err != nil {
		return "", errors.Trace(err)
	}

	var providersInOrg []*directory.Entity
	for _, member := range entity.Members {
		if member.Type == directory.EntityType_INTERNAL {
			providersInOrg = append(providersInOrg, member)
		}
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

// STEP: For each number from the forwarding list that is called, call screen
// the provider to ensure call is picked up by active provider versus automated system.

func providerCallConnected(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

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

// STEP: If provider enters the appropriate digit, connect the call, else repeat the message.

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

// STEP: If call goes to voicemail, play a default or custom greeting based on setting, and
// take message (or not) based on setting. If message is recorded, transcribe voicemail (or not) based
// based on configuration.

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

// STEP: Process the status of the incoming call.

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

// STEP: If voicemail left, then process voicemail and route voicemail to appropriate thread.

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