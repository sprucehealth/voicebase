package main

import (
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/test"
)

type mockDiagnosisDetailsIntake struct {
	diagnosisDetailsIFace
	diagnosisDetailsIntakeForCodes map[string]*common.DiagnosisDetailsIntake
}

func (m *mockDiagnosisDetailsIntake) ActiveDiagnosisDetailsIntake(codeID string, types map[string]reflect.Type) (*common.DiagnosisDetailsIntake, error) {
	return m.diagnosisDetailsIntakeForCodes[codeID], nil
}

type mockDataManager struct {
	dataManagerIFace
	visitDoctorPairs                        []*visitDoctorPair
	visitDoctorPairsThatExist               map[visitDoctorPair]bool
	diagnosisIntakeItemsForvisitDoctorPairs map[visitDoctorPair][]*diagnosisIntakeItem

	setsCreated    []*common.VisitDiagnosisSet
	intakesCreated map[visitDoctorPair][]*diagnosisDetailsIntake
}

func (m *mockDataManager) distinctVisitDoctorPairsSince(time time.Time) ([]*visitDoctorPair, error) {
	return m.visitDoctorPairs, nil
}
func (m *mockDataManager) diagnosisSetExistsForPair(p *visitDoctorPair) (bool, error) {
	return m.visitDoctorPairsThatExist[*p], nil
}
func (m *mockDataManager) diagnosisItems(p *visitDoctorPair) ([]*diagnosisIntakeItem, error) {
	return m.diagnosisIntakeItemsForvisitDoctorPairs[*p], nil
}
func (m *mockDataManager) beginTransaction() (*sql.Tx, error) {
	return &sql.Tx{}, nil
}
func (m *mockDataManager) commitTransaction(tx *sql.Tx) error {
	return nil
}
func (m *mockDataManager) rollbackTransaction(tx *sql.Tx) error {
	return nil
}
func (m *mockDataManager) createVisitDiagnosisSet(tx *sql.Tx, set *common.VisitDiagnosisSet) error {
	m.setsCreated = append(m.setsCreated, set)
	return nil
}
func (m *mockDataManager) createDiagnosisDetailsIntake(tx *sql.Tx, intake *diagnosisDetailsIntake) error {
	pair := visitDoctorPair{visitID: intake.visitID, doctorID: intake.doctorID}
	items := m.intakesCreated[pair]
	m.intakesCreated[pair] = append(items, intake)
	return nil
}

var acneQuestionIntakeLayout = diagnosis.NewQuestionIntake([]*info_intake.Question{
	{
		QuestionID:  100,
		QuestionTag: qTagDiagDetailsSeverity,
		PotentialAnswers: []*info_intake.PotentialAnswer{
			{
				AnswerID: 101,
				Answer:   "Mild",
			},
			{
				AnswerID: 102,
				Answer:   "Moderate",
			},
			{
				AnswerID: 103,
				Answer:   "Severe",
			},
		},
	},
	{
		QuestionID:  104,
		QuestionTag: qTagDiagDetailsAcneType,
		PotentialAnswers: []*info_intake.PotentialAnswer{
			{
				AnswerID: 105,
				Answer:   "Inflammatory",
			},
			{
				AnswerID: 106,
				Answer:   "Comedonal",
			},
			{
				AnswerID: 107,
				Answer:   "Hormonal",
			},
			{
				AnswerID: 108,
				Answer:   "Cystic",
			},
		},
	},
})

var perioralDermatitisQuestionIntakeLayout = diagnosis.NewQuestionIntake([]*info_intake.Question{
	{
		QuestionID:  100,
		QuestionTag: qTagDiagDetailsSeverity,
		PotentialAnswers: []*info_intake.PotentialAnswer{
			{
				AnswerID: 101,
				Answer:   "Mild",
			},
			{
				AnswerID: 102,
				Answer:   "Moderate",
			},
			{
				AnswerID: 103,
				Answer:   "Severe",
			},
		},
	},
})

var rosaceaQuestionIntakeLayout = diagnosis.NewQuestionIntake([]*info_intake.Question{
	{
		QuestionID:  100,
		QuestionTag: qTagDiagDetailsSeverity,
		PotentialAnswers: []*info_intake.PotentialAnswer{
			{
				AnswerID: 101,
				Answer:   "Mild",
			},
			{
				AnswerID: 102,
				Answer:   "Moderate",
			},
			{
				AnswerID: 103,
				Answer:   "Severe",
			},
		},
	},
	{
		QuestionID:  109,
		QuestionTag: qTagDiagDetailsRosaceaType,
		PotentialAnswers: []*info_intake.PotentialAnswer{
			{
				AnswerID: 110,
				Answer:   "Erythematotelangiectatic",
			},
			{
				AnswerID: 111,
				Answer:   "Papulopustular",
			},
			{
				AnswerID: 112,
				Answer:   "Rhinophyma",
			},
			{
				AnswerID: 113,
				Answer:   "Ocular",
			},
		},
	},
})

