package apiservice

import (
	"context"
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
)

type contextKey int

const (
	ckAccount contextKey = iota
	ckPatient
	ckDoctor
	ckCC
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

// MustCtxCC returns the doctor mapping to the care coordinator from the context or panics if it's missing or wrong type.
func MustCtxCC(ctx context.Context) *common.Doctor {
	return ctx.Value(ckDoctor).(*common.Doctor)
}

// MustCtxDoctor returns the doctor from the context or panics if it's missing or wrong type.
func MustCtxDoctor(ctx context.Context) *common.Doctor {
	return ctx.Value(ckDoctor).(*common.Doctor)
}

// MustCtxPatient returns the patient from the context or panics if it's missing or wrong type.
func MustCtxPatient(ctx context.Context) *common.Patient {
	return ctx.Value(ckPatient).(*common.Patient)
}

// MustCtxAccount returns the account from the context or panics if it's missing or wrong type.
func MustCtxAccount(ctx context.Context) *common.Account {
	return ctx.Value(ckAccount).(*common.Account)
}

// MustCtxCache returns the request cache from the context or panics if it's missing or wrong type.
func MustCtxCache(ctx context.Context) map[CacheKey]interface{} {
	return ctx.Value(ckCache).(map[CacheKey]interface{})
}

// CtxPatient returns the patient from the context and true. Otherwise it returns false.
func CtxPatient(ctx context.Context) (*common.Patient, bool) {
	v := ctx.Value(ckPatient)
	if v == nil {
		return nil, false
	}
	return v.(*common.Patient), true
}

// CtxDoctor returns the doctor from the context and true. Otherwise it returns false.
func CtxDoctor(ctx context.Context) (*common.Doctor, bool) {
	v := ctx.Value(ckDoctor)
	if v == nil {
		return nil, false
	}
	return v.(*common.Doctor), true
}

// CtxCC returns the doctor that maps to the care coordinator from the context and true. Otherwise it returns false.
func CtxCC(ctx context.Context) (*common.Doctor, bool) {
	v := ctx.Value(ckCC)
	if v == nil {
		return nil, false
	}
	return v.(*common.Doctor), true
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

// CtxWithCC returns a new context that includes the doctor mapping to the care coordinator
func CtxWithCC(ctx context.Context, d *common.Doctor) context.Context {
	return context.WithValue(ctx, ckCC, d)
}

// CtxWithDoctor returns a new context that includes the doctor
func CtxWithDoctor(ctx context.Context, d *common.Doctor) context.Context {
	return context.WithValue(ctx, ckDoctor, d)
}

// CtxWithPatient returns a new context that includes the patient
func CtxWithPatient(ctx context.Context, p *common.Patient) context.Context {
	return context.WithValue(ctx, ckPatient, p)
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
func RequestCacheHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		_, ok := CtxCache(ctx)
		if !ok {
			ctx = CtxWithCache(ctx, nil)
		}
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}
