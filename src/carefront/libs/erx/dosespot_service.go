package erx

import (
	"carefront/common"
	"carefront/libs/golog"
	pharmacySearch "carefront/libs/pharmacy"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/samuel/go-metrics/metrics"
)

type DoseSpotService struct {
	ClinicId     int64
	ClinicKey    string
	UserID       int64
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
	getMedicationRefillRequestQueueForClinicAction
	getPatientDetailsAction
	pharmacyDetailsAction
	approveRefillAction
	denyRefillAction
)

var DoseSpotApiActions = map[DoseSpotApiId]string{
	medicationQuickSearchAction:                    "MedicationQuickSearchMessage",
	selfReportedMedicationSearchAction:             "SelfReportedMedicationSearch",
	medicationStrengthSearchAction:                 "MedicationStrengthSearchMessage",
	medicationSelectAction:                         "MedicationSelectMessage",
	startPrescribingPatientAction:                  "PatientStartPrescribingMessage",
	sendMultiplPrescriptionsAction:                 "SendMultiplePrescriptions",
	searchPharmaciesAction:                         "PharmacySearchMessageDetailed",
	getPrescriptionLogDetailsAction:                "GetPrescriptionLogDetails",
	getMedicationListAction:                        "GetMedicationList",
	getTransmissionErrorDetailsAction:              "GetTransmissionErrorsDetails",
	getRefillRequestsTransmissionsErrorsAction:     "GetRefillRequestsTransmissionErrors",
	ignoreAlertAction:                              "IgnoreAlert",
	getMedicationRefillRequestQueueForClinicAction: "GetMedicationRefillRequestQueueForClinic",
	getPatientDetailsAction:                        "GetPatientDetail",
	pharmacyDetailsAction:                          "PharmacyValidateMessage",
	approveRefillAction:                            "ApproveRefill",
	denyRefillAction:                               "DenyRefill",
}

const (
	doseSpotAPIEndPoint  = "http://www.dosespot.com/API/11/"
	doseSpotSOAPEndPoint = "http://i.dosespot.com/api/11/apifull.asmx"
	resultOk             = "OK"
)

type ByLogTimeStamp []*PrescriptionLog

func (a ByLogTimeStamp) Len() int      { return len(a) }
func (a ByLogTimeStamp) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByLogTimeStamp) Less(i, j int) bool {
	return a[i].LogTimeStamp.Before(a[j].LogTimeStamp)
}

func getDoseSpotClient() *soapClient {
	return &soapClient{SoapAPIEndPoint: doseSpotSOAPEndPoint, APIEndpoint: doseSpotAPIEndPoint}
}

