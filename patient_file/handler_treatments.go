package patient_file

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"

	"github.com/sprucehealth/backend/third_party/github.com/gorilla/schema"
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
	PatientId int64 `schema:"patient_id,required"`
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

	if err := apiservice.ValidateDoctorAccessToPatientFile(currentDoctor.DoctorId.Int64(), requestData.PatientId, d.DataApi); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	patient, err := d.DataApi.GetPatientFromId(requestData.PatientId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient based on id: "+err.Error())
		return
	}

	if !patient.IsUnlinked {
		if err := apiservice.ValidateDoctorAccessToPatientFile(currentDoctor.DoctorId.Int64(), patient.PatientId.Int64(), d.DataApi); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	treatments, err := d.DataApi.GetTreatmentsForPatient(requestData.PatientId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatments for patient: "+err.Error())
		return
	}

	refillRequests, err := d.DataApi.GetRefillRequestsForPatient(requestData.PatientId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get refill requests for patient: "+err.Error())
		return
	}

	unlinkedDNTFTreatments, err := d.DataApi.GetUnlinkedDNTFTreatmentsForPatient(requestData.PatientId)
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
