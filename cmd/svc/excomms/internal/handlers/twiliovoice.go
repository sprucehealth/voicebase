package handlers

import (
	"net/http"
	"time"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/twilio"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"golang.org/x/net/context"
)

var twilioEventMapper = map[string]rawmsg.TwilioEvent{
	"process_incoming_call":             rawmsg.TwilioEvent_PROCESS_INCOMING_CALL,
	"process_outgoing_call":             rawmsg.TwilioEvent_PROCESS_OUTGOING_CALL,
	"process_incoming_call_status":      rawmsg.TwilioEvent_PROCESS_INCOMING_CALL_STATUS,
	"process_outgoing_call_status":      rawmsg.TwilioEvent_PROCESS_OUTGOING_CALL_STATUS,
	"process_voicemail":                 rawmsg.TwilioEvent_PROCESS_VOICEMAIL,
	"provider_call_connected":           rawmsg.TwilioEvent_PROVIDER_CALL_CONNECTED,
	"provider_entered_digits":           rawmsg.TwilioEvent_PROVIDER_ENTERED_DIGITS,
	"twiml_voicemail":                   rawmsg.TwilioEvent_TWIML_REQUESTED_VOICEMAIL,
	"process_sms_status":                rawmsg.TwilioEvent_PROCESS_SMS_STATUS,
	"no_op":                             rawmsg.TwilioEvent_NO_OP,
	"afterhours_greeting":               rawmsg.TwilioEvent_AFTERHOURS_GREETING,
	"afterhours_patient_entered_digits": rawmsg.TwilioEvent_AFTERHOURS_PATIENT_ENTERED_DIGITS,
	"afterhours_voicemail":              rawmsg.TwilioEvent_AFTERHOURS_VOICEMAIL,
	"afterhours_process_voicemail":      rawmsg.TwilioEvent_AFTERHOURS_PROCESS_VOICEMAIL,
}

type twilioRequestHandler struct {
	eventsHandler           twilio.EventHandler
	statRequests            *metrics.Counter
	statResponseErrors      *metrics.Counter
	statLatency             metrics.Histogram
	eventStatRequests       map[string]*metrics.Counter
	eventStatResponseErrors map[string]*metrics.Counter
	eventStatLatency        map[string]metrics.Histogram
}

func NewTwilioRequestHandler(eventsHandler twilio.EventHandler,
	metricsRegistry metrics.Registry) httputil.ContextHandler {

	statRequests := metrics.NewCounter()
	statResponseErrors := metrics.NewCounter()
	statLatency := metrics.NewUnbiasedHistogram()
	metricsRegistry.Add("requests", statRequests)
	metricsRegistry.Add("response_errors", statResponseErrors)
	metricsRegistry.Add("latency_us", statLatency)

	eventStatRequests := make(map[string]*metrics.Counter)
	eventStatResponseErrors := make(map[string]*metrics.Counter)
	eventStatLatency := make(map[string]metrics.Histogram)

	for event := range twilioEventMapper {
		sRequests := metrics.NewCounter()
		sResponseErrors := metrics.NewCounter()
		sLatency := metrics.NewUnbiasedHistogram()

		eventScope := metricsRegistry.Scope(event)
		eventScope.Add("requests", sRequests)
		eventScope.Add("response_errors", sResponseErrors)
		eventScope.Add("latency_us", sLatency)

		eventStatRequests[event] = sRequests
		eventStatResponseErrors[event] = sResponseErrors
		eventStatLatency[event] = sLatency
	}

	return &twilioRequestHandler{
		eventsHandler:           eventsHandler,
		statRequests:            statRequests,
		statResponseErrors:      statResponseErrors,
		statLatency:             statLatency,
		eventStatRequests:       eventStatRequests,
		eventStatLatency:        eventStatLatency,
		eventStatResponseErrors: eventStatResponseErrors,
	}
}

func (t *twilioRequestHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	t.statRequests.Inc(1)
	st := time.Now()
	defer func() {
		t.statLatency.Update(time.Since(st).Nanoseconds() / 1e3)
	}()

	p, err := twilio.ParamsFromRequest(r)
	if err != nil {
		golog.Errorf("Unable to parse twilio parameters from request: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		t.statResponseErrors.Inc(1)
		return
	}

	event := mux.Vars(ctx)["event"]
	twilioEvent, ok := twilioEventMapper[event]
	if !ok {
		golog.Errorf("Unable to process event %s", event)
		w.WriteHeader(http.StatusBadRequest)
		t.statResponseErrors.Inc(1)
		return
	}

	t.eventStatRequests[event].Inc(1)
	defer func() {
		t.eventStatLatency[event].Update(time.Since(st).Nanoseconds() / 1e3)
	}()

	twiml, err := t.eventsHandler.Process(ctx, twilioEvent, p)
	if err != nil {
		golog.Errorf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		t.statResponseErrors.Inc(1)
		t.eventStatResponseErrors[event].Inc(1)
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