func NewDoseSpotService(clinicId, userId int64, clinicKey string, statsRegistry metrics.Registry) *DoseSpotService {
	d := &DoseSpotService{}
	if clinicId == 0 {
		d.ClinicKey = os.Getenv("DOSESPOT_CLINIC_KEY")
		d.ClinicId, _ = strconv.ParseInt(os.Getenv("DOSESPOT_CLINIC_ID"), 10, 64)
		d.UserID, _ = strconv.ParseInt(os.Getenv("DOSESPOT_USER_ID"), 10, 64)
	} else {
		d.ClinicKey = clinicKey
		d.ClinicId = clinicId
		d.UserID = userId
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

func (d *DoseSpotService) GetDrugNamesForDoctor(clinicianId int64, prefix string) ([]string, error) {
	medicationSearch := &medicationQuickSearchRequest{
		SSO:          generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
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

func (d *DoseSpotService) GetDrugNamesForPatient(clinicianId int64, prefix string) ([]string, error) {
	selfReportedDrugsSearch := &selfReportedMedicationSearchRequest{
		SSO:        generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
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

func (d *DoseSpotService) SearchForMedicationStrength(clinicianId int64, medicationName string) ([]string, error) {
	medicationStrengthSearch := &medicationStrengthSearchRequest{
		SSO:            generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
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

func (d *DoseSpotService) SendMultiplePrescriptions(clinicianId int64, patient *common.Patient, treatments []*common.Treatment) ([]int64, error) {
	sendPrescriptionsRequest := &sendMultiplePrescriptionsRequest{
		SSO:       generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
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

func populatePatientForDoseSpot(currentPatient *common.Patient) *patient {

	newPatient := &patient{
		PatientId:   currentPatient.ERxPatientId.Int64(),
		FirstName:   currentPatient.FirstName,
		MiddleName:  currentPatient.MiddleName,
		LastName:    currentPatient.LastName,
		Suffix:      currentPatient.Suffix,
		Prefix:      currentPatient.Prefix,
		Email:       currentPatient.Email,
		City:        currentPatient.City,
		State:       currentPatient.State,
		ZipCode:     currentPatient.ZipCode,
		DateOfBirth: specialDateTime{DateTime: currentPatient.Dob, DateTimeElementName: "DateOfBirth"},
		Gender:      currentPatient.Gender,
	}

	if len(currentPatient.PhoneNumbers) > 0 {
		newPatient.PrimaryPhone = currentPatient.PhoneNumbers[0].Phone
		newPatient.PrimaryPhoneType = currentPatient.PhoneNumbers[0].PhoneType

		if len(currentPatient.PhoneNumbers) > 1 {
			newPatient.PhoneAdditional1 = currentPatient.PhoneNumbers[1].Phone
			newPatient.PhoneAdditionalType1 = currentPatient.PhoneNumbers[1].PhoneType
		}

		if len(currentPatient.PhoneNumbers) > 2 {
			newPatient.PhoneAdditional2 = currentPatient.PhoneNumbers[2].Phone
			newPatient.PhoneAdditionalType2 = currentPatient.PhoneNumbers[2].PhoneType
		}
	}

	if currentPatient.PatientAddress != nil {
		newPatient.Address1 = currentPatient.PatientAddress.AddressLine1
		newPatient.Address2 = currentPatient.PatientAddress.AddressLine2
		newPatient.City = currentPatient.PatientAddress.City
		newPatient.ZipCode = currentPatient.PatientAddress.ZipCode
	}

	if currentPatient.ERxPatientId.Int64() != 0 {
		newPatient.PatientId = currentPatient.ERxPatientId.Int64()
	}

	return newPatient
}

func (d *DoseSpotService) UpdatePatientInformation(clinicianId int64, currentPatient *common.Patient) error {
	newPatient := populatePatientForDoseSpot(currentPatient)
	patientPreferredPharmacy := &patientPharmacySelection{}
	patientPreferredPharmacy.IsPrimary = true

	pharmacyId, err := strconv.ParseInt(currentPatient.Pharmacy.SourceId, 0, 64)
	if err != nil {
		return fmt.Errorf("Unable to parse the pharmacy id: %s", err.Error())
	}

	patientPreferredPharmacy.PharmacyId = pharmacyId

	startPrescribingRequest := &patientStartPrescribingRequest{
		AddFavoritePharmacies: []*patientPharmacySelection{patientPreferredPharmacy},
		Patient:               newPatient,
		SSO:                   generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
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
	return nil
}

func (d *DoseSpotService) StartPrescribingPatient(clinicianId int64, currentPatient *common.Patient, treatments []*common.Treatment) error {

	newPatient := populatePatientForDoseSpot(currentPatient)

	patientPreferredPharmacy := &patientPharmacySelection{}
	patientPreferredPharmacy.IsPrimary = true

	pharmacyId, err := strconv.ParseInt(currentPatient.Pharmacy.SourceId, 0, 64)
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
		SSO:                   generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
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

func (d *DoseSpotService) SelectMedication(clinicianId int64, medicationName, medicationStrength string) (medication *common.Treatment, err error) {
	medicationSelect := &medicationSelectRequest{
		SSO:                generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
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

	var scheduleInt int
	if selectResult.Schedule == "" {
		scheduleInt = 0
	} else {
		scheduleInt, err = strconv.Atoi(selectResult.Schedule)
	}
	medication = &common.Treatment{
		DrugDBIds: map[string]string{
			LexiGenProductId:  strconv.FormatInt(selectResult.LexiGenProductId, 10),
			LexiDrugSynId:     strconv.FormatInt(selectResult.LexiDrugSynId, 10),
			LexiSynonymTypeId: strconv.FormatInt(selectResult.LexiSynonymTypeId, 10),
			NDC:               selectResult.RepresentativeNDC,
		},
		DispenseUnitId:          common.NewObjectId(selectResult.DispenseUnitId),
		DispenseUnitDescription: selectResult.DispenseUnitDescription,
		OTC: selectResult.OTC,
		IsControlledSubstance: err == nil && scheduleInt > 0,
	}

	return medication, err
}

func (d *DoseSpotService) SearchForPharmacies(clinicianId int64, city, state, zipcode, name string, pharmacyTypes []string) ([]*pharmacySearch.PharmacyData, error) {
	searchRequest := &pharmacySearchRequest{
		PharmacyCity:            city,
		PharmacyStateTwoLetters: state,
		PharmacyZipCode:         zipcode,
		PharmacyNameSearch:      name,
		SSO:                     generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
	}

	if len(pharmacyTypes) > 0 {
		searchRequest.PharmacyTypes = pharmacyTypes
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
			SourceId:      strconv.FormatInt(pharmacyResultItem.PharmacyId, 10),
			AddressLine1:  pharmacyResultItem.Address1,
			AddressLine2:  pharmacyResultItem.Address2,
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

func (d *DoseSpotService) GetPrescriptionStatus(clincianId int64, prescriptionId int64) ([]*PrescriptionLog, error) {
	request := &getPrescriptionLogDetailsRequest{
		SSO:            generateSingleSignOn(d.ClinicKey, clincianId, d.ClinicId),
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

	sort.Reverse(ByLogTimeStamp(prescriptionLogs))

	return prescriptionLogs, nil
}

func (d *DoseSpotService) GetMedicationList(clinicianId int64, PatientId int64) ([]*common.Treatment, error) {
	request := &getMedicationListRequest{
		PatientId: PatientId,
		SSO:       generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
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

	medications := make([]*common.Treatment, len(response.Medications))
	for i, medicationItem := range response.Medications {
		medications[i] = convertMedicationIntoTreatment(medicationItem)
	}
	return medications, nil
}

func (d *DoseSpotService) GetTransmissionErrorDetails(clinicianId int64) ([]*common.Treatment, error) {
	request := &getTransmissionErrorDetailsRequest{
		SSO: generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
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

	medicationsWithErrors := make([]*common.Treatment, len(response.TransmissionErrors))
	for i, transmissionError := range response.TransmissionErrors {
		dispenseValueInt, _ := strconv.ParseInt(transmissionError.Medication.Dispense, 10, 64)
		medicationsWithErrors[i] = &common.Treatment{
			ErxMedicationId:    common.NewObjectId(transmissionError.Medication.MedicationId),
			PrescriptionId:     common.NewObjectId(transmissionError.Medication.DoseSpotPrescriptionId),
			PrescriptionStatus: transmissionError.Medication.Status,
			ErxSentDate:        &transmissionError.Medication.DatePrescribed.DateTime,
			DrugDBIds: map[string]string{
				LexiGenProductId:  strconv.FormatInt(transmissionError.Medication.LexiGenProductId, 10),
				LexiSynonymTypeId: strconv.FormatInt(transmissionError.Medication.LexiSynonymTypeId, 10),
				LexiDrugSynId:     strconv.FormatInt(transmissionError.Medication.LexiDrugSynId, 10),
			},
			DispenseUnitId:        common.NewObjectId(transmissionError.Medication.DispenseUnitId),
			TransmissionErrorDate: &transmissionError.ErrorDateTimeStamp.DateTime,
			StatusDetails:         transmissionError.ErrorDetails,
			DrugName:              transmissionError.Medication.DrugName,
			DosageStrength:        transmissionError.Medication.Strength,
			NumberRefills:         transmissionError.Medication.Refills.Int64(),
			DaysSupply:            int64(transmissionError.Medication.DaysSupply),
			DispenseValue:         dispenseValueInt,
			PatientInstructions:   transmissionError.Medication.Instructions,
			ErxPharmacyId:         transmissionError.Medication.PharmacyId,
			PharmacyNotes:         transmissionError.Medication.PharmacyNotes,
			SubstitutionsAllowed:  !transmissionError.Medication.NoSubstitutions,
			ErxReferenceNumber:    transmissionError.Medication.RxReferenceNumber,
		}

	}

	return medicationsWithErrors, nil
}

func (d *DoseSpotService) GetTransmissionErrorRefillRequestsCount(clinicianId int64) (refillRequests int64, transactionErrors int64, err error) {
	if err != nil {
		return 0, 0, fmt.Errorf("Unable to parse clinicianId: %s", err.Error())
	}
	request := &getRefillRequestsTransmissionErrorsMessageRequest{
		SSO:         generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
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

func (d *DoseSpotService) IgnoreAlert(clinicianId, prescriptionId int64) error {
	request := &ignoreAlertRequest{
		SSO:            generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
		PrescriptionId: prescriptionId,
	}

	response := &ignoreAlertResponse{}
	return getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[ignoreAlertAction], request, response,
		d.apiLatencies[ignoreAlertAction],
		d.apiRequests[ignoreAlertAction],
		d.apiFailure[ignoreAlertAction])
}

func (d *DoseSpotService) GetPatientDetails(erxPatientId int64) (*common.Patient, error) {
	request := &getPatientDetailRequest{
		SSO:       generateSingleSignOn(d.ClinicKey, d.UserID, d.ClinicId),
		PatientId: erxPatientId,
	}

	response := &getPatientDetailResult{}
	err := getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[getPatientDetailsAction], request, response,
		d.apiLatencies[getPatientDetailsAction],
		d.apiRequests[getPatientDetailsAction],
		d.apiFailure[getPatientDetailsAction])

	if err != nil {
		return nil, err
	}

	if response.ResultCode != resultOk {
		return nil, fmt.Errorf(response.ResultDescription)
	}

	if len(response.PatientUpdates) == 0 {
		return nil, nil
	}

	if response.PatientUpdates[0].Patient == nil {
		return nil, nil
	}

	// not worrying about suffix/prefix for now
	newPatient := &common.Patient{
		ERxPatientId: common.NewObjectId(response.PatientUpdates[0].Patient.PatientId),
		FirstName:    response.PatientUpdates[0].Patient.FirstName,
		LastName:     response.PatientUpdates[0].Patient.LastName,
		Gender:       response.PatientUpdates[0].Patient.Gender,
		PatientAddress: &common.Address{
			AddressLine1: response.PatientUpdates[0].Patient.Address1,
			AddressLine2: response.PatientUpdates[0].Patient.Address2,
			City:         response.PatientUpdates[0].Patient.City,
			ZipCode:      response.PatientUpdates[0].Patient.ZipCode,
			State:        response.PatientUpdates[0].Patient.State,
		},
		Dob:     response.PatientUpdates[0].Patient.DateOfBirth.DateTime,
		Email:   response.PatientUpdates[0].Patient.Email,
		ZipCode: response.PatientUpdates[0].Patient.ZipCode,
		City:    response.PatientUpdates[0].Patient.City,
		State:   response.PatientUpdates[0].Patient.State,
		PhoneNumbers: []*common.PhoneInformation{&common.PhoneInformation{
			Phone:     response.PatientUpdates[0].Patient.PrimaryPhone,
			PhoneType: response.PatientUpdates[0].Patient.PrimaryPhoneType,
		},
		},
	}

	if response.PatientUpdates[0].Patient.PhoneAdditional1 != "" {
		newPatient.PhoneNumbers = append(newPatient.PhoneNumbers, &common.PhoneInformation{
			Phone:     response.PatientUpdates[0].Patient.PhoneAdditional1,
			PhoneType: response.PatientUpdates[0].Patient.PhoneAdditionalType1,
		})
	}

	if response.PatientUpdates[0].Patient.PhoneAdditional2 != "" {
		newPatient.PhoneNumbers = append(newPatient.PhoneNumbers, &common.PhoneInformation{
			Phone:     response.PatientUpdates[0].Patient.PhoneAdditional2,
			PhoneType: response.PatientUpdates[0].Patient.PhoneAdditionalType2,
		})
	}

	return newPatient, nil
}

func (d *DoseSpotService) GetRefillRequestQueueForClinic() ([]*common.RefillRequestItem, error) {
	request := &getMedicationRefillRequestQueueForClinicRequest{
		SSO: generateSingleSignOn(d.ClinicKey, d.UserID, d.ClinicId),
	}

	response := &getMedicationRefillRequestQueueForClinicResult{}
	err := getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[getMedicationRefillRequestQueueForClinicAction], request, response,
		d.apiLatencies[getMedicationRefillRequestQueueForClinicAction],
		d.apiRequests[getMedicationRefillRequestQueueForClinicAction],
		d.apiFailure[getMedicationRefillRequestQueueForClinicAction])

	if err != nil {
		return nil, err
	}

	if response.ResultCode != resultOk {
		return nil, fmt.Errorf(response.ResultDescription)
	}

	refillRequestQueue := make([]*common.RefillRequestItem, len(response.RefillRequestQueue))
	// translate each of the request queue items into the object to return
	for i, refillRequest := range response.RefillRequestQueue {
		refillRequestQueue[i] = &common.RefillRequestItem{
			RxRequestQueueItemId:      refillRequest.RxRequestQueueItemId,
			ReferenceNumber:           refillRequest.ReferenceNumber,
			PharmacyRxReferenceNumber: refillRequest.PharmacyRxReferenceNumber,
			RequestedDrugDescription:  refillRequest.RequestedDrugDescription,
			RequestedRefillAmount:     refillRequest.RequestedRefillAmount,
			RequestedDispense:         refillRequest.RequestedDispense,
			ErxPatientId:              refillRequest.PatientId,
			PatientAddedForRequest:    refillRequest.PatientAddedForRequest,
			RequestDateStamp:          refillRequest.RequestDateStamp.DateTime,
			ClinicianId:               refillRequest.Clinician.ClinicianId,
			RequestedPrescription:     convertMedicationIntoTreatment(refillRequest.RequestedPrescription),
			DispensedPrescription:     convertMedicationIntoTreatment(refillRequest.DispensedPrescription),
		}
	}
	return refillRequestQueue, err
}

func (d *DoseSpotService) GetPharmacyDetails(pharmacyId int64) (*pharmacySearch.PharmacyData, error) {
	request := &pharmacyDetailsRequest{
		SSO:        generateSingleSignOn(d.ClinicKey, d.UserID, d.ClinicId),
		PharmacyId: pharmacyId,
	}

	response := &pharmacyDetailsResult{}
	err := getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[pharmacyDetailsAction], request, response,
		d.apiLatencies[pharmacyDetailsAction],
		d.apiRequests[pharmacyDetailsAction], d.apiFailure[pharmacyDetailsAction])
	if err != nil {
		return nil, err
	}

	return &pharmacySearch.PharmacyData{
		SourceId:     strconv.FormatInt(response.PharmacyDetails.PharmacyId, 10),
		AddressLine1: response.PharmacyDetails.Address1,
		AddressLine2: response.PharmacyDetails.Address2,
		City:         response.PharmacyDetails.City,
		Postal:       response.PharmacyDetails.ZipCode,
		State:        response.PharmacyDetails.State,
		Phone:        response.PharmacyDetails.PrimaryPhone,
		Name:         response.PharmacyDetails.StoreName,
		Source:       pharmacySearch.PHARMACY_SOURCE_SURESCRIPTS,
	}, nil
}

func (d *DoseSpotService) ApproveRefillRequest(clinicianId, erxRefillRequestQueueItemId, approvedRefillAmount int64, comments string) (int64, error) {
	request := &approveRefillRequest{
		SSO:                  generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
		RxRequestQueueItemId: erxRefillRequestQueueItemId,
		Refills:              approvedRefillAmount,
		Comments:             comments,
	}

	response := &approveRefillResponse{}
	err := getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[approveRefillAction], request, response,
		d.apiLatencies[approveRefillAction], d.apiRequests[approveRefillAction], d.apiFailure[approveRefillAction])
	if err != nil {
		return 0, err
	}

	if response.ResultCode != resultOk {
		return 0, fmt.Errorf("Unable to approve refill request: %s", response.ResultDescription)
	}

	return response.PrescriptionId, nil
}

func (d *DoseSpotService) DenyRefillRequest(clinicianId, erxRefillRequestQueueItemId int64, denialReason string, comments string) (int64, error) {
	request := &denyRefillRequest{
		SSO:                  generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
		RxRequestQueueItemId: erxRefillRequestQueueItemId,
		DenialReason:         denialReason,
		Comments:             comments,
	}

	response := &denyRefillResponse{}
	err := getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[denyRefillAction], request, response,
		d.apiLatencies[denyRefillAction], d.apiRequests[denyRefillAction], d.apiRequests[denyRefillAction])

	if err != nil {
		return 0, err
	}

	if response.ResultCode != resultOk {
		return 0, fmt.Errorf("Unable to deny refill request: %s", response.ResultDescription)
	}

	return response.PrescriptionId, nil
}

func convertMedicationIntoTreatment(medicationItem *medication) *common.Treatment {
	if medicationItem == nil {
		return nil
	}
	scheduleInt, err := strconv.Atoi(medicationItem.Schedule)
	dispenseValue, _ := strconv.ParseInt(medicationItem.Dispense, 10, 64)
	treatment := &common.Treatment{
		PrescriptionId: common.NewObjectId(medicationItem.DoseSpotPrescriptionId),
		DrugDBIds: map[string]string{
			LexiDrugSynId:     strconv.FormatInt(medicationItem.LexiDrugSynId, 10),
			LexiGenProductId:  strconv.FormatInt(medicationItem.LexiGenProductId, 10),
			LexiSynonymTypeId: strconv.FormatInt(medicationItem.LexiSynonymTypeId, 10),
			NDC:               medicationItem.NDC,
		},
		DrugName:                medicationItem.DrugName,
		IsControlledSubstance:   err == nil && scheduleInt > 0,
		NumberRefills:           int64(medicationItem.Refills),
		DaysSupply:              int64(medicationItem.DaysSupply),
		DispenseValue:           dispenseValue,
		DispenseUnitId:          common.NewObjectId(medicationItem.DispenseUnitId),
		DispenseUnitDescription: medicationItem.DispenseUnitDescription,
		PatientInstructions:     medicationItem.Instructions,
		SubstitutionsAllowed:    !medicationItem.NoSubstitutions,
		ErxPharmacyId:           medicationItem.PharmacyId,
		PharmacyNotes:           medicationItem.PharmacyNotes,
		PrescriptionStatus:      medicationItem.PrescriptionStatus,
		ErxMedicationId:         common.NewObjectId(medicationItem.MedicationId),
		DrugRoute:               medicationItem.Route,
		DosageStrength:          medicationItem.Strength,
		DoseSpotClinicianId:     medicationItem.ClinicianId,
	}

	if medicationItem.DatePrescribed != nil {
		treatment.ErxSentDate = &medicationItem.DatePrescribed.DateTime
	}

	if medicationItem.LastDateFilled != nil {
		treatment.ErxLastDateFilled = &medicationItem.LastDateFilled.DateTime
	}
	return treatment

}
