package twilio

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/proxynumber"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"

	"context"

	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/urlutil"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/settings"
)

var (
	twilioEventsHandlers = map[rawmsg.TwilioEvent]twilioEventHandleFunc{

		// incoming calls
		rawmsg.TwilioEvent_PROCESS_INCOMING_CALL:        processIncomingCall,
		rawmsg.TwilioEvent_PROVIDER_CALL_CONNECTED:      providerCallConnected,
		rawmsg.TwilioEvent_PROVIDER_ENTERED_DIGITS:      providerEnteredDigits,
		rawmsg.TwilioEvent_TWIML_REQUESTED_VOICEMAIL:    voicemailTWIML,
		rawmsg.TwilioEvent_PROCESS_DIALED_CALL_STATUS:   processDialedCallStatus,
		rawmsg.TwilioEvent_PROCESS_INCOMING_CALL_STATUS: processIncomingCallStatus,
		rawmsg.TwilioEvent_PROCESS_VOICEMAIL:            processVoicemail,
		rawmsg.TwilioEvent_NO_OP:                        processNoOp,

		// outgoing calls
		rawmsg.TwilioEvent_PROCESS_OUTGOING_CALL:        processOutgoingCall,
		rawmsg.TwilioEvent_PROCESS_OUTGOING_CALL_STATUS: processOutgoingCallStatus,

		// sms
		rawmsg.TwilioEvent_PROCESS_SMS_STATUS: processOutgoingSMSStatus,

		// after hours
		rawmsg.TwilioEvent_AFTERHOURS_GREETING:               afterHoursGreeting,
		rawmsg.TwilioEvent_AFTERHOURS_PATIENT_ENTERED_DIGITS: afterHoursPatientEnteredDigits,
		rawmsg.TwilioEvent_AFTERHOURS_VOICEMAIL:              afterHoursVoicemailTWIML,
		rawmsg.TwilioEvent_AFTERHOURS_PROCESS_VOICEMAIL:      afterHoursProcessVoicemail,
	}
	maxPhoneNumbers = 10
)

type eventsHandler struct {
	directory            directory.DirectoryClient
	settings             settings.SettingsClient
	dal                  dal.DAL
	signer               *urlutil.Signer
	sns                  snsiface.SNSAPI
	clock                clock.Clock
	proxyNumberManager   proxynumber.Manager
	apiURL               string
	externalMessageTopic string
	incomingRawMsgTopic  string
	resourceCleanerTopic string
}

func NewEventHandler(
	directory directory.DirectoryClient,
	settingsClient settings.SettingsClient,
	dal dal.DAL,
	sns snsiface.SNSAPI,
	clock clock.Clock,
	proxyNumberManager proxynumber.Manager,
	apiURL, externalMessageTopic, incomingRawMsgTopic, resourceCleanerTopic string,
	signer *urlutil.Signer) EventHandler {
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
		signer:               signer,
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
