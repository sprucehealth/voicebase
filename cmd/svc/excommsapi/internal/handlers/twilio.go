package handlers

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/excommsapi/internal/twilio"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/svc/excomms"
	"golang.org/x/net/context"
)

var twilioEventMapper = map[string]excomms.TwilioEvent{
	"process_incoming_call":        excomms.TwilioEvent_PROCESS_INCOMING_CALL,
	"process_outgoing_call":        excomms.TwilioEvent_PROCESS_OUTGOING_CALL,
	"process_incoming_call_status": excomms.TwilioEvent_PROCESS_INCOMING_CALL_STATUS,
	"process_outgoing_call_status": excomms.TwilioEvent_PROCESS_OUTGOING_CALL_STATUS,
	"process_voicemail":            excomms.TwilioEvent_PROCESS_VOICEMAIL,
	"provider_call_connected":      excomms.TwilioEvent_PROVIDER_CALL_CONNECTED,
	"provider_entered_digits":      excomms.TwilioEvent_PROVIDER_ENTERED_DIGITS,
	"twiml_voicemail":              excomms.TwilioEvent_TWIML_REQUESTED_VOICEMAIL,
	"incoming_sms":                 excomms.TwilioEvent_INCOMING_SMS,
}

type twilioRequestHandler struct {
	h       httputil.ContextHandler
	excomms excomms.ExCommsClient
}

func NewTwilioRequestHandler(excomms excomms.ExCommsClient) httputil.ContextHandler {
	return &twilioRequestHandler{
		excomms: excomms,
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

	res, err := t.excomms.ProcessTwilioEvent(context.Background(), &excomms.ProcessTwilioEventRequest{
		Params: p,
		Event:  twilioEvent,
	})
	if err != nil {
		golog.Errorf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if res.Twiml != "" {
		w.Header().Set("Content-Type", "text/xml")

		if _, err := w.Write([]byte(res.Twiml)); err != nil {
			golog.Errorf(err.Error())
		}
	}
}
