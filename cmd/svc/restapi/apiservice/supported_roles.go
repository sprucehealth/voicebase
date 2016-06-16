package apiservice

import (
	"net/http"

	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type supportedRolesHandler struct {
	roles []string
	h     httputil.ContextHandler
}

// SupportedRoles wraps an HTTP handler with a filter that checks that the
// incoming request is made by one of the required roles.
func SupportedRoles(h httputil.ContextHandler, roles ...string) httputil.ContextHandler {
	return &supportedRolesHandler{
		h:     h,
		roles: roles,
	}
}

func (s *supportedRolesHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account, ok := CtxAccount(ctx)
	if ok {
		var roleFound bool
		for _, role := range s.roles {
			if role == account.Role {
				roleFound = true
				break
			}
		}

		if !roleFound {
			WriteAccessNotAllowedError(ctx, w, r)
			return
		}
	}
	s.h.ServeHTTP(ctx, w, r)
}
