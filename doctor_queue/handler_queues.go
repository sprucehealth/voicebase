package doctor_queue

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/libs/httputil"
)

type doctorQueueDisplayItem struct {
	PatientFirstName string                `json:"patient_first_name"`
	PatientLastName  string                `json:"patient_last_name"`
	EventDescription string                `json:"event_description"`
	EventTime        int64                 `json:"event_time"`
	ActionURL        *app_url.SpruceAction `json:"action_url"`
	AuthURL          *app_url.SpruceAction `json:"auth_url"`
}

type inboxHandler struct {
	dataAPI api.DataAPI
}

func NewInboxHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&inboxHandler{
				dataAPI: dataAPI,
			}), []string{api.DOCTOR_ROLE, api.MA_ROLE}), []string{"GET"})
}

func (i *inboxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doctorID, err := i.dataAPI.GetDoctorIDFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	queueItems, err := i.dataAPI.GetPendingItemsInDoctorQueue(doctorID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	transformQueueItems(i.dataAPI, queueItems, false, w, r)
}

type unassignedHandler struct {
	dataAPI api.DataAPI
}

func NewUnassignedHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&unassignedHandler{
				dataAPI: dataAPI,
			}), []string{api.DOCTOR_ROLE, api.MA_ROLE}), []string{"GET"})
}

func (u *unassignedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)

	doctorID, err := u.dataAPI.GetDoctorIDFromAccountID(ctxt.AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	var queueItems []*api.DoctorQueueItem
	var addAuthURL bool
	if ctxt.Role == api.MA_ROLE {
		queueItems, err = u.dataAPI.GetPendingItemsForClinic()
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	} else {
		addAuthURL = true
		queueItems, err = u.dataAPI.GetElligibleItemsInUnclaimedQueue(doctorID)
		if err != nil && !api.IsErrNotFound(err) {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	transformQueueItems(u.dataAPI, queueItems, addAuthURL, w, r)
}

type historyHandler struct {
	dataAPI api.DataAPI
}

func NewHistoryHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&historyHandler{
				dataAPI: dataAPI,
			}), []string{api.DOCTOR_ROLE, api.MA_ROLE}), []string{"GET"})
}

func (h *historyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)

	doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(ctxt.AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	var queueItems []*api.DoctorQueueItem
	if ctxt.Role == api.MA_ROLE {
		queueItems, err = h.dataAPI.GetCompletedItemsForClinic()
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	} else {
		queueItems, err = h.dataAPI.GetCompletedItemsInDoctorQueue(doctorID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	transformQueueItems(h.dataAPI, queueItems, false, w, r)
}

func transformQueueItems(
	dataAPI api.DataAPI,
	queueItems []*api.DoctorQueueItem,
	addAuthURL bool,
	w http.ResponseWriter,
	r *http.Request) {
	// collect the set of patient ids
	patientIDs := make([]int64, 0, len(queueItems))
	patientIDMap := make(map[int64]bool)
	for _, item := range queueItems {
		if !patientIDMap[item.PatientID] {
			patientIDMap[item.PatientID] = true
			patientIDs = append(patientIDs, item.PatientID)
		}
	}

	patientMap, err := dataAPI.Patients(patientIDs)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// create the display items
	items := make([]*doctorQueueDisplayItem, len(queueItems))
	for i, queueItem := range queueItems {
		patient := patientMap[queueItem.PatientID]
		items[i] = &doctorQueueDisplayItem{
			PatientFirstName: patient.FirstName,
			PatientLastName:  patient.LastName,
			EventDescription: queueItem.ShortDescription,
			EventTime:        queueItem.EnqueueDate.Unix(),
			ActionURL:        queueItem.ActionURL,
		}
		if addAuthURL {
			items[i].AuthURL = app_url.ClaimPatientCaseAction(queueItem.PatientCaseID)
		}
	}

	apiservice.WriteJSON(w, struct {
		Items []*doctorQueueDisplayItem `json:"items"`
	}{
		Items: items,
	})
}
