package test_doctor

import (
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestDoctorTransaction_TreatmentPlanCreated(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test.OK(t, err)

	// lets get the doctor to submit the treatment plan
	test_integration.SubmitPatientVisitBackToPatient(tp.ID.Int64(), doctor, testData, t)

	transactions, err := testData.DataAPI.TransactionsForDoctor(dr.DoctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(transactions))
	test.Equals(t, tp.PatientID, transactions[0].PatientID)
	test.Equals(t, (*int64)(nil), transactions[0].ItemCostID)

	// lets go ahead and version the treatment plan
	dTP := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   tp.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, nil, doctor, testData, t)

	// lets go and submit this treatment plan
	test_integration.SubmitPatientVisitBackToPatient(dTP.TreatmentPlan.ID.Int64(), doctor, testData, t)

	// at this point there should still only be 1 transaction for the doctor
	transactions, err = testData.DataAPI.TransactionsForDoctor(dr.DoctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(transactions))
}

func TestDoctorTransaction_ItemCostExists_TreatmentPlanCreated(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()

	patientVisit, stubSQSQueue, _ := test_integration.SetupTestWithActiveCostAndVisitSubmitted(testData, t)
	// now lets go ahead and start the work to consume the message
	stubStripe := testData.Config.PaymentAPI.(*test_integration.StripeStub)
	stubStripe.CreateChargeFunc = func(req *stripe.CreateChargeRequest) (*stripe.Charge, error) {
		return &stripe.Charge{
			ID: "charge_test",
		}, nil
	}

	// set an exceptionally high time period (1 day) so that the worker only runs once
	w := cost.NewWorker(testData.DataAPI, testData.Config.AnalyticsLogger, testData.Config.Dispatcher, stubStripe, nil, stubSQSQueue, metrics.NewRegistry(), 24*60*60, "", nil)
	w.Do()

	dr := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	test_integration.GrantDoctorAccessToPatientCase(t, testData, doctor, patientVisit.PatientCaseID.Int64())
	test_integration.StartReviewingPatientVisit(patientVisit.ID.Int64(), doctor, testData, t)

	dTP := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   patientVisit.ID,
		ParentType: common.TPParentTypePatientVisit,
	}, nil, doctor, testData, t)

	// lets get the doctor to submit the treatment plan
	test_integration.SubmitPatientVisitBackToPatient(dTP.TreatmentPlan.ID.Int64(), doctor, testData, t)

	transactions, err := testData.DataAPI.TransactionsForDoctor(dr.DoctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(transactions))
	test.Equals(t, dTP.TreatmentPlan.PatientID, transactions[0].PatientID)
	test.Equals(t, true, *transactions[0].ItemCostID > 0)
}

func TestDoctorTransaction_MarkedUnsuitable(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	pv, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test.OK(t, err)

	// lets mark the visit as being unsuitable for spruce
	answerBody := test_integration.PrepareAnswersForDiagnosingAsUnsuitableForSpruce(testData, t, pv.PatientVisitID)
	test_integration.SubmitPatientVisitDiagnosisWithIntake(pv.PatientVisitID, doctor.AccountID.Int64(), answerBody, testData, t)

	tranasactions, err := testData.DataAPI.TransactionsForDoctor(dr.DoctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(tranasactions))
	test.Equals(t, tp.PatientID, tranasactions[0].PatientID)
}
