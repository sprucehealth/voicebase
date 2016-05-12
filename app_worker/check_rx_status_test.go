package app_worker

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/erx"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

type mockDataAPI_CheckRXStatus struct {
	api.DataAPI
	*mock.Expector

	patient         *common.Patient
	doctor          *common.Doctor
	careCoordinator *common.Doctor
	statusEvents    []common.StatusEvent
	treatment       *common.Treatment
}

func (m *mockDataAPI_CheckRXStatus) GetPatientFromID(patientID common.PatientID) (*common.Patient, error) {
	defer m.Record(patientID)
	return m.patient, nil
}
func (m *mockDataAPI_CheckRXStatus) GetDoctorFromID(doctorID int64) (*common.Doctor, error) {
	defer m.Record(doctorID)
	return m.doctor, nil
}
func (m *mockDataAPI_CheckRXStatus) GetPrescriptionStatusEventsForPatient(erxPatientID int64) ([]common.StatusEvent, error) {
	defer m.Record(erxPatientID)
	return m.statusEvents, nil
}
func (m *mockDataAPI_CheckRXStatus) GetApprovedOrDeniedRefillRequestsForPatient(patientID common.PatientID) ([]common.StatusEvent, error) {
	defer m.Record(patientID)
	return m.statusEvents, nil
}
func (m *mockDataAPI_CheckRXStatus) GetErxStatusEventsForDNTFTreatmentBasedOnPatientID(patientID common.PatientID) ([]common.StatusEvent, error) {
	defer m.Record(patientID)
	return m.statusEvents, nil
}
func (m *mockDataAPI_CheckRXStatus) GetTreatmentBasedOnPrescriptionID(prescriptionID int64) (*common.Treatment, error) {
	defer m.Record(prescriptionID)
	return m.treatment, nil
}
func (m *mockDataAPI_CheckRXStatus) AddErxStatusEvent(treatments []*common.Treatment, statusEvent common.StatusEvent) error {
	defer m.Record(treatments, statusEvent)
	return nil
}
func (m *mockDataAPI_CheckRXStatus) AddRefillRequestStatusEvent(statusEvent common.StatusEvent) error {
	defer m.Record(statusEvent)
	return nil
}
func (m *mockDataAPI_CheckRXStatus) AddErxStatusEventForDNTFTreatment(statusEvent common.StatusEvent) error {
	defer m.Record(statusEvent)
	return nil
}
func (m *mockDataAPI_CheckRXStatus) ListCareProviders(opt api.ListCareProvidersOption) ([]*common.Doctor, error) {
	defer m.Record(opt)
	return []*common.Doctor{m.careCoordinator}, nil
}

type mockERxAPI_CheckRXStatus struct {
	erx.ERxAPI
	*mock.Expector

	prescriptionLogs []*erx.PrescriptionLog
}

func (m *mockERxAPI_CheckRXStatus) GetPrescriptionStatus(clinicianID, prescriptionID int64) ([]*erx.PrescriptionLog, error) {
	defer m.Record(clinicianID, prescriptionID)
	return m.prescriptionLogs, nil
}

