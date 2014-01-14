package apiservice

import (
	"carefront/api"
	"fmt"
	"github.com/gorilla/schema"
	"github.com/subosito/twilio"
	"log"
	"net/http"
)

type DoctorSubmitPatientVisitReviewHandler struct {
	DataApi          api.DataAPI
	TwilioCli        *twilio.Client
	TwilioFromNumber string
	accountId        int64
}

type SubmitPatientVisitReviewRequest struct {
	PatientVisitId int64  `schema:"patient_visit_id"`
	Status         string `schema:"status"`
	Message        string `schema:"message"`
}

type SubmitPatientVisitReviewResponse struct {
	Result string `json:"result"`
}

const (
	patientVisitUpdateNotification = "There is an update to your case. Tap spruce://visit.com to view."
)

func (d *DoctorSubmitPatientVisitReviewHandler) AccountIdFromAuthToken(accountId int64) {
	d.accountId = accountId
}

func (d *DoctorSubmitPatientVisitReviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		d.submitPatientVisitReview(w, r)
	default:
		WriteJSONToHTTPResponseWriter(w, http.StatusNotFound, nil)
	}
}

func (d *DoctorSubmitPatientVisitReviewHandler) submitPatientVisitReview(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(SubmitPatientVisitReviewRequest)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	doctorId, _, _, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(requestData.PatientVisitId, d.accountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	// doctor can only update the state of a patient visit that is currently in REVIEWING state
	patientVisit, err := d.DataApi.GetPatientVisitFromId(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient visit for the patient visit id specified: "+err.Error())
		return
	}

	if patientVisit.Status != api.CASE_STATUS_REVIEWING {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to change the state of a patient visit to CLOSED when its not in the reviewing state")
		return
	}

	switch requestData.Status {
	case "", api.CASE_STATUS_CLOSED, api.CASE_STATUS_TREATED, api.CASE_STATUS_TRIAGED:
		// update the status of the patient visit
		err = d.DataApi.ClosePatientVisit(requestData.PatientVisitId, requestData.Status, requestData.Message)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the status of the visit to closed: "+err.Error())
			return
		}

	case api.CASE_STATUS_PHOTOS_REJECTED:
		// reject the patient photos
		err = d.DataApi.RejectPatientVisitPhotos(requestData.PatientVisitId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to reject patient photos: "+err.Error())
			return
		}

		// mark the status on the patient visit to retake photos
		err = d.DataApi.UpdatePatientVisitStatus(requestData.PatientVisitId, requestData.Message, api.CASE_STATUS_PHOTOS_REJECTED)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to mark the status of the patient visit as rejected: "+err.Error())
			return
		}
	default:
		WriteDeveloperError(w, http.StatusBadRequest, fmt.Sprintf("Status %s is not a valid status to set for the patient visit review", requestData.Status))
		return
	}

	// mark the status on the visit in the doctor's queue to move it to the completed tab
	// so that the visit is no longer in the hands of the doctor
	err = d.DataApi.UpdateStateForPatientVisitInDoctorQueue(doctorId, requestData.PatientVisitId, api.QUEUE_ITEM_STATUS_ONGOING, requestData.Status)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the status of the patient visit in the doctor queue: "+err.Error())
		return
	}

	//  Queue up notification to patient

	if d.TwilioCli != nil {
		patient, err := d.DataApi.GetPatientFromPatientVisitId(requestData.PatientVisitId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient from id: "+err.Error())
			return
		}
		if patient.Phone != "" {
			_, _, err = d.TwilioCli.Messages.SendSMS(d.TwilioFromNumber, patient.Phone, patientVisitUpdateNotification)
			if err != nil {
				log.Println("Error sending SMS: " + err.Error())
			}
		}
	}

	// TODO Send prescriptions to pharmacy of patient's choice

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}
