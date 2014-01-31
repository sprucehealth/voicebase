package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/erx"
	"carefront/libs/golog"
	"carefront/libs/pharmacy"
	"fmt"
	"github.com/gorilla/schema"
	"github.com/subosito/twilio"
	"net/http"
)

type DoctorSubmitPatientVisitReviewHandler struct {
	IOSDeeplinkScheme string
	DataApi           api.DataAPI
	TwilioCli         *twilio.Client
	TwilioFromNumber  string
	ERxApi            erx.ERxAPI
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
	patientVisitUpdateNotification = "There is an update to your case. Tap %s://visit to view."
)

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

	doctorId, _, _, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(requestData.PatientVisitId, GetContext(r).AccountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	// doctor can only update the state of a patient visit that is currently in REVIEWING state
	err = EnsurePatientVisitInExpectedStatus(d.DataApi, requestData.PatientVisitId, api.CASE_STATUS_REVIEWING)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	switch requestData.Status {
	case "", api.CASE_STATUS_CLOSED, api.CASE_STATUS_TREATED, api.CASE_STATUS_TRIAGED:
		// update the status of the patient visit
		status := requestData.Status
		if status == "" {
			status = api.CASE_STATUS_TREATED
		}
		err = d.DataApi.ClosePatientVisit(requestData.PatientVisitId, status, requestData.Message)
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

	// if doctor treated patient, check for treatments submitted for patient visit,
	// and send to dose spot
	if requestData.Status == api.CASE_STATUS_TREATED || requestData.Status == "" {
		patient, err := d.DataApi.GetPatientFromPatientVisitId(requestData.PatientVisitId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient data from patient visit: "+err.Error())
			return
		}

		// FIX: add fake address for now
		patient.PatientAddress = &common.Address{}
		patient.PatientAddress.AddressLine1 = "1234 Main Street"
		patient.PatientAddress.City = "San Francisco"
		patient.PatientAddress.State = "CA"
		patient.PatientAddress.ZipCode = "94103"

		// FIX: add fake phone type for now
		patient.PhoneType = "Home"

		// FIX: add fake pharmacy for now
		patient.Pharmacy = &pharmacy.PharmacyData{}
		patient.Pharmacy.Id = "39203"

		treatmentPlan, err := d.DataApi.GetTreatmentPlanForPatientVisit(requestData.PatientVisitId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment plan: "+err.Error())
			return
		}

		err = d.ERxApi.StartPrescribingPatient(patient, treatmentPlan.Treatments)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to start prescribing patient: "+err.Error())
			return
		}

		// Save erx patient id to database
		err = d.DataApi.UpdatePatientWithERxPatientId(patient.PatientId, patient.ERxPatientId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to save the patient id returned from dosespot for patient: "+err.Error())
			return
		}

		// Save prescription ids for drugs to database
		err = d.DataApi.UpdateTreatmentsWithPrescriptionIds(treatmentPlan.Treatments, doctorId, requestData.PatientVisitId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to save prescription ids for treatments: "+err.Error())
			return
		}

		// Now, send the prescription to the doctor
		err = d.ERxApi.SendMultiplePrescriptions(patient, treatmentPlan.Treatments)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to send prescription to patient's pharmacy: "+err.Error())
			return
		}
	}

	//  Queue up notification to patient

	if d.TwilioCli != nil {
		patient, err := d.DataApi.GetPatientFromPatientVisitId(requestData.PatientVisitId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient from id: "+err.Error())
			return
		}
		if patient.Phone != "" {
			_, _, err = d.TwilioCli.Messages.SendSMS(d.TwilioFromNumber, patient.Phone, fmt.Sprintf(patientVisitUpdateNotification, d.IOSDeeplinkScheme))
			if err != nil {
				golog.Errorf("Error sending SMS: %s", err.Error())
			}
		}
	}

	// TODO Send prescriptions to pharmacy of patient's choice

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}
