package hint

type Client struct {
	Patient      PatientClient
	OAuth        OAuthClient
	Partner      PartnerClient
	Practitioner PractitionerClient
}

var defaultClient = getC()

func getC() *Client {
	return &Client{
		Patient:      &patientClient{B: GetBackend(), Key: Key},
		OAuth:        &oauthClient{B: GetBackend(), Key: Key},
		Partner:      &partnerClient{B: GetBackend(), Key: Key},
		Practitioner: &practitionerClient{B: GetBackend(), Key: Key},
	}
}

// SetPatientClient enables caller to provide a particular implementation of the patient client for mocking purposes.
func SetPatientClient(c PatientClient) {
	defaultClient.Patient = c
}

// SetPatientClient enables caller to provide a particular implementation of the oauth client for mocking purposes.
func SetOAuthClient(c OAuthClient) {
	defaultClient.OAuth = c
}

// SetPartnerClient enables caller to provide a particular implementation of the partner client for mocking purposes.
func SetPartnerClient(c PartnerClient) {
	defaultClient.Partner = c
}

// SetPractitionerClient enables caller to provide a particular implementation of the practitioner client for mocking purposes.
func SetPractitionerClient(c PractitionerClient) {
	defaultClient.Practitioner = c
}

// NewPatient creates a new patient based on the params.
func NewPatient(practiceKey string, params *PatientParams) (*Patient, error) {
	return defaultClient.Patient.New(practiceKey, params)
}

// GetPatient gets an existing patient in the practice account.
func GetPatient(practiceKey, id string) (*Patient, error) {
	return defaultClient.Patient.Get(practiceKey, id)
}

// UpdatePatient updates an existing patient based on the params.
func UpdatePatient(practiceKey, id string, params *PatientParams) (*Patient, error) {
	return defaultClient.Patient.Update(practiceKey, id, params)
}

// DeletePatient deletes a patient based on the id.
func DeletePatient(practiceKey, id string) error {
	return defaultClient.Patient.Delete(practiceKey, id)
}

// ListPatient returns an iterator that can be used to paginate through the list of patients
// based on the iterator.
func ListPatient(practiceKey string, params *ListParams) *Iter {
	return defaultClient.Patient.List(practiceKey, params)
}

// GrantAPIKey exchanges the OAuth token for a practice API key.
func GrantAPIKey(code string) (*PracticeGrant, error) {
	return defaultClient.OAuth.GrantAPIKey(code)
}

// GetPartner returns information about the partner.
func GetPartner() (*Partner, error) {
	return defaultClient.Partner.Get()
}

// UpdatePartner enables updating partner information and returns the updated partner.
func UpdatePartner(params *PartnerParams) (*Partner, error) {
	return defaultClient.Partner.Update(params)
}

// ListAllPractitioner lists all practitioners part of the practice.
func ListAllPractitioners(practiceKey string) ([]*Practitioner, error) {
	return defaultClient.Practitioner.List(practiceKey)
}
