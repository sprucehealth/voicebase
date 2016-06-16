package rxremind

import (
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/libs/errors"
)

// Service defines the methods required to interact with the RxReminders system
type Service interface {
	CreateRXReminder(reminder *common.RXReminder) error
	DeleteRXReminder(treatmentID int64) error
	RemindersForPatient(patientID common.PatientID) (map[int64]*common.RXReminder, error)
	UpdateRXReminder(treatmentID int64, reminder *common.RXReminder) error
}

type treatmentsDAL interface {
	GetTreatmentsForPatient(patientID common.PatientID) ([]*common.Treatment, error)
}

type rxRemindersDAL interface {
	CreateRXReminder(r *common.RXReminder) error
	DeleteRXReminder(treatmentID int64) (int64, error)
	RXReminders(treatmentIDs []int64) (map[int64]*common.RXReminder, error)
	UpdateRXReminder(treatmentID int64, reminder *common.RXReminder) (int64, error)
}

type rxReminderService struct {
	rxRemindersDAL rxRemindersDAL
	treatmentsDAL  treatmentsDAL
}

// NewService returns an initialized instance of rxReminderService
func NewService(rxRemindersDAL rxRemindersDAL, treatmentsDAL treatmentsDAL) Service {
	return &rxReminderService{
		rxRemindersDAL: rxRemindersDAL,
		treatmentsDAL:  treatmentsDAL,
	}
}

func (s *rxReminderService) RemindersForPatient(patientID common.PatientID) (map[int64]*common.RXReminder, error) {
	treatments, err := s.treatmentsDAL.GetTreatmentsForPatient(patientID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ids := make([]int64, len(treatments))
	for i, t := range treatments {
		ids[i] = t.ID.Int64()
	}

	reminders, err := s.rxRemindersDAL.RXReminders(ids)
	return reminders, errors.Trace(err)
}

func (s *rxReminderService) DeleteRXReminder(treatmentID int64) error {
	_, err := s.rxRemindersDAL.DeleteRXReminder(treatmentID)
	return errors.Trace(err)
}

func (s *rxReminderService) CreateRXReminder(reminder *common.RXReminder) error {
	return errors.Trace(s.rxRemindersDAL.CreateRXReminder(reminder))
}

func (s *rxReminderService) UpdateRXReminder(treatmentID int64, reminder *common.RXReminder) error {
	_, err := s.rxRemindersDAL.UpdateRXReminder(treatmentID, reminder)
	return errors.Trace(err)
}
