package erx

import (
	"carefront/common"
	pharmacySearch "carefront/libs/pharmacy"
	"errors"
	"fmt"
	"github.com/samuel/go-metrics/metrics"
	"os"
	"strconv"
	"strings"
)

type DoseSpotService struct {
	ClinicId     string
	ClinicKey    string
	UserId       string
	apiLatencies map[DoseSpotApiId]metrics.Histogram
	apiRequests  map[DoseSpotApiId]metrics.Counter
	apiFailure   map[DoseSpotApiId]metrics.Counter
}

type DoseSpotApiId int

const (
	medicationQuickSearchAction DoseSpotApiId = iota
	selfReportedMedicationSearchAction
	medicationStrengthSearchAction
	medicationSelectAction
	startPrescribingPatientAction
	sendMultiplPrescriptionsAction
	searchPharmaciesAction
	getPrescriptionLogDetailsAction
	getMedicationListAction
	getTransmissionErrorDetailsAction
)

var DoseSpotApiActions = map[DoseSpotApiId]string{
	medicationQuickSearchAction:        "MedicationQuickSearchMessage",
	selfReportedMedicationSearchAction: "SelfReportedMedicationSearch",
	medicationStrengthSearchAction:     "MedicationStrengthSearchMessage",
	medicationSelectAction:             "MedicationSelectMessage",
	startPrescribingPatientAction:      "PatientStartPrescribingMessage",
	sendMultiplPrescriptionsAction:     "SendMultiplePrescriptions",
	searchPharmaciesAction:             "PharmacySearchMessageDetailed",
	getPrescriptionLogDetailsAction:    "GetPrescriptionLogDetails",
	getMedicationListAction:            "GetMedicationList",
	getTransmissionErrorDetailsAction:  "GetTransmissionErrorsDetails",
}

const (
	doseSpotAPIEndPoint  = "http://www.dosespot.com/API/11/"
	doseSpotSOAPEndPoint = "http://i.dosespot.com/api/11/apifull.asmx"
	resultOk             = "OK"
)

func getDoseSpotClient() *soapClient {
	return &soapClient{SoapAPIEndPoint: doseSpotSOAPEndPoint, APIEndpoint: doseSpotAPIEndPoint}
}

func NewDoseSpotService(clinicId, clinicKey, userId string, statsRegistry metrics.Registry) *DoseSpotService {
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

	d.apiLatencies = make(map[DoseSpotApiId]metrics.Histogram)
	d.apiRequests = make(map[DoseSpotApiId]metrics.Counter)
	d.apiFailure = make(map[DoseSpotApiId]metrics.Counter)
	for apiActionId, apiAction := range DoseSpotApiActions {
		d.apiLatencies[apiActionId] = metrics.NewBiasedHistogram()
		d.apiRequests[apiActionId] = metrics.NewCounter()
		d.apiFailure[apiActionId] = metrics.NewCounter()
		if statsRegistry != nil {
			statsRegistry.Add(fmt.Sprintf("requests/latency/%s", apiAction), d.apiLatencies[apiActionId])
			statsRegistry.Add(fmt.Sprintf("requests/total/%s", apiAction), d.apiRequests[apiActionId])
			statsRegistry.Add(fmt.Sprintf("requests/failed/%s", apiAction), d.apiFailure[apiActionId])
		}
	}

	return d
}

func (d *DoseSpotService) GetDrugNamesForDoctor(prefix string) ([]string, error) {
	medicationSearch := &medicationQuickSearchRequest{
		SSO:          generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId),
		SearchString: prefix,
	}

	searchResult := &medicationQuickSearchResponse{}

	err := getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[medicationQuickSearchAction],
		medicationSearch, searchResult,
		d.apiLatencies[medicationQuickSearchAction],
		d.apiRequests[medicationQuickSearchAction],
		d.apiFailure[medicationQuickSearchAction])

	if err != nil {
		return nil, err
	}

	return searchResult.DisplayNames, nil
}

func (d *DoseSpotService) GetDrugNamesForPatient(prefix string) ([]string, error) {
	selfReportedDrugsSearch := &selfReportedMedicationSearchRequest{
		SSO:        generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId),
		SearchTerm: prefix,
	}

	searchResult := &selfReportedMedicationSearchResponse{}
	err := getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[selfReportedMedicationSearchAction],
		selfReportedDrugsSearch, searchResult,
		d.apiLatencies[selfReportedMedicationSearchAction],
		d.apiRequests[selfReportedMedicationSearchAction],
		d.apiFailure[selfReportedMedicationSearchAction])

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
	medicationStrengthSearch := &medicationStrengthSearchRequest{
		SSO:            generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId),
		MedicationName: medicationName,
	}

	searchResult := &medicationStrengthSearchResponse{}
	err := getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[medicationStrengthSearchAction],
		medicationStrengthSearch, searchResult,
		d.apiLatencies[medicationStrengthSearchAction],
		d.apiRequests[medicationStrengthSearchAction],
		d.apiFailure[medicationStrengthSearchAction])

	if err != nil {
		return nil, err
	}

	return searchResult.DisplayStrengths, nil
}

