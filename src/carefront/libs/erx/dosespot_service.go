package erx

import (
	"carefront/common"
	pharmacySearch "carefront/libs/pharmacy"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
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
	sendMultiplPrescriptionsAction     = "SendMultiplePrescriptions"
	searchPharmaciesAction             = "PharmacySearchMessageDetailed"
	getPrescriptionLogDetailsAction    = "GetPrescriptionLogDetails"
	getMedicationListAction            = "GetMedicationList"
	resultOk                           = "OK"
)

func getDoseSpotClient() *soapClient {
	return &soapClient{SoapAPIEndPoint: doseSpotSOAPEndPoint, APIEndpoint: doseSpotAPIEndPoint}
}

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

	err := getDoseSpotClient().makeSoapRequest(medicationQuickSearchAction, medicationSearch, searchResult)

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
	err := getDoseSpotClient().makeSoapRequest(selfReportedMedicationSearchAction, selfReportedDrugsSearch, searchResult)

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
	err := getDoseSpotClient().makeSoapRequest(medicationStrengthSearchAction, medicationStrengthSearch, searchResult)

	if err != nil {
		return nil, err
	}

	return searchResult.DisplayStrengths, nil
}

func (d *DoseSpotService) SendMultiplePrescriptions(Patient *common.Patient, Treatments []*common.Treatment) ([]int64, error) {
	sendPrescriptionsRequest := &sendMultiplePrescriptionsRequest{}
	sendPrescriptionsRequest.SSO = generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId)
	sendPrescriptionsRequest.PatientId = int(Patient.ERxPatientId)

	prescriptionIds := make([]int, 0)
	prescriptionIdToTreatmentIdMapping := make(map[int64]int64)
	for _, treatment := range Treatments {
		prescriptionIds = append(prescriptionIds, int(treatment.PrescriptionId))
		prescriptionIdToTreatmentIdMapping[treatment.PrescriptionId] = treatment.Id
	}

	sendPrescriptionsRequest.PrescriptionIds = prescriptionIds

	response := &sendMultiplePrescriptionsResponse{}
	err := getDoseSpotClient().makeSoapRequest(sendMultiplPrescriptionsAction, sendPrescriptionsRequest, response)
	if err != nil {
		return nil, err
	}

	unSuccessfulTreatmentIds := make([]int64, 0)
	for _, prescriptionResult := range response.SendPrescriptionResults {
		if prescriptionResult.ResultCode != resultOk {
			unSuccessfulTreatmentIds = append(unSuccessfulTreatmentIds, prescriptionIdToTreatmentIdMapping[int64(prescriptionResult.PrescriptionId)])
		}
	}

	if response.ResultCode != resultOk {
		return nil, errors.New("Unable to send multiple prescriptions")
	}
	return unSuccessfulTreatmentIds, nil
}

