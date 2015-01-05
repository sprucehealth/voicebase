package erx

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/golog"
	pharmacySearch "github.com/sprucehealth/backend/pharmacy"
)

type DoseSpotService struct {
	ClinicID     int64
	ClinicKey    string
	UserID       int64
	SOAPEndpoint string
	APIEndpoint  string
	apiLatencies map[DoseSpotAPIID]metrics.Histogram
	apiRequests  map[DoseSpotAPIID]*metrics.Counter
	apiFailure   map[DoseSpotAPIID]*metrics.Counter
}

type DoseSpotAPIID int

const (
	medicationQuickSearchAction DoseSpotAPIID = iota
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
	allergySearchAction
)

var DoseSpotAPIActions = map[DoseSpotAPIID]string{
	medicationQuickSearchAction:                    "MedicationQuickSearchMessage",
	selfReportedMedicationSearchAction:             "SelfReportedMedicationSearch",
	medicationStrengthSearchAction:                 "MedicationStrengthSearchMessage",
	medicationSelectAction:                         "MedicationSelectMessage",
	startPrescribingPatientAction:                  "PatientStartPrescribingMessage",
	sendMultiplPrescriptionsAction:                 "SendMultiplePrescriptions",
	searchPharmaciesAction:                         "PharmacySearchMessageDetailed",
	getPrescriptionLogDetailsAction:                "GetPrescriptionLogDetails",
	getTransmissionErrorDetailsAction:              "GetTransmissionErrorsDetails",
	getRefillRequestsTransmissionsErrorsAction:     "GetRefillRequestsTransmissionErrors",
	ignoreAlertAction:                              "IgnoreAlert",
	getMedicationRefillRequestQueueForClinicAction: "GetMedicationRefillRequestQueueForClinic",
	getPatientDetailsAction:                        "GetPatientDetail",
	pharmacyDetailsAction:                          "PharmacyValidateMessage",
	approveRefillAction:                            "ApproveRefill",
	denyRefillAction:                               "DenyRefill",
	allergySearchAction:                            "AllergySearch",
}

const (
	resultOk = "OK"
	prn      = "PRN"
)

type ByLogTimeStamp []*PrescriptionLog

func (a ByLogTimeStamp) Len() int      { return len(a) }
func (a ByLogTimeStamp) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByLogTimeStamp) Less(i, j int) bool {
	return a[i].LogTimestamp.Before(a[j].LogTimestamp)
}

func NewDoseSpotService(clinicID, userID int64, clinicKey, soapEndpoint, apiEndpoint string, statsRegistry metrics.Registry) ERxAPI {
	d := &DoseSpotService{
		SOAPEndpoint: soapEndpoint,
		APIEndpoint:  apiEndpoint,
	}
	if clinicID == 0 {
		d.ClinicKey = os.Getenv("DOSESPOT_CLINIC_KEY")
		d.ClinicID, _ = strconv.ParseInt(os.Getenv("DOSESPOT_CLINIC_ID"), 10, 64)
		d.UserID, _ = strconv.ParseInt(os.Getenv("DOSESPOT_USER_ID"), 10, 64)
	} else {
		d.ClinicKey = clinicKey
		d.ClinicID = clinicID
		d.UserID = userID
	}

	d.apiLatencies = make(map[DoseSpotAPIID]metrics.Histogram)
	d.apiRequests = make(map[DoseSpotAPIID]*metrics.Counter)
	d.apiFailure = make(map[DoseSpotAPIID]*metrics.Counter)
	for apiActionId, apiAction := range DoseSpotAPIActions {
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

func (d *DoseSpotService) getDoseSpotClient() *soapClient {
	return &soapClient{SoapAPIEndPoint: d.SOAPEndpoint, APIEndpoint: d.APIEndpoint}
}

func (d *DoseSpotService) GetDrugNamesForDoctor(clinicianID int64, prefix string) ([]string, error) {
	if clinicianID <= 0 {
		clinicianID = d.UserID
	}

	medicationSearch := &medicationQuickSearchRequest{
		SSO:          generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
		SearchString: prefix,
	}

	searchResult := &medicationQuickSearchResponse{}

	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[medicationQuickSearchAction],
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
		SSO:        generateSingleSignOn(d.ClinicKey, d.UserID, d.ClinicID),
		SearchTerm: prefix,
	}

	searchResult := &selfReportedMedicationSearchResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[selfReportedMedicationSearchAction],
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

func (d *DoseSpotService) SearchForAllergyRelatedMedications(searchTerm string) ([]string, error) {
	allergySearch := &allergySearchRequest{
		SSO:        generateSingleSignOn(d.ClinicKey, d.UserID, d.ClinicID),
		SearchTerm: searchTerm,
	}

	searchResults := &allergySearchResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[allergySearchAction],
		allergySearch, searchResults,
		d.apiLatencies[allergySearchAction],
		d.apiRequests[allergySearchAction],
		d.apiFailure[allergySearchAction])

	if err != nil {
		return nil, err
	}

	names := make([]string, len(searchResults.SearchResults))
	for i, searchResultItem := range searchResults.SearchResults {
		names[i] = searchResultItem.Name
	}

	return names, nil
}

func (d *DoseSpotService) SearchForMedicationStrength(clinicianID int64, medicationName string) ([]string, error) {
	if clinicianID <= 0 {
		clinicianID = d.UserID
	}

	medicationStrengthSearch := &medicationStrengthSearchRequest{
		SSO:            generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
		MedicationName: medicationName,
	}

	searchResult := &medicationStrengthSearchResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[medicationStrengthSearchAction],
		medicationStrengthSearch, searchResult,
		d.apiLatencies[medicationStrengthSearchAction],
		d.apiRequests[medicationStrengthSearchAction],
		d.apiFailure[medicationStrengthSearchAction])

	if err != nil {
		return nil, err
	}

	return searchResult.DisplayStrengths, nil
}

