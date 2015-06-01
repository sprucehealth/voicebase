package patient_file

import (
	"errors"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type doctorPatientTreatmentsHandler struct {
	DataAPI api.DataAPI
}

func NewDoctorPatientTreatmentsHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(
			&doctorPatientTreatmentsHandler{
				DataAPI: dataAPI,
			}), httputil.Get)
}

type requestData struct {
	PatientID int64 `schema:"patient_id,required"`
}

type doctorPatientTreatmentsResponse struct {
	Treatments             []*common.Treatment         `json:"treatments,omitempty"`
	UnlinkedDNTFTreatments []*common.Treatment         `json:"unlinked_dntf_treatments,omitempty"`
	RefillRequests         []*common.RefillRequestItem `json:"refill_requests,omitempty"`
}

func (d *doctorPatientTreatmentsHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.RoleDoctor {
		return false, apiservice.NewAccessForbiddenError()
	}

	requestData := &requestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	currentDoctor, err := d.DataAPI.GetDoctorFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Doctor] = currentDoctor

	patient, err := d.DataAPI.GetPatientFromID(requestData.PatientID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Patient] = patient

	if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, ctxt.Role, currentDoctor.ID.Int64(), patient.ID.Int64(), d.DataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (d *doctorPatientTreatmentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*requestData)

	treatments, err := d.DataAPI.GetTreatmentsForPatient(requestData.PatientID)
	if err != nil {
		apiservice.WriteError(errors.New("Unable to get treatments for patient: "+err.Error()), w, r)
		return
	}

	refillRequests, err := d.DataAPI.GetRefillRequestsForPatient(requestData.PatientID)
	if err != nil {
		apiservice.WriteError(errors.New("Unable to get refill requests for patient: "+err.Error()), w, r)
		return
	}

	unlinkedDNTFTreatments, err := d.DataAPI.GetUnlinkedDNTFTreatmentsForPatient(requestData.PatientID)
	if err != nil {
		apiservice.WriteError(errors.New("Unable to get unlinked dntf treatments for patient: "+err.Error()), w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &doctorPatientTreatmentsResponse{
		Treatments:             treatments,
		RefillRequests:         refillRequests,
		UnlinkedDNTFTreatments: unlinkedDNTFTreatments,
	})
}
