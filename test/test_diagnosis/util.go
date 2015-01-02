package test_diagnosis

import (
	"os"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/test"
)

func setupDiagnosisService(t *testing.T) diagnosis.API {
	diagnosisDBHost := os.Getenv("CF_LOCAL_DIAGNOSIS_DB_INSTANCE")
	if diagnosisDBHost == "" {
		t.Skipf("Skipping test for now given that diagnosis db is not setup")
	}
	diagnosisDBUsername := os.Getenv("CF_LOCAL_DIAGNOSIS_DB_USERNAME")
	diagnosisDBPassword := os.Getenv("CF_LOCAL_DIAGNOSIS_DB_PASSWORD")
	diagnosisDBName := os.Getenv("CF_LOCAL_DIAGNOSIS_DB_NAME")
	diagnosisDBPort, err := strconv.Atoi(os.Getenv("CF_LOCAL_DIAGNOSIS_DB_PORT"))
	test.OK(t, err)

	diagnosisService, err := diagnosis.NewService(&config.DB{
		User:     diagnosisDBUsername,
		Password: diagnosisDBPassword,
		Host:     diagnosisDBHost,
		Port:     diagnosisDBPort,
		Name:     diagnosisDBName,
	})
	test.OK(t, err)
	return diagnosisService
}