func (d *DoseSpotService) StartPrescribingPatient(Patient *common.Patient, Treatments []*common.Treatment) error {

	newPatient := &patient{}
	newPatient.FirstName = Patient.FirstName
	newPatient.LastName = Patient.LastName
	newPatient.Address1 = Patient.PatientAddress.AddressLine1
	newPatient.City = Patient.City
	newPatient.State = Patient.State
	newPatient.ZipCode = Patient.ZipCode
	newPatient.DateOfBirth = specialDateTime{DateTime: Patient.Dob}
	newPatient.Gender = Patient.Gender
	newPatient.PrimaryPhone = Patient.Phone
	newPatient.PrimaryPhoneType = Patient.PhoneType

	if Patient.ERxPatientId != 0 {
		fmt.Println("Using erx patient id since it exists for patient: ")
		newPatient.PatientId = int(Patient.ERxPatientId)
	}

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
	err := getDoseSpotClient().makeSoapRequest(startPrescribingPatientAction, startPrescribingRequest, response)

	if err != nil {
		return err
	}

	if response.ResultCode != resultOk {
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
	err = getDoseSpotClient().makeSoapRequest(medicationSelectAction, medicationSelect, selectResult)

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

func (d *DoseSpotService) SearchForPharmacies(city, state, zipcode, name string, pharmacyTypes []string) ([]*pharmacySearch.PharmacyData, error) {
	searchRequest := &pharmacySearchRequest{}
	if city != "" {
		searchRequest.PharmacyCity = city
	}

	if state != "" {
		searchRequest.PharmacyStateTwoLetters = state
	}

	if zipcode != "" {
		searchRequest.PharmacyZipCode = zipcode
	}

	if name != "" {
		searchRequest.PharmacyNameSearch = name
	}

	if pharmacyTypes != nil && len(pharmacyTypes) > 0 {
		searchRequest.PharmacyTypes = pharmacyTypes
	}

	searchRequest.SSO = generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId)

	searchResponse := &pharmacySearchResult{}
	err := getDoseSpotClient().makeSoapRequest(searchPharmaciesAction, searchRequest, searchResponse)
	if err != nil {
		return nil, err
	}

	if searchResponse.ResultCode != resultOk {
		return nil, errors.New("Unable to search for pharmacies: " + searchResponse.ResultDescription)
	}

	pharmacies := make([]*pharmacySearch.PharmacyData, 0)
	for _, pharmacyResultItem := range searchResponse.Pharmacies {
		pharmacyData := &pharmacySearch.PharmacyData{}
		pharmacyData.Id = strconv.FormatInt(pharmacyResultItem.PharmacyId, 10)
		pharmacyData.Address = pharmacyResultItem.Address1 + " " + pharmacyResultItem.Address2
		pharmacyData.City = pharmacyResultItem.City
		pharmacyData.State = pharmacyResultItem.State
		pharmacyData.Name = pharmacyResultItem.StoreName
		pharmacyData.Fax = pharmacyResultItem.PrimaryFax
		pharmacyData.Phone = pharmacyResultItem.PrimaryPhone
		pharmacyData.Postal = pharmacyResultItem.ZipCode
		pharmacyData.Source = pharmacySearch.PHARMACY_SOURCE_SURESCRIPTS
		pharmacyData.PharmacyTypes = strings.Split(pharmacyResultItem.PharmacySpecialties, ", ")
		pharmacies = append(pharmacies, pharmacyData)
	}

	return pharmacies, nil
}

func (d *DoseSpotService) GetPrescriptionStatus(prescriptionId int64) ([]*PrescriptionLog, error) {
	request := &getPrescriptionLogDetailsRequest{}
	request.SSO = generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId)
	request.PrescriptionId = prescriptionId

	response := &getPrescriptionLogDetailsResult{}
	err := getDoseSpotClient().makeSoapRequest(getPrescriptionLogDetailsAction, request, response)
	if err != nil {
		return nil, err
	}

	prescriptionLogs := make([]*PrescriptionLog, 0)
	if response.Log != nil {
		for _, logDetails := range response.Log {
			prescriptionLog := &PrescriptionLog{}
			prescriptionLog.LogTimeStamp = logDetails.DateTimeStamp.DateTime
			prescriptionLog.PrescriptionStatus = logDetails.Status
			prescriptionLogs = append(prescriptionLogs, prescriptionLog)
		}
	}

	return prescriptionLogs, nil
}

func (d *DoseSpotService) GetMedicationList(PatientId int64) ([]*common.Treatment, error) {
	request := &getMedicationListRequest{}
	request.PatientId = PatientId
	request.SSO = generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId)
	request.Sources = []string{"Prescription"}
	request.Status = []string{"Active"}
	response := &getMedicationListResult{}
	err := getDoseSpotClient().makeSoapRequest(getMedicationListAction, request, response)
	if err != nil {
		return nil, err
	}

	treatments := make([]*common.Treatment, 0)
	for _, medicationItem := range response.Medications {
		treatment := &common.Treatment{}
		treatment.DrugInternalName = medicationItem.DisplayName
		treatment.ErxMedicationId = medicationItem.MedicationId
		treatment.PrescriptionStatus = medicationItem.PrescriptionStatus
		treatments = append(treatments, treatment)
	}
	return treatments, nil
}