func (d *DoseSpotService) SendMultiplePrescriptions(patient *common.Patient, treatments []*common.Treatment) ([]int64, error) {
	sendPrescriptionsRequest := &sendMultiplePrescriptionsRequest{
		SSO:       generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId),
		PatientId: int(patient.PatientId),
	}

	prescriptionIds := make([]int, 0)
	prescriptionIdToTreatmentIdMapping := make(map[int64]int64)
	for _, treatment := range treatments {
		prescriptionIds = append(prescriptionIds, int(treatment.PrescriptionId))
		prescriptionIdToTreatmentIdMapping[treatment.PrescriptionId] = treatment.Id
	}

	sendPrescriptionsRequest.PrescriptionIds = prescriptionIds

	response := &sendMultiplePrescriptionsResponse{}
	err := getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[sendMultiplPrescriptionsAction],
		sendPrescriptionsRequest, response,
		d.apiLatencies[sendMultiplPrescriptionsAction],
		d.apiRequests[sendMultiplPrescriptionsAction],
		d.apiFailure[sendMultiplPrescriptionsAction])

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

func (d *DoseSpotService) StartPrescribingPatient(currentPatient *common.Patient, treatments []*common.Treatment) error {

	newPatient := &patient{
		FirstName:        currentPatient.FirstName,
		LastName:         currentPatient.LastName,
		Address1:         currentPatient.PatientAddress.AddressLine1,
		City:             currentPatient.City,
		State:            currentPatient.State,
		ZipCode:          currentPatient.ZipCode,
		DateOfBirth:      specialDateTime{DateTime: currentPatient.Dob, DateTimeElementName: "DateOfBirth"},
		Gender:           currentPatient.Gender,
		PrimaryPhone:     currentPatient.Phone,
		PrimaryPhoneType: currentPatient.PhoneType,
	}

	if currentPatient.ERxPatientId != 0 {
		fmt.Println("Using erx patient id since it exists for patient: ")
		newPatient.PatientId = int(currentPatient.ERxPatientId)
	}

	patientPreferredPharmacy := &patientPharmacySelection{}
	patientPreferredPharmacy.IsPrimary = true

	pharmacyId, _ := strconv.Atoi(currentPatient.Pharmacy.Id)
	patientPreferredPharmacy.PharmacyId = pharmacyId

	prescriptions := make([]*prescription, 0)

	for _, treatment := range treatments {
		lexiDrugSynIdInt, _ := strconv.Atoi(treatment.DrugDBIds[LexiDrugSynId])
		lexiGenProductIdInt, _ := strconv.Atoi(treatment.DrugDBIds[LexiGenProductId])
		lexiSynonymTypeIdInt, _ := strconv.Atoi(treatment.DrugDBIds[LexiSynonymTypeId])

		prescriptionMedication := &medication{
			DaysSupply:        int(treatment.DaysSupply),
			LexiDrugSynId:     lexiDrugSynIdInt,
			LexiGenProductId:  lexiGenProductIdInt,
			LexiSynonymTypeId: lexiSynonymTypeIdInt,
			Refills:           int(treatment.NumberRefills),
			Dispense:          strconv.FormatInt(treatment.DispenseValue, 10),
			DispenseUnitId:    int(treatment.DispenseUnitId),
			Instructions:      treatment.PatientInstructions,
			NoSubstitutions:   treatment.SubstitutionsAllowed,
			PharmacyNotes:     treatment.PharmacyNotes,
			PharmacyId:        pharmacyId,
		}

		patientPrescription := &prescription{Medication: prescriptionMedication}
		prescriptions = append(prescriptions, patientPrescription)
	}

	startPrescribingRequest := &patientStartPrescribingRequest{
		AddFavoritePharmacies: []*patientPharmacySelection{patientPreferredPharmacy},
		AddPrescriptions:      prescriptions,
		Patient:               newPatient,
		SSO:                   generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId),
	}

	response := &patientStartPrescribingResponse{}
	err := getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[startPrescribingPatientAction],
		startPrescribingRequest, response,
		d.apiLatencies[startPrescribingPatientAction],
		d.apiRequests[startPrescribingPatientAction],
		d.apiFailure[startPrescribingPatientAction])
	if err != nil {
		return err
	}

	if response.ResultCode != resultOk {
		return errors.New("Something went wrong when attempting to start prescriptions for patient: " + response.ResultDescription)
	}

	// populate the prescription id into the patient object
	currentPatient.ERxPatientId = int64(response.PatientUpdates[0].Patient.PatientId)

	// go through and assign medication ids to all prescriptions
	for _, patientUpdate := range response.PatientUpdates {
		for _, medication := range patientUpdate.Medications {
			for _, treatment := range treatments {
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
	medicationSelect := &medicationSelectRequest{
		SSO:                generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId),
		MedicationName:     medicationName,
		MedicationStrength: medicationStrength,
	}

	selectResult := &medicationSelectResponse{}
	err = getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[medicationSelectAction],
		medicationSelect, selectResult,
		d.apiLatencies[medicationSelectAction],
		d.apiRequests[medicationSelectAction],
		d.apiFailure[medicationSelectAction])
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
	searchRequest := &pharmacySearchRequest{
		PharmacyCity:            city,
		PharmacyStateTwoLetters: state,
		PharmacyZipCode:         zipcode,
		PharmacyNameSearch:      name,
		PharmacyTypes:           pharmacyTypes,
		SSO:                     generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId),
	}

	searchResponse := &pharmacySearchResult{}
	err := getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[searchPharmaciesAction],
		searchRequest, searchResponse,
		d.apiLatencies[searchPharmaciesAction],
		d.apiRequests[searchPharmaciesAction],
		d.apiFailure[searchPharmaciesAction])
	if err != nil {
		return nil, err
	}

	if searchResponse.ResultCode != resultOk {
		return nil, errors.New("Unable to search for pharmacies: " + searchResponse.ResultDescription)
	}

	pharmacies := make([]*pharmacySearch.PharmacyData, 0)
	for _, pharmacyResultItem := range searchResponse.Pharmacies {
		pharmacyData := &pharmacySearch.PharmacyData{
			Id:            strconv.FormatInt(pharmacyResultItem.PharmacyId, 10),
			Address:       pharmacyResultItem.Address1 + " " + pharmacyResultItem.Address2,
			City:          pharmacyResultItem.City,
			State:         pharmacyResultItem.State,
			Name:          pharmacyResultItem.StoreName,
			Fax:           pharmacyResultItem.PrimaryFax,
			Phone:         pharmacyResultItem.PrimaryPhone,
			Postal:        pharmacyResultItem.ZipCode,
			Source:        pharmacySearch.PHARMACY_SOURCE_SURESCRIPTS,
			PharmacyTypes: strings.Split(pharmacyResultItem.PharmacySpecialties, ", "),
		}

		pharmacies = append(pharmacies, pharmacyData)
	}

	return pharmacies, nil
}

