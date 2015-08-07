package test_integration

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/diagnosis/handlers"
	"github.com/sprucehealth/backend/doctor"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
)

func SignupRandomTestDoctor(t *testing.T, testData *TestData) (signedupDoctorResponse *doctor.DoctorSignedupResponse, email, password string) {
	return signupDoctor(t, testData)
}

func SignupRandomTestAdmin(t *testing.T, testData *TestData) (*doctor.DoctorSignedupResponse, string, string) {
	dr, email, password := signupRandomTestUserOfRole(t, testData, api.RoleAdmin)

	var accountID int64
	err := testData.DB.QueryRow(`SELECT account_id FROM doctor WHERE id = ?`, dr.DoctorID).Scan(&accountID)
	test.OK(t, err)
	// Make them a superuser
	_, err = testData.DB.Exec(`INSERT INTO account_group_member (group_id, account_id) SELECT id, ? FROM account_group WHERE name = 'superuser'`, accountID)
	test.OK(t, err)

	return dr, email, password
}

func SignupRandomTestCC(t *testing.T, testData *TestData, isPrimary bool) (*doctor.DoctorSignedupResponse, string, string) {
	res, email, pass := signupRandomTestUserOfRole(t, testData, api.RoleCC)
	if isPrimary {
		_, err := testData.DB.Exec(`UPDATE doctor SET primary_cc = ? WHERE id = ?`, true, res.DoctorID)
		test.OK(t, err)
	}
	return res, email, pass
}

func signupRandomTestUserOfRole(t *testing.T, testData *TestData, role string) (*doctor.DoctorSignedupResponse, string, string) {
	dr, email, password := signupDoctor(t, testData)

	// update to the desired role
	_, err := testData.DB.Exec(`UPDATE account SET role_type_id = (SELECT id FROM role_type WHERE role_type_tag = ?) WHERE email = ?`, role, email)
	test.OK(t, err)

	_, err = testData.DB.Exec(`UPDATE person SET role_type_id = (SELECT id FROM role_type WHERE role_type_tag = ?) WHERE role_id = ? `, role, dr.DoctorID)
	test.OK(t, err)

	return dr, email, password
}

func signupDoctor(t *testing.T, testData *TestData) (*doctor.DoctorSignedupResponse, string, string) {
	email := strconv.FormatInt(time.Now().UnixNano(), 10) + "@example.com"
	password := "12345"
	params := &url.Values{}
	params.Set("first_name", "Test")
	params.Set("last_name", "LastName")
	params.Set("short_display_name", "Dr. Test")
	params.Set("long_display_name", "Dr. Test LastName")
	params.Set("short_title", "Dermatologist")
	params.Set("long_title", "Board Certified Dermatologist")
	params.Set("email", email)
	params.Set("password", password)
	params.Set("dob", "1987-11-08")
	params.Set("gender", "male")
	params.Set("clinician_id", os.Getenv("DOSESPOT_USER_ID"))
	params.Set("phone", "7348465522")
	params.Set("address_line_1", "12345 Main street")
	params.Set("address_line_2", "apt 11415")
	params.Set("city", "san francisco")
	params.Set("state", "ca")
	params.Set("zip_code", "94115")

	res, err := http.Post(testData.APIServer.URL+apipaths.DoctorSignupURLPath, "application/x-www-form-urlencoded", strings.NewReader(params.Encode()))
	if err != nil {
		t.Fatal("Unable to make post request for registering patient: " + err.Error())
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read body of response: " + err.Error())
	} else if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d", http.StatusOK, res.StatusCode)
	}

	signedupDoctorResponse := &doctor.DoctorSignedupResponse{}
	err = json.Unmarshal(body, signedupDoctorResponse)
	if err != nil {
		t.Fatal("Unable to parse response from patient signed up")
	}
	return signedupDoctorResponse, email, password
}

