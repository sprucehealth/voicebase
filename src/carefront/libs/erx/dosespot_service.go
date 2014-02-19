package erx

import (
	"carefront/common"
	"carefront/libs/golog"
	pharmacySearch "carefront/libs/pharmacy"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/samuel/go-metrics/metrics"
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
	getRefillRequestsTransmissionsErrorsAction
	ignoreAlertAction
)

var DoseSpotApiActions = map[DoseSpotApiId]string{
	medicationQuickSearchAction:                "MedicationQuickSearchMessage",
	selfReportedMedicationSearchAction:         "SelfReportedMedicationSearch",
	medicationStrengthSearchAction:             "MedicationStrengthSearchMessage",
	medicationSelectAction:                     "MedicationSelectMessage",
	startPrescribingPatientAction:              "PatientStartPrescribingMessage",
	sendMultiplPrescriptionsAction:             "SendMultiplePrescriptions",
	searchPharmaciesAction:                     "PharmacySearchMessageDetailed",
	getPrescriptionLogDetailsAction:            "GetPrescriptionLogDetails",
	getMedicationListAction:                    "GetMedicationList",
	getTransmissionErrorDetailsAction:          "GetTransmissionErrorsDetails",
	getRefillRequestsTransmissionsErrorsAction: "GetRefillRequestsTransmissionErrors",
	ignoreAlertAction:                          "IgnoreAlert",
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
		PatientId: patient.ERxPatientId.Int64(),
	}

	prescriptionIds := make([]int64, 0)
	prescriptionIdToTreatmentIdMapping := make(map[int64]int64)
	for _, treatment := range treatments {
		if treatment.PrescriptionId.Int64() == 0 {
			continue
		}
		prescriptionIds = append(prescriptionIds, treatment.PrescriptionId.Int64())
		prescriptionIdToTreatmentIdMapping[treatment.PrescriptionId.Int64()] = treatment.Id.Int64()
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
			golog.Errorf("Error sending prescription with id %d : %s", prescriptionResult.PrescriptionId, prescriptionResult.ResultDescription)
		}
	}

	if response.ResultCode != resultOk {
		return nil, errors.New("Unable to send multiple prescriptions: " + response.ResultDescription)
	}
	return unSuccessfulTreatmentIds, nil
}

