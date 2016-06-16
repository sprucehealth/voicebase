package app_worker

import (
	"testing"
	"time"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/erx"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

type mockDataAPI_CheckRXErrors struct {
	api.DataAPI
	*mock.Expector
	careCoordinator       *common.Doctor
	doctor                *common.Doctor
	treatment             *common.Treatment
	treatmentError        error
	statusEvents          []common.StatusEvent
	refillRequest         *common.RefillRequestItem
	refillRequestError    error
	unlinkedDNTFTreatment *common.Treatment
}

func (m *mockDataAPI_CheckRXErrors) ListCareProviders(opt api.ListCareProvidersOption) ([]*common.Doctor, error) {
	defer m.Record(opt)
	switch opt {
	case api.LCPOptPrimaryCCOnly:
		return []*common.Doctor{m.careCoordinator}, nil
	case api.LCPOptDoctorsOnly:
		return []*common.Doctor{m.doctor}, nil
	}

	return nil, nil
}

func (m *mockDataAPI_CheckRXErrors) GetTreatmentBasedOnPrescriptionID(prescriptionID int64) (*common.Treatment, error) {
	defer m.Record(prescriptionID)
	if m.treatmentError != nil {
		return nil, m.treatmentError
	}
	return m.treatment, nil
}
func (m *mockDataAPI_CheckRXErrors) GetRefillRequestFromPrescriptionID(prescriptionID int64) (*common.RefillRequestItem, error) {
	defer m.Record(prescriptionID)
	if m.refillRequestError != nil {
		return nil, m.refillRequestError
	}
	return m.refillRequest, nil
}
func (m *mockDataAPI_CheckRXErrors) GetUnlinkedDNTFTreatmentFromPrescriptionID(prescriptionID int64) (*common.Treatment, error) {
	defer m.Record(prescriptionID)
	return m.unlinkedDNTFTreatment, nil
}

func (m *mockDataAPI_CheckRXErrors) GetPrescriptionStatusEventsForTreatment(treatmentID int64) ([]common.StatusEvent, error) {
	defer m.Record(treatmentID)
	return m.statusEvents, nil
}
func (m *mockDataAPI_CheckRXErrors) GetRefillStatusEventsForRefillRequest(refillRequestID int64) ([]common.StatusEvent, error) {
	defer m.Record(refillRequestID)
	return m.statusEvents, nil
}
func (m *mockDataAPI_CheckRXErrors) GetErxStatusEventsForDNTFTreatment(treatmentID int64) ([]common.StatusEvent, error) {
	defer m.Record(treatmentID)
	return m.statusEvents, nil
}
func (m *mockDataAPI_CheckRXErrors) AddErxStatusEvent(treatments []*common.Treatment, statusEvent common.StatusEvent) error {
	defer m.Record(treatments, statusEvent)
	return nil
}

func (m *mockDataAPI_CheckRXErrors) AddRefillRequestStatusEvent(statusEvent common.StatusEvent) error {
	defer m.Record(statusEvent)
	return nil
}

type mockERxAPI_CheckRXErrors struct {
	erx.ERxAPI
	treatmentsWithErrors []*common.Treatment
}

func (m *mockERxAPI_CheckRXErrors) GetTransmissionErrorDetails(clinicianID int64) ([]*common.Treatment, error) {
	return m.treatmentsWithErrors, nil
}

type mockPublisher_CheckRXErrors struct {
	*mock.Expector
	dispatch.Publisher
}

func (m *mockPublisher_CheckRXErrors) Publish(e interface{}) error {
	defer m.Record(e)
	return nil
}

func TestCheckRXError_Treatment_SentToError(t *testing.T) {
	testCheckRXError_Treatment_Error(t, api.ERXStatusSent)
}
func TestCheckRXError_Treatment_SendingToError(t *testing.T) {
	testCheckRXError_Treatment_Error(t, api.ERXStatusSending)
}
func TestCheckRXError_Treatment_ErrorToError(t *testing.T) {
	testCheckRXError_Treatment_Error(t, api.ERXStatusError)
}
func TestCheckRXError_Treatment_ResolvedToError(t *testing.T) {
	testCheckRXError_Treatment_Error(t, api.ERXStatusResolved)
}

func TestCheckRXError_Refill_SentToError(t *testing.T) {
	testCheckRXError_Refill_Error(t, api.RXRefillStatusSent)
}
func TestCheckRXError_Refill_SendingToError(t *testing.T) {
	testCheckRXError_Refill_Error(t, api.RXRefillStatusApproved)
}
func TestCheckRXError_Refill_ErrorToError(t *testing.T) {
	testCheckRXError_Refill_Error(t, api.RXRefillStatusError)
}
func TestCheckRXError_Refill_ResolvedToError(t *testing.T) {
	testCheckRXError_Refill_Error(t, api.RXRefillStatusErrorResolved)
}

func TestCheckRXError_UnlinkedDNTF_SentToError(t *testing.T) {
	testCheckRXError_Refill_Error(t, api.ERXStatusError)
}
func TestCheckRXError_UnlinkedDNTF_SendingToError(t *testing.T) {
	testCheckRXError_Refill_Error(t, api.ERXStatusSending)
}
func TestCheckRXError_UnlinkedDNTF_ErrorToError(t *testing.T) {
	testCheckRXError_Refill_Error(t, api.ERXStatusError)
}

func testCheckRXError_Refill_Error(t *testing.T, startStatus string) {
	prescriptionID := int64(10)
	refillRequestID := int64(20)
	statusDetails := "error"

	d := &mockDataAPI_CheckRXErrors{
		Expector: &mock.Expector{
			T: t,
		},
		doctor: &common.Doctor{
			ID:                  encoding.DeprecatedNewObjectID(100),
			DoseSpotClinicianID: 1200,
		},
		careCoordinator: &common.Doctor{
			ID: encoding.DeprecatedNewObjectID(101),
		},
		treatmentError: api.ErrNotFound("treatment not found"),
		refillRequest: &common.RefillRequestItem{
			ID:      refillRequestID,
			Doctor:  &common.Doctor{},
			Patient: &common.Patient{},
		},
		statusEvents: []common.StatusEvent{
			{
				Status: startStatus,
			},
		},
	}

	d.Expect(mock.NewExpectation(d.ListCareProviders, api.LCPOptDoctorsOnly))
	d.Expect(mock.NewExpectation(d.GetTreatmentBasedOnPrescriptionID, prescriptionID))
	d.Expect(mock.NewExpectation(d.GetRefillRequestFromPrescriptionID, prescriptionID))
	d.Expect(mock.NewExpectation(d.GetRefillStatusEventsForRefillRequest, refillRequestID))

	e := &mockERxAPI_CheckRXErrors{
		treatmentsWithErrors: []*common.Treatment{
			{
				ID:            encoding.DeprecatedNewObjectID(10),
				StatusDetails: statusDetails,
				ERx: &common.ERxData{
					PrescriptionID:        encoding.DeprecatedNewObjectID(prescriptionID),
					TransmissionErrorDate: &time.Time{},
				},
			},
		},
	}

	dp := &mockPublisher_CheckRXErrors{Expector: &mock.Expector{
		T: t,
	}}

	if !(startStatus == api.RXRefillStatusError || startStatus == api.RXRefillStatusErrorResolved) {
		d.Expect(mock.NewExpectation(d.AddRefillRequestStatusEvent, common.StatusEvent{
			Status:            api.RXRefillStatusError,
			StatusDetails:     statusDetails,
			ReportedTimestamp: time.Time{},
			ItemID:            refillRequestID,
		}))

		d.Expect(mock.NewExpectation(d.ListCareProviders, api.LCPOptPrimaryCCOnly))

		dp.Expect(mock.NewExpectation(dp.Publish, &RxTransmissionErrorEvent{
			ProviderID:   d.careCoordinator.ID.Int64(),
			ProviderRole: api.RoleCC,
			ItemID:       refillRequestID,
			EventType:    common.RefillRxType,
			Patient:      d.refillRequest.Patient,
		}))
	}

	w := NewERxErrorWorker(d, e, dp, nil, metrics.NewRegistry())
	w.Do()

	mock.FinishAll(d, dp)
}

func testCheckRXError_Treatment_Error(t *testing.T, startStatus string) {
	prescriptionID := int64(10)
	treatmentID := int64(20)
	statusDetails := "error"

	d := &mockDataAPI_CheckRXErrors{
		Expector: &mock.Expector{
			T: t,
		},
		doctor: &common.Doctor{
			ID:                  encoding.DeprecatedNewObjectID(100),
			DoseSpotClinicianID: 1200,
		},
		careCoordinator: &common.Doctor{
			ID: encoding.DeprecatedNewObjectID(101),
		},
		treatment: &common.Treatment{
			ID:      encoding.DeprecatedNewObjectID(treatmentID),
			Doctor:  &common.Doctor{},
			Patient: &common.Patient{},
		},
		statusEvents: []common.StatusEvent{
			{
				Status: startStatus,
			},
		},
	}

	d.Expect(mock.NewExpectation(d.ListCareProviders, api.LCPOptDoctorsOnly))
	d.Expect(mock.NewExpectation(d.GetTreatmentBasedOnPrescriptionID, prescriptionID))
	d.Expect(mock.NewExpectation(d.GetPrescriptionStatusEventsForTreatment, treatmentID))

	e := &mockERxAPI_CheckRXErrors{
		treatmentsWithErrors: []*common.Treatment{
			{
				ID:            encoding.DeprecatedNewObjectID(10),
				StatusDetails: statusDetails,
				ERx: &common.ERxData{
					PrescriptionID:        encoding.DeprecatedNewObjectID(prescriptionID),
					TransmissionErrorDate: &time.Time{},
				},
			},
		},
	}

	dp := &mockPublisher_CheckRXErrors{Expector: &mock.Expector{
		T: t,
	}}

	if !(startStatus == api.ERXStatusError || startStatus == api.ERXStatusResolved) {
		d.Expect(mock.NewExpectation(d.AddErxStatusEvent, []*common.Treatment{d.treatment}, common.StatusEvent{
			Status:            api.ERXStatusError,
			StatusDetails:     statusDetails,
			ReportedTimestamp: time.Time{},
			ItemID:            treatmentID,
		}))
		d.Expect(mock.NewExpectation(d.ListCareProviders, api.LCPOptPrimaryCCOnly))
		dp.Expect(mock.NewExpectation(dp.Publish, &RxTransmissionErrorEvent{
			ProviderID:   d.careCoordinator.ID.Int64(),
			ProviderRole: api.RoleCC,
			ItemID:       treatmentID,
			EventType:    common.ERxType,
			Patient:      d.treatment.Patient,
		}))
	}

	w := NewERxErrorWorker(d, e, dp, nil, metrics.NewRegistry())
	w.Do()

	mock.FinishAll(d, dp)
}

func testCheckRXError_UnlinkedDNTF_Error(t *testing.T, startStatus string) {
	prescriptionID := int64(10)
	treatmentID := int64(20)
	statusDetails := "error"

	d := &mockDataAPI_CheckRXErrors{
		Expector: &mock.Expector{
			T: t,
		},
		doctor: &common.Doctor{
			ID:                  encoding.DeprecatedNewObjectID(100),
			DoseSpotClinicianID: 1200,
		},
		careCoordinator: &common.Doctor{
			ID: encoding.DeprecatedNewObjectID(101),
		},
		treatmentError:     api.ErrNotFound("treatment not found"),
		refillRequestError: api.ErrNotFound("refill request not found"),
		unlinkedDNTFTreatment: &common.Treatment{
			ID:      encoding.DeprecatedNewObjectID(treatmentID),
			Doctor:  &common.Doctor{},
			Patient: &common.Patient{},
		},
		statusEvents: []common.StatusEvent{
			{
				Status: startStatus,
			},
		},
	}

	d.Expect(mock.NewExpectation(d.ListCareProviders, api.LCPOptDoctorsOnly))
	d.Expect(mock.NewExpectation(d.GetTreatmentBasedOnPrescriptionID, prescriptionID))
	d.Expect(mock.NewExpectation(d.GetRefillRequestFromPrescriptionID, prescriptionID))
	d.Expect(mock.NewExpectation(d.GetUnlinkedDNTFTreatmentFromPrescriptionID, prescriptionID))
	d.Expect(mock.NewExpectation(d.GetErxStatusEventsForDNTFTreatment, treatmentID))

	e := &mockERxAPI_CheckRXErrors{
		treatmentsWithErrors: []*common.Treatment{
			{
				ID:            encoding.DeprecatedNewObjectID(10),
				StatusDetails: statusDetails,
				ERx: &common.ERxData{
					PrescriptionID:        encoding.DeprecatedNewObjectID(prescriptionID),
					TransmissionErrorDate: &time.Time{},
				},
			},
		},
	}

	dp := &mockPublisher_CheckRXErrors{Expector: &mock.Expector{
		T: t,
	}}

	if startStatus != api.ERXStatusError {
		d.Expect(mock.NewExpectation(d.AddErxStatusEventForDNTFTreatment, common.StatusEvent{
			Status:            api.ERXStatusError,
			StatusDetails:     statusDetails,
			ReportedTimestamp: time.Time{},
			ItemID:            treatmentID,
		}))
		d.Expect(mock.NewExpectation(d.ListCareProviders, api.LCPOptPrimaryCCOnly))
		dp.Expect(mock.NewExpectation(dp.Publish, &RxTransmissionErrorEvent{
			ProviderID:   d.careCoordinator.ID.Int64(),
			ProviderRole: api.RoleCC,
			ItemID:       treatmentID,
			EventType:    common.UnlinkedDNTFTreatmentType,
			Patient:      d.treatment.Patient,
		}))
	}

	w := NewERxErrorWorker(d, e, dp, nil, metrics.NewRegistry())
	w.Do()

	mock.FinishAll(d, dp)
}
