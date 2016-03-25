package handlers

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/twilio"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"golang.org/x/net/context"
)

var twilioEventMapper = map[string]rawmsg.TwilioEvent{
	"process_incoming_call":        rawmsg.TwilioEvent_PROCESS_INCOMING_CALL,
	"process_outgoing_call":        rawmsg.TwilioEvent_PROCESS_OUTGOING_CALL,
	"process_incoming_call_status": rawmsg.TwilioEvent_PROCESS_INCOMING_CALL_STATUS,
	"process_outgoing_call_status": rawmsg.TwilioEvent_PROCESS_OUTGOING_CALL_STATUS,
	"process_voicemail":            rawmsg.TwilioEvent_PROCESS_VOICEMAIL,
	"provider_call_connected":      rawmsg.TwilioEvent_PROVIDER_CALL_CONNECTED,
	"provider_entered_digits":      rawmsg.TwilioEvent_PROVIDER_ENTERED_DIGITS,
	"twiml_voicemail":              rawmsg.TwilioEvent_TWIML_REQUESTED_VOICEMAIL,
	"process_sms_status":           rawmsg.TwilioEvent_PROCESS_SMS_STATUS,
	"no_op":                        rawmsg.TwilioEvent_NO_OP,
}

type twilioRequestHandler struct {
	eventsHandler twilio.EventHandler
}

func NewTwilioRequestHandler(eventsHandler twilio.EventHandler) httputil.ContextHandler {
	return &twilioRequestHandler{
		eventsHandler: eventsHandler,
	}
}

func (t *twilioRequestHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	p, err := twilio.ParamsFromRequest(r)
	if err != nil {
		golog.Errorf("Unable to parse twilio parameters from request: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	event := mux.Vars(ctx)["event"]
	twilioEvent, ok := twilioEventMapper[event]
	if !ok {
		golog.Errorf("Unable to process event %s", event)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	twiml, err := t.eventsHandler.Process(ctx, twilioEvent, p)
	if err != nil {
		golog.Errorf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if twiml != "" {
		w.Header().Set("Content-Type", "text/xml")
		if _, err := w.Write([]byte(twiml)); err != nil {
			golog.Errorf(err.Error())
		}
	}
}
