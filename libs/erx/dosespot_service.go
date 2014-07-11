package erx

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/golog"
	pharmacySearch "github.com/sprucehealth/backend/pharmacy"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

type DoseSpotService struct {
	ClinicId     int64
	ClinicKey    string
	UserID       int64
	SOAPEndpoint string
	APIEndpoint  string
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
	allergySearchAction
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
	allergySearchAction:                            "AllergySearch",
}

const (
	resultOk = "OK"
)

type ByLogTimeStamp []*PrescriptionLog

func (a ByLogTimeStamp) Len() int      { return len(a) }
func (a ByLogTimeStamp) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByLogTimeStamp) Less(i, j int) bool {
	return a[i].LogTimestamp.Before(a[j].LogTimestamp)
}

func NewDoseSpotService(clinicId, userId int64, clinicKey string, soapEndpoint string, apiEndpoint string, statsRegistry metrics.Registry) ERxAPI {
	d := &DoseSpotService{
		SOAPEndpoint: soapEndpoint,
		APIEndpoint:  apiEndpoint,
	}
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

func (d *DoseSpotService) getDoseSpotClient() *soapClient {
	return &soapClient{SoapAPIEndPoint: d.SOAPEndpoint, APIEndpoint: d.APIEndpoint}
}

func (d *DoseSpotService) GetDrugNamesForDoctor(clinicianId int64, prefix string) ([]string, error) {
	medicationSearch := &medicationQuickSearchRequest{
		SSO:          generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
		SearchString: prefix,
	}

	searchResult := &medicationQuickSearchResponse{}

	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[medicationQuickSearchAction],
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
		SSO:        generateSingleSignOn(d.ClinicKey, d.UserID, d.ClinicId),
		SearchTerm: prefix,
	}

	searchResult := &selfReportedMedicationSearchResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[selfReportedMedicationSearchAction],
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
		SSO:        generateSingleSignOn(d.ClinicKey, d.UserID, d.ClinicId),
		SearchTerm: searchTerm,
	}

	searchResults := &allergySearchResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[allergySearchAction],
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

func (d *DoseSpotService) SearchForMedicationStrength(clinicianId int64, medicationName string) ([]string, error) {
	medicationStrengthSearch := &medicationStrengthSearchRequest{
		SSO:            generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
		MedicationName: medicationName,
	}

	searchResult := &medicationStrengthSearchResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[medicationStrengthSearchAction],
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
		if treatment.ERx.PrescriptionId.Int64() == 0 {
			continue
		}
		prescriptionIds = append(prescriptionIds, treatment.ERx.PrescriptionId.Int64())
		prescriptionIdToTreatmentIdMapping[treatment.ERx.PrescriptionId.Int64()] = treatment.Id.Int64()
	}

	sendPrescriptionsRequest.PrescriptionIds = prescriptionIds

	response := &sendMultiplePrescriptionsResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[sendMultiplPrescriptionsAction],
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

