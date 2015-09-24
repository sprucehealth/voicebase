package rxremind

import (
	"errors"
	"testing"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
)

type rxReminderServiceTreatmentsDAL struct {
	getTreatmentsForPatientParam common.PatientID
	getTreatmentsForPatientErr   error
	getTreatmentsForPatient      []*common.Treatment
}

func (s *rxReminderServiceTreatmentsDAL) GetTreatmentsForPatient(patientID common.PatientID) ([]*common.Treatment, error) {
	s.getTreatmentsForPatientParam = patientID
	return s.getTreatmentsForPatient, s.getTreatmentsForPatientErr
}

type rxReminderServiceRXRemindersDAL struct {
	createRXReminderParam                *common.RXReminder
	createRXReminderErr                  error
	deleteRXReminderParam                int64
	deleteRXReminderErr                  error
	deleteRXReminder                     int64
	rxRemindersParam                     []int64
	rxRemindersErr                       error
	rxReminders                          map[int64]*common.RXReminder
	updateRXReminderTreatmentPlanIDParam int64
	updateRXReminderReminderParam        *common.RXReminder
	updateRXReminderErr                  error
	updateRXReminder                     int64
}

func (s *rxReminderServiceRXRemindersDAL) CreateRXReminder(r *common.RXReminder) error {
	s.createRXReminderParam = r
	return s.createRXReminderErr
}

func (s *rxReminderServiceRXRemindersDAL) DeleteRXReminder(treatmentID int64) (int64, error) {
	s.deleteRXReminderParam = treatmentID
	return s.deleteRXReminder, s.deleteRXReminderErr
}

func (s *rxReminderServiceRXRemindersDAL) RXReminders(treatmentIDs []int64) (map[int64]*common.RXReminder, error) {
	s.rxRemindersParam = treatmentIDs
	return s.rxReminders, s.rxRemindersErr
}

func (s *rxReminderServiceRXRemindersDAL) UpdateRXReminder(treatmentID int64, reminder *common.RXReminder) (int64, error) {
	s.updateRXReminderTreatmentPlanIDParam = treatmentID
	s.updateRXReminderReminderParam = reminder
	return s.updateRXReminder, s.updateRXReminderErr
}

func TestRXReminderServiceDeleteRXReminder(t *testing.T) {
	testData := []struct {
		treatmentID                     int64
		rxReminderServiceRXRemindersDAL *rxReminderServiceRXRemindersDAL
		dalParam                        int64
		isErr                           bool
	}{
		{
			treatmentID: 1,
			rxReminderServiceRXRemindersDAL: &rxReminderServiceRXRemindersDAL{
				deleteRXReminderErr: errors.New("Foo"),
			},
			dalParam: 1,
			isErr:    true,
		},
		{
			treatmentID:                     1,
			rxReminderServiceRXRemindersDAL: &rxReminderServiceRXRemindersDAL{},
			dalParam:                        1,
			isErr:                           false,
		},
	}

	for _, td := range testData {
		s := NewService(td.rxReminderServiceRXRemindersDAL, nil)
		err := s.DeleteRXReminder(td.treatmentID)
		test.Equals(t, td.dalParam, td.rxReminderServiceRXRemindersDAL.deleteRXReminderParam)
		if !td.isErr {
			test.Equals(t, nil, err)
		} else {
			test.Assert(t, err != nil, "Expected non nil err ro be returned")
		}
	}
}

func TestRXReminderServiceCreateRXReminder(t *testing.T) {
	reminder := &common.RXReminder{}
	testData := []struct {
		reminder                        *common.RXReminder
		rxReminderServiceRXRemindersDAL *rxReminderServiceRXRemindersDAL
		dalParam                        *common.RXReminder
		isErr                           bool
	}{
		{
			reminder: reminder,
			rxReminderServiceRXRemindersDAL: &rxReminderServiceRXRemindersDAL{
				createRXReminderErr: errors.New("Foo"),
			},
			dalParam: reminder,
			isErr:    true,
		},
		{
			reminder:                        reminder,
			rxReminderServiceRXRemindersDAL: &rxReminderServiceRXRemindersDAL{},
			dalParam:                        reminder,
			isErr:                           false,
		},
	}

	for _, td := range testData {
		s := NewService(td.rxReminderServiceRXRemindersDAL, nil)
		err := s.CreateRXReminder(td.reminder)
		test.Equals(t, td.dalParam, td.rxReminderServiceRXRemindersDAL.createRXReminderParam)
		if !td.isErr {
			test.Equals(t, nil, err)
		} else {
			test.Assert(t, err != nil, "Expected non nil err ro be returned")
		}
	}
}

