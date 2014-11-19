package test_followup

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/app_event"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost"
	"github.com/sprucehealth/backend/libs/aws/sqs"
	"github.com/sprucehealth/backend/libs/stripe"
	patientpkg "github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/patient_case"
	"github.com/sprucehealth/backend/sku"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestFollowup_CreateAndSubmit(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	stubSQSQueue := &common.SQSQueue{
		QueueUrl:     "visit_url",
		QueueService: &sqs.StubSQS{},
	}
	testData.Config.VisitQueue = stubSQSQueue
	testData.StartAPIServer(t)

	setupFollowupTest(t, testData)

	// create doctor
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)

	// create and submit visit for patient\
	pv, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	patient, err := testData.DataApi.GetPatientFromId(tp.PatientId)
	test.OK(t, err)
	patientID := patient.PatientId.Int64()
	patientAccountID := patient.AccountId.Int64()
	test_integration.AddCreditCardForPatient(patientID, testData, t)

	// ensure that a followup cannot be created until the initial visit has been treated
	_, err = patientpkg.CreatePendingFollowup(patient, testData.DataApi, testData.AuthApi, testData.Config.Dispatcher, testData.Config.Stores["media"], testData.Config.AuthTokenExpiration)
	test.Equals(t, patientpkg.InitialVisitNotTreated, err)

	// now lets treat the initial visit
	test_integration.SubmitPatientVisitDiagnosis(pv.PatientVisitId, doctor, testData, t)
	test_integration.SubmitPatientVisitBackToPatient(tp.Id.Int64(), doctor, testData, t)

	// lets wait for a moment so as to let a second elapse before creating the next followup
	// so that there is time between the creation of the initial visit and the followup
	time.Sleep(time.Second)

	// now lets try to create a followup visit
	_, err = patientpkg.CreatePendingFollowup(patient, testData.DataApi, testData.AuthApi, testData.Config.Dispatcher, testData.Config.Stores["media"], testData.Config.AuthTokenExpiration)
	test.OK(t, err)

	// at this point there should be two visits in the case for the patient
	visits, err := testData.DataApi.GetVisitsForCase(tp.PatientCaseId.Int64(), nil)
	test.OK(t, err)
	test.Equals(t, 2, len(visits))

	followupVisit := visits[0]
	test.Equals(t, sku.AcneFollowup, followupVisit.SKU)
	// the followup visit should have its state as pending
	// as the patient has not viewed it yet
	test.Equals(t, common.PVStatusPending, followupVisit.Status)

	// lets query for the visit to have its status update to OPEN
	pv = test_integration.QueryPatientVisit(
		followupVisit.PatientVisitId.Int64(),
		patientAccountID,
		map[string]string{
			"S-Version": "Patient;Test;1.0.0;0001",
			"S-OS":      "iOS;7.1",
			"S-Device":  "Phone;iPhone6,1;640;1136;2.0",
		},
		testData,
		t)
	test.Equals(t, common.PVStatusOpen, pv.Status)

	// lets generate an app event to indicate that we have viewed the treatment plan so that the
	// notification is cleared
	test_integration.GenerateAppEvent(app_event.ViewedAction, "treatment_plan", tp.Id.Int64(), patientAccountID, testData, t)

	// at this point there should be a case notification that
	// encourages the patient to complete their followup visit
	caseNotifications, err := testData.DataApi.GetNotificationsForCase(followupVisit.PatientCaseId.Int64(), patient_case.NotifyTypes)
	test.OK(t, err)
	test.Equals(t, 1, len(caseNotifications))
	test.Equals(t, patient_case.CNIncompleteFollowup, caseNotifications[0].NotificationType)

	// before submitting the response lets query the cost for the followup visit
	value, lineItems := test_integration.QueryCost(patientAccountID, sku.AcneFollowup, testData, t)
	test.Equals(t, "$20", value)
	test.Equals(t, 1, len(lineItems))

	// now lets go ahead and submit responses to the visit
	answerIntakeBody := test_integration.PrepareAnswersForQuestionsInPatientVisit(pv, t)
	test_integration.SubmitAnswersIntakeForPatient(patientID, patientAccountID, answerIntakeBody, testData, t)

	// now lets go ahead and submit the visit to the doctor. This should route the followup visit
	// directly to the doctor on the care team of the patient
	submitVisit(patientID, pv.PatientVisitId, stubSQSQueue, testData, t)

	// at this point the case notification should indicate that the patient has submitted their visit
	caseNotifications, err = testData.DataApi.GetNotificationsForCase(followupVisit.PatientCaseId.Int64(), patient_case.NotifyTypes)
	test.OK(t, err)
	test.Equals(t, 1, len(caseNotifications))
	test.Equals(t, patient_case.CNVisitSubmitted, caseNotifications[0].NotificationType)

	// that being said, the visit submitted notification should not be displayed inside the case details page
	res, err := testData.AuthGet(testData.APIServer.URL+router.PatientCaseNotificationsURLPath+"?case_id="+strconv.FormatInt(followupVisit.PatientCaseId.Int64(), 10), patientAccountID)
	test.OK(t, err)
	defer res.Body.Close()
	var resData map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&resData)
	test.OK(t, err)
	items := resData["items"].([]interface{})
	test.Equals(t, 0, len(items))

	// at this point the patient visit should be in the routed state
	followupVisit, err = testData.DataApi.GetPatientVisitFromId(followupVisit.PatientVisitId.Int64())
	test.OK(t, err)
	test.Equals(t, common.PVStatusRouted, followupVisit.Status)

	// at this point there should be a new receipt for the patient pertaining to the followup
	patientReceipt, err := testData.DataApi.GetPatientReceipt(patientID, pv.PatientVisitId, sku.AcneFollowup, true)
	test.OK(t, err)
	test.Equals(t, true, patientReceipt != nil)
	patientReceipt.CostBreakdown.CalculateTotal()
	test.Equals(t, 2000, patientReceipt.CostBreakdown.TotalCost.Amount)

	// at this point the doctor should have a pending item in their inbox
	pendingItems, err := testData.DataApi.GetPendingItemsInDoctorQueue(doctor.DoctorId.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(pendingItems))
	test.Equals(t, followupVisit.PatientVisitId.Int64(), pendingItems[0].ItemId)
	test.Equals(t, api.DQEventTypePatientVisit, pendingItems[0].EventType)

	// lets get the doctor to start revieiwng the visit
	test_integration.StartReviewingPatientVisit(followupVisit.PatientVisitId.Int64(), doctor, testData, t)

	// at this point the visit should be in reviewing state
	followupVisit, err = testData.DataApi.GetPatientVisitFromId(followupVisit.PatientVisitId.Int64())
	test.OK(t, err)
	test.Equals(t, common.PVStatusReviewing, followupVisit.Status)

	// now lets get the doctor to submit diagnosis for the followup visit
	test_integration.SubmitPatientVisitDiagnosis(followupVisit.PatientVisitId.Int64(), doctor, testData, t)

	// start treatment plan
	newTP := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentType: common.TPParentTypeTreatmentPlan,
		ParentId:   tp.Id,
	}, nil, doctor, testData, t)

	// add treatments
	test_integration.AddTreatmentsToTreatmentPlan(newTP.TreatmentPlan.Id.Int64(), doctor, t, testData)

	// add regimen steps
	test_integration.AddRegimenPlanForTreatmentPlan(newTP.TreatmentPlan.Id.Int64(), doctor, t, testData)

	// now lets go ahead and submit the treatment plan to the patient
	test_integration.SubmitPatientVisitBackToPatient(newTP.TreatmentPlan.Id.Int64(), doctor, testData, t)

	// at this point there should be a message notification for the patient
	caseNotifications, err = testData.DataApi.GetNotificationsForCase(newTP.TreatmentPlan.PatientCaseId.Int64(), patient_case.NotifyTypes)
	test.OK(t, err)
	test.Equals(t, 1, len(caseNotifications))
	test.Equals(t, patient_case.CNMessage, caseNotifications[0].NotificationType)

	// there should no longer be an item in the pending list for the doctor, but there should be an item in the completed list
	pendingItems, err = testData.DataApi.GetPendingItemsInDoctorQueue(doctor.DoctorId.Int64())
	test.OK(t, err)
	test.Equals(t, 0, len(pendingItems))

	completedItems, err := testData.DataApi.GetCompletedItemsInDoctorQueue(doctor.DoctorId.Int64())
	test.OK(t, err)
	test.Equals(t, 2, len(completedItems))

	// followup visit should be in treated state
	followupVisit, err = testData.DataApi.GetPatientVisitFromId(followupVisit.PatientVisitId.Int64())
	test.OK(t, err)
	test.Equals(t, common.PVStatusTreated, followupVisit.Status)

	// there should be a doctor transaction for treating the followup visit
	transactions, err := testData.DataApi.TransactionsForDoctor(doctor.DoctorId.Int64())
	test.OK(t, err)
	test.Equals(t, 2, len(transactions))
	test.Equals(t, sku.AcneFollowup, transactions[0].ItemType)
	test.Equals(t, sku.AcneVisit, transactions[1].ItemType)
}