func (d *DoseSpotService) SendMultiplePrescriptions(clinicianID int64, patient *common.Patient, treatments []*common.Treatment) ([]*common.Treatment, error) {
	sendPrescriptionsRequest := &sendMultiplePrescriptionsRequest{
		SSO:       generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
		PatientID: patient.ERxPatientID.Int64(),
	}

	prescriptionIds := make([]int64, 0)
	prescriptionIdToTreatmentMapping := make(map[int64]*common.Treatment)
	for _, treatment := range treatments {
		if treatment.ERx.PrescriptionID.Int64() == 0 {
			continue
		}
		prescriptionIds = append(prescriptionIds, treatment.ERx.PrescriptionID.Int64())
		prescriptionIdToTreatmentMapping[treatment.ERx.PrescriptionID.Int64()] = treatment
	}

	sendPrescriptionsRequest.PrescriptionIds = prescriptionIds

	response := &sendMultiplePrescriptionsResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[sendMultiplPrescriptionsAction],
		sendPrescriptionsRequest, response,
		d.apiLatencies[sendMultiplPrescriptionsAction],
		d.apiRequests[sendMultiplPrescriptionsAction],
		d.apiFailure[sendMultiplPrescriptionsAction])

	if err != nil {
		return nil, err
	}

	unSuccessfulTreatments := make([]*common.Treatment, 0)
	for _, prescriptionResult := range response.SendPrescriptionResults {
		if prescriptionResult.ResultCode != resultOk {
			unSuccessfulTreatments = append(unSuccessfulTreatments, prescriptionIdToTreatmentMapping[int64(prescriptionResult.PrescriptionID)])
			golog.Errorf("Error sending prescription with id %d : %s", prescriptionResult.PrescriptionID, prescriptionResult.ResultDescription)
		}
	}

	if response.ResultCode != resultOk {
		return nil, errors.New("Unable to send multiple prescriptions: " + response.ResultDescription)
	}
	return unSuccessfulTreatments, nil
}

func populatePatientForDoseSpot(currentPatient *common.Patient) (*patient, error) {
	newPatient := &patient{
		PatientID:   currentPatient.ERxPatientID.Int64(),
		FirstName:   currentPatient.FirstName,
		MiddleName:  currentPatient.MiddleName,
		LastName:    currentPatient.LastName,
		Suffix:      currentPatient.Suffix,
		Prefix:      currentPatient.Prefix,
		Email:       currentPatient.Email,
		DateOfBirth: specialDateTime{DateTime: currentPatient.DOB.ToTime(), DateTimeElementName: "DateOfBirth"},
		Gender:      currentPatient.Gender,
	}

	if len(currentPatient.PhoneNumbers) > 0 {
		newPatient.PrimaryPhone = currentPatient.PhoneNumbers[0].Phone.String()
		newPatient.PrimaryPhoneType = currentPatient.PhoneNumbers[0].Type

		if len(currentPatient.PhoneNumbers) > 1 {
			newPatient.PhoneAdditional1 = currentPatient.PhoneNumbers[1].Phone.String()
			newPatient.PhoneAdditionalType1 = currentPatient.PhoneNumbers[1].Type
		}

		if len(currentPatient.PhoneNumbers) > 2 {
			newPatient.PhoneAdditional2 = currentPatient.PhoneNumbers[2].Phone.String()
			newPatient.PhoneAdditionalType2 = currentPatient.PhoneNumbers[2].Type
		}
	}

	if currentPatient.PatientAddress != nil {
		newPatient.Address1 = currentPatient.PatientAddress.AddressLine1
		newPatient.Address2 = currentPatient.PatientAddress.AddressLine2
		newPatient.City = currentPatient.PatientAddress.City
		newPatient.ZipCode = currentPatient.PatientAddress.ZipCode
		newPatient.State = currentPatient.PatientAddress.State
	}

	if currentPatient.ERxPatientID.Int64() != 0 {
		newPatient.PatientID = currentPatient.ERxPatientID.Int64()
	}

	return newPatient, nil
}

