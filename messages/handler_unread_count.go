package messages

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
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

func NewUnreadCountHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.RequestCacheHandler(
			apiservice.AuthorizationRequired(
				&unreadCountHandler{
					dataAPI: dataAPI,
				})),
		httputil.Get)
}

func (h *unreadCountHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)

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
	requestCache[apiservice.CKPatientCase] = cas

	personID, _, err := validateAccess(h.dataAPI, r, apiservice.MustCtxAccount(ctx), cas)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKPersonID] = personID

	return true, nil
}

func (h *unreadCountHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	cas := requestCache[apiservice.CKPatientCase].(*common.PatientCase)
	personID := requestCache[apiservice.CKPersonID].(int64)
	count, err := h.dataAPI.UnreadMessageCount(cas.ID.Int64(), personID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, unreadCountResponse{UnreadCount: count})
}