var diagnosisDetailsForCodes = map[string]*common.DiagnosisDetailsIntake{
	"diag_l700": &common.DiagnosisDetailsIntake{
		ID:      10,
		CodeID:  "diag_l700",
		Version: &common.Version{Major: 1, Minor: 1, Patch: 0},
		Layout:  &acneQuestionIntakeLayout,
	},
	"diag_l710": &common.DiagnosisDetailsIntake{
		ID:      11,
		CodeID:  "diag_l710",
		Version: &common.Version{Major: 1, Minor: 1, Patch: 0},
		Layout:  &perioralDermatitisQuestionIntakeLayout,
	},
	"diag_l719": &common.DiagnosisDetailsIntake{
		ID:      12,
		CodeID:  "diag_l719",
		Version: &common.Version{Major: 1, Minor: 1, Patch: 0},
		Layout:  &rosaceaQuestionIntakeLayout,
	},
}

// this test is to ensure that if all pre-existing diagnosis
// were either acne, rosacea and perioral dermatitis, then
// they'd get successfully migrated
func TestMigration_AllStaticMappings(t *testing.T) {

	mild := "Mild"
	moderate := "Moderate"
	severe := "Severe"
	hormonal := "Hormonal"
	cystic := "Cystic"
	rosaceaTypeR := "Rhinophyma"
	rosaceaTypeE := "Erythematotelangiectatic"
	rosaceaTypeO := "Ocular"

	mdm := &mockDataManager{
		visitDoctorPairs: []*visitDoctorPair{
			{
				visitID:  5,
				doctorID: 6,
			},
			{
				visitID:  7,
				doctorID: 8,
			},
			{
				visitID:  9,
				doctorID: 10,
			},
		},
		visitDoctorPairsThatExist: make(map[visitDoctorPair]bool),
		intakesCreated:            make(map[visitDoctorPair][]*diagnosisDetailsIntake),
		diagnosisIntakeItemsForvisitDoctorPairs: map[visitDoctorPair][]*diagnosisIntakeItem{
			visitDoctorPair{visitID: 5, doctorID: 6}: []*diagnosisIntakeItem{
				{
					questionTag:    qTagDiagnosis,
					answerSelected: &aAcneVulgars,
					created:        time.Date(2014, 11, 11, 11, 11, 11, 11, time.UTC),
				},
				{
					questionTag:    qTagDiagnosisSeverity,
					answerSelected: &mild,
					created:        time.Date(2014, 12, 12, 11, 11, 11, 11, time.UTC),
				},
				{
					questionTag:    qTagAcneType,
					answerSelected: &hormonal,
					created:        time.Date(2014, 10, 10, 11, 11, 11, 11, time.UTC),
				},
				{
					questionTag:    qTagAcneType,
					answerSelected: &cystic,
					created:        time.Date(2014, 9, 9, 9, 11, 11, 11, time.UTC),
				},
			},
			visitDoctorPair{visitID: 7, doctorID: 8}: []*diagnosisIntakeItem{
				{
					questionTag:    qTagDiagnosis,
					answerSelected: &aPerioralDermatitis,
					created:        time.Date(2014, 9, 9, 9, 11, 11, 11, time.UTC),
				},
				{
					questionTag:    qTagDiagnosisSeverity,
					answerSelected: &moderate,
					created:        time.Date(2014, 10, 9, 9, 11, 11, 11, time.UTC),
				},
			},
			visitDoctorPair{visitID: 9, doctorID: 10}: []*diagnosisIntakeItem{
				{
					questionTag:    qTagDiagnosis,
					answerSelected: &aRosacea,
					created:        time.Date(2014, 9, 9, 9, 11, 11, 11, time.UTC),
				},
				{
					questionTag:    qTagDiagnosisSeverity,
					answerSelected: &severe,
					created:        time.Date(2014, 12, 12, 11, 11, 11, 11, time.UTC),
				},
				{
					questionTag:    qTagRosaceaType,
					answerSelected: &rosaceaTypeE,
					created:        time.Date(2014, 10, 9, 9, 11, 11, 11, time.UTC),
				},
				{
					questionTag:    qTagRosaceaType,
					answerSelected: &rosaceaTypeR,
					created:        time.Date(2014, 10, 9, 9, 11, 11, 11, time.UTC),
				},

				{
					questionTag:    qTagRosaceaType,
					answerSelected: &rosaceaTypeO,
					created:        time.Date(2014, 10, 9, 9, 11, 11, 11, time.UTC),
				},
			},
		},
	}

	mddi := &mockDiagnosisDetailsIntake{
		diagnosisDetailsIntakeForCodes: diagnosisDetailsForCodes,
	}

	test.OK(t, migrateDiagnosisIntakeToNewModel(&sql.Tx{}, make(map[visitDoctorPair][]string), mdm, mddi))
	test.Equals(t, 3, len(mdm.setsCreated))

	for i, p := range mdm.visitDoctorPairs {
		test.Equals(t, p.visitID, mdm.setsCreated[i].VisitID)
		test.Equals(t, p.doctorID, mdm.setsCreated[i].DoctorID)
		test.Equals(t, mdm.diagnosisIntakeItemsForvisitDoctorPairs[*p][0].created, mdm.setsCreated[i].Created)
		test.Equals(t, 1, len(mdm.setsCreated[i].Items))
		test.Equals(t, len(mdm.diagnosisIntakeItemsForvisitDoctorPairs[*p])-1, len(mdm.intakesCreated[*p]))
	}
}