func ensurePatientInformationIsConsistent(currentPatient *common.Patient, patientUpdatesFromDoseSpot []*patientUpdate) error {
	if len(patientUpdatesFromDoseSpot) != 1 {
		return fmt.Errorf("Expected a single patient to be returned from dosespot instead got back %d", len(patientUpdatesFromDoseSpot))
	}

	patientFromDoseSpot := patientUpdatesFromDoseSpot[0].Patient

	if currentPatient.FirstName != patientFromDoseSpot.FirstName {
		return errors.New("PATIENT_INFO_MISMATCH: firstName")
	}

	if currentPatient.LastName != patientFromDoseSpot.LastName {
		return errors.New("PATIENT_INFO_MISTMATCH: lastName")
	}

	if currentPatient.MiddleName != patientFromDoseSpot.MiddleName {
		return errors.New("PATIENT_INFO_MISTMATCH: middleName")
	}

	if currentPatient.Suffix != patientFromDoseSpot.Suffix {
		return errors.New("PATIENT_INFO_MISTMATCH: suffix")
	}

	if currentPatient.Prefix != patientFromDoseSpot.Prefix {
		return errors.New("PATIENT_INFO_MISTMATCH: prefix")
	}

	if currentPatient.LastName != patientFromDoseSpot.LastName {
		return errors.New("PATIENT_INFO_MISTMATCH: lastName")
	}

	// lets compare the day, month and year components
	doseSpotPatientDOBYear, doseSpotPatientDOBMonth, doseSpotPatientDay := patientFromDoseSpot.DateOfBirth.DateTime.Date()

	if currentPatient.DOB.Day != doseSpotPatientDay || currentPatient.DOB.Month != int(doseSpotPatientDOBMonth) || currentPatient.DOB.Year != doseSpotPatientDOBYear {
		return fmt.Errorf("PATIENT_INFO_MISTMATCH: dob %+v %+v", currentPatient.DOB, patientFromDoseSpot.DateOfBirth.DateTime)
	}

	if strings.ToLower(currentPatient.Gender) != strings.ToLower(patientFromDoseSpot.Gender) {
		return errors.New("PATIENT_INFO_MISTMATCH: gender")
	}

	if currentPatient.Email != patientFromDoseSpot.Email {
		return errors.New("PATIENT_INFO_MISTMATCH: email")
	}

	if currentPatient.PatientAddress.AddressLine1 != patientFromDoseSpot.Address1 {
		return errors.New("PATIENT_INFO_MISTMATCH: address1")
	}

	if currentPatient.PatientAddress.AddressLine2 != patientFromDoseSpot.Address2 {
		return errors.New("PATIENT_INFO_MISTMATCH: email")
	}

	if currentPatient.PatientAddress.City != patientFromDoseSpot.City {
		return errors.New("PATIENT_INFO_MISTMATCH: city")
	}

	if strings.ToLower(currentPatient.PatientAddress.State) != strings.ToLower(patientFromDoseSpot.State) {
		return errors.New("PATIENT_INFO_MISTMATCH: state")
	}

	if currentPatient.PatientAddress.ZipCode != patientFromDoseSpot.ZipCode {
		return errors.New("PATIENT_INFO_MISTMATCH: zipCode")
	}

	if currentPatient.PhoneNumbers[0].Phone.String() != patientFromDoseSpot.PrimaryPhone {
		return errors.New("PATIENT_INFO_MISTMATCH: primaryPhone")
	}

	if currentPatient.PhoneNumbers[0].Type != patientFromDoseSpot.PrimaryPhoneType {
		return errors.New("PATIENT_INFO_MISTMATCH: primaryPhoneType")
	}

	return nil
}

