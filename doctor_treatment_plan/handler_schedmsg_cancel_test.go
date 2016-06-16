package doctor_treatment_plan

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"golang.org/x/net/context"
)

type mockDataAPI_schedmsg struct {
	api.DataAPI
	*mock.Expector
	tpSchedMsg *common.TreatmentPlanScheduledMessage
	tp         *common.TreatmentPlan
	cancelled  bool
	doctorID   int64
}

func (m *mockDataAPI_schedmsg) TreatmentPlanScheduledMessage(id int64) (*common.TreatmentPlanScheduledMessage, error) {
	defer m.Record(id)
	return m.tpSchedMsg, nil
}
func (m *mockDataAPI_schedmsg) GetDoctorIDFromAccountID(accountID int64) (int64, error) {
	defer m.Record(accountID)
	return m.doctorID, nil
}
func (m *mockDataAPI_schedmsg) GetAbridgedTreatmentPlan(tpID, doctorID int64) (*common.TreatmentPlan, error) {
	defer m.Record(tpID, doctorID)
	return m.tp, nil
}
func (m *mockDataAPI_schedmsg) CancelTreatmentPlanScheduledMessage(messageID int64, undo bool) (bool, error) {
	defer m.Record(messageID, undo)
	return m.cancelled, nil
}

func TestCancelSchedMsg_TPDraft(t *testing.T) {
	testCancelSchedMsg_fail(t, common.TPStatusInactive)
}

func TestCancelSchedMsg_TPInactive(t *testing.T) {
	testCancelSchedMsg_fail(t, common.TPStatusDraft)
}

func testCancelSchedMsg_fail(t *testing.T, tpStatus common.TreatmentPlanStatus) {
	treatmentPlanID := uint64(10)
	doctorID := uint64(20)
	messageID := int64(30)
	caseID := uint64(40)
	patientID := uint64(50)
	accountID := int64(60)

	m := &mockDataAPI_schedmsg{
		Expector: &mock.Expector{
			T: t,
		},
		tpSchedMsg: &common.TreatmentPlanScheduledMessage{
			TreatmentPlanID: int64(treatmentPlanID),
		},
		tp: &common.TreatmentPlan{
			DoctorID:      encoding.NewObjectID(doctorID),
			ID:            encoding.NewObjectID(treatmentPlanID),
			PatientID:     common.NewPatientID(patientID),
			PatientCaseID: encoding.NewObjectID(caseID),
			Status:        tpStatus,
		},
	}

	m.Expect(mock.NewExpectation(m.TreatmentPlanScheduledMessage, messageID))
	m.Expect(mock.NewExpectation(m.GetDoctorIDFromAccountID, accountID))
	m.Expect(mock.NewExpectation(m.GetAbridgedTreatmentPlan, int64(treatmentPlanID), int64(0)))

	dp := &mock.Publisher{
		Expector: &mock.Expector{
			T: t,
		},
	}

	jsonData, err := json.Marshal(CancelScheduledMessageRequest{
		MessageID: messageID,
	})
	if err != nil {
		t.Fatal(err)
	}

	r, err := http.NewRequest("PUT", "api.spruce.loc", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{Role: api.RoleCC, ID: accountID})

	h := NewCancelScheduledMessageHandler(m, dp)
	w := httptest.NewRecorder()

	h.ServeHTTP(ctx, w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected bad request but instead got %d", w.Code)
	}

	mock.FinishAll(m, dp)
}

func TestCancelSchedMsg_alreadySent(t *testing.T) {
	treatmentPlanID := uint64(10)
	doctorID := uint64(20)
	messageID := int64(30)
	caseID := uint64(40)
	patientID := uint64(50)
	accountID := int64(60)

	m := &mockDataAPI_schedmsg{
		Expector: &mock.Expector{
			T: t,
		},
		tpSchedMsg: &common.TreatmentPlanScheduledMessage{
			TreatmentPlanID: int64(treatmentPlanID),
			SentTime:        &time.Time{},
		},
		tp: &common.TreatmentPlan{
			DoctorID:      encoding.NewObjectID(doctorID),
			ID:            encoding.NewObjectID(treatmentPlanID),
			PatientID:     common.NewPatientID(patientID),
			PatientCaseID: encoding.NewObjectID(caseID),
			Status:        common.TPStatusActive,
		},
	}

	m.Expect(mock.NewExpectation(m.TreatmentPlanScheduledMessage, messageID))

	dp := &mock.Publisher{
		Expector: &mock.Expector{
			T: t,
		},
	}

	jsonData, err := json.Marshal(CancelScheduledMessageRequest{
		MessageID: messageID,
	})
	if err != nil {
		t.Fatal(err)
	}

	r, err := http.NewRequest("PUT", "api.spruce.loc", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{Role: api.RoleCC, ID: accountID})

	h := NewCancelScheduledMessageHandler(m, dp)
	w := httptest.NewRecorder()

	h.ServeHTTP(ctx, w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected bad request but instead got %d", w.Code)
	}

	mock.FinishAll(m, dp)
}