// This test is to ensure that any visit that was marked as being unsuitable is correctly ported over
func TestMigration_Unsuitable(t *testing.T) {
	unsuitableReason := "foo"
	mdm := &mockDataManager{
		visitDoctorPairs: []*visitDoctorPair{
			{
				visitID:  5,
				doctorID: 6,
			},
		},
		visitDoctorPairsThatExist: make(map[visitDoctorPair]bool),
		intakesCreated:            make(map[visitDoctorPair][]*diagnosisDetailsIntake),
		diagnosisIntakeItemsForvisitDoctorPairs: map[visitDoctorPair][]*diagnosisIntakeItem{
			visitDoctorPair{visitID: 5, doctorID: 6}: []*diagnosisIntakeItem{
				{
					questionTag:    qTagDiagnosis,
					answerSelected: &aNotSuitable,
					created:        time.Date(2014, 11, 11, 11, 11, 11, 11, time.UTC),
				},
				{
					questionTag: qTagNotSuitableReason,
					text:        unsuitableReason,
					created:     time.Date(2014, 12, 12, 11, 11, 11, 11, time.UTC),
				},
			},
		},
	}

	test.OK(t, migrateDiagnosisIntakeToNewModel(&sql.Tx{}, make(map[visitDoctorPair][]string), mdm, nil))
	test.Equals(t, 1, len(mdm.setsCreated))
	test.Equals(t, true, mdm.setsCreated[0].Unsuitable)
	test.Equals(t, unsuitableReason, mdm.setsCreated[0].UnsuitableReason)
	test.Equals(t, mdm.visitDoctorPairs[0].visitID, mdm.setsCreated[0].VisitID)
	test.Equals(t, mdm.visitDoctorPairs[0].doctorID, mdm.setsCreated[0].DoctorID)
	test.Equals(t, mdm.diagnosisIntakeItemsForvisitDoctorPairs[*mdm.visitDoctorPairs[0]][1].created, mdm.setsCreated[0].Created)
}

