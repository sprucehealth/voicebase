package patient_file

import (
	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/surescripts"

	"net/http"
)

type doctorPatientHandler struct {
	DataApi              api.DataAPI
	ErxApi               erx.ERxAPI
	AddressValidationApi address.AddressValidationAPI
}

func NewDoctorPatientHandler(dataApi api.DataAPI, erxApi erx.ERxAPI, addressValidationApi address.AddressValidationAPI) *doctorPatientHandler {
	return &doctorPatientHandler{
		DataApi:              dataApi,
		ErxApi:               erxApi,
		AddressValidationApi: addressValidationApi,
	}
}

func (d *doctorPatientHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case apiservice.HTTP_GET:
		d.getPatientInformation(w, r)
	case apiservice.HTTP_PUT:
		d.updatePatientInformation(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

type requestResponstData struct {
	Patient   *common.Patient `json:"patient"`
	PatientId int64           `schema:"patient_id,required" json:"-"`
}

func (d *doctorPatientHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.DOCTOR_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	requestData := &requestResponstData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	doctor, err := d.DataApi.GetDoctorFromAccountId(ctxt.AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Doctor] = doctor

	patientId := requestData.PatientId
	if patientId == 0 {
		patientId = requestData.Patient.PatientId.Int64()
	}

	patient, err := d.DataApi.GetPatientFromId(patientId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Patient] = patient

	if !patient.IsUnlinked {
		if err := apiservice.ValidateDoctorAccessToPatientFile(doctor.DoctorId.Int64(), patient.PatientId.Int64(), d.DataApi); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (d *doctorPatientHandler) getPatientInformation(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	patient := ctxt.RequestCache[apiservice.Patient].(*common.Patient)

	apiservice.WriteJSON(w, &requestResponstData{Patient: patient})
}

func (d *doctorPatientHandler) updatePatientInformation(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*requestResponstData)
	currentDoctor := ctxt.RequestCache[apiservice.Doctor].(*common.Doctor)
	existingPatientInfo := ctxt.RequestCache[apiservice.Patient].(*common.Patient)

	err := surescripts.ValidatePatientInformation(requestData.Patient, d.AddressValidationApi, d.DataApi)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	requestData.Patient.ERxPatientId = existingPatientInfo.ERxPatientId
	requestData.Patient.Pharmacy = existingPatientInfo.Pharmacy

	if err := d.ErxApi.UpdatePatientInformation(currentDoctor.DoseSpotClinicianId, requestData.Patient); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, `Unable to update patient information on dosespot. 
			Failing to avoid out of sync issues between surescripts and our platform `+err.Error())
		return
	}

	// update the doseSpot Id for the patient in our system now that we got one
	if !existingPatientInfo.ERxPatientId.IsValid {
		if err := d.DataApi.UpdatePatientWithERxPatientId(requestData.Patient.PatientId.Int64(), requestData.Patient.ERxPatientId.Int64()); err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the patientId from dosespot: "+err.Error())
			return
		}
	}

	// go ahead and udpate the doctor's information in our system now that dosespot has it
	if err := d.DataApi.UpdatePatientInformation(requestData.Patient, true); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update patient information on our databsae: "+err.Error())
		return
	}

	apiservice.WriteJSONSuccess(w)
}
