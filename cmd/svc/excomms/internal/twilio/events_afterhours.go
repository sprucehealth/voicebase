package twilio

import (
	"fmt"
	"html"
	"net/url"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/sns"
	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/twilio/twiml"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/settings"
)

// STEP: If after hours call triage, then prompt the patient with the binary phone tree

func afterHoursCallTriage(ctx context.Context, orgEntity *directory.Entity, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

	// mark the fact that the incoming call was an after hours call
	if rowsUpdated, err := eh.dal.UpdateIncomingCall(params.CallSID, &dal.IncomingCallUpdate{
		Afterhours: ptr.Bool(true),
	}); err != nil {
		return "", errors.Trace(err)
	} else if rowsUpdated > 1 {
		return "", errors.Trace(fmt.Errorf("Expected 1 row to be updated instead got %d rows updated", rowsUpdated))
	}

	// check whether to use custom voicemail or not
	var afterHoursGreetingURL string
	singleSelectValue, err := settings.GetSingleSelectValue(ctx, eh.settings, &settings.GetValuesRequest{
		NodeID: orgEntity.ID,
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeyVoicemailOption,
				Subkey: params.To,
			},
		},
	})
	if err != nil {
		golog.Errorf("Unable to read setting for afterhours greeting option for orgID %s phone number %s: %s", orgEntity.ID, params.To, err.Error())
	}

	if singleSelectValue.GetItem().ID == excommsSettings.VoicemailOptionCustom {
		if url := singleSelectValue.GetItem().FreeTextResponse; url == "" {
			golog.Errorf("URL for custom afterhours greeting not specified for orgID %s when custom voicemail selected", orgEntity.ID)
		}
		afterHoursGreetingMediaID := singleSelectValue.GetItem().FreeTextResponse
		afterHoursGreetingURL, err = eh.signer.SignedURL(fmt.Sprintf("/media/%s", afterHoursGreetingMediaID), url.Values{}, ptr.Time(eh.clock.Now().Add(time.Hour)))
		if err != nil {
			golog.Errorf("Unable to generate expiring url for %s:%s", afterHoursGreetingMediaID, afterHoursGreetingURL)
		}
	}

	var verbs []interface{}
	if len(afterHoursGreetingURL) > 0 {
		verbs = []interface{}{
			&twiml.Play{
				Text: html.EscapeString(afterHoursGreetingURL),
			},
		}
	} else {
		verbs = []interface{}{
			&twiml.Say{
				Voice: "alice",
				Text:  fmt.Sprintf("You have reached %s. If this is an emergency, please hang up and dial 9 1 1.", orgEntity.Info.DisplayName),
			},
			&twiml.Say{
				Voice: "alice",
				Text:  "Otherwise, press 1 to leave an urgent message, 2 to leave a non-urgent message.",
			},
		}
	}

	tw := twiml.NewResponse(
		&twiml.Gather{
			Action:           "/twilio/call/afterhours_patient_entered_digits",
			Method:           "POST",
			TimeoutInSeconds: 5,
			NumDigits:        1,
			Verbs:            verbs,
		},
		// In the event that no key is entered, we repeate the
		// message.
		&twiml.Redirect{
			Text: "/twilio/call/afterhours_greeting",
		},
	)

	return tw.GenerateTwiML()
}

// STEP: Play after hours call greeting, which is either custom or the default.

func afterHoursGreeting(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	orgEntity, err := directory.SingleEntityByContact(ctx, eh.directory, &directory.LookupEntitiesByContactRequest{
		ContactValue: params.To,
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	})
	if err != nil {
		return "", errors.Trace(err)
	}
	return afterHoursCallTriage(ctx, orgEntity, params, eh)
}

// STEP: If patient presses 1 prompt them to leave an urgent voicemail; if patient presses 2 leave a non-urgent voicemail.

func afterHoursPatientEnteredDigits(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	var urgent bool
	switch params.Digits {
	case "1":
		urgent = true
	case "2":
		urgent = false
	default:
		// repeat message if patient entered a digit that is not recognized
		return afterHoursGreeting(ctx, params, eh)
	}

	// record the fact that we are dealing with an after hours call
	if rowsUpdated, err := eh.dal.UpdateIncomingCall(params.CallSID, &dal.IncomingCallUpdate{
		Urgent: ptr.Bool(urgent),
	}); err != nil {
		return "", errors.Trace(fmt.Errorf("Unable to update incoming call %s : %s", params.CallSID, err))
	} else if rowsUpdated > 1 {
		return "", errors.Trace(fmt.Errorf("Expected 1 row to be updated for call %s but got %d", params.CallSID, rowsUpdated))
	}

	tw := twiml.NewResponse(
		&twiml.Redirect{
			Text: "/twilio/call/afterhours_voicemail",
		},
	)

	return tw.GenerateTwiML()
}

// STEP: Prompt the user to leave a voicemail and transcribe the voicemail if the user has that setting on.

func afterHoursVoicemailTWIML(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
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
	}

	// check whether or not to transcribe voicemail
	var transcribeVoicemail bool
	booleanValue, err := settings.GetBooleanValue(ctx, eh.settings, &settings.GetValuesRequest{
		NodeID: entity.ID,
		Keys: []*settings.ConfigKey{
			{
				Key: excommsSettings.ConfigKeyTranscribeVoicemail,
			},
		},
	})
	if err != nil {
		golog.Errorf("Unable to get transcribe voicemail setting for orgID %s", entity.ID)
	}
	transcribeVoicemail = booleanValue.Value

	var action, transcribeCallback, transcriptionInfoInVoicemailMessage string
	if transcribeVoicemail {
		transcribeCallback = "/twilio/call/afterhours_process_voicemail"
		action = "/twilio/call/no_op"
		transcriptionInfoInVoicemailMessage = " Speak slowly and clearly as your message will be transcribed."
	} else {
		action = "/twilio/call/afterhours_process_voicemail"
		transcribeCallback = "/twilio/call/no_op"
	}

	tw := &twiml.Response{
		Verbs: []interface{}{
			&twiml.Say{
				Voice: "alice",
				Text:  fmt.Sprintf("Please leave a message after the tone.%s", transcriptionInfoInVoicemailMessage),
			},
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

// STEP: Process the voicemail that was left.

func afterHoursProcessVoicemail(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

	// mark the call as completed
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
		return "", errors.Errorf("Unable to lookup call %s: %s", params.CallSID, err)
	}

	urgentAfterHoursVoicemail := incomingCall.AfterHours && incomingCall.Urgent

	rawMessageID, err := eh.dal.StoreIncomingRawMessage(&rawmsg.Incoming{
		Type: rawmsg.Incoming_TWILIO_VOICEMAIL,
		Message: &rawmsg.Incoming_Twilio{
			Twilio: params,
		},
	})
	if err != nil {
		return "", errors.Trace(err)
	}

	if urgentAfterHoursVoicemail {
		trackInboundCall(eh, params.CallSID, "urgent-voicemail")
	} else {
		trackInboundCall(eh, params.CallSID, "voicemail")
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