func TestCheckStatus_Treatment_Sent(t *testing.T) {
	erxPatientID := int64(10)
	patientID := uint64(20)
	clinicianID := int64(30)
	prescriptionID := int64(40)
	doctorID := int64(50)
	m := &mockDataAPI_CheckRXStatus{
		Expector: &mock.Expector{
			T: t,
		},
		patient: &common.Patient{
			ID:           common.NewPatientID(patientID),
			ERxPatientID: encoding.DeprecatedNewObjectID(erxPatientID),
		},
		doctor: &common.Doctor{
			ID:                  encoding.DeprecatedNewObjectID(doctorID),
			DoseSpotClinicianID: 30,
		},
		statusEvents: []common.StatusEvent{
			{
				Status:            api.ERXStatusSending,
				ReportedTimestamp: time.Time{},
				PrescriptionID:    prescriptionID,
			},
		},
		treatment: &common.Treatment{},
	}
	m.Expect(mock.NewExpectation(m.GetPatientFromID, common.NewPatientID(patientID)))
	m.Expect(mock.NewExpectation(m.GetDoctorFromID, doctorID))
	m.Expect(mock.NewExpectation(m.GetPrescriptionStatusEventsForPatient, erxPatientID))
	m.Expect(mock.NewExpectation(m.GetTreatmentBasedOnPrescriptionID, prescriptionID))
	m.Expect(mock.NewExpectation(m.AddErxStatusEvent, []*common.Treatment{m.treatment}, common.StatusEvent{
		Status:            api.ERXStatusSent,
		ReportedTimestamp: time.Time{},
	}))

	e := &mockERxAPI_CheckRXStatus{
		Expector: &mock.Expector{
			T: t,
		},
		prescriptionLogs: []*erx.PrescriptionLog{
			{
				PrescriptionStatus: api.ERXStatusSent,
				LogTimestamp:       time.Time{},
				AdditionalInfo:     "no error",
			},
		},
	}
	e.Expect(mock.NewExpectation(e.GetPrescriptionStatus, clinicianID, prescriptionID))

	dp := &mock.Publisher{
		Expector: &mock.Expector{
			T: t,
		},
	}
	queue := &common.SQSQueue{
		QueueService: &awsutil.SQS{},
		QueueURL:     "testing",
	}
	w := NewERxStatusWorker(m, e, dp, queue, metrics.NewRegistry())

	jsonData, err := json.Marshal(&common.PrescriptionStatusCheckMessage{
		PatientID:      common.NewPatientID(patientID),
		DoctorID:       doctorID,
		EventCheckType: common.ERxType,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = queue.QueueService.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    ptr.String(queue.QueueURL),
		MessageBody: ptr.String(string(jsonData)),
	})
	if err != nil {
		t.Fatal(err)
	}

	w.Do()
	mock.FinishAll(m, e, dp)
}

func TestCheckStatus_Treatment_Error(t *testing.T) {
	erxPatientID := int64(10)
	patientID := uint64(20)
	clinicianID := int64(30)
	prescriptionID := int64(40)
	doctorID := int64(50)
	ccID := int64(70)

	m := &mockDataAPI_CheckRXStatus{
		Expector: &mock.Expector{
			T: t,
		},
		patient: &common.Patient{
			ID:           common.NewPatientID(patientID),
			ERxPatientID: encoding.DeprecatedNewObjectID(erxPatientID),
		},
		doctor: &common.Doctor{
			ID:                  encoding.DeprecatedNewObjectID(doctorID),
			DoseSpotClinicianID: 30,
		},
		careCoordinator: &common.Doctor{
			ID: encoding.DeprecatedNewObjectID(ccID),
		},
		statusEvents: []common.StatusEvent{
			{
				Status:            api.ERXStatusSending,
				ReportedTimestamp: time.Time{},
				PrescriptionID:    prescriptionID,
				ItemID:            prescriptionID,
			},
		},
		treatment: &common.Treatment{},
	}
	m.Expect(mock.NewExpectation(m.GetPatientFromID, common.NewPatientID(patientID)))
	m.Expect(mock.NewExpectation(m.GetDoctorFromID, doctorID))
	m.Expect(mock.NewExpectation(m.GetPrescriptionStatusEventsForPatient, erxPatientID))
	m.Expect(mock.NewExpectation(m.GetTreatmentBasedOnPrescriptionID, prescriptionID))
	m.Expect(mock.NewExpectation(m.AddErxStatusEvent, []*common.Treatment{m.treatment}, common.StatusEvent{
		Status:            api.ERXStatusError,
		ReportedTimestamp: time.Time{},
		StatusDetails:     "error!",
	}))
	m.Expect(mock.NewExpectation(m.ListCareProviders, api.LCPOptPrimaryCCOnly))

	e := &mockERxAPI_CheckRXStatus{
		Expector: &mock.Expector{
			T: t,
		},
		prescriptionLogs: []*erx.PrescriptionLog{
			{
				PrescriptionStatus: api.ERXStatusError,
				LogTimestamp:       time.Time{},
				AdditionalInfo:     "error!",
			},
		},
	}
	e.Expect(mock.NewExpectation(e.GetPrescriptionStatus, clinicianID, prescriptionID))

	dp := &mock.Publisher{
		Expector: &mock.Expector{
			T: t,
		},
	}

	dp.Expect(mock.NewExpectation(dp.Publish, &RxTransmissionErrorEvent{
		ProviderID:   m.careCoordinator.ID.Int64(),
		ProviderRole: api.RoleCC,
		ItemID:       prescriptionID,
		EventType:    common.ERxType,
		Patient:      m.patient,
	}))
	queue := &common.SQSQueue{
		QueueService: &awsutil.SQS{},
		QueueURL:     "testing",
	}
	w := NewERxStatusWorker(m, e, dp, queue, metrics.NewRegistry())

	jsonData, err := json.Marshal(&common.PrescriptionStatusCheckMessage{
		PatientID:      common.NewPatientID(patientID),
		DoctorID:       doctorID,
		EventCheckType: common.ERxType,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = queue.QueueService.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    ptr.String(queue.QueueURL),
		MessageBody: ptr.String(string(jsonData)),
	})
	if err != nil {
		t.Fatal(err)
	}

	w.Do()
	mock.FinishAll(m, e, dp)
}