func populatePatientForDoseSpot(currentPatient *common.Patient) (*patient, error) {
	newPatient := &patient{
		PatientId:   currentPatient.ERxPatientId.Int64(),
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
		newPatient.PrimaryPhone = currentPatient.PhoneNumbers[0].Phone
		newPatient.PrimaryPhoneType = currentPatient.PhoneNumbers[0].Type

		if len(currentPatient.PhoneNumbers) > 1 {
			newPatient.PhoneAdditional1 = currentPatient.PhoneNumbers[1].Phone
			newPatient.PhoneAdditionalType1 = currentPatient.PhoneNumbers[1].Type
		}

		if len(currentPatient.PhoneNumbers) > 2 {
			newPatient.PhoneAdditional2 = currentPatient.PhoneNumbers[2].Phone
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

	if currentPatient.ERxPatientId.Int64() != 0 {
		newPatient.PatientId = currentPatient.ERxPatientId.Int64()
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

	if currentPatient.PhoneNumbers[0].Phone != patientFromDoseSpot.PrimaryPhone {
		return errors.New("PATIENT_INFO_MISTMATCH: primaryPhone")
	}

	if currentPatient.PhoneNumbers[0].Type != patientFromDoseSpot.PrimaryPhoneType {
		return errors.New("PATIENT_INFO_MISTMATCH: primaryPhoneType")
	}

	return nil
}

func (d *DoseSpotService) UpdatePatientInformation(clinicianId int64, currentPatient *common.Patient) error {
	newPatient, err := populatePatientForDoseSpot(currentPatient)
	if err != nil {
		return err
	}
	patientPreferredPharmacy := &patientPharmacySelection{}
	patientPreferredPharmacy.IsPrimary = true

	patientPreferredPharmacy.PharmacyId = currentPatient.Pharmacy.SourceId

	startPrescribingRequest := &patientStartPrescribingRequest{
		AddFavoritePharmacies: []*patientPharmacySelection{patientPreferredPharmacy},
		Patient:               newPatient,
		SSO:                   generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
	}

	response := &patientStartPrescribingResponse{}
	err = d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[startPrescribingPatientAction],
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

	if err := ensurePatientInformationIsConsistent(currentPatient, response.PatientUpdates); err != nil {
		return err
	}

	// populate the prescription id into the patient object
	currentPatient.ERxPatientId = encoding.NewObjectId(response.PatientUpdates[0].Patient.PatientId)
	return nil
}

func (d *DoseSpotService) StartPrescribingPatient(clinicianId int64, currentPatient *common.Patient, treatments []*common.Treatment, pharmacySourceId int64) error {

	newPatient, err := populatePatientForDoseSpot(currentPatient)
	if err != nil {
		return err
	}

	patientPreferredPharmacy := &patientPharmacySelection{}
	patientPreferredPharmacy.IsPrimary = true
	patientPreferredPharmacy.PharmacyId = pharmacySourceId

	prescriptions := make([]*prescription, 0)

	for _, treatment := range treatments {
		lexiDrugSynIdInt, _ := strconv.ParseInt(treatment.DrugDBIds[LexiDrugSynId], 0, 64)
		lexiGenProductIdInt, _ := strconv.ParseInt(treatment.DrugDBIds[LexiGenProductId], 0, 64)
		lexiSynonymTypeIdInt, _ := strconv.ParseInt(treatment.DrugDBIds[LexiSynonymTypeId], 0, 64)

		prescriptionMedication := &medication{
			LexiDrugSynId:     lexiDrugSynIdInt,
			LexiGenProductId:  lexiGenProductIdInt,
			LexiSynonymTypeId: lexiSynonymTypeIdInt,
			Refills:           treatment.NumberRefills,
			Dispense:          treatment.DispenseValue.String(),
			DaysSupply:        treatment.DaysSupply,
			DispenseUnitId:    treatment.DispenseUnitId.Int64(),
			Instructions:      treatment.PatientInstructions,
			NoSubstitutions:   !treatment.SubstitutionsAllowed,
			PharmacyNotes:     treatment.PharmacyNotes,
			PharmacyId:        pharmacySourceId,
		}

		if treatment.ERx != nil && treatment.ERx.ErxReferenceNumber != "" {
			prescriptionMedication.RxReferenceNumber = treatment.ERx.ErxReferenceNumber
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
	err = d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[startPrescribingPatientAction],
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

	if err := ensurePatientInformationIsConsistent(currentPatient, response.PatientUpdates); err != nil {
		return err
	}

	// populate the prescription id into the patient object
	currentPatient.ERxPatientId = encoding.NewObjectId(response.PatientUpdates[0].Patient.PatientId)

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
					if treatment.ERx == nil {
						treatment.ERx = &common.ERxData{}
					}
					treatment.ERx.PrescriptionId = encoding.NewObjectId(medication.DoseSpotPrescriptionId)
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
	err = d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[medicationSelectAction],
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

	if selectResult.LexiGenProductId == 0 && selectResult.LexiDrugSynId == 0 && selectResult.LexiSynonymTypeId == 0 {
		// this drug does not exist
		return nil, nil
	}

	// starting refills at 0 because we default to 0 even when doctor
	// does not enter something
	medication = &common.Treatment{
		DrugDBIds: map[string]string{
			LexiGenProductId:  strconv.FormatInt(selectResult.LexiGenProductId, 10),
			LexiDrugSynId:     strconv.FormatInt(selectResult.LexiDrugSynId, 10),
			LexiSynonymTypeId: strconv.FormatInt(selectResult.LexiSynonymTypeId, 10),
			NDC:               selectResult.RepresentativeNDC,
		},
		DispenseUnitId:          encoding.NewObjectId(selectResult.DispenseUnitId),
		DispenseUnitDescription: selectResult.DispenseUnitDescription,
		DrugInternalName:        medicationName,
		OTC:                     selectResult.OTC,
		SubstitutionsAllowed:    true, // defaulting to substitutions being allowed as required by surescripts
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 0,
		},
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
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[searchPharmaciesAction],
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
			SourceId:      pharmacyResultItem.PharmacyId,
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
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[getPrescriptionLogDetailsAction],
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

func (d *DoseSpotService) GetMedicationList(clinicianId int64, PatientId int64) ([]*common.Treatment, error) {
	request := &getMedicationListRequest{
		PatientId: PatientId,
		SSO:       generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
		Sources:   []string{"Prescription"},
		Status:    []string{"Active"},
	}
	response := &getMedicationListResult{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[getMedicationListAction],
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
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[getTransmissionErrorDetailsAction],
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
				ErxMedicationId:       encoding.NewObjectId(transmissionError.Medication.MedicationId),
				PrescriptionId:        encoding.NewObjectId(transmissionError.Medication.DoseSpotPrescriptionId),
				PrescriptionStatus:    transmissionError.Medication.Status,
				ErxSentDate:           &transmissionError.Medication.DatePrescribed.DateTime,
				TransmissionErrorDate: &transmissionError.ErrorDateTimeStamp.DateTime,
				ErxReferenceNumber:    transmissionError.Medication.RxReferenceNumber,
				ErxPharmacyId:         transmissionError.Medication.PharmacyId,
			},
			DrugDBIds: map[string]string{
				LexiGenProductId:  strconv.FormatInt(transmissionError.Medication.LexiGenProductId, 10),
				LexiSynonymTypeId: strconv.FormatInt(transmissionError.Medication.LexiSynonymTypeId, 10),
				LexiDrugSynId:     strconv.FormatInt(transmissionError.Medication.LexiDrugSynId, 10),
			},
			DispenseUnitId:       encoding.NewObjectId(transmissionError.Medication.DispenseUnitId),
			StatusDetails:        transmissionError.ErrorDetails,
			DrugName:             transmissionError.Medication.DrugName,
			DosageStrength:       transmissionError.Medication.Strength,
			NumberRefills:        transmissionError.Medication.Refills,
			DaysSupply:           transmissionError.Medication.DaysSupply,
			DispenseValue:        encoding.HighPrecisionFloat64(dispenseValueFloat),
			PatientInstructions:  transmissionError.Medication.Instructions,
			PharmacyNotes:        transmissionError.Medication.PharmacyNotes,
			SubstitutionsAllowed: !transmissionError.Medication.NoSubstitutions,
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
	err = d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[getRefillRequestsTransmissionsErrorsAction],
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
	return d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[ignoreAlertAction], request, response,
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
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[getPatientDetailsAction], request, response,
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
		ERxPatientId: encoding.NewObjectId(response.PatientUpdates[0].Patient.PatientId),
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
			Phone: response.PatientUpdates[0].Patient.PrimaryPhone,
			Type:  response.PatientUpdates[0].Patient.PrimaryPhoneType,
		},
		},
	}

	if response.PatientUpdates[0].Patient.PhoneAdditional1 != "" {
		newPatient.PhoneNumbers = append(newPatient.PhoneNumbers, &common.PhoneNumber{
			Phone: response.PatientUpdates[0].Patient.PhoneAdditional1,
			Type:  response.PatientUpdates[0].Patient.PhoneAdditionalType1,
		})
	}

	if response.PatientUpdates[0].Patient.PhoneAdditional2 != "" {
		newPatient.PhoneNumbers = append(newPatient.PhoneNumbers, &common.PhoneNumber{
			Phone: response.PatientUpdates[0].Patient.PhoneAdditional2,
			Type:  response.PatientUpdates[0].Patient.PhoneAdditionalType2,
		})
	}

	if len(response.PatientUpdates[0].Pharmacies) > 0 {
		newPatient.Pharmacy = &pharmacySearch.PharmacyData{
			Source:       pharmacySearch.PHARMACY_SOURCE_SURESCRIPTS,
			SourceId:     response.PatientUpdates[0].Pharmacies[0].PharmacyId,
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

func (d *DoseSpotService) GetRefillRequestQueueForClinic(clinicianId int64) ([]*common.RefillRequestItem, error) {
	request := &getMedicationRefillRequestQueueForClinicRequest{
		SSO: generateSingleSignOn(d.ClinicKey, clinicianId, d.ClinicId),
	}

	response := &getMedicationRefillRequestQueueForClinicResult{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[getMedicationRefillRequestQueueForClinicAction], request, response,
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
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[pharmacyDetailsAction], request, response,
		d.apiLatencies[pharmacyDetailsAction],
		d.apiRequests[pharmacyDetailsAction], d.apiFailure[pharmacyDetailsAction])
	if err != nil {
		return nil, err
	}

	return &pharmacySearch.PharmacyData{
		SourceId:     response.PharmacyDetails.PharmacyId,
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
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[approveRefillAction], request, response,
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
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotApiActions[denyRefillAction], request, response,
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
	dispenseValue, _ := strconv.ParseFloat(medicationItem.Dispense, 64)
	treatment := &common.Treatment{
		DrugDBIds: map[string]string{
			LexiDrugSynId:     strconv.FormatInt(medicationItem.LexiDrugSynId, 10),
			LexiGenProductId:  strconv.FormatInt(medicationItem.LexiGenProductId, 10),
			LexiSynonymTypeId: strconv.FormatInt(medicationItem.LexiSynonymTypeId, 10),
			NDC:               medicationItem.NDC,
		},
		DrugName:                medicationItem.DrugName,
		IsControlledSubstance:   err == nil && scheduleInt > 0,
		NumberRefills:           medicationItem.Refills,
		DaysSupply:              medicationItem.DaysSupply,
		DispenseValue:           encoding.HighPrecisionFloat64(dispenseValue),
		DispenseUnitId:          encoding.NewObjectId(medicationItem.DispenseUnitId),
		DispenseUnitDescription: medicationItem.DispenseUnitDescription,
		PatientInstructions:     medicationItem.Instructions,
		SubstitutionsAllowed:    !medicationItem.NoSubstitutions,
		PharmacyNotes:           medicationItem.PharmacyNotes,
		DrugRoute:               medicationItem.Route,
		DosageStrength:          medicationItem.Strength,
		ERx: &common.ERxData{
			PrescriptionId:      encoding.NewObjectId(medicationItem.DoseSpotPrescriptionId),
			ErxPharmacyId:       medicationItem.PharmacyId,
			PrescriptionStatus:  medicationItem.PrescriptionStatus,
			ErxMedicationId:     encoding.NewObjectId(medicationItem.MedicationId),
			DoseSpotClinicianId: medicationItem.ClinicianId,
		},
	}

	if medicationItem.DatePrescribed != nil {
		treatment.ERx.ErxSentDate = &medicationItem.DatePrescribed.DateTime
	}

	if medicationItem.LastDateFilled != nil {
		treatment.ERx.ErxLastDateFilled = &medicationItem.LastDateFilled.DateTime
	}
	return treatment

}
