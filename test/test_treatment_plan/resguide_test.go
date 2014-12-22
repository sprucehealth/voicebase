package test_treatment_plan

import (
	"testing"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestTPResourceGuides(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)
	doctorCli := test_integration.DoctorClient(testData, t, doctorID)

	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	tp, err = doctorCli.TreatmentPlan(tp.ID.Int64(), false, doctor_treatment_plan.ResourceGuidesSection)
	test.OK(t, err)
	test.Equals(t, 0, len(tp.ResourceGuides))

	_, guideIDs := createTestResourceGuides(t, testData)

	test.OK(t, doctorCli.AddResourceGuidesToTreatmentPlan(tp.ID.Int64(), guideIDs))

	tp, err = doctorCli.TreatmentPlan(tp.ID.Int64(), false, doctor_treatment_plan.ResourceGuidesSection)
	test.OK(t, err)
	test.Equals(t, 2, len(tp.ResourceGuides))

	// Should be idempotent
	test.OK(t, doctorCli.AddResourceGuidesToTreatmentPlan(tp.ID.Int64(), guideIDs))

	tp, err = doctorCli.TreatmentPlan(tp.ID.Int64(), false, doctor_treatment_plan.ResourceGuidesSection)
	test.OK(t, err)
	test.Equals(t, 2, len(tp.ResourceGuides))

	test.OK(t, doctorCli.RemoveResourceGuideFromTreatmentPlan(tp.ID.Int64(), guideIDs[1]))

	tp, err = doctorCli.TreatmentPlan(tp.ID.Int64(), false, doctor_treatment_plan.ResourceGuidesSection)
	test.OK(t, err)
	test.Equals(t, 1, len(tp.ResourceGuides))

	test.OK(t, doctorCli.AddResourceGuidesToTreatmentPlan(tp.ID.Int64(), guideIDs))
	test.OK(t, testData.DataAPI.RemoveResourceGuidesFromTreatmentPlan(tp.ID.Int64(), guideIDs))
	tp, err = doctorCli.TreatmentPlan(tp.ID.Int64(), false, doctor_treatment_plan.ResourceGuidesSection)
	test.OK(t, err)
	test.Equals(t, 0, len(tp.ResourceGuides))
}

func TestFTPResourceGuides(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)
	doctorCli := test_integration.DoctorClient(testData, t, doctorID)

	_, guideIDs := createTestResourceGuides(t, testData)

	// Create a patient treatment plan, and save a draft message
	visit, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test_integration.AddTreatmentsToTreatmentPlan(tp.ID.Int64(), doctor, t, testData)
	test_integration.AddRegimenPlanForTreatmentPlan(tp.ID.Int64(), doctor, t, testData)
	test.OK(t, doctorCli.AddResourceGuidesToTreatmentPlan(tp.ID.Int64(), guideIDs))

	// Refetch the treatment plan to fill in with recent updates
	tp, err = doctorCli.TreatmentPlan(tp.ID.Int64(), false, doctor_treatment_plan.AllSections)
	test.OK(t, err)
	test.Equals(t, 2, len(tp.ResourceGuides))

	ftp := &common.FavoriteTreatmentPlan{
		Name:          "Test FTP",
		TreatmentList: tp.TreatmentList,
		RegimenPlan:   tp.RegimenPlan,
	}

	// Test creating ftp when resource guides don't match
	_, err = doctorCli.CreateFavoriteTreatmentPlanFromTreatmentPlan(ftp, tp.ID.Int64())
	test.Equals(t, false, err == nil)

	ftp.ResourceGuides = tp.ResourceGuides
	_, err = doctorCli.CreateFavoriteTreatmentPlanFromTreatmentPlan(ftp, tp.ID.Int64())
	test.OK(t, err)

	ftps, err := doctorCli.ListFavoriteTreatmentPlans()
	test.OK(t, err)
	test.Equals(t, 1, len(ftps))
	test.Equals(t, len(tp.ResourceGuides), len(ftps[0].ResourceGuides))

	// Make sure treatment plan created from an ftp that has resource guides also
	// gets the guides.
	tp, err = doctorCli.PickTreatmentPlanForVisit(visit.PatientVisitID, ftps[0])
	test.OK(t, err)
	test.Equals(t, len(ftps[0].ResourceGuides), len(tp.ResourceGuides))

	err = doctorCli.DeleteFavoriteTreatmentPlan(ftps[0].ID.Int64())
	test.OK(t, err)

	err = doctorCli.DeleteTreatmentPlan(tp.ID.Int64())
	test.OK(t, err)
}

func createTestResourceGuides(t *testing.T, testData *test_integration.TestData) (int64, []int64) {
	secID, err := testData.DataAPI.CreateResourceGuideSection(&common.ResourceGuideSection{
		Ordinal: 1,
		Title:   "Test Section",
	})
	test.OK(t, err)

	guide1ID, err := testData.DataAPI.CreateResourceGuide(&common.ResourceGuide{
		SectionID: secID,
		Ordinal:   1,
		Title:     "Guide 1",
		PhotoURL:  "http://example.com/blah.png",
		Layout:    &struct{}{},
	})
	test.OK(t, err)

	guide2ID, err := testData.DataAPI.CreateResourceGuide(&common.ResourceGuide{
		SectionID: secID,
		Ordinal:   2,
		Title:     "Guide 2",
		PhotoURL:  "http://example.com/blah.png",
		Layout:    &struct{}{},
	})
	test.OK(t, err)

	return secID, []int64{guide1ID, guide2ID}
}
