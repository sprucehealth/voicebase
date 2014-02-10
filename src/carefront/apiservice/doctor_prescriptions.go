package apiservice

// import (
// 	"carefront/api"
// 	"github.com/gorilla/schema"
// 	"net/http"
// 	"time"
// )

// type DoctorPrescriptionsHandler struct {
// 	DataApi api.DataAPI
// }

// type DoctorPrescriptionsRequestData struct {
// 	FromTimeUnix int64 `schema:"from"`
// 	ToTimeUnix   int64 `schema:"to"`
// }

// func (d *DoctorSubmitPatientVisitReviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	r.ParseForm()
// 	requestData := new(PatientVisitRequestData)
// 	decoder := schema.NewDecoder()
// 	err := decoder.Decode(requestData, r.Form)
// 	if err != nil {
// 		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
// 		return
// 	}

// 	// STEP 1: Get a list of treatments (including the prescription status from the erx_status_events table)
// 	// prescribed by the doctor within the given time period

// 	// STEP 2: Group each of the treatments by day

// 	// STEP 3: Within each day, group each of the treatments by patient

// 	// STEP 4: Get patient information for each grouping of the prescriptions

// 	// STEP 5: Get

// }
