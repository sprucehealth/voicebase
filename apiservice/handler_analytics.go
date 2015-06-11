package apiservice

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
)

const (
	maxEventNameLength   = 256
	invalidTimeThreshold = 60 * 60 * 24 * 30 // number of seconds after which an event is dropped
)

type properties map[string]interface{}

func (p properties) popString(name string) string {
	s, ok := p[name].(string)
	if !ok {
		return ""
	}
	delete(p, name)
	return s
}

func (p properties) popFloat64Ptr(name string) *float64 {
	f, ok := p[name].(float64)
	if !ok {
		return nil
	}
	delete(p, name)
	return &f
}

func (p properties) popFloat64(name string) float64 {
	f := p.popFloat64Ptr(name)
	if f == nil {
		return 0.0
	}
	return *f
}

func (p properties) popInt64(name string) int64 {
	i, ok := p[name].(float64)
	if !ok {
		if s := p.popString(name); s != "" {
			if i, err := strconv.ParseInt(s, 10, 64); err == nil {
				return i
			}
		}
		return 0
	}
	delete(p, name)
	return int64(i)
}

func (p properties) popInt(name string) int {
	return int(p.popInt64(name))
}

func (p properties) popBoolPtr(name string) *bool {
	b, ok := p[name].(bool)
	if !ok {
		return nil
	}
	delete(p, name)
	return &b
}

type eventRequest struct {
	CurrentTime float64 `json:"current_time"`
	Events      []event `json:"events"`
}

type event struct {
	Name       string     `json:"event"`
	Properties properties `json:"properties"`
}

type analyticsHandler struct {
	publisher          dispatch.Publisher
	statEventsReceived *metrics.Counter
	statEventsDropped  *metrics.Counter
}

func newAnalyticsHandler(publisher dispatch.Publisher, statsRegistry metrics.Registry) *analyticsHandler {
	h := &analyticsHandler{
		publisher:          publisher,
		statEventsReceived: metrics.NewCounter(),
		statEventsDropped:  metrics.NewCounter(),
	}
	statsRegistry.Add("events/received", h.statEventsReceived)
	statsRegistry.Add("events/dropped", h.statEventsDropped)
	return h
}

func NewAnalyticsHandler(publisher dispatch.Publisher, statsRegistry metrics.Registry) http.Handler {
	return httputil.SupportedMethods(
		NoAuthorizationRequired(newAnalyticsHandler(publisher, statsRegistry)),
		httputil.Post)
}

func (h *analyticsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req eventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequestError(err, w, r)
		return
	}

	h.statEventsReceived.Inc(uint64(len(req.Events)))

	ch := ExtractSpruceHeaders(r)

	nowUnix := float64(time.Now().UTC().UnixNano()) / 1e9
	var eventsOut []analytics.Event
	var dropped uint64
	for _, ev := range req.Events {
		name, err := analytics.MangleEventName(ev.Name)
		if err != nil || len(name) > maxEventNameLength {
			dropped++
			// Truncate really long names to avoid flooding the analytics
			if len(ev.Name) > maxEventNameLength {
				ev.Name = ev.Name[:maxEventNameLength]
			}
			eventsOut = append(eventsOut,
				analytics.BadAnalyticsEvent("restapi", "client_event", ev.Name, "invalid name"))
			continue
		}
		if ev.Properties == nil {
			dropped++
			eventsOut = append(eventsOut,
				analytics.BadAnalyticsEvent("restapi", "client_event", name, "missing properties"))
			continue
		}
		// Calculate delta time for the event from the client provided current time.
		// Use this delta to generate the absolute event time based on the server's time.
		// This accounts for the client clock being off.
		td := req.CurrentTime - ev.Properties.popFloat64("time")
		if td > invalidTimeThreshold || td < 0 {
			dropped++
			eventsOut = append(eventsOut, analytics.BadAnalyticsEvent("restapi", "client_event", name, "invalid time"))
			continue
		}
		tf := nowUnix - td
		tm := time.Unix(int64(math.Floor(tf)), int64(1e9*(tf-math.Floor(tf))))
		evo := &analytics.ClientEvent{
			Event:      name,
			Timestamp:  analytics.Time(tm),
			Error:      ev.Properties.popString("error"),
			SessionID:  ev.Properties.popString("session_id"),
			AccountID:  ev.Properties.popInt64("account_id"),
			PatientID:  ev.Properties.popInt64("patient_id"),
			DoctorID:   ev.Properties.popInt64("doctor_id"),
			CaseID:     ev.Properties.popInt64("case_id"),
			VisitID:    ev.Properties.popInt64("visit_id"),
			ScreenID:   ev.Properties.popString("screen_id"),
			QuestionID: ev.Properties.popString("question_id"),
			TimeSpent:  ev.Properties.popFloat64Ptr("time_spent"),
			DeviceID:   ch.DeviceID,
			AppType:    ch.AppType,
			AppEnv:     ch.AppEnvironment,
			// Use app_version from properties intead of relying on the HTTP headers
			// because the events could be collected from a different version of
			// the app then the version that's sending them (incase they get
			// stored and later sent).
			AppVersion:       ev.Properties.popString("app_version"),
			AppBuild:         ch.AppBuild,
			Platform:         ch.Platform.String(),
			PlatformVersion:  ch.PlatformVersion,
			DeviceType:       ch.Device,
			DeviceModel:      ch.DeviceModel,
			ScreenWidth:      int(ch.ScreenWidth),
			ScreenHeight:     int(ch.ScreenHeight),
			ScreenResolution: ch.DeviceResolution,
		}
		// Put anything left over into ExtraJSON if it's a valid format
		for k, v := range ev.Properties {
			switch v.(type) {
			case string, float64, bool:
			default:
				delete(ev.Properties, k)
			}
		}
		if len(ev.Properties) != 0 {
			extraJSON, err := json.Marshal(ev.Properties)
			if err != nil {
				golog.Errorf("Failed to marshal extra properties: %s", err.Error())
			} else {
				evo.ExtraJSON = string(extraJSON)
			}
		}
		eventsOut = append(eventsOut, evo)
	}
	h.statEventsDropped.Inc(dropped)

	if len(eventsOut) == 0 {
		return
	}

	h.publisher.Publish(analytics.Events(eventsOut))
}
