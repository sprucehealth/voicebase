package info_intake

import (
	"encoding/json"
	"io/ioutil"
	"testing"
)

type PatientVisitReview struct {
	VisitOverview *PatientVisitOverview `json:"patient_visit_overview"`
}

func parseFileToGetPatientVisitOverview(t *testing.T) (visitOverview *PatientVisitOverview) {
	fileContents, err := ioutil.ReadFile("../api-response-examples/v1/doctor/visit/review.json")
	if err != nil {
		t.Fatal("Unable to open the json representation of the patient visit for testing:" + err.Error())
	}
	patientVisitReview := &PatientVisitReview{}
	err = json.Unmarshal(fileContents, &patientVisitReview)
	visitOverview = patientVisitReview.VisitOverview
	if err != nil {
		t.Fatal("Unable to parse the json representation of a patient visit :" + err.Error())
	}
	return
}

func TestParsingOfPatientVisitOverview(t *testing.T) {
	parseFileToGetPatientVisitOverview(t)
}

func TestInitialPatientVisitInformation(t *testing.T) {
	visitReview := parseFileToGetPatientVisitOverview(t)
	if visitReview.PatientVisitId == 0 {
		t.Fatal("Patient visit overview does not contain patient visit id when it should")
	}
	if visitReview.PatientVisitTime.IsZero() {
		t.Fatal("Patient visit overview does not contain patient visit time when it should")
	}

	if visitReview.HealthConditionId == 0 {
		t.Fatal("Patient visit overview does not contain health condition id when it should")
	}
}

func TestSectionsInPatientVisitReviewParsing(t *testing.T) {
	visitReview := parseFileToGetPatientVisitOverview(t)
	if visitReview.Sections == nil || len(visitReview.Sections) == 0 {
		t.Fatal("No sections present inside the patient visit review")
	}

	for _, section := range visitReview.Sections {
		if section.SectionTitle == "" {
			t.Fatal("Section title should be set for section")
		}
		if section.SectionTypes == nil || len(section.SectionTypes) == 0 {
			t.Fatal("Section types should be set for section")
		}
	}
}

func TestSubSectionsInPatientVisitReviewParsing(t *testing.T) {
	visitReview := parseFileToGetPatientVisitOverview(t)
	for _, section := range visitReview.Sections {
		if section.SubSections == nil || len(section.SubSections) == 0 {
			t.Fatal("Every section must have a subsection")
		}

		for _, subSection := range section.SubSections {
			if subSection.SubSectionTypes == nil || len(subSection.SubSectionTypes) == 0 {
				t.Fatal("Every sub section must have a type. Inside section " + section.SectionTitle)
			}
		}
	}
}
