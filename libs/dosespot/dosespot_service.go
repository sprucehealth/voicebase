package dosespot

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/samuel/go-metrics/metrics"
)

type Service struct {
	ClinicID     int64
	ClinicKey    string
	UserID       int64
	SOAPEndpoint string
	APIEndpoint  string
	apiLatencies map[DoseSpotAPIID]metrics.Histogram
	apiSuccess   map[DoseSpotAPIID]*metrics.Counter
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

func New(clinicID, userID int64, clinicKey, soapEndpoint, apiEndpoint string, statsRegistry metrics.Registry) *Service {
	d := &Service{
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
	d.apiSuccess = make(map[DoseSpotAPIID]*metrics.Counter)
	d.apiFailure = make(map[DoseSpotAPIID]*metrics.Counter)
	for id, apiAction := range DoseSpotAPIActions {
		d.apiLatencies[id] = metrics.NewBiasedHistogram()
		d.apiSuccess[id] = metrics.NewCounter()
		d.apiFailure[id] = metrics.NewCounter()
		if statsRegistry != nil {
			statsRegistry.Add(fmt.Sprintf("requests/latency/%s", apiAction), d.apiLatencies[id])
			statsRegistry.Add(fmt.Sprintf("requests/succeeded/%s", apiAction), d.apiSuccess[id])
			statsRegistry.Add(fmt.Sprintf("requests/failed/%s", apiAction), d.apiFailure[id])
		}
	}

	return d
}

func (d *Service) getDoseSpotClient() *soapClient {
	return &soapClient{SoapAPIEndPoint: d.SOAPEndpoint, APIEndpoint: d.APIEndpoint}
}

func (d *Service) GetDrugNamesForDoctor(clinicianID int64, prefix string) ([]string, error) {
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
		d.apiSuccess[medicationQuickSearchAction],
		d.apiFailure[medicationQuickSearchAction])

	if err != nil {
		return nil, err
	}

	return searchResult.DisplayNames, nil
}

func (d *Service) GetDrugNamesForPatient(prefix string) ([]string, error) {
	selfReportedDrugsSearch := &selfReportedMedicationSearchRequest{
		SSO:        generateSingleSignOn(d.ClinicKey, d.UserID, d.ClinicID),
		SearchTerm: prefix,
	}

	searchResult := &selfReportedMedicationSearchResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[selfReportedMedicationSearchAction],
		selfReportedDrugsSearch, searchResult,
		d.apiLatencies[selfReportedMedicationSearchAction],
		d.apiSuccess[selfReportedMedicationSearchAction],
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

func (d *Service) SearchForAllergyRelatedMedications(searchTerm string) ([]string, error) {
	allergySearch := &allergySearchRequest{
		SSO:        generateSingleSignOn(d.ClinicKey, d.UserID, d.ClinicID),
		SearchTerm: searchTerm,
	}

	searchResults := &allergySearchResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[allergySearchAction],
		allergySearch, searchResults,
		d.apiLatencies[allergySearchAction],
		d.apiSuccess[allergySearchAction],
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

func (d *Service) SearchForMedicationStrength(clinicianID int64, medicationName string) ([]string, error) {
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
		d.apiSuccess[medicationStrengthSearchAction],
		d.apiFailure[medicationStrengthSearchAction])

	if err != nil {
		return nil, err
	}

	return searchResult.DisplayStrengths, nil
}

// SendMultiplePrescriptions sends a batch of prescriptions and returns the set of IDs that failed
func (d *Service) SendMultiplePrescriptions(clinicianID, eRxPatientID int64, prescriptionIDs []int64) ([]*SendPrescriptionResult, error) {
	req := &sendMultiplePrescriptionsRequest{
		SSO:             generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
		PatientID:       eRxPatientID,
		PrescriptionIds: prescriptionIDs,
	}
	response := &sendMultiplePrescriptionsResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[sendMultiplPrescriptionsAction],
		req, response,
		d.apiLatencies[sendMultiplPrescriptionsAction],
		d.apiSuccess[sendMultiplPrescriptionsAction],
		d.apiFailure[sendMultiplPrescriptionsAction])
	if err != nil {
		return nil, err
	}
	return response.SendPrescriptionResults, nil
}

func (d *Service) UpdatePatientInformation(clinicianID int64, patient *Patient, pharmacyID int64) ([]*PatientUpdate, error) {
	patientPreferredPharmacy := &patientPharmacySelection{
		IsPrimary:  true,
		PharmacyID: pharmacyID,
	}
	req := &patientStartPrescribingRequest{
		AddFavoritePharmacies: []*patientPharmacySelection{patientPreferredPharmacy},
		Patient:               patient,
		SSO:                   generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
	}
	response := &patientStartPrescribingResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[startPrescribingPatientAction],
		req, response,
		d.apiLatencies[startPrescribingPatientAction],
		d.apiSuccess[startPrescribingPatientAction],
		d.apiFailure[startPrescribingPatientAction])
	return response.PatientUpdates, err
}