func (d *DoseSpotService) GetPrescriptionStatus(prescriptionId int64) ([]*PrescriptionLog, error) {
	request := &getPrescriptionLogDetailsRequest{
		SSO:            generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId),
		PrescriptionId: prescriptionId,
	}

	response := &getPrescriptionLogDetailsResult{}
	err := getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[getPrescriptionLogDetailsAction],
		request, response,
		d.apiLatencies[getPrescriptionLogDetailsAction],
		d.apiRequests[getPrescriptionLogDetailsAction],
		d.apiFailure[getPrescriptionLogDetailsAction])
	if err != nil {
		return nil, err
	}

	prescriptionLogs := make([]*PrescriptionLog, 0)
	if response.Log != nil {
		for _, logDetails := range response.Log {
			prescriptionLog := &PrescriptionLog{
				LogTimeStamp:       logDetails.DateTimeStamp.DateTime,
				PrescriptionStatus: logDetails.Status,
			}
			prescriptionLogs = append(prescriptionLogs, prescriptionLog)
		}
	}

	return prescriptionLogs, nil
}

func (d *DoseSpotService) GetMedicationList(PatientId int64) ([]*Medication, error) {
	request := &getMedicationListRequest{
		PatientId: PatientId,
		SSO:       generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId),
		Sources:   []string{"Prescription"},
		Status:    []string{"Active"},
	}
	response := &getMedicationListResult{}
	err := getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[getMedicationListAction],
		request, response,
		d.apiLatencies[getMedicationListAction],
		d.apiRequests[getMedicationListAction],
		d.apiFailure[getMedicationListAction])
	if err != nil {
		return nil, err
	}

	medications := make([]*Medication, 0)
	for _, medicationItem := range response.Medications {
		medication := &Medication{
			ErxMedicationId:    medicationItem.MedicationId,
			PrescriptionStatus: medicationItem.PrescriptionStatus,
		}
		medications = append(medications, medication)
	}
	return medications, nil
}

func (d *DoseSpotService) GetTransmissionErrorDetails() error {
	request := &getTransmissionErrorDetailsRequest{
		SSO: generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId),
	}
	response := &getTransmissionErrorDetailsResponse{}
	err := getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[getTransmissionErrorDetailsAction],
		request, response,
		d.apiLatencies[getTransmissionErrorDetailsAction],
		d.apiRequests[getTransmissionErrorDetailsAction],
		d.apiFailure[getTransmissionErrorDetailsAction])
	if err != nil {
		return err
	}

	for _, detailsItem := range response.TransmissionErrors {
		fmt.Println(detailsItem.ErrorDetails)
	}
	return nil
}
