package test_doctor_queue

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_queue"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

func TestNotifyDoctorsOfUnclaimedCases(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// lets register multiple doctors in CA
	dr1 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	dr2 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	dr3 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)

	// lets ensure that all doctors listed above have notifications turned on for case notifications
	_, err := testData.DB.Exec(`update care_provider_state_elligibility set notify = 1 where provider_id in (?,?,?)`, dr1.DoctorId, dr2.DoctorId, dr3.DoctorId)
	test.OK(t, err)

	// now lets go ahead and submit a visit in CA
	test_integration.CreateRandomPatientVisitInState("CA", t, testData)

	// now start the worker to notify the doctors
	testLock := &test_integration.TestLock{}
	w := doctor_queue.StartWorker(testData.DataApi, testData.AuthApi, testLock, testData.Config.NotificationManager, metrics.NewRegistry())
	time.Sleep(500 * time.Millisecond)
	defer w.Stop()

	// at this point just one of the doctors should have been notified
	smsAPI := testData.SMSAPI
	test.Equals(t, 1, smsAPI.Len())

	var count int
	err = testData.DB.QueryRow(`select count(*) from doctor_case_notification`).Scan(&count)
	test.OK(t, err)
	test.Equals(t, 1, count)

	err = testData.DB.QueryRow(`select count(*) from care_providing_state_notification`).Scan(&count)
	test.OK(t, err)
	test.Equals(t, 1, count)

	// lets delete the care providing state notification so that we can notify a doctor in the state again
	_, err = testData.DB.Exec(`delete from care_providing_state_notification`)
	test.OK(t, err)

	// lets get the worker to run again and we should ensure that the doctor notified this time
	// is different than the previous doctor notified
	testLock = &test_integration.TestLock{}
	w = doctor_queue.StartWorker(testData.DataApi, testData.AuthApi, testLock, testData.Config.NotificationManager, metrics.NewRegistry())
	time.Sleep(500 * time.Millisecond)
	defer w.Stop()
	err = testData.DB.QueryRow(`select count(*) from doctor_case_notification`).Scan(&count)
	test.OK(t, err)
	test.Equals(t, 2, count)
	test.Equals(t, 2, smsAPI.Len())

	_, err = testData.DB.Exec(`delete from care_providing_state_notification`)
	test.OK(t, err)

	testLock = &test_integration.TestLock{}
	w = doctor_queue.StartWorker(testData.DataApi, testData.AuthApi, testLock, testData.Config.NotificationManager, metrics.NewRegistry())
	time.Sleep(500 * time.Millisecond)
	defer w.Stop()
	err = testData.DB.QueryRow(`select count(*) from doctor_case_notification`).Scan(&count)
	test.OK(t, err)
	test.Equals(t, 3, count)
	test.Equals(t, 3, smsAPI.Len())

	// at this point ensure to check that all 3 notfications went to different doctors
	doctorsNotified := make(map[int64]bool)
	rows, err := testData.DB.Query(`select doctor_id from doctor_case_notification`)
	test.OK(t, err)
	defer rows.Close()

	for rows.Next() {
		var doctorID int64
		if err := rows.Scan(&doctorID); err != nil {
			t.Fatal(err)
		}

		test.Equals(t, false, doctorsNotified[doctorID])
		doctorsNotified[doctorID] = true
	}
	test.OK(t, rows.Err())
}