func (d *Service) StartPrescribingPatient(clinicianID int64, patient *Patient, prescriptions []*Prescription, pharmacySourceID int64) ([]*PatientUpdate, error) {
	patientPreferredPharmacy := &patientPharmacySelection{
		IsPrimary:  true,
		PharmacyID: pharmacySourceID,
	}
	req := &patientStartPrescribingRequest{
		AddFavoritePharmacies: []*patientPharmacySelection{patientPreferredPharmacy},
		Patient:               patient,
		SSO:                   generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
		AddPrescriptions:      prescriptions,
	}

	response := &patientStartPrescribingResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[startPrescribingPatientAction],
		req, response,
		d.apiLatencies[startPrescribingPatientAction],
		d.apiSuccess[startPrescribingPatientAction],
		d.apiFailure[startPrescribingPatientAction])
	if err != nil {
		return nil, err
	}
	return response.PatientUpdates, nil
}

func (d *Service) SelectMedication(clinicianID int64, medicationName, medicationStrength string) (*MedicationSelectResponse, error) {
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
		d.apiSuccess[medicationSelectAction],
		d.apiFailure[medicationSelectAction])
	if err != nil {
		return nil, err
	}

	if selectResult.LexiGenProductID == 0 && selectResult.LexiDrugSynID == 0 && selectResult.LexiSynonymTypeID == 0 {
		// this drug does not exist
		return nil, nil
	}

	return selectResult, nil
}

func (d *Service) SearchForPharmacies(clinicianID int64, city, state, zipcode, name string, pharmacyTypes []string) ([]*Pharmacy, error) {
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
		d.apiSuccess[searchPharmaciesAction],
		d.apiFailure[searchPharmaciesAction])
	if err != nil {
		return nil, err
	}

	return searchResponse.Pharmacies, nil
}

func (d *Service) GetPrescriptionStatus(clincianID int64, prescriptionID int64) ([]*PrescriptionLogInfo, error) {
	request := &getPrescriptionLogDetailsRequest{
		SSO:            generateSingleSignOn(d.ClinicKey, clincianID, d.ClinicID),
		PrescriptionID: prescriptionID,
	}

	response := &getPrescriptionLogDetailsResult{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[getPrescriptionLogDetailsAction],
		request, response,
		d.apiLatencies[getPrescriptionLogDetailsAction],
		d.apiSuccess[getPrescriptionLogDetailsAction],
		d.apiFailure[getPrescriptionLogDetailsAction])
	return response.Log, err
}

func (d *Service) GetTransmissionErrorDetails(clinicianID int64) ([]*TransmissionErrorDetails, error) {
	request := &getTransmissionErrorDetailsRequest{
		SSO: generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
	}
	response := &getTransmissionErrorDetailsResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[getTransmissionErrorDetailsAction],
		request, response,
		d.apiLatencies[getTransmissionErrorDetailsAction],
		d.apiSuccess[getTransmissionErrorDetailsAction],
		d.apiFailure[getTransmissionErrorDetailsAction])
	if err != nil {
		return nil, err
	}
	return response.TransmissionErrors, nil
}

func (d *Service) GetTransmissionErrorRefillRequestsCount(clinicianID int64) (refillRequests int64, transactionErrors int64, err error) {
	request := &getRefillRequestsTransmissionErrorsMessageRequest{
		SSO:         generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
		ClinicianID: clinicianID,
	}

	response := &getRefillRequestsTransmissionErrorsResult{}
	err = d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[getRefillRequestsTransmissionsErrorsAction],
		request, response,
		d.apiLatencies[getRefillRequestsTransmissionsErrorsAction],
		d.apiSuccess[getRefillRequestsTransmissionsErrorsAction],
		d.apiSuccess[getRefillRequestsTransmissionsErrorsAction])
	if err != nil {
		return 0, 0, err
	}

	if len(response.RefillRequestsTransmissionErrors) == 0 {
		return 0, 0, nil
	}

	return response.RefillRequestsTransmissionErrors[0].RefillRequestsCount, response.RefillRequestsTransmissionErrors[0].TransactionErrorsCount, nil
}