func TestRXReminderServiceUpdateRXReminder(t *testing.T) {
	reminder := &common.RXReminder{}
	testData := []struct {
		treatmentID                     int64
		reminder                        *common.RXReminder
		rxReminderServiceRXRemindersDAL *rxReminderServiceRXRemindersDAL
		dalIDParam                      int64
		dalRemParam                     *common.RXReminder
		isErr                           bool
	}{
		{
			treatmentID: 1,
			reminder:    reminder,
			rxReminderServiceRXRemindersDAL: &rxReminderServiceRXRemindersDAL{
				updateRXReminderErr: errors.New("Foo"),
			},
			dalIDParam:  1,
			dalRemParam: reminder,
			isErr:       true,
		},
		{
			treatmentID:                     1,
			reminder:                        reminder,
			rxReminderServiceRXRemindersDAL: &rxReminderServiceRXRemindersDAL{},
			dalIDParam:                      1,
			dalRemParam:                     reminder,
			isErr:                           false,
		},
	}

	for _, td := range testData {
		s := NewService(td.rxReminderServiceRXRemindersDAL, nil)
		err := s.UpdateRXReminder(td.treatmentID, td.reminder)
		test.Equals(t, td.dalIDParam, td.rxReminderServiceRXRemindersDAL.updateRXReminderTreatmentPlanIDParam)
		test.Equals(t, td.dalRemParam, td.rxReminderServiceRXRemindersDAL.updateRXReminderReminderParam)
		if !td.isErr {
			test.Equals(t, nil, err)
		} else {
			test.Assert(t, err != nil, "Expected non nil err ro be returned")
		}
	}
}

func TestRXReminderServiceRemindersForPatient(t *testing.T) {
	reminderMap := map[int64]*common.RXReminder{
		1: &common.RXReminder{},
		2: &common.RXReminder{},
	}
	testData := []struct {
		patientID                       common.PatientID
		rxReminderServiceTreatmentsDAL  *rxReminderServiceTreatmentsDAL
		rxReminderServiceRXRemindersDAL *rxReminderServiceRXRemindersDAL
		treatmentsParam                 common.PatientID
		remindersParam                  []int64
		reminders                       map[int64]*common.RXReminder
		isErr                           bool
	}{
		{
			patientID: common.NewPatientID(1),
			rxReminderServiceTreatmentsDAL: &rxReminderServiceTreatmentsDAL{
				getTreatmentsForPatientErr: errors.New("Foo"),
			},
			rxReminderServiceRXRemindersDAL: &rxReminderServiceRXRemindersDAL{},
			treatmentsParam:                 common.NewPatientID(1),
			isErr:                           true,
		},
		{
			patientID: common.NewPatientID(1),
			rxReminderServiceTreatmentsDAL: &rxReminderServiceTreatmentsDAL{
				getTreatmentsForPatient: []*common.Treatment{{ID: encoding.NewObjectID(2)}},
			},
			rxReminderServiceRXRemindersDAL: &rxReminderServiceRXRemindersDAL{
				rxRemindersErr: errors.New("Foo"),
			},
			treatmentsParam: common.NewPatientID(1),
			remindersParam:  []int64{2},
			isErr:           true,
		},
		{
			patientID: common.NewPatientID(1),
			rxReminderServiceTreatmentsDAL: &rxReminderServiceTreatmentsDAL{
				getTreatmentsForPatient: []*common.Treatment{
					{ID: encoding.NewObjectID(2)},
					{ID: encoding.NewObjectID(3)},
				},
			},
			rxReminderServiceRXRemindersDAL: &rxReminderServiceRXRemindersDAL{
				rxReminders: reminderMap,
			},
			treatmentsParam: common.NewPatientID(1),
			remindersParam:  []int64{2, 3},
			reminders:       reminderMap,
			isErr:           false,
		},
	}

	for _, td := range testData {
		s := NewService(td.rxReminderServiceRXRemindersDAL, td.rxReminderServiceTreatmentsDAL)
		reminders, err := s.RemindersForPatient(td.patientID)
		test.Equals(t, td.treatmentsParam, td.rxReminderServiceTreatmentsDAL.getTreatmentsForPatientParam)
		test.Equals(t, td.remindersParam, td.rxReminderServiceRXRemindersDAL.rxRemindersParam)
		test.Equals(t, td.reminders, reminders)
		if !td.isErr {
			test.Equals(t, nil, err)
		} else {
			test.Assert(t, err != nil, "Expected non nil err ro be returned")
		}
	}
}