func TestCancelSchedmsg_TPActive(t *testing.T) {
	treatmentPlanID := uint64(10)
	doctorID := uint64(20)
	messageID := int64(30)
	caseID := uint64(40)
	patientID := uint64(50)
	accountID := int64(60)

	m := &mockDataAPI_schedmsg{
		Expector: &mock.Expector{
			T: t,
		},
		tpSchedMsg: &common.TreatmentPlanScheduledMessage{
			TreatmentPlanID: int64(treatmentPlanID),
		},
		tp: &common.TreatmentPlan{
			DoctorID:      encoding.NewObjectID(doctorID),
			ID:            encoding.NewObjectID(treatmentPlanID),
			PatientID:     common.NewPatientID(patientID),
			PatientCaseID: encoding.NewObjectID(caseID),
			Status:        common.TPStatusActive,
		},
		doctorID:  int64(doctorID),
		cancelled: true,
	}

	m.Expect(mock.NewExpectation(m.TreatmentPlanScheduledMessage, messageID))
	m.Expect(mock.NewExpectation(m.GetDoctorIDFromAccountID, accountID))
	m.Expect(mock.NewExpectation(m.GetAbridgedTreatmentPlan, int64(treatmentPlanID), int64(0)))
	m.Expect(mock.NewExpectation(m.CancelTreatmentPlanScheduledMessage, messageID, false))

	dp := &mock.Publisher{
		Expector: &mock.Expector{
			T: t,
		},
	}

	dp.Expect(mock.NewExpectation(dp.Publish, &TreatmentPlanScheduledMessageCancelledEvent{
		DoctorID:        int64(doctorID),
		TreatmentPlanID: int64(treatmentPlanID),
		PatientID:       m.tp.PatientID,
		CaseID:          int64(caseID),
		Undone:          false,
	}))

	jsonData, err := json.Marshal(CancelScheduledMessageRequest{
		MessageID: messageID,
	})
	if err != nil {
		t.Fatal(err)
	}

	r, err := http.NewRequest("PUT", "api.spruce.loc", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{Role: api.RoleCC, ID: accountID})

	h := NewCancelScheduledMessageHandler(m, dp)
	w := httptest.NewRecorder()

	h.ServeHTTP(ctx, w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected success but instead got %d", w.Code)
	}

	mock.FinishAll(m, dp)
}

func TestCancelSchedmsg_Undo_TPActive(t *testing.T) {
	treatmentPlanID := uint64(10)
	doctorID := uint64(20)
	messageID := int64(30)
	caseID := uint64(40)
	patientID := uint64(50)
	accountID := int64(60)

	m := &mockDataAPI_schedmsg{
		Expector: &mock.Expector{
			T: t,
		},
		tpSchedMsg: &common.TreatmentPlanScheduledMessage{
			TreatmentPlanID: int64(treatmentPlanID),
		},
		tp: &common.TreatmentPlan{
			DoctorID:      encoding.NewObjectID(doctorID),
			ID:            encoding.NewObjectID(treatmentPlanID),
			PatientID:     common.NewPatientID(patientID),
			PatientCaseID: encoding.NewObjectID(caseID),
			Status:        common.TPStatusActive,
		},
		doctorID:  int64(doctorID),
		cancelled: true,
	}

	m.Expect(mock.NewExpectation(m.TreatmentPlanScheduledMessage, messageID))
	m.Expect(mock.NewExpectation(m.GetDoctorIDFromAccountID, accountID))
	m.Expect(mock.NewExpectation(m.GetAbridgedTreatmentPlan, int64(treatmentPlanID), int64(0)))
	m.Expect(mock.NewExpectation(m.CancelTreatmentPlanScheduledMessage, messageID, true))

	dp := &mock.Publisher{
		Expector: &mock.Expector{
			T: t,
		},
	}

	dp.Expect(mock.NewExpectation(dp.Publish, &TreatmentPlanScheduledMessageCancelledEvent{
		DoctorID:        int64(doctorID),
		TreatmentPlanID: int64(treatmentPlanID),
		PatientID:       m.tp.PatientID,
		CaseID:          int64(caseID),
		Undone:          true,
	}))

	jsonData, err := json.Marshal(CancelScheduledMessageRequest{
		MessageID: messageID,
		Undo:      true,
	})
	if err != nil {
		t.Fatal(err)
	}

	r, err := http.NewRequest("PUT", "api.spruce.loc", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{Role: api.RoleCC, ID: accountID})

	h := NewCancelScheduledMessageHandler(m, dp)
	w := httptest.NewRecorder()

	h.ServeHTTP(ctx, w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected success but instead got %d", w.Code)
	}

	mock.FinishAll(m, dp)
}
