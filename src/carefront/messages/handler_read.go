package messages

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/libs/dispatch"
	"net/http"
)

type ReadRequest struct {
	CaseID int64 `json:"case_id,string"`
}

type readHandler struct {
	dataAPI api.DataAPI
}

func NewReadHandler(dataAPI api.DataAPI) http.Handler {
	return &readHandler{dataAPI: dataAPI}
}

func (h *readHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_POST {
		http.NotFound(w, r)
		return
	}

	var req ReadRequest
	if err := apiservice.DecodeRequestData(&req, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	cas, err := h.dataAPI.GetPatientCaseFromId(req.CaseID)
	if err == api.NoRowsError {
		apiservice.WriteDeveloperError(w, http.StatusNotFound, "Case with the given ID does not exist")
		return
	}

	personID, _, err := validateAccess(h.dataAPI, r, cas)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := h.dataAPI.MarkCaseMessagesAsRead(req.CaseID, personID); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	people, err := h.dataAPI.GetPeople([]int64{personID})
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	dispatch.Default.Publish(&ReadEvent{
		CaseID: req.CaseID,
		Person: people[personID],
	})

	apiservice.WriteJSONSuccess(w)
}
