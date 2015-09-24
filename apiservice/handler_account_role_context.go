package apiservice

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type accountRoleDAL interface {
	GetPatientFromAccountID(accountID int64) (*common.Patient, error)
	GetDoctorFromAccountID(accountID int64) (*common.Doctor, error)
}

type accountRoleContextHandler struct {
	h              httputil.ContextHandler
	methods        map[string]bool
	accountRoleDAL accountRoleDAL
}

// NewAccountRoleContextHandler wraps the provided handler in a layer that adds the caller role information to the context. This is currently either a patient or a doctor object
// The lookup will only be done for methods provided, or for all calls if no methods are provided
// Note: This asserts that the account ID is already present in the context for panics
func NewAccountRoleContextHandler(h httputil.ContextHandler, accountRoleDAL accountRoleDAL, methods ...string) httputil.ContextHandler {
	ms := make(map[string]bool, len(methods))
	for _, m := range methods {
		ms[m] = true
	}
	return &accountRoleContextHandler{
		h:              h,
		methods:        ms,
		accountRoleDAL: accountRoleDAL,
	}
}

func (h *accountRoleContextHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Bypass this setup if this is a test reuqest
	// TODO: We should think of a possible better way for these assertions
	if isTestRequest(r) {
		h.h.ServeHTTP(ctx, w, r)
		return
	} else if _, ok := h.methods[r.Method]; ok || len(h.methods) == 0 {
		a := MustCtxAccount(ctx)
		switch a.Role {
		case api.RolePatient:
			patient, err := h.accountRoleDAL.GetPatientFromAccountID(a.ID)
			if err != nil {
				WriteError(ctx, err, w, r)
				return
			}
			ctx = CtxWithPatient(ctx, patient)
		case api.RoleDoctor, api.RoleCC:
			doctor, err := h.accountRoleDAL.GetDoctorFromAccountID(a.ID)
			if err != nil {
				WriteError(ctx, err, w, r)
				return
			}
			switch a.Role {
			case api.RoleDoctor:
				ctx = CtxWithDoctor(ctx, doctor)
			case api.RoleCC:
				ctx = CtxWithCC(ctx, doctor)
			}
		default:
			WriteError(ctx, fmt.Errorf("Unknown account role %s - Unable to locate account role sub object", a.Role), w, r)
			return
		}
	}

	h.h.ServeHTTP(ctx, w, r)
}