func (d *DoseSpotService) UpdatePatientInformation(clinicianID int64, currentPatient *common.Patient) error {
	newPatient, err := populatePatientForDoseSpot(currentPatient)
	if err != nil {
		return err
	}
	patientPreferredPharmacy := &patientPharmacySelection{}
	patientPreferredPharmacy.IsPrimary = true

	patientPreferredPharmacy.PharmacyID = currentPatient.Pharmacy.SourceID

	startPrescribingRequest := &patientStartPrescribingRequest{
		AddFavoritePharmacies: []*patientPharmacySelection{patientPreferredPharmacy},
		Patient:               newPatient,
		SSO:                   generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
	}

	response := &patientStartPrescribingResponse{}
	err = d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[startPrescribingPatientAction],
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

	// if err := ensurePatientInformationIsConsistent(currentPatient, response.PatientUpdates); err != nil {
	// 	return err
	// }

	// populate the prescription id into the patient object
	currentPatient.ERxPatientID = encoding.NewObjectID(response.PatientUpdates[0].Patient.PatientID)
	return nil
}

func (d *DoseSpotService) StartPrescribingPatient(clinicianID int64, currentPatient *common.Patient, treatments []*common.Treatment, pharmacySourceId int64) error {

	newPatient, err := populatePatientForDoseSpot(currentPatient)
	if err != nil {
		return err
	}

	patientPreferredPharmacy := &patientPharmacySelection{}
	patientPreferredPharmacy.IsPrimary = true
	patientPreferredPharmacy.PharmacyID = pharmacySourceId

	prescriptions := make([]*prescription, 0)

	for _, treatment := range treatments {
		lexiDrugSynIDInt, _ := strconv.ParseInt(treatment.DrugDBIDs[LexiDrugSynID], 0, 64)
		lexiGenProductIDInt, _ := strconv.ParseInt(treatment.DrugDBIDs[LexiGenProductID], 0, 64)
		lexiSynonymTypeIDInt, _ := strconv.ParseInt(treatment.DrugDBIDs[LexiSynonymTypeID], 0, 64)

		patientPrescription := &prescription{
			LexiDrugSynID:     lexiDrugSynIDInt,
			LexiGenProductID:  lexiGenProductIDInt,
			LexiSynonymTypeID: lexiSynonymTypeIDInt,
			Refills:           treatment.NumberRefills.Int64(),
			Dispense:          treatment.DispenseValue.String(),
			DaysSupply:        treatment.DaysSupply.Int64(),
			DispenseUnitID:    treatment.DispenseUnitID.Int64(),
			Instructions:      treatment.PatientInstructions,
			NoSubstitutions:   !treatment.SubstitutionsAllowed,
			PharmacyNotes:     treatment.PharmacyNotes,
			PharmacyID:        pharmacySourceId,
		}

		if treatment.ERx != nil && treatment.ERx.ErxReferenceNumber != "" {
			patientPrescription.RxReferenceNumber = treatment.ERx.ErxReferenceNumber
		}

		prescriptions = append(prescriptions, patientPrescription)
	}

	startPrescribingRequest := &patientStartPrescribingRequest{
		AddFavoritePharmacies: []*patientPharmacySelection{patientPreferredPharmacy},
		Patient:               newPatient,
		SSO:                   generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
	}

	if len(prescriptions) > 0 {
		startPrescribingRequest.AddPrescriptions = prescriptions
	}

	response := &patientStartPrescribingResponse{}
	err = d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[startPrescribingPatientAction],
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

	// if err := ensurePatientInformationIsConsistent(currentPatient, response.PatientUpdates); err != nil {
	// 	return err
	// }

	// populate the prescription id into the patient object
	currentPatient.ERxPatientID = encoding.NewObjectID(response.PatientUpdates[0].Patient.PatientID)

	// go through and assign medication ids to all prescriptions
	for _, patientUpdate := range response.PatientUpdates {
		for _, medication := range patientUpdate.Medications {
			for _, treatment := range treatments {
				LexiDrugSynIdInt, _ := strconv.ParseInt(treatment.DrugDBIDs[LexiDrugSynID], 0, 64)
				LexiGenProductIdInt, _ := strconv.ParseInt(treatment.DrugDBIDs[LexiGenProductID], 0, 64)
				LexiSynonymTypeIdInt, _ := strconv.ParseInt(treatment.DrugDBIDs[LexiSynonymTypeID], 0, 64)
				if medication.LexiDrugSynID == LexiDrugSynIdInt &&
					medication.LexiGenProductID == LexiGenProductIdInt &&
					medication.LexiSynonymTypeID == LexiSynonymTypeIdInt {
					if treatment.ERx == nil {
						treatment.ERx = &common.ERxData{}
					}
					treatment.ERx.PrescriptionID = encoding.NewObjectID(medication.DoseSpotPrescriptionId)
				}
			}
		}
	}

	return err
}

