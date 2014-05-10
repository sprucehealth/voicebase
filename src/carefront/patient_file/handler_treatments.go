package patient_file

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"net/http"
	"strconv"

	"github.com/gorilla/schema"
)

type doctorPatientTreatmentsHandler struct {
	DataApi api.DataAPI
}

func NewDoctorPatientTreatmentsHandler(dataApi api.DataAPI) *doctorPatientTreatmentsHandler {
	return &doctorPatientTreatmentsHandler{
		DataApi: dataApi,
	}
}

type requestData struct {
	PatientId string `schema:"patient_id,required"`
}

type doctorPatientTreatmentsResponse struct {
	Treatments             []*common.Treatment         `json:"treatments,omitempty"`
	UnlinkedDNTFTreatments []*common.Treatment         `json:"unlinked_dntf_treatments,omitempty"`
	RefillRequests         []*common.RefillRequestItem `json:"refill_requests,omitempty"`
}

func (d *doctorPatientTreatmentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request parameters: "+err.Error())
		return
	}

	requestData := requestData{}
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	currentDoctor, err := d.DataApi.GetDoctorFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the doctor based on account id: "+err.Error())
		return
	}

	patientId, err := strconv.ParseInt(requestData.PatientId, 10, 64)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse patient id: "+err.Error())
		return
	}

	patient, err := d.DataApi.GetPatientFromId(patientId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient based on id: "+err.Error())
		return
	}

	if !patient.IsUnlinked {
		careTeam, err := d.DataApi.GetCareTeamForPatient(patientId)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get care team based on patient id: "+err.Error())
			return
		}

		primaryDoctorId := apiservice.GetPrimaryDoctorIdFromCareTeam(careTeam)

		if currentDoctor.DoctorId.Int64() != primaryDoctorId {
			apiservice.WriteDeveloperError(w, http.StatusForbidden, "Unable to get the patient information by doctor when this doctor is not the primary doctor for patient")
			return
		}
	}

	treatments, err := d.DataApi.GetTreatmentsForPatient(patientId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatments for patient: "+err.Error())
		return
	}

	refillRequests, err := d.DataApi.GetRefillRequestsForPatient(patientId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get refill requests for patient: "+err.Error())
		return
	}

	unlinkedDNTFTreatments, err := d.DataApi.GetUnlinkedDNTFTreatmentsForPatient(patientId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get unlinked dntf treatments for patient: "+err.Error())
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &doctorPatientTreatmentsResponse{
		Treatments:             treatments,
		RefillRequests:         refillRequests,
		UnlinkedDNTFTreatments: unlinkedDNTFTreatments,
	})
}
