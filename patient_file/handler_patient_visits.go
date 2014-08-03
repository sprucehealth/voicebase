package patient_file

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

type patientVisitsHandler struct {
	DataApi api.DataAPI
}

type request struct {
	PatientId int64 `schema:"patient_id,required"`
}

type response struct {
	PatientVisits []*common.PatientVisit `json:"patient_visits"`
}

func NewPatientVisitsHandler(dataApi api.DataAPI) http.Handler {
	return apiservice.SupportedMethods(&patientVisitsHandler{
		DataApi: dataApi,
	}, []string{apiservice.HTTP_GET})
}

func (p *patientVisitsHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.DOCTOR_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	requestData := &request{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	doctor, err := p.DataApi.GetDoctorFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Doctor] = doctor

	patient, err := p.DataApi.GetPatientFromId(requestData.PatientId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Patient] = patient

	if err := apiservice.ValidateDoctorAccessToPatientFile(doctor.DoctorId.Int64(), patient.PatientId.Int64(), p.DataApi); err != nil {
		return false, err
	}

	return true, nil
}

func (p *patientVisitsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	patient := ctxt.RequestCache[apiservice.Patient].(*common.Patient)

	patientVisits, err := p.DataApi.GetPatientVisitsForPatient(patient.PatientId.Int64())
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient visits for patient: "+err.Error())
		return
	}

	apiservice.WriteJSON(w, response{PatientVisits: patientVisits})
}