func (d *Service) IgnoreAlert(clinicianID, prescriptionID int64) error {
	request := &ignoreAlertRequest{
		SSO:            generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
		PrescriptionID: prescriptionID,
	}

	response := &ignoreAlertResponse{}
	return d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[ignoreAlertAction], request, response,
		d.apiLatencies[ignoreAlertAction],
		d.apiSuccess[ignoreAlertAction],
		d.apiFailure[ignoreAlertAction])
}

func (d *Service) GetPatientDetails(erxPatientID int64) (*PatientUpdate, error) {
	request := &getPatientDetailRequest{
		SSO:       generateSingleSignOn(d.ClinicKey, d.UserID, d.ClinicID),
		PatientID: erxPatientID,
	}

	response := &getPatientDetailResult{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[getPatientDetailsAction], request, response,
		d.apiLatencies[getPatientDetailsAction],
		d.apiSuccess[getPatientDetailsAction],
		d.apiFailure[getPatientDetailsAction])
	if err != nil {
		return nil, err
	}

	if len(response.PatientUpdates) == 0 {
		return nil, nil
	}
	return response.PatientUpdates[0], nil
}

func (d *Service) GetRefillRequestQueueForClinic(clinicianID int64) ([]*RefillRequestQueueItem, error) {
	request := &getMedicationRefillRequestQueueForClinicRequest{
		SSO: generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
	}
	response := &getMedicationRefillRequestQueueForClinicResult{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[getMedicationRefillRequestQueueForClinicAction], request, response,
		d.apiLatencies[getMedicationRefillRequestQueueForClinicAction],
		d.apiSuccess[getMedicationRefillRequestQueueForClinicAction],
		d.apiFailure[getMedicationRefillRequestQueueForClinicAction])
	if err != nil {
		return nil, err
	}
	return response.RefillRequestQueue, nil
}

func (d *Service) GetPharmacyDetails(pharmacyID int64) (*Pharmacy, error) {
	request := &pharmacyDetailsRequest{
		SSO:        generateSingleSignOn(d.ClinicKey, d.UserID, d.ClinicID),
		PharmacyID: pharmacyID,
	}

	response := &pharmacyDetailsResult{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[pharmacyDetailsAction], request, response,
		d.apiLatencies[pharmacyDetailsAction],
		d.apiSuccess[pharmacyDetailsAction], d.apiFailure[pharmacyDetailsAction])
	if err != nil {
		return nil, err
	}

	return response.PharmacyDetails, nil
}

func (d *Service) ApproveRefillRequest(clinicianID, erxRefillRequestQueueItemID, approvedRefillAmount int64, comments string) (int64, error) {
	request := &approveRefillRequest{
		SSO:                  generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
		RxRequestQueueItemID: erxRefillRequestQueueItemID,
		Refills:              approvedRefillAmount,
		Comments:             comments,
	}

	response := &approveRefillResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[approveRefillAction], request, response,
		d.apiLatencies[approveRefillAction], d.apiSuccess[approveRefillAction], d.apiFailure[approveRefillAction])
	if err != nil {
		return 0, err
	}

	return response.PrescriptionID, nil
}

func (d *Service) DenyRefillRequest(clinicianID, erxRefillRequestQueueItemID int64, denialReason string, comments string) (int64, error) {
	request := &denyRefillRequest{
		SSO:                  generateSingleSignOn(d.ClinicKey, clinicianID, d.ClinicID),
		RxRequestQueueItemID: erxRefillRequestQueueItemID,
		DenialReason:         denialReason,
		Comments:             comments,
	}

	response := &denyRefillResponse{}
	err := d.getDoseSpotClient().makeSoapRequest(DoseSpotAPIActions[denyRefillAction], request, response,
		d.apiLatencies[denyRefillAction], d.apiSuccess[denyRefillAction], d.apiSuccess[denyRefillAction])
	if err != nil {
		return 0, err
	}

	return response.PrescriptionID, nil
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
	// The generic name is at the beginning of the string so find the lowest index in
	// the string for route, form, and strength and truncate the string to it.
	ix := strings.Index(name, m.DoseFormDescription)
	if i := strings.Index(name, m.RouteDescription); ix < 0 || (i >= 0 && i < ix) {
		ix = i
	}
	if i := strings.Index(name, m.StrengthDescription); ix < 0 || (i >= 0 && i < ix) {
		ix = i
	}
	// If none were found then something is terribly wrong
	if ix <= 0 {
		return "", fmt.Errorf("dosespot: no route, form, or strength found for '%s'", name)
	}
	return strings.TrimRightFunc(name[:ix], trimFn), nil
}