func TestNotifyDoctorsOfUnclaimedCases_SnoozeNotifications(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// lets register a doctor in CA
	dr1 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr1.DoctorId)
	test.OK(t, err)

	// lets ensure that the doctor listed above has notifications turned on for case notifications
	_, err = testData.DB.Exec(`update care_provider_state_elligibility set notify = 1 where provider_id = ?`, dr1.DoctorId)
	test.OK(t, err)

	// lets specify the timezone in which the doctor is in
	tzName := "America/Los_Angeles"
	location, err := time.LoadLocation(tzName)
	test.OK(t, err)

	_, err = testData.DB.Exec(`insert into account_timezone (account_id, iana_timezone) values (?,?)`, doctor.AccountId.Int64(), tzName)
	test.OK(t, err)

	timeInTz := time.Now().In(location)

	// lets specify a snooze period for the doctor
	_, err = testData.DB.Exec(`insert into communication_snooze (account_id, start_hour, num_hours) values (?,?,?)`,
		doctor.AccountId.Int64(), timeInTz.Add(-4*time.Hour).Hour(), 8)
	test.OK(t, err)

	// now lets go ahead and submit a visit in CA
	test_integration.CreateRandomPatientVisitInState("CA", t, testData)

	// now start the worker to notify the doctor
	testLock := &test_integration.TestLock{}
	w := doctor_queue.StartWorker(testData.DataApi, testData.AuthApi, testLock, testData.Config.NotificationManager, metrics.NewRegistry())
	time.Sleep(500 * time.Millisecond)
	defer w.Stop()

	// at this point none of the doctors should have been notified since
	// the doctor has its communications snoozed
	smsAPI := testData.SMSAPI
	test.Equals(t, 0, smsAPI.Len())

	// now lets change the timezone for the doctor
	tzName = "Asia/Shanghai"
	_, err = testData.DB.Exec(`replace into account_timezone (account_id, iana_timezone) values (?,?)`,
		doctor.AccountId.Int64(), tzName)
	test.OK(t, err)

	_, err = testData.DB.Exec(`delete from communication_snooze`)
	test.OK(t, err)

	location, err = time.LoadLocation(tzName)
	test.OK(t, err)
	timeInTz = time.Now().In(location)
	test.OK(t, err)

	// insert an entry in a different timezone where the current time is included in the range
	_, err = testData.DB.Exec(`insert into communication_snooze (account_id, start_hour, num_hours) 
		values (?,?,?)`, doctor.AccountId.Int64(), timeInTz.Add(-4*time.Hour).Hour(), 8)
	test.OK(t, err)

	// lets attempt to notify the doctor again
	w = doctor_queue.StartWorker(testData.DataApi, testData.AuthApi, &test_integration.TestLock{}, testData.Config.NotificationManager, metrics.NewRegistry())
	time.Sleep(500 * time.Millisecond)
	defer w.Stop()

	// doctor should not have been notified again
	// as the doctor has their communication snoozed
	smsAPI = testData.SMSAPI
	test.Equals(t, 0, smsAPI.Len())

	// now lets switch the doctor back to a timezone with 0 offset from UTC
	// to ensure that the doctor will be notified when the snooze time is outside the timezone
	tzName = "Africa/Abidjan"
	_, err = testData.DB.Exec(`replace into account_timezone (account_id, iana_timezone) values (?,?)`, doctor.AccountId.Int64(), tzName)
	test.OK(t, err)

	w = doctor_queue.StartWorker(testData.DataApi, testData.AuthApi, &test_integration.TestLock{}, testData.Config.NotificationManager, metrics.NewRegistry())
	time.Sleep(500 * time.Millisecond)
	defer w.Stop()

	// doctor should have been notified
	smsAPI = testData.SMSAPI
	test.Equals(t, 1, smsAPI.Len())
}

