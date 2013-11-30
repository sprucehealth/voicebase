package integration

import (
	// "carefront/api"
	// "carefront/apiservice"
	// "carefront/config"
	// "encoding/json"
	// _ "github.com/go-sql-driver/mysql"
	// "io/ioutil"
	// "net/http"
	// "net/http/httptest"
	"testing"
)

func TestSingleSelectIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	// dbConfig := GetDBConfig(t)
	// db := ConnectToDB(t, dbConfig)
	// defer db.Close()

	// patientSignedUpResponse := SignupRandomTestPatient(t, dataApi, authApi)
}

func TestMultipleChoiceIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

}

func TestSingleEntryIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

}

func TestFreeTextEntryIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

}

func TestSubQuestionEntryIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

}

func TestMultipleAnswersForSamePotentialAnswerIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

}

func TestPhotoAnswerIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

}
