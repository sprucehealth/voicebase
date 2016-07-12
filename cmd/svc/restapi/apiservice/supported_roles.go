package apiservice

import (
	"net/http"
)

type supportedRolesHandler struct {
	roles []string
	h     http.Handler
}

// SupportedRoles wraps an HTTP handler with a filter that checks that the
// incoming request is made by one of the required roles.
func SupportedRoles(h http.Handler, roles ...string) http.Handler {
	return &supportedRolesHandler{
		h:     h,
		roles: roles,
	}
}

func (s *supportedRolesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account, ok := CtxAccount(r.Context())
	if ok {
		var roleFound bool
		for _, role := range s.roles {
			if role == account.Role {
				roleFound = true
				break
			}
		}

		if !roleFound {
			WriteAccessNotAllowedError(w, r)
			return
		}
	}
	s.h.ServeHTTP(w, r)
}
