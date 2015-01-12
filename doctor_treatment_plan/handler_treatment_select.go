package doctor_treatment_plan

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
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
		apiservice.WriteJSON(w, &NewTreatmentResponse{})
		return
	}

	var scheduleInt int
	if medication.Schedule == "" {
		scheduleInt = 0
	} else {
		scheduleInt, err = strconv.Atoi(medication.Schedule)
		if err != nil {
			scheduleInt = 0
		}
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
		DrugInternalName:        requestData.MedicationName,
		OTC:                     medication.OTC,
		SubstitutionsAllowed:    true, // defaulting to substitutions being allowed as required by surescripts
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 0,
		},
		IsControlledSubstance: scheduleInt > 0,
	}
	treatment.DrugName, treatment.DrugForm, treatment.DrugRoute = apiservice.BreakDrugInternalNameIntoComponents(requestData.MedicationName)

	treatment.GenericDrugName, err = erx.ParseGenericName(medication)
	if err != nil {
		golog.Errorf("Failed to parse generic drug name '%s': %s", medication.GenericProductName, err.Error())
	}

	if treatment.IsControlledSubstance {
		apiservice.WriteUserError(w, apiservice.StatusUnprocessableEntity, "Unfortunately, we do not support electronic routing of controlled substances using the platform. If you have any questions, feel free to contact support. Apologies for any inconvenience!")
		return
	}

	// store the drug description so that we are able to look it up
	// and use it as source of authority to describe a treatment that a
	// doctor adds to the treatment plan
	if err := m.dataAPI.SetDrugDescription(&api.DrugDescription{
		InternalName:            treatment.DrugInternalName,
		DosageStrength:          requestData.MedicationStrength,
		DrugDBIDs:               treatment.DrugDBIDs,
		DispenseUnitID:          treatment.DispenseUnitID.Int64(),
		DispenseUnitDescription: treatment.DispenseUnitDescription,
		OTC:             treatment.OTC,
		Schedule:        scheduleInt,
		DrugName:        treatment.DrugName,
		DrugForm:        treatment.DrugForm,
		DrugRoute:       treatment.DrugRoute,
		GenericDrugName: treatment.GenericDrugName,
	}); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	newTreatmentResponse := &NewTreatmentResponse{
		Treatment: treatment,
	}
	apiservice.WriteJSON(w, newTreatmentResponse)
}
