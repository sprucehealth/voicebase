package www

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
)

type PermissionsAPI interface {
	PermissionsForAccount(accountID int64) ([]string, error)
}

// Permissions is is the set of permissions for an account
type Permissions map[string]bool

// Has returns true iff the permissions set allows the requested permission
func (p Permissions) Has(perm string) bool {
	return p[perm]
}

// HasAny returns true iff the permissions set allows any of the request permissions
func (p Permissions) HasAny(perms []string) bool {
	for _, per := range perms {
		if p[per] {
			return true
		}
	}
	return false
}

// HasAll returns true iff the permissions set allows all of the request permissions.
// If even one permissions is not allowed then it will return false.
func (p Permissions) HasAll(perms []string) bool {
	for _, per := range perms {
		if !p[per] {
			return false
		}
	}
	return true
}

var permissionDeniedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusForbidden)
	return
})

type permsRequiredHandler struct {
	api           PermissionsAPI
	perms         map[string][]string
	okHandler     http.Handler
	failedHandler http.Handler
}

func PermissionsRequiredHandler(api PermissionsAPI, perms map[string][]string, ok, failed http.Handler) http.Handler {
	if failed == nil {
		failed = permissionDeniedHandler
	}
	return &permsRequiredHandler{
		api:           api,
		perms:         perms,
		okHandler:     ok,
		failedHandler: failed,
	}
}

func (h *permsRequiredHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	account := MustCtxAccount(ctx)

	var permsMap Permissions

	// Check if someone has already fetched the permissions (likely nested instances of this handler)
	if p, ok := CtxPermissions(ctx); ok {
		permsMap = p
	} else {
		perms, err := h.api.PermissionsForAccount(account.ID)
		if err != nil {
			InternalServerError(w, r, err)
			return
		}
		permsMap = Permissions(make(map[string]bool, len(perms)))
		for _, p := range perms {
			permsMap[p] = true
		}
		ctx = CtxWithPermissions(ctx, permsMap)
	}

	if permsMap.HasAny(h.perms[r.Method]) {
		h.okHandler.ServeHTTP(w, r.WithContext(ctx))
		return
	}

	h.failedHandler.ServeHTTP(w, r.WithContext(ctx))
}

type noPermsRequiredHandler struct {
	authAPI api.AuthAPI
	h       http.Handler
}

func NoPermissionsRequiredFilter(authAPI api.AuthAPI) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return NoPermissionsRequiredHandler(authAPI, h)
	}
}

// NoPermissionsRequiredHandler pulls down and caches permissions but
// doesn't check them. The use is when a handler itself will validate
// the permissions.
func NoPermissionsRequiredHandler(authAPI api.AuthAPI, h http.Handler) http.Handler {
	return &noPermsRequiredHandler{
		authAPI: authAPI,
		h:       h,
	}
}

func (h *noPermsRequiredHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	account := MustCtxAccount(ctx)

	perms, err := h.authAPI.PermissionsForAccount(account.ID)
	if err != nil {
		InternalServerError(w, r, err)
		return
	}
	permsMap := Permissions(make(map[string]bool, len(perms)))
	for _, p := range perms {
		permsMap[p] = true
	}
	ctx = CtxWithPermissions(ctx, permsMap)

	h.h.ServeHTTP(w, r.WithContext(ctx))
}