func (d *DoseSpotService) StartPrescribingPatient(currentPatient *common.Patient, treatments []*common.Treatment) error {

	newPatient := &patient{
		PatientId:        currentPatient.ERxPatientId.Int64(),
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

	if currentPatient.ERxPatientId.Int64() != 0 {
		newPatient.PatientId = currentPatient.ERxPatientId.Int64()
	}

	patientPreferredPharmacy := &patientPharmacySelection{}
	patientPreferredPharmacy.IsPrimary = true

	pharmacyId, err := strconv.ParseInt(currentPatient.Pharmacy.Id, 0, 64)
	if err != nil {
		return fmt.Errorf("Unable to parse the pharmacy id: %s", err.Error())
	}

	patientPreferredPharmacy.PharmacyId = pharmacyId

	prescriptions := make([]*prescription, 0)

	for _, treatment := range treatments {
		lexiDrugSynIdInt, _ := strconv.ParseInt(treatment.DrugDBIds[LexiDrugSynId], 0, 64)
		lexiGenProductIdInt, _ := strconv.ParseInt(treatment.DrugDBIds[LexiGenProductId], 0, 64)
		lexiSynonymTypeIdInt, _ := strconv.ParseInt(treatment.DrugDBIds[LexiSynonymTypeId], 0, 64)

		daysSupply := nullInt64(treatment.DaysSupply)
		prescriptionMedication := &medication{
			DaysSupply:        daysSupply,
			LexiDrugSynId:     lexiDrugSynIdInt,
			LexiGenProductId:  lexiGenProductIdInt,
			LexiSynonymTypeId: lexiSynonymTypeIdInt,
			Refills:           nullInt64(treatment.NumberRefills),
			Dispense:          strconv.FormatInt(treatment.DispenseValue, 10),
			DispenseUnitId:    treatment.DispenseUnitId.Int64(),
			Instructions:      treatment.PatientInstructions,
			NoSubstitutions:   !treatment.SubstitutionsAllowed,
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
	err = getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[startPrescribingPatientAction],
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
	currentPatient.ERxPatientId = common.NewObjectId(response.PatientUpdates[0].Patient.PatientId)

	// go through and assign medication ids to all prescriptions
	for _, patientUpdate := range response.PatientUpdates {
		for _, medication := range patientUpdate.Medications {
			for _, treatment := range treatments {
				LexiDrugSynIdInt, _ := strconv.ParseInt(treatment.DrugDBIds[LexiDrugSynId], 0, 64)
				LexiGenProductIdInt, _ := strconv.ParseInt(treatment.DrugDBIds[LexiGenProductId], 0, 64)
				LexiSynonymTypeIdInt, _ := strconv.ParseInt(treatment.DrugDBIds[LexiSynonymTypeId], 0, 64)
				if medication.LexiDrugSynId == LexiDrugSynIdInt &&
					medication.LexiGenProductId == LexiGenProductIdInt &&
					medication.LexiSynonymTypeId == LexiSynonymTypeIdInt {
					treatment.PrescriptionId = common.NewObjectId(medication.DoseSpotPrescriptionId)
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
	medication.DrugDBIds[LexiGenProductId] = strconv.FormatInt(selectResult.LexiGenProductId, 10)
	medication.DrugDBIds[LexiDrugSynId] = strconv.FormatInt(selectResult.LexiDrugSynId, 10)
	medication.DrugDBIds[LexiSynonymTypeId] = strconv.FormatInt(selectResult.LexiSynonymTypeId, 10)
	medication.DispenseUnitId = selectResult.DispenseUnitId
	medication.DispenseUnitDescription = selectResult.DispenseUnitDescription
	medication.OTC = selectResult.OTC

	scheduleInt, err := strconv.Atoi(selectResult.Schedule)
	medication.IsControlledSubstance = err == nil && scheduleInt > 0
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
				AdditionalInfo:     logDetails.AdditionalInfo,
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

func (d *DoseSpotService) GetTransmissionErrorDetails() ([]*Medication, error) {
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
		return nil, err
	}

	medicationsWithErrors := make([]*Medication, 0)
	for _, transmissionError := range response.TransmissionErrors {
		medicationWithError := &Medication{
			ErxMedicationId:        transmissionError.Medication.MedicationId,
			DoseSpotPrescriptionId: transmissionError.Medication.DoseSpotPrescriptionId,
			PrescriptionStatus:     transmissionError.Medication.Status,
			PrescriptionDate:       &transmissionError.Medication.DatePrescribed.DateTime,
			DrugDBIds: map[string]string{
				LexiGenProductId:  strconv.FormatInt(transmissionError.Medication.LexiGenProductId, 10),
				LexiSynonymTypeId: strconv.FormatInt(transmissionError.Medication.LexiSynonymTypeId, 10),
				LexiDrugSynId:     strconv.FormatInt(transmissionError.Medication.LexiDrugSynId, 10),
			},
			DispenseUnitId:    transmissionError.Medication.DispenseUnitId,
			ErrorTimeStamp:    &transmissionError.ErrorDateTimeStamp.DateTime,
			ErrorDetails:      transmissionError.ErrorDetails,
			DisplayName:       transmissionError.Medication.DrugName,
			Strength:          transmissionError.Medication.Strength,
			Refills:           transmissionError.Medication.Refills.Int64(),
			DaysSupply:        int64(transmissionError.Medication.DaysSupply),
			Dispense:          transmissionError.Medication.Dispense,
			Instructions:      transmissionError.Medication.Instructions,
			PharmacyId:        transmissionError.Medication.PharmacyId,
			PharmacyNotes:     transmissionError.Medication.PharmacyNotes,
			NoSubstitutions:   transmissionError.Medication.NoSubstitutions,
			RxReferenceNumber: transmissionError.Medication.RxReferenceNumber,
		}

		medicationsWithErrors = append(medicationsWithErrors, medicationWithError)
	}

	return medicationsWithErrors, nil
}

func (d *DoseSpotService) GetTransmissionErrorRefillRequestsCount() (refillRequests int64, transactionErrors int64, err error) {
	clinicianId, err := strconv.ParseInt(d.UserId, 0, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("Unable to parse clinicianId: %s", err.Error())
	}
	request := &getRefillRequestsTransmissionErrorsMessageRequest{
		SSO:         generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId),
		ClinicianId: clinicianId,
	}

	response := &getRefillRequestsTransmissionErrorsResult{}
	err = getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[getRefillRequestsTransmissionsErrorsAction],
		request, response,
		d.apiLatencies[getRefillRequestsTransmissionsErrorsAction],
		d.apiRequests[getRefillRequestsTransmissionsErrorsAction],
		d.apiRequests[getRefillRequestsTransmissionsErrorsAction])

	if err != nil {
		return
	}

	if len(response.RefillRequestsTransmissionErrors) == 0 {
		return
	}

	refillRequests = response.RefillRequestsTransmissionErrors[0].RefillRequestsCount
	transactionErrors = response.RefillRequestsTransmissionErrors[0].TransactionErrorsCount
	return
}

func (d *DoseSpotService) IgnoreAlert(prescriptionId int64) error {
	request := &ignoreAlertRequest{
		SSO:            generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId),
		PrescriptionId: prescriptionId,
	}

	response := &ignoreAlertResponse{}
	err := getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[ignoreAlertAction], request, response,
		d.apiLatencies[ignoreAlertAction],
		d.apiRequests[ignoreAlertAction],
		d.apiRequests[ignoreAlertAction])
	return err
}
