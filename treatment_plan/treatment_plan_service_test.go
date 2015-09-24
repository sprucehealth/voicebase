package treatment_plan

import (
	"errors"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
)

type treatmentPlanServiceTreatmentDAL struct {
	getTreatmentFromIDParam      int64
	getTreatmentFromIDErr        error
	getTreatmentFromID           *common.Treatment
	getTreatmentsForPatientParam common.PatientID
	getTreatmentsForPatientErr   error
	getTreatmentsForPatient      []*common.Treatment
}

func (s *treatmentPlanServiceTreatmentDAL) GetTreatmentFromID(treatmentID int64) (*common.Treatment, error) {
	s.getTreatmentFromIDParam = treatmentID
	return s.getTreatmentFromID, s.getTreatmentFromIDErr
}

func (s *treatmentPlanServiceTreatmentDAL) GetTreatmentsForPatient(patientID common.PatientID) ([]*common.Treatment, error) {
	s.getTreatmentsForPatientParam = patientID
	return s.getTreatmentsForPatient, s.getTreatmentsForPatientErr
}

type treatmentPlanServiceTreatmentPlanDAL struct {
	getTreatmentPlanForPatientPatientIDParam       common.PatientID
	getTreatmentPlanForPatientTreatmentPlanIDParam int64
	getTreatmentPlanForPatientErr                  error
	getTreatmentPlanForPatient                     *common.TreatmentPlan
}

func (s *treatmentPlanServiceTreatmentPlanDAL) GetTreatmentPlanForPatient(patientID common.PatientID, treatmentPlanID int64) (*common.TreatmentPlan, error) {
	s.getTreatmentPlanForPatientPatientIDParam = patientID
	s.getTreatmentPlanForPatientTreatmentPlanIDParam = treatmentPlanID
	return s.getTreatmentPlanForPatient, s.getTreatmentPlanForPatientErr
}