func SignupRandomTestDoctorInState(state string, t *testing.T, testData *TestData) *doctor.DoctorSignedupResponse {
	doctorSignedupResponse, _, _ := signupDoctor(t, testData)

	// check to see if the state already exists in the system
	careProvidingStateID, err := testData.DataAPI.GetCareProvidingStateID(state, api.AcnePathwayTag)
	if api.IsErrNotFound(err) {
		// this means that the state does not exist and we need to add it
		careProvidingStateID, err = testData.DataAPI.AddCareProvidingState(state, state, api.AcnePathwayTag)
		if err != nil {
			t.Fatal(err)
		}
	} else if err != nil {
		t.Fatal(err)
	}

	// add doctor as elligible to serve in this state for the default condition of acne
	err = testData.DataAPI.MakeDoctorElligibleinCareProvidingState(careProvidingStateID, doctorSignedupResponse.DoctorID)
	test.OK(t, err)
	return doctorSignedupResponse
}

func SetupAnswerIntakeForDiagnosis(questionIDToAnswerTagMapping map[int64][]string, patientVisitID int64, testData *TestData, t *testing.T) *apiservice.IntakeData {
	intakeData := &apiservice.IntakeData{}
	intakeData.PatientVisitID = patientVisitID

	i := 0
	intakeData.Questions = make([]*apiservice.QuestionAnswerItem, len(questionIDToAnswerTagMapping))
	for questionID, answerTags := range questionIDToAnswerTagMapping {
		answerInfoList, err := testData.DataAPI.GetAnswerInfoForTags(answerTags, api.LanguageIDEnglish)
		if err != nil {
			t.Fatal(err)
		}
		intakeData.Questions[i] = &apiservice.QuestionAnswerItem{
			QuestionID:    questionID,
			AnswerIntakes: make([]*apiservice.AnswerItem, len(answerInfoList)),
		}
		for j, answerInfoItem := range answerInfoList {
			intakeData.Questions[i].AnswerIntakes[j] = &apiservice.AnswerItem{PotentialAnswerID: answerInfoItem.AnswerID}
		}
		i++
	}
	return intakeData
}

func PrepareAnswersForDiagnosis(testData *TestData, t *testing.T, patientVisitID int64) *apiservice.IntakeData {
	intakeData := &apiservice.IntakeData{
		PatientVisitID: patientVisitID,
	}
	var diagnosisQuestionID, severityQuestionID, acneTypeQuestionID int64
	if qi, err := testData.DataAPI.GetQuestionInfo("q_acne_diagnosis", 1, 1); err != nil {
		t.Fatalf("Unable to get the questionIds for the question tags requested for the doctor to diagnose patient visit: %s", err.Error())
	} else {
		diagnosisQuestionID = qi.QuestionID
	}
	if qi, err := testData.DataAPI.GetQuestionInfo("q_acne_severity", 1, 1); err != nil {
		t.Fatalf("Unable to get the questionIds for the question tags requested for the doctor to diagnose patient visit: %s", err.Error())
	} else {
		severityQuestionID = qi.QuestionID
	}
	if qi, err := testData.DataAPI.GetQuestionInfo("q_acne_type", 1, 1); err != nil {
		t.Fatalf("Unable to get the questionIds for the question tags requested for the doctor to diagnose patient visit: %s", err.Error())
	} else {
		acneTypeQuestionID = qi.QuestionID
	}

	answerInfo, err := testData.DataAPI.GetAnswerInfoForTags([]string{"a_doctor_acne_vulgaris"}, api.LanguageIDEnglish)
	test.OK(t, err)
	QuestionAnswerItem := &apiservice.QuestionAnswerItem{
		QuestionID:    diagnosisQuestionID,
		AnswerIntakes: []*apiservice.AnswerItem{&apiservice.AnswerItem{PotentialAnswerID: answerInfo[0].AnswerID}},
	}

	answerInfo, err = testData.DataAPI.GetAnswerInfoForTags([]string{"a_doctor_acne_severity_severity"}, api.LanguageIDEnglish)
	test.OK(t, err)
	QuestionAnswerItem2 := &apiservice.QuestionAnswerItem{
		QuestionID:    severityQuestionID,
		AnswerIntakes: []*apiservice.AnswerItem{&apiservice.AnswerItem{PotentialAnswerID: answerInfo[0].AnswerID}},
	}

	answerInfo, err = testData.DataAPI.GetAnswerInfoForTags([]string{"a_acne_inflammatory"}, api.LanguageIDEnglish)
	if err != nil {
		t.Fatal(err.Error())
	}
	QuestionAnswerItem3 := &apiservice.QuestionAnswerItem{
		QuestionID:    acneTypeQuestionID,
		AnswerIntakes: []*apiservice.AnswerItem{&apiservice.AnswerItem{PotentialAnswerID: answerInfo[0].AnswerID}},
	}

	intakeData.Questions = []*apiservice.QuestionAnswerItem{QuestionAnswerItem, QuestionAnswerItem2, QuestionAnswerItem3}

	return intakeData
}

