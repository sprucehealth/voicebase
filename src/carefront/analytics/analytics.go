package analytics

import (
	"carefront/apiservice"
	"carefront/common"
	"carefront/libs/golog"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/samuel/go-metrics/metrics"
)

const (
	timeTag = "time"

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

type event struct {
	Name       string     `json:"event"`
	Properties properties `json:"properties"`
}

type Handler struct {
	logger             Logger
	statEventsReceived metrics.Counter
	statEventsDropped  metrics.Counter
}

func NewHandler(logger Logger, statsRegistry metrics.Registry) (*Handler, error) {
	h := &Handler{
		logger:             logger,
		statEventsReceived: metrics.NewCounter(),
		statEventsDropped:  metrics.NewCounter(),
	}
	statsRegistry.Add("events/received", h.statEventsReceived)
	statsRegistry.Add("events/dropped", h.statEventsDropped)
	return h, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	var events []event
	if err := json.NewDecoder(r.Body).Decode(&events); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Failed to decode body: "+err.Error())
		return
	}

	h.statEventsReceived.Inc(int64(len(events)))

	ch := common.ParseClientHeaders(r)

	now := time.Now().UTC()
	nowUnix := now.Unix()
	var eventsOut []Event
	for _, ev := range events {
		if ev.Name == "" || ev.Properties == nil {
			continue
		}
		tm := ev.Properties.popInt64("time")
		if tm < nowUnix-invalidTimeThreshold {
			continue
		}
		id, err := newID()
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to generate ID: "+err.Error())
			return
		}
		evo := &ClientEvent{
			ID:           id,
			Event:        ev.Name,
			Time:         Time(time.Unix(tm, 0)),
			Error:        ev.Properties.popString("error"),
			SessionID:    ev.Properties.popString("session_id"),
			AccountID:    ev.Properties.popInt64("account_id"),
			PatientID:    ev.Properties.popInt64("patient_id"),
			VisitID:      ev.Properties.popInt64("visit_id"),
			ScreenID:     ev.Properties.popString("screen_id"),
			QuestionID:   ev.Properties.popString("question_id"),
			TimeSpent:    ev.Properties.popInt("time_spent"),
			DeviceID:     ch.DeviceID,
			AppType:      ch.AppType,
			AppEnv:       ch.AppEnv,
			AppVersion:   ch.AppVersion,
			AppBuild:     ch.AppBuild,
			OS:           ch.OS,
			OSVersion:    ch.OSVersion,
			DeviceType:   ch.DeviceType,
			DeviceModel:  ch.DeviceModel,
			ScreenWidth:  ch.ScreenWidth,
			ScreenHeight: ch.ScreenHeight,
			DPI:          ch.DPI,
			Scale:        ch.Scale,
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
			var err error
			evo.ExtraJSON, err = json.Marshal(ev.Properties)
			if err != nil {
				golog.Errorf("Failed to marshal extra properties: %s", err.Error())
			}
		}
		eventsOut = append(eventsOut, evo)
	}
	h.statEventsDropped.Inc(int64(len(events) - len(eventsOut)))

	if len(eventsOut) == 0 {
		return
	}

	h.logger.WriteEvents(eventsOut)
}
