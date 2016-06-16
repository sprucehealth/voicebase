package treatment_plan

import (
	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/errors"
)

// Service represents the methods needed to provide the business logic layer related to treatment plans
type Service interface {
	PatientCanAccessTreatment(patientID common.PatientID, treatmentID int64) (bool, error)
	TreatmentsForPatient(patientID common.PatientID) ([]*common.Treatment, error)
}

type treatmentDAL interface {
	GetTreatmentFromID(treatmentID int64) (*common.Treatment, error)
	GetTreatmentsForPatient(patientID common.PatientID) ([]*common.Treatment, error)
}

type treatmentPlanDAL interface {
	GetTreatmentPlanForPatient(patientID common.PatientID, treatmentPlanID int64) (*common.TreatmentPlan, error)
}

type treatmentPlanService struct {
	treatmentDAL     treatmentDAL
	treatmentPlanDAL treatmentPlanDAL
}

// NewService returns an initialized instance of treatmentPlanService
func NewService(treatmentDAL treatmentDAL, treatmentPlanDAL treatmentPlanDAL) Service {
	return &treatmentPlanService{
		treatmentDAL:     treatmentDAL,
		treatmentPlanDAL: treatmentPlanDAL,
	}
}

// TODO: This is a pretty heavy weight access check since building the TP is non trivial perhaps think up something better
func (s *treatmentPlanService) PatientCanAccessTreatment(patientID common.PatientID, treatmentID int64) (bool, error) {
	treatment, err := s.treatmentDAL.GetTreatmentFromID(treatmentID)
	if api.IsErrNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, errors.Trace(err)
	}

	_, err = s.treatmentPlanDAL.GetTreatmentPlanForPatient(patientID, treatment.TreatmentPlanID.Int64())
	if err != nil {
		return false, errors.Trace(err)
	}

	return true, nil
}

func (s *treatmentPlanService) TreatmentsForPatient(patientID common.PatientID) ([]*common.Treatment, error) {
	ts, err := s.treatmentDAL.GetTreatmentsForPatient(patientID)
	return ts, errors.Trace(err)
}