func TestFollowup_LayoutVersionUpdateOnRead(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// create doctor
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)

	// create and submit visit for patient\
	pv, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	patient, err := testData.DataApi.GetPatientFromId(tp.PatientId)
	test.OK(t, err)
	patientID := patient.PatientId.Int64()
	patientAccountID := patient.AccountId.Int64()
	test_integration.AddCreditCardForPatient(patientID, testData, t)
	// now lets treat the initial visit
	test_integration.SubmitPatientVisitDiagnosis(pv.PatientVisitId, doctor, testData, t)
	test_integration.SubmitPatientVisitBackToPatient(tp.Id.Int64(), doctor, testData, t)

	// lets wait for a moment so as to let a second elapse before creating the next followup
	// so that there is time between the creation of the initial visit and the followup
	time.Sleep(time.Second)

	// now lets try to create a followup visit
	_, err = patientpkg.CreatePendingFollowup(patient, testData.DataApi, testData.AuthApi, testData.Config.Dispatcher, testData.Config.Stores["media"], testData.Config.AuthTokenExpiration)
	test.OK(t, err)

	followupVisit, err := testData.DataApi.GetPatientVisitForSKU(patient.PatientId.Int64(), sku.AcneFollowup)
	test.OK(t, err)
	layoutVersionIDBeforeUpdate := followupVisit.LayoutVersionId.Int64()
	test.Equals(t, true, layoutVersionIDBeforeUpdate != 0)

	// before the patient opens the followup, lets go ahead and simulate a scenario where there is a new followup layout
	// for an updated version of the app, which the patient updates to.
	// Upload first versions of the intake, review and diagnosis layouts
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-0-0.json", test_integration.FollowupIntakeFileLocation, t)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-2-0-0.json", test_integration.FollowupReviewFileLocation, t)

	// specify the app versions and the platform information
	test_integration.AddFieldToMultipartWriter(writer, "patient_app_version", "1.1.0", t)
	test_integration.AddFieldToMultipartWriter(writer, "doctor_app_version", "1.2.0", t)
	test_integration.AddFieldToMultipartWriter(writer, "platform", "iOS", t)

	err = writer.Close()
	test.OK(t, err)

	admin := test_integration.CreateRandomAdmin(t, testData)
	resp, err := testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// now lets have the patient query for the followup visit with the newly updated information
	pv = test_integration.QueryPatientVisit(
		followupVisit.PatientVisitId.Int64(),
		patientAccountID,
		map[string]string{
			"S-Version": "Patient;Test;1.1.0;0001",
			"S-OS":      "iOS;7.1",
			"S-Device":  "Phone;iPhone6,1;640;1136;2.0",
		},
		testData,
		t)
	test.Equals(t, common.PVStatusOpen, pv.Status)

	fVisit, err := testData.DataApi.GetPatientVisitFromId(pv.PatientVisitId)
	test.OK(t, err)
	layoutVersionIDAfterUpdate := fVisit.LayoutVersionId.Int64()
	test.Equals(t, true, layoutVersionIDBeforeUpdate < layoutVersionIDAfterUpdate)
	test.Equals(t, fVisit.PatientVisitId.Int64(), followupVisit.PatientVisitId.Int64())

}

