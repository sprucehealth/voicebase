package homelog

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/libs/golog"
	"fmt"
	"net/http"
	"strconv"
)

type ListHandler struct {
	dataAPI api.DataAPI
}

type DismissHandler struct {
	dataAPI api.DataAPI
}

type response struct {
	Notifications []view `json:"notifications"`
}

func NewListHandler(dataAPI api.DataAPI) *ListHandler {
	return &ListHandler{
		dataAPI: dataAPI,
	}
}

func NewDismissHandler(dataAPI api.DataAPI) *DismissHandler {
	return &DismissHandler{
		dataAPI: dataAPI,
	}
}

func (h *ListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	patientId, err := h.dataAPI.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get patient: "+err.Error())
		return
	}

	notes, err := h.dataAPI.GetHomeNotificationsForPatient(patientId, notifyTypes)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get home notifications: "+err.Error())
		return
	}

	noteViews := make([]view, 0, len(notes))
	for _, n := range notes {
		view, err := n.Data.(notification).makeView(h.dataAPI, patientId)
		if err != nil {
			golog.Errorf("Failed to create view for notification %d of type %s", n.Id, n.Data.TypeName())
			continue
		}
		noteViews = append(noteViews, view)
	}

	res := &response{
		Notifications: noteViews,
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, res)
}

func (h *DismissHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_POST {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var noteIDs []int64
	for _, s := range r.PostForm["notification_id"] {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			apiservice.WriteUserError(w, http.StatusBadRequest, fmt.Sprintf("Notification ID '%s' not an integer", s))
			return
		}
		noteIDs = append(noteIDs, id)
	}
	if len(noteIDs) == 0 {
		apiservice.WriteUserError(w, http.StatusBadRequest, "notification_id required")
		return
	}

	if err := h.dataAPI.DeleteHomeNotifications(noteIDs); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to delete notifications: "+err.Error())
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, map[string]bool{"success": true})
}