func TestCheckStatus_RefillRequest_Sent(t *testing.T) {
	erxPatientID := int64(10)
	patientID := uint64(20)
	clinicianID := int64(30)
	prescriptionID := int64(40)
	doctorID := int64(50)
	ccID := int64(70)

	m := &mockDataAPI_CheckRXStatus{
		Expector: &mock.Expector{
			T: t,
		},
		patient: &common.Patient{
			ID:           common.NewPatientID(patientID),
			ERxPatientID: encoding.DeprecatedNewObjectID(erxPatientID),
		},
		doctor: &common.Doctor{
			ID:                  encoding.DeprecatedNewObjectID(doctorID),
			DoseSpotClinicianID: 30,
		},
		careCoordinator: &common.Doctor{
			ID: encoding.DeprecatedNewObjectID(ccID),
		},
		statusEvents: []common.StatusEvent{
			{
				Status:            api.RXRefillStatusApproved,
				ReportedTimestamp: time.Time{},
				PrescriptionID:    prescriptionID,
				ItemID:            prescriptionID,
			},
		},
	}
	m.Expect(mock.NewExpectation(m.GetPatientFromID, common.NewPatientID(patientID)))
	m.Expect(mock.NewExpectation(m.GetDoctorFromID, doctorID))
	m.Expect(mock.NewExpectation(m.GetApprovedOrDeniedRefillRequestsForPatient, common.NewPatientID(patientID)))
	m.Expect(mock.NewExpectation(m.AddRefillRequestStatusEvent, common.StatusEvent{
		Status:            api.RXRefillStatusSent,
		ReportedTimestamp: time.Time{},
		ItemID:            prescriptionID,
	}))

	e := &mockERxAPI_CheckRXStatus{
		Expector: &mock.Expector{
			T: t,
		},
		prescriptionLogs: []*erx.PrescriptionLog{
			{
				PrescriptionStatus: api.ERXStatusSent,
				LogTimestamp:       time.Time{},
			},
		},
	}
	e.Expect(mock.NewExpectation(e.GetPrescriptionStatus, clinicianID, prescriptionID))

	dp := &mock.Publisher{
		Expector: &mock.Expector{
			T: t,
		},
	}

	queue := &common.SQSQueue{
		QueueService: &awsutil.SQS{},
		QueueURL:     "testing",
	}
	w := NewERxStatusWorker(m, e, dp, queue, metrics.NewRegistry())

	jsonData, err := json.Marshal(&common.PrescriptionStatusCheckMessage{
		PatientID:      common.NewPatientID(patientID),
		DoctorID:       doctorID,
		EventCheckType: common.RefillRxType,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = queue.QueueService.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    ptr.String(queue.QueueURL),
		MessageBody: ptr.String(string(jsonData)),
	})
	if err != nil {
		t.Fatal(err)
	}

	w.Do()
	mock.FinishAll(m, e, dp)
}

