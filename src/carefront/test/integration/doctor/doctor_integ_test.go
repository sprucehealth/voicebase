package doctor

import (
	"carefront/test/integration"
	"testing"
)

func TestDoctorRegistration(t *testing.T) {
	testData := integration.SetupIntegrationTest(t)
	defer testData.DB.Close()
	SignupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)
}