func (d *DoseSpotService) SelectMedication(clinicianID int64, medicationName, medicationStrength string) (*MedicationSelectResponse, error) {
	if clinicianID <= 0 {
		clinicianID = d.UserID
	}

	medicationSelect := &medicationSelectRequest{
		SSO:                generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
		MedicationName:     medicationName,
		MedicationStrength: medicationStrength,
	}

	selectResult := &MedicationSelectResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[medicationSelectAction],
		medicationSelect, selectResult,
		d.apiLatencies[medicationSelectAction],
		d.apiRequests[medicationSelectAction],
		d.apiFailure[medicationSelectAction])
	if err != nil {
		return nil, err
	}

	if selectResult.LexiGenProductID == 0 && selectResult.LexiDrugSynID == 0 && selectResult.LexiSynonymTypeID == 0 {
		// this drug does not exist
		return nil, nil
	}

	return selectResult, err
}

func (d *DoseSpotService) SearchForPharmacies(clinicianID int64, city, state, zipcode, name string, pharmacyTypes []string) ([]*pharmacySearch.PharmacyData, error) {
	searchRequest := &pharmacySearchRequest{
		PharmacyCity:            city,
		PharmacyStateTwoLetters: state,
		PharmacyZipCode:         zipcode,
		PharmacyNameSearch:      name,
		SSO:                     generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
	}

	if len(pharmacyTypes) > 0 {
		searchRequest.PharmacyTypes = pharmacyTypes
	}

	searchResponse := &pharmacySearchResult{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[searchPharmaciesAction],
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
			SourceID:      pharmacyResultItem.PharmacyID,
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

func (d *DoseSpotService) GetPrescriptionStatus(clincianID int64, prescriptionID int64) ([]*PrescriptionLog, error) {
	request := &getPrescriptionLogDetailsRequest{
		SSO:            generateSingleSignOn(d.ClinicKey, clincianID, d.ClinicID),
		PrescriptionID: prescriptionID,
	}

	response := &getPrescriptionLogDetailsResult{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[getPrescriptionLogDetailsAction],
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
				LogTimestamp:       logDetails.DateTimeStamp.DateTime,
				PrescriptionStatus: logDetails.Status,
				AdditionalInfo:     logDetails.AdditionalInfo,
			}
			prescriptionLogs = append(prescriptionLogs, prescriptionLog)
		}
	}

	sort.Reverse(ByLogTimeStamp(prescriptionLogs))

	return prescriptionLogs, nil
}

func (d *DoseSpotService) GetTransmissionErrorDetails(clinicianID int64) ([]*common.Treatment, error) {
	request := &getTransmissionErrorDetailsRequest{
		SSO: generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
	}
	response := &getTransmissionErrorDetailsResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[getTransmissionErrorDetailsAction],
		request, response,
		d.apiLatencies[getTransmissionErrorDetailsAction],
		d.apiRequests[getTransmissionErrorDetailsAction],
		d.apiFailure[getTransmissionErrorDetailsAction])
	if err != nil {
		return nil, err
	}

	medicationsWithErrors := make([]*common.Treatment, len(response.TransmissionErrors))
	for i, transmissionError := range response.TransmissionErrors {
		dispenseValueFloat, _ := strconv.ParseFloat(transmissionError.Medication.Dispense, 64)
		medicationsWithErrors[i] = &common.Treatment{
			ERx: &common.ERxData{
				ErxMedicationID:       encoding.NewObjectID(transmissionError.Medication.MedicationId),
				PrescriptionID:        encoding.NewObjectID(transmissionError.Medication.DoseSpotPrescriptionId),
				PrescriptionStatus:    transmissionError.Medication.Status,
				ErxSentDate:           &transmissionError.Medication.DatePrescribed.DateTime,
				TransmissionErrorDate: &transmissionError.ErrorDateTimeStamp.DateTime,
				ErxReferenceNumber:    transmissionError.Medication.RxReferenceNumber,
				ErxPharmacyID:         transmissionError.Medication.PharmacyID,
			},
			DrugDBIDs: map[string]string{
				LexiGenProductID:  strconv.FormatInt(transmissionError.Medication.LexiGenProductID, 10),
				LexiSynonymTypeID: strconv.FormatInt(transmissionError.Medication.LexiSynonymTypeID, 10),
				LexiDrugSynID:     strconv.FormatInt(transmissionError.Medication.LexiDrugSynID, 10),
			},
			DispenseUnitID:       encoding.NewObjectID(transmissionError.Medication.DispenseUnitID),
			StatusDetails:        transmissionError.ErrorDetails,
			DrugName:             transmissionError.Medication.DrugName,
			DosageStrength:       transmissionError.Medication.Strength,
			DaysSupply:           transmissionError.Medication.DaysSupply,
			DispenseValue:        encoding.HighPrecisionFloat64(dispenseValueFloat),
			PatientInstructions:  transmissionError.Medication.Instructions,
			PharmacyNotes:        transmissionError.Medication.PharmacyNotes,
			SubstitutionsAllowed: !transmissionError.Medication.NoSubstitutions,
		}

		// we expect the refills to be a number so if it errors out this is expected
		medicationsWithErrors[i].NumberRefills, err = encoding.NullInt64FromString(transmissionError.Medication.Refills)
		if err != nil {
			return nil, err
		}
	}

	return medicationsWithErrors, nil
}

