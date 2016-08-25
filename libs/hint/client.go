package hint

type Client struct {
	Patient PatientClient
	OAuth   OAuthClient
}

var defaultClient = getC()

func getC() *Client {
	return &Client{
		Patient: &patientClient{B: GetBackend(), Key: Key},
	}
}

func SetPatientClient(c PatientClient) {
	defaultClient.Patient = c
}

func SetOAuthClient(c OAuthClient) {
	defaultClient.OAuth = c
}

func NewPatient(practiceKey string, params *PatientParams) (*Patient, error) {
	return defaultClient.Patient.New(practiceKey, params)
}

func GetPatient(practiceKey, id string) (*Patient, error) {
	return defaultClient.Patient.Get(practiceKey, id)
}

func UpdatePatient(practiceKey, id string, params *PatientParams) (*Patient, error) {
	return defaultClient.Patient.Update(practiceKey, id, params)
}

func DeletePatient(practiceKey, id string) error {
	return defaultClient.Patient.Delete(practiceKey, id)
}

func ListPatient(practiceKey string, params *ListParams) *Iter {
	return defaultClient.Patient.List(practiceKey, params)
}

func GrantAPIKey(code string) (*PracticeGrant, error) {
	return defaultClient.OAuth.GrantAPIKey(code)
}