// This test ensures that when a doctor is notified of a case route to a state,
// then we bias towards picking doctors that are not registered in an overlapping state
// to notify of case routes to other states
func TestNotifyDoctorsOfUnclaimedCases_AvoidOverlap(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// lets setup the scenario to have 1 doctor registered in PA
	// and another doctor registered in FL and PA and another doctor registered only in FL
	dr1 := test_integration.SignupRandomTestDoctorInState("PA", t, testData)
	dr2 := test_integration.SignupRandomTestDoctorInState("FL", t, testData)
	dr3 := test_integration.SignupRandomTestDoctorInState("FL", t, testData)

	careProvidingStateIDPA, err := testData.DataApi.GetCareProvidingStateId("PA", apiservice.HEALTH_CONDITION_ACNE_ID)
	test.OK(t, err)

	// register doctor2 in PA
	err = testData.DataApi.MakeDoctorElligibleinCareProvidingState(careProvidingStateIDPA, dr2.DoctorId)
	test.OK(t, err)

	_, err = testData.DB.Exec(`update care_provider_state_elligibility set notify = 1 where provider_id in (?,?,?)`, dr1.DoctorId, dr2.DoctorId, dr3.DoctorId)
	test.OK(t, err)

	// submit a visit in CA and FL
	test_integration.CreateRandomPatientVisitInState("PA", t, testData)
	test_integration.CreateRandomPatientVisitInState("FL", t, testData)

	// ensure that a doctor (doctor1 or doctor2) was notified about the visit in CA
	// and the doctor only registered in FL was notified about the visit in FL (doctor3)
	testLock := &test_integration.TestLock{}
	w := doctor_queue.StartWorker(testData.DataApi, testData.AuthApi, testLock, testData.Config.NotificationManager, metrics.NewRegistry())
	time.Sleep(500 * time.Millisecond)
	defer w.Stop()

	smsAPI := testData.SMSAPI
	test.Equals(t, 2, smsAPI.Len())

	var doctorID int64
	err = testData.DB.QueryRow(`SELECT doctor_id from doctor_case_notification WHERE doctor_id in (?,?)`, dr1.DoctorId, dr2.DoctorId).Scan(&doctorID)
	test.OK(t, err)
	err = testData.DB.QueryRow(`SELECT doctor_id from doctor_case_notification WHERE doctor_id = ?`, dr3.DoctorId).Scan(&doctorID)
	test.OK(t, err)
}