func (d *DoseSpotService) GetTransmissionErrorRefillRequestsCount(clinicianID int64) (refillRequests int64, transactionErrors int64, err error) {
	if err != nil {
		return 0, 0, fmt.Errorf("Unable to parse clinicianId: %s", err.Error())
	}
	request := &getRefillRequestsTransmissionErrorsMessageRequest{
		SSO:         generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
		ClinicianID: clinicianID,
	}

	response := &getRefillRequestsTransmissionErrorsResult{}
	err = d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[getRefillRequestsTransmissionsErrorsAction],
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

func (d *DoseSpotService) IgnoreAlert(clinicianID, prescriptionID int64) error {
	request := &ignoreAlertRequest{
		SSO:            generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
		PrescriptionID: prescriptionID,
	}

	response := &ignoreAlertResponse{}
	return d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[ignoreAlertAction], request, response,
		d.apiLatencies[ignoreAlertAction],
		d.apiRequests[ignoreAlertAction],
		d.apiFailure[ignoreAlertAction])
}

func (d *DoseSpotService) GetPatientDetails(erxPatientID int64) (*common.Patient, error) {
	request := &getPatientDetailRequest{
		SSO:       generateSingleSignOn(d.ClinicKey, d.UserID, d.ClinicID),
		PatientID: erxPatientID,
	}

	response := &getPatientDetailResult{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[getPatientDetailsAction], request, response,
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
		ERxPatientID: encoding.NewObjectID(response.PatientUpdates[0].Patient.PatientID),
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
		Email:   response.PatientUpdates[0].Patient.Email,
		ZipCode: response.PatientUpdates[0].Patient.ZipCode,
		DOB:     encoding.NewDOBFromTime(response.PatientUpdates[0].Patient.DateOfBirth.DateTime),
		PhoneNumbers: []*common.PhoneNumber{&common.PhoneNumber{
			Phone: parsePhone(response.PatientUpdates[0].Patient.PrimaryPhone),
			Type:  response.PatientUpdates[0].Patient.PrimaryPhoneType,
		},
		},
	}

	if response.PatientUpdates[0].Patient.PhoneAdditional1 != "" {
		newPatient.PhoneNumbers = append(newPatient.PhoneNumbers, &common.PhoneNumber{
			Phone: parsePhone(response.PatientUpdates[0].Patient.PhoneAdditional1),
			Type:  response.PatientUpdates[0].Patient.PhoneAdditionalType1,
		})
	}

	if response.PatientUpdates[0].Patient.PhoneAdditional2 != "" {
		newPatient.PhoneNumbers = append(newPatient.PhoneNumbers, &common.PhoneNumber{
			Phone: parsePhone(response.PatientUpdates[0].Patient.PhoneAdditional2),
			Type:  response.PatientUpdates[0].Patient.PhoneAdditionalType2,
		})
	}

	if len(response.PatientUpdates[0].Pharmacies) > 0 {
		newPatient.Pharmacy = &pharmacySearch.PharmacyData{
			Source:       pharmacySearch.PHARMACY_SOURCE_SURESCRIPTS,
			SourceID:     response.PatientUpdates[0].Pharmacies[0].PharmacyID,
			Name:         response.PatientUpdates[0].Pharmacies[0].StoreName,
			AddressLine1: response.PatientUpdates[0].Pharmacies[0].Address1,
			AddressLine2: response.PatientUpdates[0].Pharmacies[0].Address2,
			City:         response.PatientUpdates[0].Pharmacies[0].City,
			State:        response.PatientUpdates[0].Pharmacies[0].State,
			Postal:       response.PatientUpdates[0].Pharmacies[0].ZipCode,
			Phone:        response.PatientUpdates[0].Pharmacies[0].PrimaryPhone,
			Fax:          response.PatientUpdates[0].Pharmacies[0].PrimaryFax,
		}
	}

	return newPatient, nil
}

