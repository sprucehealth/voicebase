package test_integration

import (
	"runtime/debug"
	"testing"
)

// Object that should contain common assertions made in integration tests
type Assertion struct {
	testData *TestData
	t        *testing.T
}

func NewAssertion(testData *TestData, t *testing.T) *Assertion {
	return &Assertion{
		testData: testData,
		t:        t,
	}
}

func (i *Assertion) ProviderIsAssignedToCase(patientCaseID, providerID int64, expectedStatus string) {
	doctorAssignments, err := i.testData.DataAPI.GetDoctorsAssignedToPatientCase(patientCaseID)
	if err != nil {
		i.Fatal(err)
	}
	for _, v := range doctorAssignments {
		if v.ProviderID == providerID {
			if doctorAssignments[0].Status != expectedStatus {
				i.Fatalf("Expected the doctor to have status %v but it had %v", expectedStatus, doctorAssignments[0].Status)
			}
			return
		}
		i.Fatalf("Expected doctor %v to be assigned to case but was not", providerID)
	}
}

func (i *Assertion) ProviderIsMemberOfCareTeam(patientID, providerID, caseID int64, expectedStatus string) {
	careTeams, err := i.testData.DataAPI.GetCareTeamsForPatientByCase(patientID)
	if err != nil {
		i.Fatal(err)
	}

	if careTeam, ok := careTeams[caseID]; ok {
		if len(careTeam.Assignments) != 1 {
			i.Fatalf("Expected at least 1 doctor to exist in care team instead got %d", len(careTeam.Assignments))
		}
		for _, v := range careTeam.Assignments {
			if v.ProviderID == providerID {
				if v.Status != expectedStatus {
					i.Fatalf("Found doctor as memver of care team - expected status %v but got %v", v.Status, expectedStatus)
				}
				return
			}
		}
		i.Fatalf("The expected doctor %v was not found in the care team mapped to case %v", providerID, caseID)
	} else {
		i.Fatalf("No care team exists for case %v", caseID)
	}
}

func (i *Assertion) CaseStatusFromVisitIs(patientVisitId int64, expectedStatus string) {
	patientCase, err := i.testData.DataAPI.GetPatientCaseFromPatientVisitID(patientVisitId)
	if err != nil {
		i.Fatal(err)
	} else if patientCase == nil {
		i.Fatalf("Expected to find case associated with visit %v but did not", patientVisitId)
	} else if patientCase.Status != expectedStatus {
		i.Fatalf("Expected patient case to be %s but it was %s", expectedStatus, patientCase.Status)
	}
}

func (i *Assertion) Fatalf(format string, args ...interface{}) {
	debug.PrintStack()
	i.t.Fatalf(format, args...)
}

func (i *Assertion) Fatal(f interface{}) {
	debug.PrintStack()
	i.t.Fatal(f)
}