func TestCheckStatus_RefillRequest_Error(t *testing.T) {
	erxPatientID := int64(10)
	patientID := uint64(20)
	clinicianID := int64(30)
	prescriptionID := int64(40)
	doctorID := int64(50)
	ccID := int64(70)

	m := &mockDataAPI_CheckRXStatus{
		Expector: &mock.Expector{
			T: t,
		},
		patient: &common.Patient{
			ID:           common.NewPatientID(patientID),
			ERxPatientID: encoding.DeprecatedNewObjectID(erxPatientID),
		},
		doctor: &common.Doctor{
			ID:                  encoding.DeprecatedNewObjectID(doctorID),
			DoseSpotClinicianID: 30,
		},
		careCoordinator: &common.Doctor{
			ID: encoding.DeprecatedNewObjectID(ccID),
		},
		statusEvents: []common.StatusEvent{
			{
				Status:            api.RXRefillStatusApproved,
				ReportedTimestamp: time.Time{},
				PrescriptionID:    prescriptionID,
				ItemID:            prescriptionID,
			},
		},
	}
	m.Expect(mock.NewExpectation(m.GetPatientFromID, common.NewPatientID(patientID)))
	m.Expect(mock.NewExpectation(m.GetDoctorFromID, doctorID))
	m.Expect(mock.NewExpectation(m.GetApprovedOrDeniedRefillRequestsForPatient, common.NewPatientID(patientID)))
	m.Expect(mock.NewExpectation(m.AddRefillRequestStatusEvent, common.StatusEvent{
		Status:            api.RXRefillStatusError,
		ReportedTimestamp: time.Time{},
		ItemID:            prescriptionID,
		StatusDetails:     "error!",
	}))
	m.Expect(mock.NewExpectation(m.ListCareProviders, api.LCPOptPrimaryCCOnly))

	e := &mockERxAPI_CheckRXStatus{
		Expector: &mock.Expector{
			T: t,
		},
		prescriptionLogs: []*erx.PrescriptionLog{
			{
				PrescriptionStatus: api.ERXStatusError,
				LogTimestamp:       time.Time{},
				AdditionalInfo:     "error!",
			},
		},
	}
	e.Expect(mock.NewExpectation(e.GetPrescriptionStatus, clinicianID, prescriptionID))

	dp := &mock.Publisher{
		Expector: &mock.Expector{
			T: t,
		},
	}
	dp.Expect(mock.NewExpectation(dp.Publish, &RxTransmissionErrorEvent{
		ProviderID:   m.careCoordinator.ID.Int64(),
		ProviderRole: api.RoleCC,
		ItemID:       prescriptionID,
		EventType:    common.RefillRxType,
		Patient:      m.patient,
	}))

	queue := &common.SQSQueue{
		QueueService: &awsutil.SQS{},
		QueueURL:     "testing",
	}
	w := NewERxStatusWorker(m, e, dp, queue, metrics.NewRegistry())

	jsonData, err := json.Marshal(&common.PrescriptionStatusCheckMessage{
		PatientID:      common.NewPatientID(patientID),
		DoctorID:       doctorID,
		EventCheckType: common.RefillRxType,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = queue.QueueService.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    ptr.String(queue.QueueURL),
		MessageBody: ptr.String(string(jsonData)),
	})
	if err != nil {
		t.Fatal(err)
	}

	w.Do()
	mock.FinishAll(m, e, dp)
}

func TestCheckStatus_DNTF_Sent(t *testing.T) {
	erxPatientID := int64(10)
	patientID := uint64(20)
	clinicianID := int64(30)
	prescriptionID := int64(40)
	doctorID := int64(50)
	ccID := int64(70)

	m := &mockDataAPI_CheckRXStatus{
		Expector: &mock.Expector{
			T: t,
		},
		patient: &common.Patient{
			ID:           common.NewPatientID(patientID),
			ERxPatientID: encoding.DeprecatedNewObjectID(erxPatientID),
		},
		doctor: &common.Doctor{
			ID:                  encoding.DeprecatedNewObjectID(doctorID),
			DoseSpotClinicianID: 30,
		},
		careCoordinator: &common.Doctor{
			ID: encoding.DeprecatedNewObjectID(ccID),
		},
		statusEvents: []common.StatusEvent{
			{
				Status:            api.ERXStatusSending,
				ReportedTimestamp: time.Time{},
				PrescriptionID:    prescriptionID,
				ItemID:            prescriptionID,
			},
		},
	}
	m.Expect(mock.NewExpectation(m.GetPatientFromID, common.NewPatientID(patientID)))
	m.Expect(mock.NewExpectation(m.GetDoctorFromID, doctorID))
	m.Expect(mock.NewExpectation(m.GetErxStatusEventsForDNTFTreatmentBasedOnPatientID, common.NewPatientID(patientID)))
	m.Expect(mock.NewExpectation(m.AddErxStatusEventForDNTFTreatment, common.StatusEvent{
		Status:            api.ERXStatusSent,
		ReportedTimestamp: time.Time{},
		ItemID:            prescriptionID,
	}))

	e := &mockERxAPI_CheckRXStatus{
		Expector: &mock.Expector{
			T: t,
		},
		prescriptionLogs: []*erx.PrescriptionLog{
			{
				PrescriptionStatus: api.ERXStatusSent,
				LogTimestamp:       time.Time{},
			},
		},
	}
	e.Expect(mock.NewExpectation(e.GetPrescriptionStatus, clinicianID, prescriptionID))

	dp := &mock.Publisher{
		Expector: &mock.Expector{
			T: t,
		},
	}

	queue := &common.SQSQueue{
		QueueService: &awsutil.SQS{},
		QueueURL:     "testing",
	}
	w := NewERxStatusWorker(m, e, dp, queue, metrics.NewRegistry())

	jsonData, err := json.Marshal(&common.PrescriptionStatusCheckMessage{
		PatientID:      common.NewPatientID(patientID),
		DoctorID:       doctorID,
		EventCheckType: common.UnlinkedDNTFTreatmentType,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = queue.QueueService.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    ptr.String(queue.QueueURL),
		MessageBody: ptr.String(string(jsonData)),
	})
	if err != nil {
		t.Fatal(err)
	}

	w.Do()
	mock.FinishAll(m, e, dp)
}