func parsePhone(phoneNumber string) common.Phone {
	p, err := common.ParsePhone(phoneNumber)
	if err != nil {
		golog.Errorf("Unable to parse phone number for dosespot patient from string: %s", err)
	}
	return p
}

func (d *DoseSpotService) GetRefillRequestQueueForClinic(clinicianID int64) ([]*common.RefillRequestItem, error) {
	request := &getMedicationRefillRequestQueueForClinicRequest{
		SSO: generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
	}

	response := &getMedicationRefillRequestQueueForClinicResult{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[getMedicationRefillRequestQueueForClinicAction], request, response,
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
			RxRequestQueueItemID:      refillRequest.RxRequestQueueItemID,
			ReferenceNumber:           refillRequest.ReferenceNumber,
			PharmacyRxReferenceNumber: refillRequest.PharmacyRxReferenceNumber,
			RequestedRefillAmount:     refillRequest.RequestedRefillAmount,
			ErxPatientID:              refillRequest.PatientID,
			PatientAddedForRequest:    refillRequest.PatientAddedForRequest,
			RequestDateStamp:          refillRequest.RequestDateStamp.DateTime,
			ClinicianID:               refillRequest.Clinician.ClinicianID,
			RequestedPrescription:     convertMedicationIntoTreatment(refillRequest.RequestedPrescription),
			DispensedPrescription:     convertMedicationIntoTreatment(refillRequest.DispensedPrescription),
		}

		// FIX: We will read the number refill values from RefillRequest.RequestedRefillAmount for now
		// due to the discrepancy on Dosespot's end with this value and the RefillRequested.RequestedPrescription.Refills value.
		// In theory, these two values should be the same. Also, its possible for RequestedRefillAmount to indicate "PRN"
		// or a number. In the event it indicates "PRN" we will handle it via a -1 on our end to indicate this. If its any value other than a number
		// or "PRN", we error out given that the number is not parseable.
		if refillRequest.RequestedRefillAmount == prn {
			refillRequestQueue[i].RequestedPrescription.NumberRefills = encoding.NullInt64{IsValid: true, Int64Value: -1}
		} else {
			refillRequestQueue[i].RequestedPrescription.NumberRefills, err = encoding.NullInt64FromString(refillRequest.RequestedRefillAmount)
			if err != nil {
				return nil, err
			}
		}

		// FIX: the refill quantity for dispensed and requested prescription are expected to be the same. So enforcing that until Dosespot
		// has a fix to ensure that all three (RequestedRefillAmount, RequestedPrescription.Refills, DispensedPrescription.Refills) are the same
		refillRequestQueue[i].DispensedPrescription.NumberRefills = refillRequestQueue[i].RequestedPrescription.NumberRefills

	}

	return refillRequestQueue, err
}

func (d *DoseSpotService) GetPharmacyDetails(pharmacyID int64) (*pharmacySearch.PharmacyData, error) {
	request := &pharmacyDetailsRequest{
		SSO:        generateSingleSignOn(d.ClinicKey, d.UserID, d.ClinicID),
		PharmacyID: pharmacyID,
	}

	response := &pharmacyDetailsResult{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[pharmacyDetailsAction], request, response,
		d.apiLatencies[pharmacyDetailsAction],
		d.apiRequests[pharmacyDetailsAction], d.apiFailure[pharmacyDetailsAction])
	if err != nil {
		return nil, err
	}

	return &pharmacySearch.PharmacyData{
		SourceID:     response.PharmacyDetails.PharmacyID,
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

func (d *DoseSpotService) ApproveRefillRequest(clinicianID, erxRefillRequestQueueItemId, approvedRefillAmount int64, comments string) (int64, error) {
	request := &approveRefillRequest{
		SSO:                  generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
		RxRequestQueueItemID: erxRefillRequestQueueItemId,
		Refills:              approvedRefillAmount,
		Comments:             comments,
	}

	response := &approveRefillResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[approveRefillAction], request, response,
		d.apiLatencies[approveRefillAction], d.apiRequests[approveRefillAction], d.apiFailure[approveRefillAction])
	if err != nil {
		return 0, err
	}

	if response.ResultCode != resultOk {
		return 0, fmt.Errorf("Unable to approve refill request: %s", response.ResultDescription)
	}

	return response.PrescriptionID, nil
}

func (d *DoseSpotService) DenyRefillRequest(clinicianID, erxRefillRequestQueueItemId int64, denialReason string, comments string) (int64, error) {
	request := &denyRefillRequest{
		SSO:                  generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
		RxRequestQueueItemID: erxRefillRequestQueueItemId,
		DenialReason:         denialReason,
		Comments:             comments,
	}

	response := &denyRefillResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[denyRefillAction], request, response,
		d.apiLatencies[denyRefillAction], d.apiRequests[denyRefillAction], d.apiRequests[denyRefillAction])

	if err != nil {
		return 0, err
	}

	if response.ResultCode != resultOk {
		return 0, fmt.Errorf("Unable to deny refill request: %s", response.ResultDescription)
	}

	return response.PrescriptionID, nil
}

