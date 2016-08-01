package twilio

import (
	"fmt"
	"html"
	"net/url"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/cleaner"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/sns"
	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/twilio/twiml"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/settings"
)

// STEP: Determine what to do with incoming call (answering service triage or call forwarding list?)

func processIncomingCall(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

	source, err := phone.ParseNumber(params.From)
	if err != nil {
		golog.Errorf("Invalid from phone number: %s when calling %s", params.From, params.To)
		// if we are dealing with an invalid phone number of the caller, then inform the caller
		// that their call cannot be completed as dialed.
		tw := &twiml.Response{
			Verbs: []interface{}{
				&twiml.Say{
					Voice: "alice",
					Text:  "Sorry, your call cannot be completed as dialed.",
				},
			},
		}

		return tw.GenerateTwiML()
	}

	entity, err := directory.SingleEntityByContact(ctx, eh.directory, &directory.LookupEntitiesByContactRequest{
		ContactValue: params.To,
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_ORGANIZATION},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION, directory.EntityType_INTERNAL},
	})
	if err != nil {
		return "", errors.Trace(err)
	} else if entity.Type != directory.EntityType_ORGANIZATION {
		return "", errors.Trace(fmt.Errorf("expected entity %s of type %s but got %s", entity.ID, directory.EntityType_ORGANIZATION, entity.Type))
	}
	organizationID := entity.ID

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

	return callForwardingList(ctx, entity, params, eh)
}

func processIncomingCallStatus(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	switch params.CallStatus {
	default:
		// do nothing until end state reached

	case rawmsg.TwilioParams_COMPLETED, rawmsg.TwilioParams_NO_ANSWER:

		// end state reached
		if rowsUpdated, err := eh.dal.UpdateIncomingCall(params.CallSID, &dal.IncomingCallUpdate{
			Completed:     ptr.Bool(true),
			CompletedTime: ptr.Time(eh.clock.Now()),
		}); err != nil {
			return "", errors.Trace(err)
		} else if rowsUpdated != 1 {
			return "", errors.Errorf("Expected to update 1 row for %s but updated %d instead", params.CallSID, rowsUpdated)
		}

		incomingCall, err := eh.dal.LookupIncomingCall(params.CallSID)
		if err != nil {
			return "", errors.Trace(err)
		}

		// only consider the call answered if the call has been active for more than 2 seconds for the patient
		if incomingCall.Answered && eh.clock.Now().Sub(*incomingCall.AnsweredTime) > 2*time.Second {
			conc.Go(func() {
				durationInSeconds := params.CallDuration
				if incomingCall.AnsweredTime != nil {
					durationInSeconds = uint32(eh.clock.Now().Sub(*incomingCall.AnsweredTime).Seconds())
				}

				if err := sns.Publish(eh.sns, eh.externalMessageTopic, &excomms.PublishedExternalMessage{
					FromChannelID: params.From,
					ToChannelID:   params.To,
					Timestamp:     uint64(eh.clock.Now().Unix()),
					Direction:     excomms.PublishedExternalMessage_INBOUND,
					Type:          excomms.PublishedExternalMessage_INCOMING_CALL_EVENT,
					Item: &excomms.PublishedExternalMessage_Incoming{
						Incoming: &excomms.IncomingCallEventItem{
							Type:              excomms.IncomingCallEventItem_ANSWERED,
							DurationInSeconds: durationInSeconds,
						},
					},
				}); err != nil {
					golog.Errorf(err.Error())
				}
			})

			trackInboundCall(eh, params.CallSID, "answered")
		} else {
			conc.Go(func() {

				// check if send all calls to voicemail is turned on for organization in which case
				// don't log missed call
				sendAllCallsToVoicemailValue, err := settings.GetBooleanValue(context.Background(), eh.settings, &settings.GetValuesRequest{
					NodeID: incomingCall.OrganizationID,
					Keys: []*settings.ConfigKey{
						{
							Key:    excommsSettings.ConfigKeySendCallsToVoicemail,
							Subkey: params.To,
						},
					},
				})
				if err != nil {
					golog.Errorf("Unable to get %s value for %s: %s", excommsSettings.ConfigKeySendCallsToVoicemail, incomingCall.OrganizationID, err)
					return
				} else if sendAllCallsToVoicemailValue.Value {
					// dont track missed calls
					return
				}

				if err := sns.Publish(eh.sns, eh.externalMessageTopic, &excomms.PublishedExternalMessage{
					FromChannelID: params.From,
					ToChannelID:   params.To,
					Timestamp:     uint64(time.Now().Unix()),
					Direction:     excomms.PublishedExternalMessage_INBOUND,
					Type:          excomms.PublishedExternalMessage_INCOMING_CALL_EVENT,
					Item: &excomms.PublishedExternalMessage_Incoming{
						Incoming: &excomms.IncomingCallEventItem{
							Type: excomms.IncomingCallEventItem_UNANSWERED,
						},
					},
				}); err != nil {
					golog.Errorf(err.Error())
				}
			})
			trackInboundCall(eh, params.CallSID, "missed-call")
		}
	}

	return "", nil
}

