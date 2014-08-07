package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/surescripts"
)

type selectHandler struct {
	dataAPI api.DataAPI
	erxAPI  erx.ERxAPI
}

func NewMedicationSelectHandler(dataAPI api.DataAPI, erxAPI erx.ERxAPI) http.Handler {
	return &selectHandler{
		dataAPI: dataAPI,
		erxAPI:  erxAPI,
	}
}

type NewTreatmentRequestData struct {
	MedicationName     string `schema:"drug_internal_name,required"`
	MedicationStrength string `schema:"medication_strength,required"`
}

type NewTreatmentResponse struct {
	Treatment *common.Treatment `json:"treatment"`
}

func (m *selectHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_GET {
		return false, apiservice.NewResourceNotFoundError("", r)
	}

	if apiservice.GetContext(r).Role != api.DOCTOR_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (m *selectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestData := new(NewTreatmentRequestData)
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	if (len(requestData.MedicationName) + len(requestData.MedicationStrength)) > surescripts.MaxMedicationDescriptionLength {
		apiservice.WriteUserError(w, apiservice.StatusUnprocessableEntity, "Any medication name + dosage strength longer than 105 characters cannot be sent electronically and instead must be called in. Please call in this prescription to the patient's preferred pharmacy if you would like to route it.")
		return
	}

	doctor, err := m.dataAPI.GetDoctorFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	medication, err := m.erxAPI.SelectMedication(doctor.DoseSpotClinicianId, requestData.MedicationName, requestData.MedicationStrength)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if medication == nil {
		apiservice.WriteJSON(w, &NewTreatmentResponse{})
		return
	}

	medication.DrugName, medication.DrugForm, medication.DrugRoute = apiservice.BreakDrugInternalNameIntoComponents(requestData.MedicationName)

	if medication.IsControlledSubstance {
		apiservice.WriteUserError(w, apiservice.StatusUnprocessableEntity, "Unfortunately, we do not support electronic routing of controlled substances using the platform. If you have any questions, feel free to contact support. Apologies for any inconvenience!")
		return
	}

	newTreatmentResponse := &NewTreatmentResponse{
		Treatment: medication,
	}
	apiservice.WriteJSON(w, newTreatmentResponse)
}
