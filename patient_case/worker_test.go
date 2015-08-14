package patient_case

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
)

type mockDataAPI_worker struct {
	api.DataAPI
	timedoutCases []*common.PatientCase

	updates map[int64]*api.PatientCaseUpdate
}

func (m *mockDataAPI_worker) TimedOutCases() ([]*common.PatientCase, error) {
	return m.timedoutCases, nil
}
func (m *mockDataAPI_worker) UpdatePatientCase(caseID int64, update *api.PatientCaseUpdate) error {
	m.updates[caseID] = update
	return nil
}

type mockLockAPI struct {
	api.LockAPI
}

func (m *mockLockAPI) Locked() bool {
	return false
}
func (m *mockLockAPI) Wait() bool {
	return true
}
func (m *mockLockAPI) Release() {}

func TestWorker_ActiveInactive(t *testing.T) {
	mLock := &mockLockAPI{}
	mDataAPI := &mockDataAPI_worker{
		updates: make(map[int64]*api.PatientCaseUpdate),
		timedoutCases: []*common.PatientCase{
			{
				ID:     encoding.DeprecatedNewObjectID(1),
				Status: common.PCStatusActive,
			},
			{
				ID:     encoding.DeprecatedNewObjectID(2),
				Status: common.PCStatusActive,
			},
		},
	}

	w := NewWorker(mDataAPI, mLock)
	test.OK(t, w.Do())

	// at this point the cases should've transitioned to inactive with the timeout being nullified
	test.Equals(t, 2, len(mDataAPI.updates))
	test.Equals(t, common.PCStatusInactive, *mDataAPI.updates[1].Status)
	test.Equals(t, true, mDataAPI.updates[1].TimeoutDate.Valid)
	test.Equals(t, true, mDataAPI.updates[1].TimeoutDate.Time == nil)
	test.Equals(t, common.PCStatusInactive, *mDataAPI.updates[2].Status)
	test.Equals(t, true, mDataAPI.updates[2].TimeoutDate.Valid)
	test.Equals(t, true, mDataAPI.updates[2].TimeoutDate.Time == nil)
}

func TestWorker_TriageDeleted(t *testing.T) {
	mLock := &mockLockAPI{}
	mDataAPI := &mockDataAPI_worker{
		updates: make(map[int64]*api.PatientCaseUpdate),
		timedoutCases: []*common.PatientCase{
			{
				ID:     encoding.DeprecatedNewObjectID(1),
				Status: common.PCStatusPreSubmissionTriage,
			},
			{
				ID:     encoding.DeprecatedNewObjectID(2),
				Status: common.PCStatusPreSubmissionTriage,
			},
		},
	}

	w := NewWorker(mDataAPI, mLock)
	test.OK(t, w.Do())

	// at this point the cases should've transitioned to inactive with the timeout being nullified
	test.Equals(t, 2, len(mDataAPI.updates))
	test.Equals(t, common.PCStatusPreSubmissionTriageDeleted, *mDataAPI.updates[1].Status)
	test.Equals(t, true, mDataAPI.updates[1].TimeoutDate.Valid)
	test.Equals(t, true, mDataAPI.updates[1].TimeoutDate.Time == nil)
	test.Equals(t, common.PCStatusPreSubmissionTriageDeleted, *mDataAPI.updates[2].Status)
	test.Equals(t, true, mDataAPI.updates[2].TimeoutDate.Valid)
	test.Equals(t, true, mDataAPI.updates[2].TimeoutDate.Time == nil)
}

func TestWorker_Idempotent(t *testing.T) {
	mLock := &mockLockAPI{}
	mDataAPI := &mockDataAPI_worker{
		updates: make(map[int64]*api.PatientCaseUpdate),
		timedoutCases: []*common.PatientCase{
			{
				ID:     encoding.DeprecatedNewObjectID(1),
				Status: common.PCStatusPreSubmissionTriageDeleted,
			},
			{
				ID:     encoding.DeprecatedNewObjectID(2),
				Status: common.PCStatusInactive,
			},
		},
	}

	w := NewWorker(mDataAPI, mLock)
	test.OK(t, w.Do())

	// at this point the cases should've transitioned to inactive with the timeout being nullified
	test.Equals(t, 2, len(mDataAPI.updates))
	test.Equals(t, true, mDataAPI.updates[1].TimeoutDate.Valid)
	test.Equals(t, true, mDataAPI.updates[1].TimeoutDate.Time == nil)
	test.Equals(t, true, mDataAPI.updates[2].TimeoutDate.Valid)
	test.Equals(t, true, mDataAPI.updates[2].TimeoutDate.Time == nil)
}