func TestCheckStatus_DNTF_Error(t *testing.T) {
	erxPatientID := int64(10)
	patientID := uint64(20)
	clinicianID := int64(30)
	prescriptionID := int64(40)
	doctorID := int64(50)
	ccID := int64(70)

	m := &mockDataAPI_CheckRXStatus{
		Expector: &mock.Expector{
			T: t,
		},
		patient: &common.Patient{
			ID:           common.NewPatientID(patientID),
			ERxPatientID: encoding.DeprecatedNewObjectID(erxPatientID),
		},
		doctor: &common.Doctor{
			ID:                  encoding.DeprecatedNewObjectID(doctorID),
			DoseSpotClinicianID: 30,
		},
		careCoordinator: &common.Doctor{
			ID: encoding.DeprecatedNewObjectID(ccID),
		},
		statusEvents: []common.StatusEvent{
			{
				Status:            api.ERXStatusSending,
				ReportedTimestamp: time.Time{},
				PrescriptionID:    prescriptionID,
				ItemID:            prescriptionID,
			},
		},
	}
	m.Expect(mock.NewExpectation(m.GetPatientFromID, common.NewPatientID(patientID)))
	m.Expect(mock.NewExpectation(m.GetDoctorFromID, doctorID))
	m.Expect(mock.NewExpectation(m.GetErxStatusEventsForDNTFTreatmentBasedOnPatientID, common.NewPatientID(patientID)))
	m.Expect(mock.NewExpectation(m.AddErxStatusEventForDNTFTreatment, common.StatusEvent{
		Status:            api.ERXStatusError,
		ReportedTimestamp: time.Time{},
		ItemID:            prescriptionID,
	}))
	m.Expect(mock.NewExpectation(m.ListCareProviders, api.LCPOptPrimaryCCOnly))

	e := &mockERxAPI_CheckRXStatus{
		Expector: &mock.Expector{
			T: t,
		},
		prescriptionLogs: []*erx.PrescriptionLog{
			{
				PrescriptionStatus: api.ERXStatusError,
				LogTimestamp:       time.Time{},
			},
		},
	}
	e.Expect(mock.NewExpectation(e.GetPrescriptionStatus, clinicianID, prescriptionID))

	dp := &mock.Publisher{
		Expector: &mock.Expector{
			T: t,
		},
	}
	dp.Expect(mock.NewExpectation(dp.Publish, &RxTransmissionErrorEvent{
		ProviderID:   m.careCoordinator.ID.Int64(),
		ProviderRole: api.RoleCC,
		ItemID:       prescriptionID,
		EventType:    common.UnlinkedDNTFTreatmentType,
		Patient:      m.patient,
	}))

	queue := &common.SQSQueue{
		QueueService: &awsutil.SQS{},
		QueueURL:     "testing",
	}
	w := NewERxStatusWorker(m, e, dp, queue, metrics.NewRegistry())

	jsonData, err := json.Marshal(&common.PrescriptionStatusCheckMessage{
		PatientID:      common.NewPatientID(patientID),
		DoctorID:       doctorID,
		EventCheckType: common.UnlinkedDNTFTreatmentType,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = queue.QueueService.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    ptr.String(queue.QueueURL),
		MessageBody: ptr.String(string(jsonData)),
	})
	if err != nil {
		t.Fatal(err)
	}

	w.Do()
	mock.FinishAll(m, e, dp)
}
