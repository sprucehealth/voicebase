package dosespot

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/samuel/go-metrics/metrics"
)

type Service struct {
	clinicID     int64
	clinicKey    string
	userID       int64
	client       *soapClient
	apiLatencies map[doseSpotAPIID]metrics.Histogram
	apiSuccess   map[doseSpotAPIID]*metrics.Counter
	apiFailure   map[doseSpotAPIID]*metrics.Counter
}

type doseSpotAPIID int

const (
	medicationQuickSearchAction doseSpotAPIID = iota
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

var doseSpotAPIActions = map[doseSpotAPIID]string{
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
)

func New(clinicID, userID int64, clinicKey, soapEndpoint, apiEndpoint string, statsRegistry metrics.Registry) *Service {
	d := &Service{
		client: &soapClient{SoapAPIEndPoint: soapEndpoint, APIEndpoint: apiEndpoint},
	}
	if clinicID == 0 {
		d.clinicKey = os.Getenv("DOSESPOT_CLINIC_KEY")
		d.clinicID, _ = strconv.ParseInt(os.Getenv("DOSESPOT_CLINIC_ID"), 10, 64)
		d.userID, _ = strconv.ParseInt(os.Getenv("DOSESPOT_USER_ID"), 10, 64)
	} else {
		d.clinicKey = clinicKey
		d.clinicID = clinicID
		d.userID = userID
	}

	d.apiLatencies = make(map[doseSpotAPIID]metrics.Histogram)
	d.apiSuccess = make(map[doseSpotAPIID]*metrics.Counter)
	d.apiFailure = make(map[doseSpotAPIID]*metrics.Counter)
	for id, apiAction := range doseSpotAPIActions {
		d.apiLatencies[id] = metrics.NewUnbiasedHistogram()
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

func (d *Service) makeSoapRequest(action doseSpotAPIID, req interface{}, res response) error {
	return d.client.makeSoapRequest(doseSpotAPIActions[action], req, res, d.apiLatencies[action], d.apiSuccess[action], d.apiFailure[action])
}

func (d *Service) GetDrugNamesForDoctor(clinicianID int64, prefix string) ([]string, error) {
	if clinicianID <= 0 {
		clinicianID = d.userID
	}
	req := &medicationQuickSearchRequest{
		SSO:          generateSingleSignOn(d.clinicKey, clinicianID, d.clinicID),
		SearchString: prefix,
	}
	res := &medicationQuickSearchResponse{}
	if err := d.makeSoapRequest(medicationQuickSearchAction, req, res); err != nil {
		return nil, err
	}
	return res.DisplayNames, nil
}

func (d *Service) GetDrugNamesForPatient(prefix string) ([]string, error) {
	req := &selfReportedMedicationSearchRequest{
		SSO:        generateSingleSignOn(d.clinicKey, d.userID, d.clinicID),
		SearchTerm: prefix,
	}
	res := &selfReportedMedicationSearchResponse{}
	if err := d.makeSoapRequest(selfReportedMedicationSearchAction, req, res); err != nil {
		return nil, err
	}
	drugNames := make([]string, len(res.SearchResults))
	for i, searchResultItem := range res.SearchResults {
		drugNames[i] = searchResultItem.DisplayName
	}
	return drugNames, nil
}

func (d *Service) SearchForAllergyRelatedMedications(searchTerm string) ([]string, error) {
	req := &allergySearchRequest{
		SSO:        generateSingleSignOn(d.clinicKey, d.userID, d.clinicID),
		SearchTerm: searchTerm,
	}
	res := &allergySearchResponse{}
	if err := d.makeSoapRequest(allergySearchAction, req, res); err != nil {
		return nil, err
	}
	names := make([]string, len(res.SearchResults))
	for i, searchResultItem := range res.SearchResults {
		names[i] = searchResultItem.Name
	}
	return names, nil
}

func (d *Service) SearchForMedicationStrength(clinicianID int64, medicationName string) ([]string, error) {
	if clinicianID <= 0 {
		clinicianID = d.userID
	}
	req := &medicationStrengthSearchRequest{
		SSO:            generateSingleSignOn(d.clinicKey, clinicianID, d.clinicID),
		MedicationName: medicationName,
	}
	res := &medicationStrengthSearchResponse{}
	if err := d.makeSoapRequest(medicationStrengthSearchAction, req, res); err != nil {
		return nil, err
	}
	return res.DisplayStrengths, nil
}

// SendMultiplePrescriptions sends a batch of prescriptions and returns the set of IDs that failed
func (d *Service) SendMultiplePrescriptions(clinicianID, eRxPatientID int64, prescriptionIDs []int64) ([]*SendPrescriptionResult, error) {
	req := &sendMultiplePrescriptionsRequest{
		SSO:             generateSingleSignOn(d.clinicKey, clinicianID, d.clinicID),
		PatientID:       eRxPatientID,
		PrescriptionIds: prescriptionIDs,
	}
	res := &sendMultiplePrescriptionsResponse{}
	if err := d.makeSoapRequest(sendMultiplPrescriptionsAction, req, res); err != nil {
		return nil, err
	}
	return res.SendPrescriptionResults, nil
}

func (d *Service) UpdatePatientInformation(clinicianID int64, patient *Patient, pharmacyID int64) ([]*PatientUpdate, error) {
	patientPreferredPharmacy := &patientPharmacySelection{
		IsPrimary:  true,
		PharmacyID: pharmacyID,
	}
	req := &patientStartPrescribingRequest{
		AddFavoritePharmacies: []*patientPharmacySelection{patientPreferredPharmacy},
		Patient:               patient,
		SSO:                   generateSingleSignOn(d.clinicKey, clinicianID, d.clinicID),
	}
	res := &patientStartPrescribingResponse{}
	if err := d.makeSoapRequest(startPrescribingPatientAction, req, res); err != nil {
		return nil, err
	}
	return res.PatientUpdates, nil
}

func (d *Service) StartPrescribingPatient(clinicianID int64, patient *Patient, prescriptions []*Prescription, pharmacyID int64) ([]*PatientUpdate, error) {
	patientPreferredPharmacy := &patientPharmacySelection{
		IsPrimary:  true,
		PharmacyID: pharmacyID,
	}
	req := &patientStartPrescribingRequest{
		AddFavoritePharmacies: []*patientPharmacySelection{patientPreferredPharmacy},
		Patient:               patient,
		SSO:                   generateSingleSignOn(d.clinicKey, clinicianID, d.clinicID),
		AddPrescriptions:      prescriptions,
	}
	res := &patientStartPrescribingResponse{}
	if err := d.makeSoapRequest(startPrescribingPatientAction, req, res); err != nil {
		return nil, err
	}
	return res.PatientUpdates, nil
}

func (d *Service) SelectMedication(clinicianID int64, medicationName, medicationStrength string) (*MedicationSelectResponse, error) {
	if clinicianID <= 0 {
		clinicianID = d.userID
	}
	req := &medicationSelectRequest{
		SSO:                generateSingleSignOn(d.clinicKey, clinicianID, d.clinicID),
		MedicationName:     medicationName,
		MedicationStrength: medicationStrength,
	}
	res := &MedicationSelectResponse{}
	if err := d.makeSoapRequest(medicationSelectAction, req, res); err != nil {
		return nil, err
	}
	if res.LexiGenProductID == 0 && res.LexiDrugSynID == 0 && res.LexiSynonymTypeID == 0 {
		// this drug does not exist
		return nil, nil
	}
	return res, nil
}

func (d *Service) SearchForPharmacies(clinicianID int64, city, state, zipcode, name string, pharmacyTypes []string) ([]*Pharmacy, error) {
	req := &pharmacySearchRequest{
		PharmacyCity:            city,
		PharmacyStateTwoLetters: state,
		PharmacyZipCode:         zipcode,
		PharmacyNameSearch:      name,
		SSO:                     generateSingleSignOn(d.clinicKey, clinicianID, d.clinicID),
		PharmacyTypes:           pharmacyTypes,
	}
	res := &pharmacySearchResult{}
	if err := d.makeSoapRequest(searchPharmaciesAction, req, res); err != nil {
		return nil, err
	}
	return res.Pharmacies, nil
}

func (d *Service) GetPrescriptionStatus(clincianID int64, prescriptionID int64) ([]*PrescriptionLogInfo, error) {
	req := &getPrescriptionLogDetailsRequest{
		SSO:            generateSingleSignOn(d.clinicKey, clincianID, d.clinicID),
		PrescriptionID: prescriptionID,
	}
	res := &getPrescriptionLogDetailsResult{}
	if err := d.makeSoapRequest(getPrescriptionLogDetailsAction, req, res); err != nil {
		return nil, err
	}
	return res.Log, nil
}

func (d *Service) GetTransmissionErrorDetails(clinicianID int64) ([]*TransmissionErrorDetails, error) {
	req := &getTransmissionErrorDetailsRequest{
		SSO: generateSingleSignOn(d.clinicKey, clinicianID, d.clinicID),
	}
	res := &getTransmissionErrorDetailsResponse{}
	if err := d.makeSoapRequest(getTransmissionErrorDetailsAction, req, res); err != nil {
		return nil, err
	}
	return res.TransmissionErrors, nil
}

func (d *Service) GetTransmissionErrorRefillRequestsCount(clinicianID int64) (refillRequests int64, transactionErrors int64, err error) {
	req := &getRefillRequestsTransmissionErrorsMessageRequest{
		SSO:         generateSingleSignOn(d.clinicKey, clinicianID, d.clinicID),
		ClinicianID: clinicianID,
	}
	res := &getRefillRequestsTransmissionErrorsResult{}
	if err := d.makeSoapRequest(getRefillRequestsTransmissionsErrorsAction, req, res); err != nil {
		return 0, 0, err
	}
	if len(res.RefillRequestsTransmissionErrors) == 0 {
		return 0, 0, nil
	}
	return res.RefillRequestsTransmissionErrors[0].RefillRequestsCount, res.RefillRequestsTransmissionErrors[0].TransactionErrorsCount, nil
}

func (d *Service) IgnoreAlert(clinicianID, prescriptionID int64) error {
	req := &ignoreAlertRequest{
		SSO:            generateSingleSignOn(d.clinicKey, clinicianID, d.clinicID),
		PrescriptionID: prescriptionID,
	}
	res := &ignoreAlertResponse{}
	return d.makeSoapRequest(ignoreAlertAction, req, res)
}

func (d *Service) GetPatientDetails(erxPatientID int64) (*PatientUpdate, error) {
	req := &getPatientDetailRequest{
		SSO:       generateSingleSignOn(d.clinicKey, d.userID, d.clinicID),
		PatientID: erxPatientID,
	}
	res := &getPatientDetailResult{}
	if err := d.makeSoapRequest(getPatientDetailsAction, req, res); err != nil {
		return nil, err
	}
	if len(res.PatientUpdates) == 0 {
		return nil, nil
	}
	return res.PatientUpdates[0], nil
}

func (d *Service) GetRefillRequestQueueForClinic(clinicianID int64) ([]*RefillRequestQueueItem, error) {
	req := &getMedicationRefillRequestQueueForClinicRequest{
		SSO: generateSingleSignOn(d.clinicKey, clinicianID, d.clinicID),
	}
	res := &getMedicationRefillRequestQueueForClinicResult{}
	if err := d.makeSoapRequest(getMedicationRefillRequestQueueForClinicAction, req, res); err != nil {
		return nil, err
	}
	return res.RefillRequestQueue, nil
}

func (d *Service) GetPharmacyDetails(pharmacyID int64) (*Pharmacy, error) {
	req := &pharmacyDetailsRequest{
		SSO:        generateSingleSignOn(d.clinicKey, d.userID, d.clinicID),
		PharmacyID: pharmacyID,
	}
	res := &pharmacyDetailsResult{}
	if err := d.makeSoapRequest(pharmacyDetailsAction, req, res); err != nil {
		return nil, err
	}
	return res.PharmacyDetails, nil
}

func (d *Service) ApproveRefillRequest(clinicianID, erxRefillRequestQueueItemID, approvedRefillAmount int64, comments string) (int64, error) {
	req := &approveRefillRequest{
		SSO:                  generateSingleSignOn(d.clinicKey, clinicianID, d.clinicID),
		RxRequestQueueItemID: erxRefillRequestQueueItemID,
		Refills:              approvedRefillAmount,
		Comments:             comments,
	}
	res := &approveRefillResponse{}
	if err := d.makeSoapRequest(approveRefillAction, req, res); err != nil {
		return 0, err
	}
	return res.PrescriptionID, nil
}

func (d *Service) DenyRefillRequest(clinicianID, erxRefillRequestQueueItemID int64, denialReason string, comments string) (int64, error) {
	req := &denyRefillRequest{
		SSO:                  generateSingleSignOn(d.clinicKey, clinicianID, d.clinicID),
		RxRequestQueueItemID: erxRefillRequestQueueItemID,
		DenialReason:         denialReason,
		Comments:             comments,
	}
	res := &denyRefillResponse{}
	if err := d.makeSoapRequest(denyRefillAction, req, res); err != nil {
		return 0, err
	}
	return res.PrescriptionID, nil
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
