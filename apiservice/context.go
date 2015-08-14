package apiservice

import (
	"net/http"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type contextKey int

const (
	ckAccount contextKey = iota
	ckCache
)

// CacheKey represents a type used to key a cache map
type CacheKey int

// Available cache keys
const (
	CKPatient CacheKey = iota
	CKPatientID
	CKPersonID
	CKDoctor
	CKDoctorID
	CKPatientCase
	CKPatientCaseID
	CKPatientVisit
	CKPatientVisitID
	CKTreatmentPlan
	CKTreatmentPlanID
	CKTreatment
	CKTreatmentID
	CKRefillRequestID
	CKRefillRequest
	CKRequestData
	CKERxSource
	CKFavoriteTreatmentPlan
	CKAccount
)

// MustCtxAccount returns the account from the context or panics if it's missing or wrong type.
func MustCtxAccount(ctx context.Context) *common.Account {
	return ctx.Value(ckAccount).(*common.Account)
}

// MustCtxCache returns the request cache from the context or panics if it's missing or wrong type.
func MustCtxCache(ctx context.Context) map[CacheKey]interface{} {
	return ctx.Value(ckCache).(map[CacheKey]interface{})
}

// CtxAccount returns the account from the context and true. Otherwise it returns false.
func CtxAccount(ctx context.Context) (*common.Account, bool) {
	v := ctx.Value(ckAccount)
	if v == nil {
		return nil, false
	}
	return v.(*common.Account), true
}

// CtxCache returns the request cache from the context and true. Otherwise it returns false.
func CtxCache(ctx context.Context) (map[CacheKey]interface{}, bool) {
	v := ctx.Value(ckCache)
	if v == nil {
		return nil, false
	}
	return v.(map[CacheKey]interface{}), true
}

// CtxWithAccount returns a new context that includes the account
func CtxWithAccount(ctx context.Context, a *common.Account) context.Context {
	return context.WithValue(ctx, ckAccount, a)
}

// CtxWithCache returns a new context that includes a request cache. If the cache
// is nil then a new map is allocated.
func CtxWithCache(ctx context.Context, c map[CacheKey]interface{}) context.Context {
	if c == nil {
		c = make(map[CacheKey]interface{})
	}
	return context.WithValue(ctx, ckCache, c)
}

// RequestCacheHandler wraps a handler to provide a cache in the request cache
func RequestCacheHandler(h httputil.ContextHandler) httputil.ContextHandler {
	return httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		_, ok := CtxCache(ctx)
		if !ok {
			ctx = CtxWithCache(ctx, nil)
		}
		h.ServeHTTP(ctx, w, r)
	})
}
