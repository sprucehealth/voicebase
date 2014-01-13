package apiservice

import (
	"carefront/api"
	"github.com/gorilla/schema"
	"net/http"
)

type DoctorRejectPhotosHandler struct {
	DataApi   api.DataAPI
	accountId int64
}

type DoctorRejectPhotosRequestDatas struct {
	PatientVisitId int64  `schema:"patient_visit_id"`
	Message        string `schema:"message"`
}

func (d *DoctorRejectPhotosHandler) AccountIdFromAuthToken(accountId int64) {
	d.accountId = accountId
}

func (d *DoctorRejectPhotosHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		d.rejectPhotosForPatientVisit(w, r)
	default:
		WriteJSONToHTTPResponseWriter(w, http.StatusNotFound, nil)
	}
}

func (d *DoctorRejectPhotosHandler) rejectPhotosForPatientVisit(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(DoctorRejectPhotosRequestDatas)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	// ensure that the doctor is the one authorized to work on the case
	doctorId, _, _, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(requestData.PatientVisitId, d.accountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	// reject the patient photos
	err = d.DataApi.RejectPatientVisitPhotos(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to reject patient photos: "+err.Error())
		return
	}

	// mark the status on the patient visit to retake photos
	if requestData.Message != "" {
		err = d.DataApi.UpdatePatientVisitStatusWithMessage(requestData.PatientVisitId, requestData.Message, api.CASE_STATUS_PHOTOS_REJECTED)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to mark the status of the patient visit as rejected: "+err.Error())
			return
		}
	} else {
		err = d.DataApi.UpdatePatientVisitStatus(requestData.PatientVisitId, api.CASE_STATUS_PHOTOS_REJECTED)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to mark the status of the patient visit as rejected: "+err.Error())
			return
		}

	}

	// mark the status on the visit in the doctor's queue to move it to the completed tab
	// so that the visit is no longer in the hands of the doctor
	err = d.DataApi.UpdateStateForPatientVisitInDoctorQueue(doctorId, requestData.PatientVisitId, api.QUEUE_ITEM_STATUS_ONGOING, api.QUEUE_ITEM_STATUS_PHOTOS_REJECTED)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the status of the patient visit in the doctor queue: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}
