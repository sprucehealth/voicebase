package patient_file

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/encoding"
	"carefront/libs/pharmacy"
	"net/http"
)

type doctorUpdatePatientPharmacyHandler struct {
	dataAPI api.DataAPI
}

func NewDoctorUpdatePatientPharmacyHandler(dataAPI api.DataAPI) *doctorUpdatePatientPharmacyHandler {
	return &doctorUpdatePatientPharmacyHandler{
		dataAPI: dataAPI,
	}
}

type DoctorUpdatePatientPharmacyRequestData struct {
	PatientId encoding.ObjectId      `json:"patient_id"`
	Pharmacy  *pharmacy.PharmacyData `json:"pharmacy"`
}

func (d *doctorUpdatePatientPharmacyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_PUT {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestData := &DoctorUpdatePatientPharmacyRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError("Unable to parse input parameters", w, r)
		return
	}

	patient, err := d.dataAPI.GetPatientFromId(requestData.PatientId.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	doctor, err := d.dataAPI.GetDoctorFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	if err := apiservice.ValidateDoctorAccessToPatientFile(doctor.DoctorId.Int64(), patient.PatientId.Int64(), d.dataAPI, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := d.dataAPI.UpdatePatientPharmacy(patient.PatientId.Int64(), requestData.Pharmacy); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.SuccessfulGenericJSONResponse())
}
