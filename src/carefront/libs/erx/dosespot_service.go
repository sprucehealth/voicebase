package erx

import (
	"carefront/common"
	"errors"
	"os"
	"strconv"
)

type DoseSpotService struct {
	ClinicId  string
	ClinicKey string
	UserId    string
}

const (
	doseSpotAPIEndPoint                = "http://www.dosespot.com/API/11/"
	doseSpotSOAPEndPoint               = "http://i.dosespot.com/api/11/apifull.asmx"
	medicationQuickSearchAction        = "MedicationQuickSearchMessage"
	selfReportedMedicationSearchAction = "SelfReportedMedicationSearch"
	medicationStrengthSearchAction     = "MedicationStrengthSearchMessage"
	medicationSelectAction             = "MedicationSelectMessage"
	startPrescribingPatientAction      = "PatientStartPrescribingMessage"
)

var (
	doseSpotClient = soapClient{SoapAPIEndPoint: doseSpotSOAPEndPoint, APIEndpoint: doseSpotAPIEndPoint}
)

func NewDoseSpotService(clinicId, clinicKey, userId string) *DoseSpotService {
	d := &DoseSpotService{}
	if clinicId == "" {
		d.ClinicKey = os.Getenv("DOSESPOT_CLINIC_KEY")
		d.UserId = os.Getenv("DOSESPOT_USER_ID")
		d.ClinicId = os.Getenv("DOSESPOT_CLINIC_ID")
	} else {
		d.ClinicKey = clinicKey
		d.ClinicId = clinicId
		d.UserId = userId
	}
	return d
}

func (d *DoseSpotService) GetDrugNamesForDoctor(prefix string) ([]string, error) {
	medicationSearch := &medicationQuickSearchRequest{}
	medicationSearch.SSO = generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId)
	medicationSearch.SearchString = prefix

	searchResult := &medicationQuickSearchResponse{}
	err := doseSpotClient.makeSoapRequest(medicationQuickSearchAction, medicationSearch, searchResult)

	if err != nil {
		return nil, err
	}

	return searchResult.DisplayNames, nil
}

func (d *DoseSpotService) GetDrugNamesForPatient(prefix string) ([]string, error) {
	selfReportedDrugsSearch := &selfReportedMedicationSearchRequest{}
	selfReportedDrugsSearch.SSO = generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId)
	selfReportedDrugsSearch.SearchTerm = prefix

	searchResult := &selfReportedMedicationSearchResponse{}
	err := doseSpotClient.makeSoapRequest(selfReportedMedicationSearchAction, selfReportedDrugsSearch, searchResult)

	if err != nil {
		return nil, err
	}

	drugNames := make([]string, len(searchResult.SearchResults))
	for i, searchResultItem := range searchResult.SearchResults {
		drugNames[i] = searchResultItem.DisplayName
	}

	return drugNames, nil
}

func (d *DoseSpotService) SearchForMedicationStrength(medicationName string) ([]string, error) {
	medicationStrengthSearch := &medicationStrengthSearchRequest{}
	medicationStrengthSearch.SSO = generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId)
	medicationStrengthSearch.MedicationName = medicationName

	searchResult := &medicationStrengthSearchResponse{}
	err := doseSpotClient.makeSoapRequest(medicationStrengthSearchAction, medicationStrengthSearch, searchResult)

	if err != nil {
		return nil, err
	}

	return searchResult.DisplayStrengths, nil
}

