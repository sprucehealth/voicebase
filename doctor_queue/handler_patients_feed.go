package doctor_queue

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type patientsFeedHandler struct {
	dataAPI api.DataAPI
}

type PatientsFeedItem struct {
	ID               string               `json:"id"` // Unique to the content of the item
	PatientFirstName string               `json:"patient_first_name"`
	PatientLastName  string               `json:"patient_last_name"`
	EventDescription string               `json:"event_description"`
	EventTime        int64                `json:"event_time"`      // unix timestamp
	LastVisitTime    int64                `json:"last_visit_time"` // unix timestamp
	LastVisitDoctor  string               `json:"last_visit_doctor"`
	ActionURL        app_url.SpruceAction `json:"action_url"`
	Tags             []string             `json:"tags"`
}

type PatientsFeedResponse struct {
	Items []*PatientsFeedItem `json:"items"`
}

func NewPatientsFeedHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(
				&patientsFeedHandler{
					dataAPI: dataAPI,
				}), []string{api.DOCTOR_ROLE, api.MA_ROLE}),
		[]string{"GET"})
}

func (h *patientsFeedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := apiservice.GetContext(r)

	// Query items. MA gets all items. Doctors get only the cases they're involved with.

	var items []*common.PatientCaseFeedItem
	var err error
	if ctx.Role == api.MA_ROLE {
		items, err = h.dataAPI.PatientCaseFeed()
	} else {
		var doctorID int64
		doctorID, err = h.dataAPI.GetDoctorIDFromAccountID(ctx.AccountID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		items, err = h.dataAPI.PatientCaseFeedForDoctor(doctorID)
	}
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// Transform from data models to response

	res := &PatientsFeedResponse{
		Items: make([]*PatientsFeedItem, len(items)),
	}
	for i, it := range items {
		var tags []string
		tags = []string{it.PathwayName}
		eventTime := it.LastEventTime.Unix()
		res.Items[i] = &PatientsFeedItem{
			// Generate an ID unique to the contents of the item
			ID:               fmt.Sprintf("%d:%d:%d:%d", it.DoctorID, it.PatientID, it.CaseID, eventTime),
			PatientFirstName: it.PatientFirstName,
			PatientLastName:  it.PatientLastName,
			EventDescription: it.LastEvent,
			EventTime:        eventTime,
			LastVisitTime:    it.LastVisitTime.Unix(),
			LastVisitDoctor:  it.LastVisitDoctor,
			ActionURL:        it.ActionURL,
			Tags:             tags,
		}
	}

	httputil.JSONResponse(w, http.StatusOK, res)
}