// STEP: If call forwarding list, then send all calls to voicemail or simultaneously call numbers in list?

func callForwardingList(ctx context.Context, orgEntity *directory.Entity, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

	valuesRes, err := eh.settings.GetValues(ctx, &settings.GetValuesRequest{
		NodeID: orgEntity.ID,
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeySendCallsToVoicemail,
				Subkey: params.To,
			},
			{
				Key:    excommsSettings.ConfigKeyAfterHoursVociemailEnabled,
				Subkey: params.To,
			},
			{
				Key:    excommsSettings.ConfigKeyForwardingListTimeout,
				Subkey: params.To,
			},
			{
				Key:    excommsSettings.ConfigKeyForwardingList,
				Subkey: params.To,
			},
			{
				Key:    excommsSettings.ConfigKeyPauseBeforeCallConnect,
				Subkey: params.To,
			},
		},
	})
	if err != nil {
		return "", errors.Trace(fmt.Errorf("Unable to get settings for org %s: %s", orgEntity.ID, err.Error()))
	} else if len(valuesRes.Values) != 5 {
		return "", errors.Trace(fmt.Errorf("Expected 5 values to be returned but got %d for org %s", len(valuesRes.Values), orgEntity.ID))
	}

	sendAllCallsToVoicemail := valuesRes.Values[0].GetBoolean().Value
	afterHoursVoicemailEnabled := valuesRes.Values[1].GetBoolean().Value
	timeoutInSeconds := valuesRes.Values[2].GetInteger().Value
	forwardingList := valuesRes.Values[3].GetStringList().Values
	pauseBeforeCallConnectInSeconds := valuesRes.Values[4].GetInteger().Value

	if sendAllCallsToVoicemail && afterHoursVoicemailEnabled {
		return afterHoursCallTriage(ctx, orgEntity, params, eh)
	} else if sendAllCallsToVoicemail {
		return voicemailTWIML(ctx, params, eh)
	}

	numbers := make([]interface{}, 0, maxPhoneNumbers)
	forwardingListMap := make(map[string]struct{}, len(forwardingList))
	for _, p := range forwardingList {

		if _, ok := forwardingListMap[p]; ok {
			continue
		}
		forwardingListMap[p] = struct{}{}

		parsedPn, err := phone.Format(p, phone.E164)
		if err != nil {
			golog.Errorf("Unable to parse phone number %s: %s", p, err.Error())
			continue
		}
		if len(numbers) == maxPhoneNumbers {
			golog.Errorf("Org %s is currently configured to simultaneously call more than 10 numbers when that is the maximum that twilio supports.", orgEntity.ID)
			break
		}

		// don't include phone number in the list if it matches the incoming number
		if parsedPn == params.To {
			golog.Warningf("Found a phone number in the forwarding list that matches the destination number: %s", params.To)
			continue
		}

		numbers = append(numbers, &twiml.Number{
			URL:  "/twilio/call/provider_call_connected",
			Text: parsedPn,
		})
	}

	// if there are no numbers in the forwarding list, then direct calls to voicemail
	if len(numbers) == 0 {
		if afterHoursVoicemailEnabled {
			return afterHoursCallTriage(ctx, orgEntity, params, eh)
		}
		return voicemailTWIML(ctx, params, eh)
	}

	// put the incoming call into the queue to be deleted once the call is complete.
	cleaner.Publish(eh.sns, eh.resourceCleanerTopic, &models.DeleteResourceRequest{
		Type:       models.DeleteResourceRequest_TWILIO_CALL,
		ResourceID: params.CallSID,
	})

	verbs := make([]interface{}, 0, 2)
	if pauseBeforeCallConnectInSeconds > 0 {
		verbs = append(verbs, &twiml.Pause{
			Length: uint(pauseBeforeCallConnectInSeconds),
		})
	}
	verbs = append(verbs, &twiml.Dial{
		CallerID:         params.To,
		TimeoutInSeconds: uint(timeoutInSeconds),
		Action:           "/twilio/call/process_dialed_call_status",
		Nouns:            numbers,
	})

	tw := twiml.NewResponse(verbs...)
	return tw.GenerateTwiML()
}

