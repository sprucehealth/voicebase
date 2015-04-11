package admin

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/events"
	"github.com/sprucehealth/backend/events/model"
	"github.com/sprucehealth/backend/events/query"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type serverEventsHandler struct {
	eventsClient events.Client
}

type serverEventsGETRequest struct {
	Event           *string    `schema:"name"`
	Begin           *time.Time `schema:"begin"`
	End             *time.Time `schema:"end"`
	SessionID       *string    `schema:"session_id"`
	AccountID       *int64     `schema:"account_id"`
	PatientID       *int64     `schema:"patient_id"`
	DoctorID        *int64     `schema:"doctor_id"`
	VisitID         *int64     `schema:"visit_id"`
	CaseID          *int64     `schema:"case_id"`
	TreatmentPlanID *int64     `schema:"treatment_plan_id"`
	Role            *string    `schema:"role"`
}

type serverEventsGETResponse struct {
	Events []*analytics.ServerEvent `json:"events"`
}

func NewServerEventsHandler(eventsClient events.Client) http.Handler {
	return httputil.SupportedMethods(&serverEventsHandler{eventsClient: eventsClient}, []string{"GET"})
}

func (h *serverEventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		req, err := h.parseGETRequest(r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.serveGET(w, r, req)
	}
}

func (h *serverEventsHandler) parseGETRequest(r *http.Request) (*serverEventsGETRequest, error) {
	rd := &serverEventsGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	return rd, nil
}

func (h *serverEventsHandler) serveGET(w http.ResponseWriter, r *http.Request, req *serverEventsGETRequest) {
	events, err := h.eventsClient.ServerEvents(&query.ServerEventQuery{
		TimestampQuery:  query.TimestampQuery{Begin: req.Begin, End: req.End},
		Event:           req.Event,
		SessionID:       req.SessionID,
		AccountID:       req.AccountID,
		PatientID:       req.PatientID,
		DoctorID:        req.DoctorID,
		VisitID:         req.VisitID,
		CaseID:          req.CaseID,
		TreatmentPlanID: req.TreatmentPlanID,
		Role:            req.Role,
	})
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	resp := &serverEventsGETResponse{
		Events: make([]*analytics.ServerEvent, len(events)),
	}
	for i, v := range events {
		resp.Events[i] = model.FromServerEventModel(v)
	}

	httputil.JSONResponse(w, http.StatusOK, resp)
}