func convertMedicationIntoTreatment(medicationItem *medication) *common.Treatment {
	if medicationItem == nil {
		return nil
	}
	scheduleInt, err := strconv.Atoi(medicationItem.Schedule)
	dispenseValue, _ := strconv.ParseFloat(medicationItem.Dispense, 64)
	treatment := &common.Treatment{
		DrugDBIDs: map[string]string{
			LexiDrugSynID:     strconv.FormatInt(medicationItem.LexiDrugSynID, 10),
			LexiGenProductID:  strconv.FormatInt(medicationItem.LexiGenProductID, 10),
			LexiSynonymTypeID: strconv.FormatInt(medicationItem.LexiSynonymTypeID, 10),
			NDC:               medicationItem.NDC,
		},
		DrugName:                medicationItem.DrugName,
		IsControlledSubstance:   err == nil && scheduleInt > 0,
		DaysSupply:              medicationItem.DaysSupply,
		DispenseValue:           encoding.HighPrecisionFloat64(dispenseValue),
		DispenseUnitID:          encoding.NewObjectID(medicationItem.DispenseUnitID),
		DispenseUnitDescription: medicationItem.DispenseUnitDescription,
		PatientInstructions:     medicationItem.Instructions,
		SubstitutionsAllowed:    !medicationItem.NoSubstitutions,
		PharmacyNotes:           medicationItem.PharmacyNotes,
		DrugRoute:               medicationItem.Route,
		DosageStrength:          medicationItem.Strength,
		ERx: &common.ERxData{
			PrescriptionID:      encoding.NewObjectID(medicationItem.DoseSpotPrescriptionId),
			ErxPharmacyID:       medicationItem.PharmacyID,
			PrescriptionStatus:  medicationItem.PrescriptionStatus,
			ErxMedicationID:     encoding.NewObjectID(medicationItem.MedicationId),
			DoseSpotClinicianID: medicationItem.ClinicianID,
		},
	}

	treatment.NumberRefills, _ = encoding.NullInt64FromString(medicationItem.Refills)

	if medicationItem.DatePrescribed != nil {
		treatment.ERx.ErxSentDate = &medicationItem.DatePrescribed.DateTime
	}

	if medicationItem.LastDateFilled != nil {
		treatment.ERx.ErxLastDateFilled = &medicationItem.LastDateFilled.DateTime
	}
	return treatment

}

// ParseGenericName parses and returns the generic drug name from a medication select
// response. The .GenericName in that struct seems to almost always be in the format
// "name strength route form".. except for odd cases. (for examples search for
// "PruClair", "Pruet", "Tums", "Pepcid")
func ParseGenericName(m *MedicationSelectResponse) (string, error) {
	trimFn := func(r rune) bool {
		switch r {
		case ' ':
			return true
		case ',':
			return true
		}
		return false
	}
	name := m.GenericProductName
	if !strings.HasSuffix(name, m.DoseFormDescription) {
		return "", errors.New("missing form")
	}
	name = strings.TrimRightFunc(name[:len(name)-len(m.DoseFormDescription)], trimFn)
	if !strings.HasSuffix(name, m.RouteDescription) {
		return "", errors.New("missing route")
	}
	name = strings.TrimRightFunc(name[:len(name)-len(m.RouteDescription)], trimFn)
	if m.StrengthDescription == "-" || m.StrengthDescription == "" {
		return name, nil
	}
	if !strings.HasSuffix(name, m.StrengthDescription) {
		return "", errors.New("missing strength")
	}
	return strings.TrimRightFunc(name[:len(name)-len(m.StrengthDescription)], trimFn), nil
}
