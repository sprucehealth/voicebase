package messages

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type unreadCountResponse struct {
	UnreadCount int `json:"unread_count"`
}

type unreadCountHandler struct {
	dataAPI api.DataAPI
}

func NewUnreadCountHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(
			&unreadCountHandler{
				dataAPI: dataAPI,
			}), httputil.Get)
}

func (h *unreadCountHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	caseID, err := strconv.ParseInt(r.FormValue("case_id"), 10, 64)
	if err != nil {
		return false, apiservice.NewValidationError("bad case_id")
	}

	cas, err := h.dataAPI.GetPatientCaseFromID(caseID)
	if api.IsErrNotFound(err) {
		return false, apiservice.NewResourceNotFoundError("Case not found", r)
	} else if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.PatientCase] = cas

	personID, _, err := validateAccess(h.dataAPI, r, cas)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.PersonID] = personID

	return true, nil
}

func (h *unreadCountHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	cas := ctxt.RequestCache[apiservice.PatientCase].(*common.PatientCase)
	personID := ctxt.RequestCache[apiservice.PersonID].(int64)
	count, err := h.dataAPI.UnreadMessageCount(cas.ID.Int64(), personID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, unreadCountResponse{UnreadCount: count})
}
