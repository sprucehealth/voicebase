package admin

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type providerSearchAPIHandler struct {
	dataAPI api.DataAPI
	authAPI api.AuthAPI
}

type createProviderRequest struct {
	Role      string        `json:"role"`
	Email     string        `json:"email"`
	FirstName string        `json:"first_name"`
	LastName  string        `json:"last_name"`
	DOB       encoding.Date `json:"dob"`
	Gender    string        `json:"gender"`
	CellPhone common.Phone  `json:"cell_phone"`
}

type createProviderResponse struct {
	AccountID  int64 `json:"account_id,string"`
	ProviderID int64 `json:"provider_id,string"`
}

func (r *createProviderRequest) validate() (string, bool) {
	switch r.Role {
	case "":
		return "role required", false
	case api.RoleCC, api.RoleDoctor:
	default:
		return "role must be " + api.RoleCC + " or " + api.RoleDoctor, false
	}
	if r.Email == "" {
		return "email required", false
	}
	if r.FirstName == "" {
		return "first_name required", false
	}
	if r.LastName == "" {
		return "last_name required", false
	}
	if r.DOB.IsZero() {
		return "dob required", false
	}
	switch r.Gender {
	case "":
		return "gender required", false
	case "male", "female":
	default:
		return "gender must be 'male' or 'female'", false
	}
	if r.CellPhone == "" {
		return "cell_phone required", false
	}
	return "", true
}

func newProviderSearchAPIHandler(dataAPI api.DataAPI, authAPI api.AuthAPI) httputil.ContextHandler {
	return httputil.ContextSupportedMethods(&providerSearchAPIHandler{
		dataAPI: dataAPI,
		authAPI: authAPI,
	}, httputil.Get, httputil.Post)
}

func (h *providerSearchAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		h.get(ctx, w, r)
	case httputil.Post:
		h.post(ctx, w, r)
	}
}

func (h *providerSearchAPIHandler) get(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var results []*common.DoctorSearchResult

	query := r.FormValue("q")

	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "SearchProviders", map[string]interface{}{"query": query})

	if query != "" {
		var err error
		results, err = h.dataAPI.SearchDoctors(query)
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}
	}

	httputil.JSONResponse(w, http.StatusOK, &struct {
		Results []*common.DoctorSearchResult `json:"results"`
	}{
		Results: results,
	})
}

func (h *providerSearchAPIHandler) post(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "CreateProvider", nil)

	var req createProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		www.APIBadRequestError(w, r, err.Error())
		return
	}

	if msg, ok := req.validate(); !ok {
		www.APIBadRequestError(w, r, msg)
		return
	}

	accountID, err := h.authAPI.CreateAccount(req.Email, "", req.Role)
	switch err {
	case api.ErrLoginAlreadyExists:
		www.APIBadRequestError(w, r, "An account with the email already exists")
		return
	case nil:
	default:
		www.APIInternalError(w, r, err)
		return
	}

	id, err := h.dataAPI.RegisterProvider(&common.Doctor{
		AccountID: encoding.NewObjectID(accountID),
		Email:     req.Email,
		DOB:       req.DOB,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Gender:    req.Gender,
		CellPhone: req.CellPhone,
	}, req.Role)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &createProviderResponse{
		AccountID:  accountID,
		ProviderID: id,
	})
}
