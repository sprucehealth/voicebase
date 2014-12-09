package patient_file

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/pharmacy"
)

type doctorUpdatePatientPharmacyHandler struct {
	dataAPI api.DataAPI
}

func NewDoctorUpdatePatientPharmacyHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(apiservice.AuthorizationRequired(
		&doctorUpdatePatientPharmacyHandler{
			dataAPI: dataAPI,
		}), []string{"PUT"})
}

type DoctorUpdatePatientPharmacyRequestData struct {
	PatientID encoding.ObjectID      `json:"patient_id"`
	Pharmacy  *pharmacy.PharmacyData `json:"pharmacy"`
}

func (d *doctorUpdatePatientPharmacyHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.DOCTOR_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	requestData := &DoctorUpdatePatientPharmacyRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	patient, err := d.dataAPI.GetPatientFromID(requestData.PatientID.Int64())
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Patient] = patient

	doctor, err := d.dataAPI.GetDoctorFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Doctor] = doctor

	if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, ctxt.Role, doctor.DoctorID.Int64(), patient.PatientID.Int64(), d.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (d *doctorUpdatePatientPharmacyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	patient := ctxt.RequestCache[apiservice.Patient].(*common.Patient)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*DoctorUpdatePatientPharmacyRequestData)

	if err := d.dataAPI.UpdatePatientPharmacy(patient.PatientID.Int64(), requestData.Pharmacy); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}