// This test is to ensure that any visit with a custom diagnosis is correctly ported
// over by picking the diagnosis from the icd10 diagnosis determined by a doctor
func TestMigration_CustomMapping(t *testing.T) {

	diagnosis1 := "d1"
	diagnosis2 := "d2"
	diagnosis3 := "d3,d4,d5"

	mdm := &mockDataManager{
		visitDoctorPairs: []*visitDoctorPair{
			{
				visitID:  5,
				doctorID: 6,
			},
			{
				visitID:  7,
				doctorID: 8,
			},
			{
				visitID:  9,
				doctorID: 10,
			},
		},
		visitDoctorPairsThatExist: make(map[visitDoctorPair]bool),
		intakesCreated:            make(map[visitDoctorPair][]*diagnosisDetailsIntake),
		diagnosisIntakeItemsForvisitDoctorPairs: map[visitDoctorPair][]*diagnosisIntakeItem{
			visitDoctorPair{visitID: 5, doctorID: 6}: []*diagnosisIntakeItem{
				{
					questionTag:    qTagDiagnosis,
					answerSelected: &aOther,
					created:        time.Date(2014, 11, 11, 11, 11, 11, 11, time.UTC),
				},
				{
					questionTag: qTagDiagnosisDescription,
					text:        diagnosis1,
					created:     time.Date(2014, 12, 12, 11, 11, 11, 11, time.UTC),
				},
			},
			visitDoctorPair{visitID: 7, doctorID: 8}: []*diagnosisIntakeItem{
				{
					questionTag:    qTagDiagnosis,
					answerSelected: &aOther,
					created:        time.Date(2014, 11, 11, 11, 11, 11, 11, time.UTC),
				},
				{
					questionTag: qTagDiagnosisDescription,
					text:        diagnosis2,
					created:     time.Date(2014, 12, 12, 11, 11, 11, 11, time.UTC),
				},
			},
			visitDoctorPair{visitID: 9, doctorID: 10}: []*diagnosisIntakeItem{
				{
					questionTag:    qTagDiagnosis,
					answerSelected: &aOther,
					created:        time.Date(2014, 11, 11, 11, 11, 11, 11, time.UTC),
				},
				{
					questionTag: qTagDiagnosisDescription,
					text:        diagnosis3,
					created:     time.Date(2014, 12, 12, 11, 11, 11, 11, time.UTC),
				},
			},
		},
	}

	diagnosisMapping := map[visitDoctorPair][]string{
		visitDoctorPair{visitID: 5, doctorID: 6}:  []string{diagnosis1},
		visitDoctorPair{visitID: 7, doctorID: 8}:  []string{diagnosis2},
		visitDoctorPair{visitID: 9, doctorID: 10}: []string{"d3", "d4", "d5"},
	}

	test.OK(t, migrateDiagnosisIntakeToNewModel(&sql.Tx{}, diagnosisMapping, mdm, nil))
	test.Equals(t, 3, len(mdm.setsCreated))

	test.Equals(t, 1, len(mdm.setsCreated[0].Items))
	test.Equals(t, diagnosis1, mdm.setsCreated[0].Items[0].CodeID)

	test.Equals(t, 1, len(mdm.setsCreated[1].Items))
	test.Equals(t, diagnosis2, mdm.setsCreated[1].Items[0].CodeID)

	test.Equals(t, 3, len(mdm.setsCreated[2].Items))
	test.Equals(t, "d3", mdm.setsCreated[2].Items[0].CodeID)
	test.Equals(t, "d4", mdm.setsCreated[2].Items[1].CodeID)
	test.Equals(t, "d5", mdm.setsCreated[2].Items[2].CodeID)
}

// this test is to ensure that we skip over the completed migrations
func TestMigration_SkipCompletedc(t *testing.T) {
	diagnosis3 := "d3,d4,d5"

	mdm := &mockDataManager{
		visitDoctorPairs: []*visitDoctorPair{
			{
				visitID:  5,
				doctorID: 6,
			},
			{
				visitID:  7,
				doctorID: 8,
			},
			{
				visitID:  9,
				doctorID: 10,
			},
		},
		visitDoctorPairsThatExist: map[visitDoctorPair]bool{
			visitDoctorPair{visitID: 5, doctorID: 6}: true,
			visitDoctorPair{visitID: 7, doctorID: 8}: true,
		},
		intakesCreated: make(map[visitDoctorPair][]*diagnosisDetailsIntake),
		diagnosisIntakeItemsForvisitDoctorPairs: map[visitDoctorPair][]*diagnosisIntakeItem{
			visitDoctorPair{visitID: 9, doctorID: 10}: []*diagnosisIntakeItem{
				{
					questionTag:    qTagDiagnosis,
					answerSelected: &aOther,
					created:        time.Date(2014, 11, 11, 11, 11, 11, 11, time.UTC),
				},
				{
					questionTag: qTagDiagnosisDescription,
					text:        diagnosis3,
					created:     time.Date(2014, 12, 12, 11, 11, 11, 11, time.UTC),
				},
			},
		},
	}

	diagnosisMapping := map[visitDoctorPair][]string{
		visitDoctorPair{visitID: 9, doctorID: 10}: []string{"d3", "d4", "d5"},
	}

	test.OK(t, migrateDiagnosisIntakeToNewModel(&sql.Tx{}, diagnosisMapping, mdm, nil))
	test.Equals(t, 1, len(mdm.setsCreated))

	test.Equals(t, 3, len(mdm.setsCreated[0].Items))
	test.Equals(t, "d3", mdm.setsCreated[0].Items[0].CodeID)
	test.Equals(t, "d4", mdm.setsCreated[0].Items[1].CodeID)
	test.Equals(t, "d5", mdm.setsCreated[0].Items[2].CodeID)

}
