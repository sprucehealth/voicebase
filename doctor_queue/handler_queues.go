package doctor_queue

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type DoctorQueueDisplayItem struct {
	ID               string                `json:"id"`
	PatientFirstName string                `json:"patient_first_name"`
	PatientLastName  string                `json:"patient_last_name"`
	EventDescription string                `json:"event_description"`
	EventTime        int64                 `json:"event_time"`
	ActionURL        *app_url.SpruceAction `json:"action_url"`
	AuthURL          *app_url.SpruceAction `json:"auth_url"`
	Tags             []string              `json:"tags"`
}

type inboxHandler struct {
	dataAPI api.DataAPI
}

func NewInboxHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&inboxHandler{
				dataAPI: dataAPI,
			}), api.RoleDoctor, api.RoleCC),
		httputil.Get)
}

func (i *inboxHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := apiservice.MustCtxAccount(ctx)

	var queueItems []*api.DoctorQueueItem

	if account.Role == api.RoleCC {
		// Care coordinates see a unified inbox across all CC accounts
		var err error
		queueItems, err = i.dataAPI.GetPendingItemsInCCQueues()
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	} else {
		doctorID, err := i.dataAPI.GetDoctorIDFromAccountID(account.ID)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		queueItems, err = i.dataAPI.GetPendingItemsInDoctorQueue(doctorID)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	}

	transformQueueItems(ctx, i.dataAPI, queueItems, false, w, r)
}

type unassignedHandler struct {
	dataAPI api.DataAPI
}

func NewUnassignedHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&unassignedHandler{
				dataAPI: dataAPI,
			}), api.RoleDoctor, api.RoleCC),
		httputil.Get)
}

func (u *unassignedHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := apiservice.MustCtxAccount(ctx)

	doctorID, err := u.dataAPI.GetDoctorIDFromAccountID(account.ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	var queueItems []*api.DoctorQueueItem
	var addAuthURL bool
	if account.Role == api.RoleCC {
		queueItems, err = u.dataAPI.GetPendingItemsForClinic()
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	} else {
		addAuthURL = true
		queueItems, err = u.dataAPI.GetElligibleItemsInUnclaimedQueue(doctorID)
		if err != nil && !api.IsErrNotFound(err) {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	}

	transformQueueItems(ctx, u.dataAPI, queueItems, addAuthURL, w, r)
}

type historyHandler struct {
	dataAPI api.DataAPI
}

func NewHistoryHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&historyHandler{
				dataAPI: dataAPI,
			}), api.RoleDoctor, api.RoleCC),
		httputil.Get)
}

func (h *historyHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := apiservice.MustCtxAccount(ctx)

	doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(account.ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	var queueItems []*api.DoctorQueueItem
	if account.Role == api.RoleCC {
		queueItems, err = h.dataAPI.GetCompletedItemsForClinic()
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	} else {
		queueItems, err = h.dataAPI.GetCompletedItemsInDoctorQueue(doctorID)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	}

	transformQueueItems(ctx, h.dataAPI, queueItems, false, w, r)
}

func transformQueueItems(
	ctx context.Context,
	dataAPI api.DataAPI,
	queueItems []*api.DoctorQueueItem,
	addAuthURL bool,
	w http.ResponseWriter,
	r *http.Request,
) {
	// collect the set of patient ids
	patientIDs := make([]common.PatientID, 0, len(queueItems))
	patientIDMap := make(map[common.PatientID]bool)
	for _, item := range queueItems {
		if !patientIDMap[item.PatientID] {
			patientIDMap[item.PatientID] = true
			patientIDs = append(patientIDs, item.PatientID)
		}
	}

	patientMap, err := dataAPI.Patients(patientIDs)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// create the display items
	items := make([]*DoctorQueueDisplayItem, len(queueItems))
	for i, queueItem := range queueItems {
		patient := patientMap[queueItem.PatientID]
		items[i] = &DoctorQueueDisplayItem{
			ID:               constructIDFromItem(queueItem),
			PatientFirstName: patient.FirstName,
			PatientLastName:  patient.LastName,
			EventDescription: queueItem.ShortDescription,
			EventTime:        queueItem.EnqueueDate.Unix(),
			ActionURL:        queueItem.ActionURL,
			Tags:             queueItem.Tags,
		}
		if addAuthURL {
			items[i].AuthURL = app_url.ClaimPatientCaseAction(queueItem.PatientCaseID)
		}
	}

	httputil.JSONResponse(w, http.StatusOK, struct {
		Items []*DoctorQueueDisplayItem `json:"items"`
	}{
		Items: items,
	})
}