func (d *DoseSpotService) StartPrescribingPatient(Patient *common.Patient, Treatments []*common.Treatment) error {

	newPatient := &patient{}
	newPatient.FirstName = Patient.FirstName
	newPatient.LastName = Patient.LastName
	newPatient.Address1 = Patient.PatientAddress.AddressLine1
	newPatient.City = Patient.City
	newPatient.State = Patient.State
	newPatient.ZipCode = Patient.ZipCode
	newPatient.DateOfBirth = DateOfBirthType{DateOfBirth: Patient.Dob}
	newPatient.Gender = Patient.Gender
	newPatient.PrimaryPhone = Patient.Phone
	newPatient.PrimaryPhoneType = Patient.PhoneType

	patientPreferredPharmacy := &patientPharmacySelection{}
	patientPreferredPharmacy.IsPrimary = true

	pharmacyId, _ := strconv.Atoi(Patient.Pharmacy.Id)
	patientPreferredPharmacy.PharmacyId = pharmacyId

	prescriptions := make([]*prescription, 0)

	for _, treatment := range Treatments {
		prescriptionMedication := &medication{}
		prescriptionMedication.DaysSupply = int(treatment.DaysSupply)
		lexiDrugSynIdInt, _ := strconv.Atoi(treatment.DrugDBIds[LexiDrugSynId])
		prescriptionMedication.LexiDrugSynId = lexiDrugSynIdInt

		lexiGenProductIdInt, _ := strconv.Atoi(treatment.DrugDBIds[LexiGenProductId])
		prescriptionMedication.LexiGenProductId = lexiGenProductIdInt

		lexiSynonymTypeIdInt, _ := strconv.Atoi(treatment.DrugDBIds[LexiSynonymTypeId])
		prescriptionMedication.LexiSynonymTypeId = lexiSynonymTypeIdInt

		prescriptionMedication.Refills = int(treatment.NumberRefills)
		prescriptionMedication.Dispense = strconv.FormatInt(treatment.DispenseValue, 10)
		prescriptionMedication.DispenseUnitId = int(treatment.DispenseUnitId)
		prescriptionMedication.Instructions = treatment.PatientInstructions
		prescriptionMedication.NoSubstitutions = treatment.SubstitutionsAllowed
		prescriptionMedication.PharmacyNotes = treatment.PharmacyNotes
		prescriptionMedication.PharmacyId = pharmacyId

		patientPrescription := &prescription{}
		patientPrescription.Medication = prescriptionMedication
		prescriptions = append(prescriptions, patientPrescription)
	}

	startPrescribingRequest := &patientStartPrescribingRequest{}
	startPrescribingRequest.AddFavoritePharmacies = []*patientPharmacySelection{patientPreferredPharmacy}
	startPrescribingRequest.AddPrescriptions = prescriptions
	startPrescribingRequest.Patient = newPatient
	startPrescribingRequest.SSO = generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId)

	response := &patientStartPrescribingResponse{}
	err := doseSpotClient.makeSoapRequest(startPrescribingPatientAction, startPrescribingRequest, response)

	if err != nil {
		return err
	}

	if response.ResultCode != "OK" {
		return errors.New("Something went wrong when attempting to start prescriptions for patient: " + response.ResultDescription)
	}

	// populate the prescription id into the patient object
	Patient.ERxPatientId = int64(response.PatientUpdates[0].Patient.PatientId)

	// go through and assign medication ids to all prescriptions
	for _, patientUpdate := range response.PatientUpdates {
		for _, medication := range patientUpdate.Medications {
			for _, treatment := range Treatments {
				LexiDrugSynIdInt, _ := strconv.Atoi(treatment.DrugDBIds[LexiDrugSynId])
				LexiGenProductIdInt, _ := strconv.Atoi(treatment.DrugDBIds[LexiGenProductId])
				LexiSynonymTypeIdInt, _ := strconv.Atoi(treatment.DrugDBIds[LexiSynonymTypeId])
				if medication.LexiDrugSynId == LexiDrugSynIdInt &&
					medication.LexiGenProductId == LexiGenProductIdInt &&
					medication.LexiSynonymTypeId == LexiSynonymTypeIdInt {
					treatment.PrescriptionId = int64(medication.DoseSpotPrescriptionId)
					break
				}
			}
		}
	}

	return err
}

func (d *DoseSpotService) SelectMedication(medicationName, medicationStrength string) (medication *Medication, err error) {
	medicationSelect := &medicationSelectRequest{}
	medicationSelect.SSO = generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId)
	medicationSelect.MedicationName = medicationName
	medicationSelect.MedicationStrength = medicationStrength

	selectResult := &medicationSelectResponse{}
	err = doseSpotClient.makeSoapRequest(medicationSelectAction, medicationSelect, selectResult)

	if err != nil {
		return nil, err
	}

	medication = &Medication{}
	medication.DrugDBIds = make(map[string]string)
	medication.DrugDBIds[LexiGenProductId] = strconv.Itoa(selectResult.LexiGenProductId)
	medication.DrugDBIds[LexiDrugSynId] = strconv.Itoa(selectResult.LexiDrugSynId)
	medication.DrugDBIds[LexiSynonymTypeId] = strconv.Itoa(selectResult.LexiSynonymTypeId)
	medication.DispenseUnitId = selectResult.DispenseUnitId
	medication.DispenseUnitDescription = selectResult.DispenseUnitDescription
	medication.OTC = selectResult.OTC
	return medication, err
}
