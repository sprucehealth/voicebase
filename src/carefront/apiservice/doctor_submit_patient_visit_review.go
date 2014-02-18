package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/erx"
	"carefront/libs/golog"
	"carefront/libs/pharmacy"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/schema"
	"github.com/subosito/twilio"
)

type DoctorSubmitPatientVisitReviewHandler struct {
	IOSDeeplinkScheme string
	DataApi           api.DataAPI
	TwilioCli         *twilio.Client
	TwilioFromNumber  string
	ERxApi            erx.ERxAPI
	ErxStatusQueue    *common.SQSQueue
	ERxRouting        bool
}

type SubmitPatientVisitReviewRequest struct {
	PatientVisitId int64  `schema:"patient_visit_id"`
	Status         string `schema:"status"`
	Message        string `schema:"message"`
}

type SubmitPatientVisitReviewResponse struct {
	Result string `json:"result"`
}

type PrescriptionStatusCheckMessage struct {
	PatientId int64
	DoctorId  int64
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

	treatmentPlanId, err := d.DataApi.GetActiveTreatmentPlanForPatientVisit(doctorId, requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get current active treatment plan for patient visit: "+err.Error())
		return
	}

	patient, err := d.DataApi.GetPatientFromPatientVisitId(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient data from patient visit: "+err.Error())
		return
	}

	// if doctor treated patient, check for treatments submitted for patient visit,
	// and send to dose spot
	if requestData.Status == api.CASE_STATUS_TREATED || requestData.Status == "" {

		// FIX: add fake address for now until we start accepting address from client
		patient.PatientAddress = &common.Address{
			AddressLine1: "1234 Main Street",
			City:         "San Francisco",
			State:        "CA",
			ZipCode:      "94103",
		}

		pharmacySelection, err := d.DataApi.GetPatientPharmacySelection(patient.PatientId.Int64())
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get pharmacy selection for patient: "+err.Error())
			return
		}
		// FIX: Undo this when we are using surescripts as our backing database for pharmacies
		// patient.pharmacy = pharmacySelection

		// FIX: add fake pharmacy for now
		patient.Pharmacy = &pharmacy.PharmacyData{
			Id:      "39203",
			Source:  pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
			Address: "123 TEST TEST",
			City:    "San Francisco",
			State:   "CA",
			Postal:  "94115",
		}

		treatments, err := d.DataApi.GetTreatmentsBasedOnTreatmentPlanId(requestData.PatientVisitId, treatmentPlanId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatments based on active treatment plan: "+err.Error())
			return
		}

		if d.ERxRouting == true && d.ERxApi != nil && len(treatments) > 0 {
			err = d.ERxApi.StartPrescribingPatient(patient, treatments)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to start prescribing patient: "+err.Error())
				return
			}

			// Save erx patient id to database
			err = d.DataApi.UpdatePatientWithERxPatientId(patient.PatientId.Int64(), patient.ERxPatientId.Int64())
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to save the patient id returned from dosespot for patient: "+err.Error())
				return
			}

			// Save prescription ids for drugs to database
			err = d.DataApi.MarkTreatmentsAsPrescriptionsSent(treatments, pharmacySelection, doctorId, requestData.PatientVisitId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to save prescription ids for treatments: "+err.Error())
				return
			}

			// Now, send the prescription to the pharmacy
			unSuccessfulTreatmentIds, err := d.ERxApi.SendMultiplePrescriptions(patient, treatments)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to send prescription to patient's pharmacy: "+err.Error())
				return
			}

			successfulTreatments := make([]*common.Treatment, 0)
			unSuccessfulTreatments := make([]*common.Treatment, 0)
			for _, treatment := range treatments {
				treatmentFound := false
				for _, unSuccessfulTreatmentId := range unSuccessfulTreatmentIds {
					if unSuccessfulTreatmentId == treatment.Id.Int64() {
						treatmentFound = true
						break
					}
				}
				if !treatmentFound {
					successfulTreatments = append(successfulTreatments, treatment)
				} else {
					unSuccessfulTreatments = append(unSuccessfulTreatments, treatment)
				}
			}

			if len(successfulTreatments) > 0 {
				err = d.DataApi.AddErxStatusEvent(successfulTreatments, api.ERX_STATUS_SENDING)
				if err != nil {
					WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add an erx status event: "+err.Error())
					return
				}
			}

			if len(unSuccessfulTreatments) > 0 {
				err = d.DataApi.AddErxStatusEvent(unSuccessfulTreatments, api.ERX_STATUS_SEND_ERROR)
				if err != nil {
					WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add an erx status event: "+err.Error())
					return
				}
			}

			//  Queue up notification to patient
			err = d.queueUpJobForErxStatus(patient.PatientId.Int64(), doctorId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to queue up job for getting prescription status: "+err.Error())
				return
			}
		}
	}

	switch requestData.Status {
	case "", api.CASE_STATUS_CLOSED, api.CASE_STATUS_TREATED, api.CASE_STATUS_TRIAGED:
		// update the status of the patient visit
		status := requestData.Status
		if status == "" {
			status = api.CASE_STATUS_TREATED
		}
		err = d.DataApi.ClosePatientVisit(requestData.PatientVisitId, treatmentPlanId, status, requestData.Message)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the status of the visit to closed: "+err.Error())
			return
		}

	case api.CASE_STATUS_PHOTOS_REJECTED:
		// reject the  patient photos
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

	err = d.sendSMSToNotifyPatient(patient, requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to SMS notification to patient: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}

func (d *DoctorSubmitPatientVisitReviewHandler) queueUpJobForErxStatus(patientId, doctorId int64) error {
	// queue up a job to get the updated status of the prescription
	// to know when exatly the message was sent to the pharmacy
	erxMessage := &PrescriptionStatusCheckMessage{
		PatientId: patientId,
		DoctorId:  doctorId,
	}
	jsonData, err := json.Marshal(erxMessage)
	if err != nil {
		return err
	}

	// queue up a job
	return d.ErxStatusQueue.QueueService.SendMessage(d.ErxStatusQueue.QueueUrl, 0, string(jsonData))
}

func (d *DoctorSubmitPatientVisitReviewHandler) sendSMSToNotifyPatient(patient *common.Patient, patientVisitId int64) error {

	if d.TwilioCli != nil {

		if patient.Phone != "" {
			_, _, err := d.TwilioCli.Messages.SendSMS(d.TwilioFromNumber, patient.Phone, fmt.Sprintf(patientVisitUpdateNotification, d.IOSDeeplinkScheme))
			if err != nil {
				golog.Errorf("Error sending SMS: %s", err.Error())
			}
		}
	}

	return nil
}