func PrepareAnswersForDiagnosingAsUnsuitableForSpruce(testData *TestData, t *testing.T, patientVisitID int64) *apiservice.IntakeData {
	intakeData := &apiservice.IntakeData{}
	intakeData.PatientVisitID = patientVisitID

	var diagnosisQuestionID int64
	if qi, err := testData.DataAPI.GetQuestionInfo("q_acne_diagnosis", 1, 1); err != nil {
		t.Fatalf("Unable to get the questionIds for the question tags requested for the doctor to diagnose patient visit: %s", err.Error())
	} else {
		diagnosisQuestionID = qi.QuestionID
	}

	answerItemList, err := testData.DataAPI.GetAnswerInfoForTags([]string{"a_doctor_acne_not_suitable_spruce"}, api.LanguageIDEnglish)
	test.OK(t, err)

	QuestionAnswerItem := &apiservice.QuestionAnswerItem{
		QuestionID:    diagnosisQuestionID,
		AnswerIntakes: []*apiservice.AnswerItem{&apiservice.AnswerItem{PotentialAnswerID: answerItemList[0].AnswerID}},
	}
	intakeData.Questions = []*apiservice.QuestionAnswerItem{QuestionAnswerItem}
	return intakeData
}

func MarkUnsuitableForSpruce(testData *TestData, t *testing.T, patientVisitID, doctorAccountID int64) {
	requestData, err := json.Marshal(handlers.DiagnosisListRequestData{
		VisitID: patientVisitID,
		CaseManagement: handlers.CaseManagementItem{
			Unsuitable: true,
			Reason:     "Unsuitable for spruce",
		},
	})
	if err != nil {
		t.Fatal("Unable to marshal request body")
	}

	resp, err := testData.AuthPut(testData.APIServer.URL+apipaths.DoctorVisitDiagnosisListURLPath, "application/json", bytes.NewBuffer(requestData), doctorAccountID)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
}

func CompareDiagnosisWithDoctorIntake(answerIntakeBody *apiservice.IntakeData, diagnosisLayout *info_intake.DiagnosisIntake, testData *TestData, t *testing.T) {

	if diagnosisLayout == nil {
		t.Fatal("Diagnosis response not as expected after doctor submitted diagnosis")
	}

	// create a mapping of question to expected answer
	questionToAnswerMapping := make(map[int64]int64)
	for _, item := range answerIntakeBody.Questions {
		questionToAnswerMapping[item.QuestionID] = item.AnswerIntakes[0].PotentialAnswerID
	}

	for _, section := range diagnosisLayout.InfoIntakeLayout.Sections {
		for _, question := range section.Questions {

			for _, response := range GetAnswerIntakesFromAnswers(question.Answers, t) {
				if questionToAnswerMapping[response.QuestionID.Int64()] != response.PotentialAnswerID.Int64() {
					t.Fatal("Answer returned for question does not match what was supplied")
				}
			}
		}
	}

}

func StartReviewingPatientVisit(patientVisitID int64, doctor *common.Doctor, testData *TestData, t *testing.T) {
	resp, err := testData.AuthGet(testData.APIServer.URL+apipaths.DoctorVisitReviewURLPath+"?patient_visit_id="+strconv.FormatInt(patientVisitID, 10), doctor.AccountID.Int64())
	if err != nil {
		t.Fatal("Unable to make call to get patient visit review for patient: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d", http.StatusOK, resp.StatusCode)
	}

}

