package homelog

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/libs/golog"
	"net/http"
)

type Handler struct {
	dataAPI api.DataAPI
}

type response struct {
	Notifications []view `json:"notifications"`
}

func NewHandler(dataAPI api.DataAPI) *Handler {
	return &Handler{
		dataAPI: dataAPI,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
			golog.Errorf("Failed to create view for notification %d of type %s", n.Id, n.Type)
			continue
		}
		noteViews = append(noteViews, view)
	}

	res := &response{
		Notifications: noteViews,
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, res)
}
