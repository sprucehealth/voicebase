package doctor_treatment_plan

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/surescripts"
)

type selectHandler struct {
	dataAPI api.DataAPI
	erxAPI  erx.ERxAPI
}

func NewMedicationSelectHandler(dataAPI api.DataAPI, erxAPI erx.ERxAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&selectHandler{
			dataAPI: dataAPI,
			erxAPI:  erxAPI,
		}), []string{"GET"})
}

type NewTreatmentRequestData struct {
	MedicationName     string `schema:"drug_internal_name,required"`
	MedicationStrength string `schema:"medication_strength,required"`
}

type NewTreatmentResponse struct {
	Treatment *common.Treatment `json:"treatment"`
}

func (m *selectHandler) IsAuthorized(r *http.Request) (bool, error) {
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

	doctor, err := m.dataAPI.GetDoctorFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	medication, err := m.erxAPI.SelectMedication(doctor.DoseSpotClinicianID, requestData.MedicationName, requestData.MedicationStrength)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if medication == nil {
		httputil.JSONResponse(w, http.StatusOK, &NewTreatmentResponse{})
		return
	}

	// starting refills at 0 because we default to 0 even when doctor
	// does not enter something
	treatment := &common.Treatment{
		DrugDBIDs: map[string]string{
			erx.LexiGenProductID:  strconv.FormatInt(medication.LexiGenProductID, 10),
			erx.LexiDrugSynID:     strconv.FormatInt(medication.LexiDrugSynID, 10),
			erx.LexiSynonymTypeID: strconv.FormatInt(medication.LexiSynonymTypeID, 10),
			erx.NDC:               medication.RepresentativeNDC,
		},
		DispenseUnitID:          encoding.NewObjectID(medication.DispenseUnitID),
		DispenseUnitDescription: medication.DispenseUnitDescription,
		DosageStrength:          requestData.MedicationStrength,
		DrugInternalName:        requestData.MedicationName,
		OTC:                     medication.OTC,
		SubstitutionsAllowed:    true, // defaulting to substitutions being allowed as required by surescripts
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 0,
		},
	}

	description := createDrugDescription(treatment, medication)
	treatment.DrugName = description.DrugName
	treatment.DrugForm = description.DrugForm
	treatment.DrugRoute = description.DrugRoute
	treatment.IsControlledSubstance = description.Schedule > 0
	treatment.GenericDrugName = description.GenericDrugName

	if treatment.IsControlledSubstance {
		apiservice.WriteUserError(w, apiservice.StatusUnprocessableEntity, "Unfortunately, we do not support electronic routing of controlled substances using the platform. If you have any questions, feel free to contact support. Apologies for any inconvenience!")
		return
	}

	// store the drug description so that we are able to look it up
	// and use it as source of authority to describe a treatment that a
	// doctor adds to the treatment plan
	if err := m.dataAPI.SetDrugDescription(description); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	newTreatmentResponse := &NewTreatmentResponse{
		Treatment: treatment,
	}
	httputil.JSONResponse(w, http.StatusOK, newTreatmentResponse)
}
