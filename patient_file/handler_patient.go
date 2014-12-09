package patient_file

import (
	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/surescripts"

	"net/http"
)

type doctorPatientHandler struct {
	dataAPI              api.DataAPI
	erxAPI               erx.ERxAPI
	addressValidationAPI address.AddressValidationAPI
}

func NewDoctorPatientHandler(
	dataAPI api.DataAPI,
	erxAPI erx.ERxAPI,
	addressValidationAPI address.AddressValidationAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&doctorPatientHandler{
			dataAPI:              dataAPI,
			erxAPI:               erxAPI,
			addressValidationAPI: addressValidationAPI,
		}), []string{"GET", "PUT"})
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
	PatientID int64           `schema:"patient_id,required" json:"-"`
}

func (d *doctorPatientHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	requestData := &requestResponstData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	doctor, err := d.dataAPI.GetDoctorFromAccountID(ctxt.AccountID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Doctor] = doctor

	patientID := requestData.PatientID
	if patientID == 0 {
		patientID = requestData.Patient.PatientID.Int64()
	}

	patient, err := d.dataAPI.GetPatientFromID(patientID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Patient] = patient

	if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method,
		ctxt.Role,
		doctor.DoctorID.Int64(),
		patient.PatientID.Int64(),
		d.dataAPI); err != nil {
		return false, err
	}
	return true, nil
}

func (d *doctorPatientHandler) getPatientInformation(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	patient := ctxt.RequestCache[apiservice.Patient].(*common.Patient)

	apiservice.WriteJSON(w, &requestResponstData{
		Patient: patient,
	})
}

func (d *doctorPatientHandler) updatePatientInformation(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*requestResponstData)
	currentDoctor := ctxt.RequestCache[apiservice.Doctor].(*common.Doctor)
	existingPatientInfo := ctxt.RequestCache[apiservice.Patient].(*common.Patient)

	err := surescripts.ValidatePatientInformation(requestData.Patient, d.addressValidationAPI, d.dataAPI)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	requestData.Patient.ERxPatientID = existingPatientInfo.ERxPatientID
	requestData.Patient.Pharmacy = existingPatientInfo.Pharmacy

	if err := d.erxAPI.UpdatePatientInformation(currentDoctor.DoseSpotClinicianID, requestData.Patient); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, `Unable to update patient information on dosespot. 
			Failing to avoid out of sync issues between surescripts and our platform `+err.Error())
		return
	}

	// update the doseSpot Id for the patient in our system now that we got one
	if !existingPatientInfo.ERxPatientID.IsValid {
		if err := d.dataAPI.UpdatePatientWithERxPatientID(requestData.Patient.PatientID.Int64(), requestData.Patient.ERxPatientID.Int64()); err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the patientId from dosespot: "+err.Error())
			return
		}
	}

	// go ahead and udpate the doctor's information in our system now that dosespot has it
	if err := d.dataAPI.UpdatePatientInformation(requestData.Patient, true); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update patient information on our databsae: "+err.Error())
		return
	}

	apiservice.WriteJSONSuccess(w)
}
