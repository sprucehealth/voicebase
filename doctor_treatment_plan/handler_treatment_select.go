package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/surescripts"
)

type selectHandler struct {
	dataAPI api.DataAPI
	erxAPI  erx.ERxAPI
}

func NewMedicationSelectHandler(dataAPI api.DataAPI, erxAPI erx.ERxAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&selectHandler{
			dataAPI: dataAPI,
			erxAPI:  erxAPI,
		}), httputil.Get)
}

type NewTreatmentRequestData struct {
	MedicationName     string `schema:"drug_internal_name,required"`
	MedicationStrength string `schema:"medication_strength,required"`
}

type NewTreatmentResponse struct {
	Treatment *common.Treatment `json:"treatment"`
}

func (m *selectHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	if apiservice.MustCtxAccount(ctx).Role != api.RoleDoctor {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (m *selectHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestData := new(NewTreatmentRequestData)
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	}

	if (len(requestData.MedicationName) + len(requestData.MedicationStrength)) > surescripts.MaxMedicationDescriptionLength {
		apiservice.WriteUserError(w, apiservice.StatusUnprocessableEntity, "Any medication name + dosage strength longer than 105 characters cannot be sent electronically and instead must be called in. Please call in this prescription to the patient's preferred pharmacy if you would like to route it.")
		return
	}

	doctor, err := m.dataAPI.GetDoctorFromAccountID(apiservice.MustCtxAccount(ctx).ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	medication, err := m.erxAPI.SelectMedication(doctor.DoseSpotClinicianID, requestData.MedicationName, requestData.MedicationStrength)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	if medication == nil {
		httputil.JSONResponse(w, http.StatusOK, &NewTreatmentResponse{})
		return
	}

	treatment, description := CreateTreatmentFromMedication(medication, requestData.MedicationStrength, requestData.MedicationName)

	if treatment.IsControlledSubstance {
		apiservice.WriteUserError(w, apiservice.StatusUnprocessableEntity, "Unfortunately, we do not support electronic routing of controlled substances using the platform. If you have any questions, feel free to contact support. Apologies for any inconvenience!")
		return
	}

	// store the drug description so that we are able to look it up
	// and use it as source of authority to describe a treatment that a
	// doctor adds to the treatment plan
	if err := m.dataAPI.SetDrugDescription(description); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	newTreatmentResponse := &NewTreatmentResponse{
		Treatment: treatment,
	}
	httputil.JSONResponse(w, http.StatusOK, newTreatmentResponse)
}