func sendToVoicemail(ctx context.Context, orgEntity *directory.Entity, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	afterHoursVoicemailValue, err := settings.GetBooleanValue(ctx, eh.settings, &settings.GetValuesRequest{
		NodeID: orgEntity.ID,
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeyAfterHoursVociemailEnabled,
				Subkey: params.To,
			},
		},
	})
	if err != nil {
		return "", errors.Trace(err)
	} else if afterHoursVoicemailValue.Value {
		return afterHoursCallTriage(ctx, orgEntity, params, eh)
	}
	return voicemailTWIML(ctx, params, eh)
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

		// update the call metadata to indicate that the provider answered the call
		if rowsUpdated, err := eh.dal.UpdateIncomingCall(params.ParentCallSID, &dal.IncomingCallUpdate{
			Answered:     ptr.Bool(true),
			AnsweredTime: ptr.Time(eh.clock.Now()),
		}); err != nil {
			return "", errors.Trace(err)
		} else if rowsUpdated != 1 {
			return "", errors.Errorf("Expected 1 row to be updated for %s but %d rows updated", params.ParentCallSID, rowsUpdated)
		}

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
	entity, err := directory.SingleEntityByContact(ctx, eh.directory, &directory.LookupEntitiesByContactRequest{
		ContactValue: params.To,
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	})
	if err != nil {
		return "", errors.Trace(err)
	} else if entity.Type != directory.EntityType_ORGANIZATION {
		return "", errors.Trace(fmt.Errorf("Expected entity %s to be of type %s but got %s", entity.ID, directory.EntityType_ORGANIZATION, entity.Type))
	}

	orgName := entity.Info.DisplayName
	orgID := entity.ID

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

	if singleSelectValue != nil && singleSelectValue.GetItem().ID == excommsSettings.VoicemailOptionCustom {
		if url := singleSelectValue.GetItem().FreeTextResponse; url == "" {
			golog.Errorf("URL for custom voicemail not specified for orgID %s when custom voicemail selected", orgID)
		}
		customVoicemailMediaID := singleSelectValue.GetItem().FreeTextResponse
		customVoicemailURL, err = eh.signer.SignedURL(fmt.Sprintf("/media/%s", customVoicemailMediaID), url.Values{}, ptr.Time(eh.clock.Now().Add(time.Hour)))
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

	// update the incoming call status to indicate that the patient was sent to voicemail
	callSID := params.CallSID
	// if the parentCallSID is specified, that means we are trying to direct the
	// dialed call to the voicemail prompt for the patient.
	if params.ParentCallSID != "" {
		callSID = params.ParentCallSID
	}

	// update the call metadata to indicate that the provider answered the call
	if rowsUpdated, err := eh.dal.UpdateIncomingCall(callSID, &dal.IncomingCallUpdate{
		SentToVoicemail: ptr.Bool(true),
	}); err != nil {
		return "", errors.Trace(err)
	} else if rowsUpdated != 1 {
		return "", errors.Trace(fmt.Errorf("Expected 1 row to be updated for %s but %d rows updated", params.ParentCallSID, rowsUpdated))
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

func processDialedCallStatus(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

	// nothing to do if the incoming call is not in progress. This is because the status call back
	// for the incoming call will manage the state of the call
	if params.CallStatus != rawmsg.TwilioParams_IN_PROGRESS {
		return "", nil
	}

	switch params.DialCallStatus {
	case rawmsg.TwilioParams_ANSWERED, rawmsg.TwilioParams_COMPLETED:
		// do nothing because the processing of the call status of the patient call
		// will handle adding the right events
	case rawmsg.TwilioParams_CALL_STATUS_UNDEFINED:
	default:
		incomingCall, err := eh.dal.LookupIncomingCall(params.CallSID)
		if err != nil {
			return "", errors.Trace(err)
		}

		entity, err := directory.SingleEntity(ctx, eh.directory, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: incomingCall.OrganizationID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
			},
			Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		})
		if err != nil {
			return "", errors.Trace(err)
		}

		return sendToVoicemail(ctx, entity, params, eh)
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

	// update the call metadata to indicate that the patient left a voicemail
	if rowsUpdated, err := eh.dal.UpdateIncomingCall(params.CallSID, &dal.IncomingCallUpdate{
		LeftVoicemail:     ptr.Bool(true),
		LeftVoicemailTime: ptr.Time(eh.clock.Now()),
	}); err != nil {
		return "", errors.Trace(err)
	} else if rowsUpdated != 1 {
		return "", errors.Trace(fmt.Errorf("Expected 1 row to be updated for %s but %d rows updated", params.ParentCallSID, rowsUpdated))
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
