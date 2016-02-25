package handlers

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/twilio"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type twilioSMSStatusHandler struct {
	eventsHandler twilio.EventHandler
}

func NewTwilioSMSStatusHandler(eventsHandler twilio.EventHandler) httputil.ContextHandler {
	return &twilioSMSStatusHandler{
		eventsHandler: eventsHandler,
	}
}

func (t *twilioSMSStatusHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	p, err := twilio.ParamsFromRequest(r)
	if err != nil {
		golog.Errorf("Unable to parse twilio parameters from request: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	twilioEvent, ok := twilioEventMapper["process_sms_status"]
	if !ok {
		golog.Errorf("Unable to process event %s", "process_sms_status")
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