func TestTreatmentPlanServicePatientCanAccessTreatment(t *testing.T) {
	patientID := common.NewPatientID(1)
	var treatmentID int64 = 2
	var treatmentPlanID int64 = 3
	testData := []struct {
		inPatientID                                    common.PatientID
		inTreatmentID                                  int64
		treatmentPlanServiceTreatmentDAL               *treatmentPlanServiceTreatmentDAL
		treatmentPlanServiceTreatmentPlanDAL           *treatmentPlanServiceTreatmentPlanDAL
		getTreatmentFromIDParam                        int64
		getTreatmentPlanForPatientPatientIDParam       common.PatientID
		getTreatmentPlanForPatientTreatmentPlanIDParam int64
		out                                            bool
		isErr                                          bool
	}{
		{
			inPatientID:   patientID,
			inTreatmentID: treatmentID,
			treatmentPlanServiceTreatmentDAL: &treatmentPlanServiceTreatmentDAL{
				getTreatmentFromIDErr: errors.New("Foo"),
			},
			treatmentPlanServiceTreatmentPlanDAL: &treatmentPlanServiceTreatmentPlanDAL{},
			getTreatmentFromIDParam:              treatmentID,
			out:   false,
			isErr: true,
		},
		{
			inPatientID:   patientID,
			inTreatmentID: treatmentID,
			treatmentPlanServiceTreatmentDAL: &treatmentPlanServiceTreatmentDAL{
				getTreatmentFromIDErr: api.ErrNotFound(`treatment`),
			},
			treatmentPlanServiceTreatmentPlanDAL: &treatmentPlanServiceTreatmentPlanDAL{},
			getTreatmentFromIDParam:              treatmentID,
			out:   false,
			isErr: false,
		},
		{
			inPatientID:   patientID,
			inTreatmentID: treatmentID,
			treatmentPlanServiceTreatmentDAL: &treatmentPlanServiceTreatmentDAL{
				getTreatmentFromID: &common.Treatment{TreatmentPlanID: encoding.NewObjectID(uint64(treatmentPlanID))},
			},
			treatmentPlanServiceTreatmentPlanDAL: &treatmentPlanServiceTreatmentPlanDAL{
				getTreatmentPlanForPatientErr: errors.New("Foo"),
			},
			getTreatmentFromIDParam:                        treatmentID,
			getTreatmentPlanForPatientPatientIDParam:       patientID,
			getTreatmentPlanForPatientTreatmentPlanIDParam: treatmentPlanID,
			out:   false,
			isErr: true,
		},
		{
			inPatientID:   patientID,
			inTreatmentID: treatmentID,
			treatmentPlanServiceTreatmentDAL: &treatmentPlanServiceTreatmentDAL{
				getTreatmentFromID: &common.Treatment{TreatmentPlanID: encoding.NewObjectID(uint64(treatmentPlanID))},
			},
			treatmentPlanServiceTreatmentPlanDAL:           &treatmentPlanServiceTreatmentPlanDAL{},
			getTreatmentFromIDParam:                        treatmentID,
			getTreatmentPlanForPatientPatientIDParam:       patientID,
			getTreatmentPlanForPatientTreatmentPlanIDParam: treatmentPlanID,
			out:   true,
			isErr: false,
		},
	}

	for _, td := range testData {
		s := NewService(td.treatmentPlanServiceTreatmentDAL, td.treatmentPlanServiceTreatmentPlanDAL)
		o, err := s.PatientCanAccessTreatment(td.inPatientID, td.inTreatmentID)
		test.Equals(t, td.getTreatmentFromIDParam, td.treatmentPlanServiceTreatmentDAL.getTreatmentFromIDParam)
		test.Equals(t, td.getTreatmentPlanForPatientPatientIDParam, td.treatmentPlanServiceTreatmentPlanDAL.getTreatmentPlanForPatientPatientIDParam)
		test.Equals(t, td.getTreatmentPlanForPatientTreatmentPlanIDParam, td.treatmentPlanServiceTreatmentPlanDAL.getTreatmentPlanForPatientTreatmentPlanIDParam)
		test.Equals(t, td.out, o)
		if !td.isErr {
			test.Equals(t, nil, err)
		} else {
			test.Assert(t, err != nil, "Expected an error to be returned")
		}
	}
}

func TestTreatmentPlanServiceTreatmentsForPatient(t *testing.T) {
	patientID := common.NewPatientID(1)
	treatments := []*common.Treatment{&common.Treatment{}}
	testData := []struct {
		inPatientID                      common.PatientID
		treatmentPlanServiceTreatmentDAL *treatmentPlanServiceTreatmentDAL
		getTreatmentsForPatientParam     common.PatientID
		out                              []*common.Treatment
		isErr                            bool
	}{
		{
			inPatientID: patientID,
			treatmentPlanServiceTreatmentDAL: &treatmentPlanServiceTreatmentDAL{
				getTreatmentsForPatientErr: errors.New("Foo"),
			},
			getTreatmentsForPatientParam: patientID,
			isErr: true,
		},
		{
			inPatientID: patientID,
			treatmentPlanServiceTreatmentDAL: &treatmentPlanServiceTreatmentDAL{
				getTreatmentsForPatient: treatments,
			},
			getTreatmentsForPatientParam: patientID,
			out:   treatments,
			isErr: false,
		},
	}

	for _, td := range testData {
		s := NewService(td.treatmentPlanServiceTreatmentDAL, nil)
		o, err := s.TreatmentsForPatient(td.inPatientID)
		test.Equals(t, td.getTreatmentsForPatientParam, td.treatmentPlanServiceTreatmentDAL.getTreatmentsForPatientParam)
		test.Equals(t, td.out, o)
		if !td.isErr {
			test.Equals(t, nil, err)
		} else {
			test.Assert(t, err != nil, "Expected an error to be returned")
		}
	}
}