func submitVisit(patientID, patientVisitID int64, stubSQSQueue *common.SQSQueue, testData *test_integration.TestData, t *testing.T) {
	stubStripe := testData.Config.PaymentAPI.(*test_integration.StripeStub)
	stubStripe.CreateChargeFunc = func(req *stripe.CreateChargeRequest) (*stripe.Charge, error) {
		return &stripe.Charge{
			ID: "charge_test",
		}, nil
	}

	test_integration.SubmitPatientVisitForPatient(patientID, patientVisitID, testData, t)
	// wait for the patient's card to be charged, and the followup visit to be routed
	w := cost.StartWorker(testData.DataApi, testData.Config.AnalyticsLogger, testData.Config.Dispatcher,
		stubStripe, nil, stubSQSQueue, metrics.NewRegistry(), 0, "")
	time.Sleep(500 * time.Millisecond)
	w.Stop()
}

func setupFollowupTest(t *testing.T, testData *test_integration.TestData) {
	// lets setup a cost for followup
	skuID, err := testData.DataApi.SKUID(sku.AcneFollowup)

	res, err := testData.DB.Exec(`insert into item_cost (sku_id, status) values (?,?)`, skuID, api.STATUS_ACTIVE)
	test.OK(t, err)
	itemCostId, err := res.LastInsertId()
	test.OK(t, err)
	_, err = testData.DB.Exec(`insert into line_item (currency, description, amount, item_cost_id) values ('USD','Acne Followup',2000,?)`, itemCostId)
	test.OK(t, err)

}
