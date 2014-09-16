package apiservice

import (
	"net/http"
	"sync"
	"time"
)

var (
	ctxMu          sync.Mutex
	requestContext = map[*http.Request]*Context{}
)

type CacheKey int

const (
	Patient CacheKey = iota
	PatientID
	PersonID
	Doctor
	DoctorID
	PatientCase
	PatientCaseID
	PatientVisit
	PatientVisitID
	TreatmentPlan
	TreatmentPlanID
	Treatment
	TreatmentID
	RefillRequestID
	RefillRequest
	RequestData
	ERxSource
	FavoriteTreatmentPlan
	Account
)

type Context struct {
	AccountId        int64
	Role             string
	RequestStartTime time.Time
	RequestID        int64
	RequestCache     map[CacheKey]interface{}
}

func GetContext(req *http.Request) *Context {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	if ctx := requestContext[req]; ctx != nil {
		return ctx
	}
	ctx := &Context{}
	ctx.RequestCache = make(map[CacheKey]interface{})
	requestContext[req] = ctx
	return ctx
}

func DeleteContext(req *http.Request) {
	ctxMu.Lock()
	delete(requestContext, req)
	ctxMu.Unlock()
}