func PickATreatmentPlan(parent *common.TreatmentPlanParent, contentSource *common.TreatmentPlanContentSource, doctor *common.Doctor, testData *TestData, t *testing.T) *doctor_treatment_plan.DoctorTreatmentPlanResponse {
	requestData := doctor_treatment_plan.TreatmentPlanRequestData{
		TPParent: parent,
	}

	if contentSource != nil {
		requestData.TPContentSource = contentSource
	}

	jsonData, err := json.Marshal(requestData)
	test.OK(t, err)

	resp, err := testData.AuthPost(testData.APIServer.URL+apipaths.DoctorTreatmentPlansURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountID.Int64())
	if err != nil {
		t.Fatalf("Unable to pick a treatment plan for the patient visit doctor %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(resp.Body)
		t.Fatalf("Expected successful picking up of treatment plan instead got %d: %s [%s]", resp.StatusCode, string(b), test.CallerString(1))
	}

	responseData := &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	if err := json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		t.Fatalf("Unable to unmarshal response into response json: %s", err)
	}

	return responseData
}

func PickATreatmentPlanForPatientVisit(patientVisitID int64, doctor *common.Doctor, ftp *responses.FavoriteTreatmentPlan, testData *TestData, t *testing.T) *doctor_treatment_plan.DoctorTreatmentPlanResponse {
	cli := DoctorClient(testData, t, doctor.ID.Int64())
	tp, err := cli.PickTreatmentPlanForVisit(patientVisitID, ftp)
	if err != nil {
		t.Fatalf("Failed to pick treatment plan from visit: %s [%s]", err.Error(), test.CallerString(1))
	}
	return &doctor_treatment_plan.DoctorTreatmentPlanResponse{TreatmentPlan: tp}
}

func SubmitPatientVisitBackToPatient(treatmentPlanID int64, doctor *common.Doctor, testData *TestData, t *testing.T) {
	cli := DoctorClient(testData, t, doctor.ID.Int64())
	test.OK(t, cli.UpdateTreatmentPlanNote(treatmentPlanID, "test note"))
	test.OK(t, cli.SubmitTreatmentPlan(treatmentPlanID))
}

func AddTreatmentsToTreatmentPlan(treatmentPlanID int64, doctor *common.Doctor, t *testing.T, testData *TestData) {
	treatment1 := &common.Treatment{
		DrugDBIDs: map[string]string{
			erx.LexiDrugSynID:     "1234",
			erx.LexiGenProductID:  "12345",
			erx.LexiSynonymTypeID: "123556",
			erx.NDC:               "2415",
		},
		DrugInternalName:        "Drug1 (Route1 - Form1)",
		GenericDrugName:         "LSD",
		DosageStrength:          "Strength1",
		DispenseValue:           5,
		DispenseUnitDescription: "Tablet",
		DispenseUnitID:          encoding.NewObjectID(19),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		},
		SubstitutionsAllowed: false,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		},
		PatientInstructions: "Take once daily",
		OTC:                 false,
	}

	AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountID.Int64(), treatmentPlanID, t)
}

func AddRegimenPlanForTreatmentPlan(treatmentPlanID int64, doctor *common.Doctor, t *testing.T, testData *TestData) {

	regimenPlanRequest := &common.RegimenPlan{
		TreatmentPlanID: encoding.NewObjectID(treatmentPlanID),
	}

	regimenStep1 := &common.DoctorInstructionItem{
		Text:  "Regimen Step 1",
		State: common.StateAdded,
	}

	regimenStep2 := &common.DoctorInstructionItem{
		Text:  "Regimen Step 2",
		State: common.StateAdded,
	}

	regimenSection := &common.RegimenSection{
		Name: "morning",
		Steps: []*common.DoctorInstructionItem{{
			Text:  regimenStep1.Text,
			State: common.StateAdded,
		}},
	}

	regimenSection2 := &common.RegimenSection{
		Name: "night",
		Steps: []*common.DoctorInstructionItem{{
			Text:  regimenStep2.Text,
			State: common.StateAdded,
		}},
	}

	regimenPlanRequest.AllSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}
	regimenPlanRequest.Sections = []*common.RegimenSection{regimenSection, regimenSection2}

	regimenPlanResponse := CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

}