// This test is to ensure that we are notifying doctors that are configured to receive SMS notifications
// of unclaimed cases submitted in the states they are activated in
func TestNotifyDoctorsOfUnclaimedCases_NotifyFlag(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// lets create three doctors in three different state
	dr1 := test_integration.SignupRandomTestDoctorInState("FL", t, testData)
	dr2 := test_integration.SignupRandomTestDoctorInState("NY", t, testData)
	dr3 := test_integration.SignupRandomTestDoctorInState("WA", t, testData)
	test_integration.SignupRandomTestDoctorInState("PA", t, testData)

	careProvidingStateIDFL, err := testData.DataApi.GetCareProvidingStateId("FL", apiservice.HEALTH_CONDITION_ACNE_ID)
	test.OK(t, err)

	careProvidingStateIDWA, err := testData.DataApi.GetCareProvidingStateId("WA", apiservice.HEALTH_CONDITION_ACNE_ID)
	test.OK(t, err)

	careProvidingStateIDNY, err := testData.DataApi.GetCareProvidingStateId("NY", apiservice.HEALTH_CONDITION_ACNE_ID)
	test.OK(t, err)

	// lets register doctor1 to get notified for visits in CA and NY
	_, err = testData.DB.Exec(`UPDATE care_provider_state_elligibility SET notify = 1 WHERE provider_id = ? and care_providing_state_id = ?`, dr1.DoctorId, careProvidingStateIDFL)
	test.OK(t, err)

	err = testData.DataApi.MakeDoctorElligibleinCareProvidingState(careProvidingStateIDNY, dr1.DoctorId)
	test.OK(t, err)
	_, err = testData.DB.Exec(`UPDATE care_provider_state_elligibility SET notify = 1 WHERE provider_id = ? and care_providing_state_id = ?`, dr1.DoctorId, careProvidingStateIDNY)
	test.OK(t, err)

	// lets update doctor1's phone number to make it something that is distinguishable
	doctor1, err := testData.DataApi.GetDoctorFromId(dr1.DoctorId)
	test.OK(t, err)
	err = testData.AuthApi.ReplacePhoneNumbersForAccount(doctor1.AccountId.Int64(), []*common.PhoneNumber{
		&common.PhoneNumber{
			Phone:  common.Phone("734-846-5520"),
			Type:   api.PHONE_CELL,
			Status: api.STATUS_ACTIVE,
		},
	})
	test.OK(t, err)

	// now lets create and submit a visit in the state of FL
	test_integration.CreateRandomPatientVisitInState("FL", t, testData)

	w := doctor_queue.StartWorker(testData.DataApi, testData.AuthApi, &test_integration.TestLock{}, testData.Config.NotificationManager, metrics.NewRegistry())
	time.Sleep(500 * time.Millisecond)
	defer w.Stop()

	// at this point doctor1 should have received an SMS about the visit
	test.Equals(t, 1, testData.SMSAPI.Len())
	test.Equals(t, "734-846-5520", testData.SMSAPI.Sent[0].To)

	// lets change doctor2's phone number to something unique
	doctor2, err := testData.DataApi.GetDoctorFromId(dr2.DoctorId)
	test.OK(t, err)
	err = testData.AuthApi.ReplacePhoneNumbersForAccount(doctor2.AccountId.Int64(), []*common.PhoneNumber{
		&common.PhoneNumber{
			Phone:  common.Phone("734-846-5521"),
			Type:   api.PHONE_CELL,
			Status: api.STATUS_ACTIVE,
		},
	})
	test.OK(t, err)

	// now lets create and submit a visit in the state of NY. Given that doctor1 was already notified
	// about the case in CA, the doctor should not be notified about the visit in NY as they are likely to pick it up
	test_integration.CreateRandomPatientVisitInState("NY", t, testData)

	w = doctor_queue.StartWorker(testData.DataApi, testData.AuthApi, &test_integration.TestLock{}, testData.Config.NotificationManager, metrics.NewRegistry())
	time.Sleep(500 * time.Millisecond)
	defer w.Stop()

	// at this point doctor1 should have received an SMS about the visit in NY
	test.Equals(t, 1, testData.SMSAPI.Len())

	// lets register doctor2 to get notified for visits in NY
	_, err = testData.DB.Exec(`UPDATE care_provider_state_elligibility SET notify = 1 WHERE provider_id = ? and care_providing_state_id = ?`, dr2.DoctorId, careProvidingStateIDNY)
	test.OK(t, err)

	// lets register doctor3 to get notified for visits in WA
	_, err = testData.DB.Exec(`UPDATE care_provider_state_elligibility SET notify = 1 WHERE provider_id = ? and care_providing_state_id = ?`, dr3.DoctorId, careProvidingStateIDWA)
	test.OK(t, err)

	// now lets submit another visit in NY
	// both doctors should be notified
	test_integration.CreateRandomPatientVisitInState("NY", t, testData)

	w = doctor_queue.StartWorker(testData.DataApi, testData.AuthApi, &test_integration.TestLock{}, testData.Config.NotificationManager, metrics.NewRegistry())
	time.Sleep(500 * time.Millisecond)
	defer w.Stop()

	test.Equals(t, 2, testData.SMSAPI.Len())
	test.Equals(t, "734-846-5521", testData.SMSAPI.Sent[1].To)

	// lets change doctor3's phone number to something unique
	doctor3, err := testData.DataApi.GetDoctorFromId(dr3.DoctorId)
	test.OK(t, err)
	err = testData.AuthApi.ReplacePhoneNumbersForAccount(doctor3.AccountId.Int64(), []*common.PhoneNumber{
		&common.PhoneNumber{
			Phone:  common.Phone("734-846-5525"),
			Type:   api.PHONE_CELL,
			Status: api.STATUS_ACTIVE,
		},
	})
	test.OK(t, err)

	// now lets submit a visit in WA and only doctor3 should be notified
	test_integration.CreateRandomPatientVisitInState("WA", t, testData)

	w = doctor_queue.StartWorker(testData.DataApi, testData.AuthApi, &test_integration.TestLock{}, testData.Config.NotificationManager, metrics.NewRegistry())
	time.Sleep(500 * time.Millisecond)
	defer w.Stop()

	test.Equals(t, 3, testData.SMSAPI.Len())
	test.Equals(t, "734-846-5525", testData.SMSAPI.Sent[2].To)

	w = doctor_queue.StartWorker(testData.DataApi, testData.AuthApi, &test_integration.TestLock{}, testData.Config.NotificationManager, metrics.NewRegistry())
	time.Sleep(500 * time.Millisecond)
	defer w.Stop()

	// now submit a visit in PA and no one should be notified
	test_integration.CreateRandomPatientVisitInState("PA", t, testData)
	test.Equals(t, 3, testData.SMSAPI.Len())
}
