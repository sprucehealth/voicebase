package demo

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost"
	"github.com/sprucehealth/backend/doctor"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/patient_file"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
)

type mockDataAPI_DemoListener struct {
	api.DataAPI
	patient             *common.Patient
	careTeamMember      *common.CareProviderAssignment
	careTeamMemberError error
	visit               *common.PatientVisit

	doctorLookupByEmail func(email string) (*common.Doctor, error)
	doctorLookupByID    func(id int64) (*common.Doctor, error)
}

func (m *mockDataAPI_DemoListener) GetPatientFromID(id int64) (*common.Patient, error) {
	return m.patient, nil
}
func (m *mockDataAPI_DemoListener) GetDoctorFromID(id int64) (*common.Doctor, error) {
	if m.doctorLookupByID != nil {
		return m.doctorLookupByID(id)
	}
	return nil, nil
}
func (m *mockDataAPI_DemoListener) GetDoctorWithEmail(email string) (*common.Doctor, error) {
	if m.doctorLookupByEmail != nil {
		return m.doctorLookupByEmail(email)
	}
	return nil, nil
}
func (m *mockDataAPI_DemoListener) GetPatientVisitFromID(id int64) (*common.PatientVisit, error) {
	return m.visit, nil
}
func (m *mockDataAPI_DemoListener) GetActiveCareTeamMemberForCase(role string, caseID int64) (*common.CareProviderAssignment, error) {
	return m.careTeamMember, nil
}

type mockDoctorCLI struct {
	ftps     []*responses.PathwayFTPGroup
	ftpsList []*responses.FavoriteTreatmentPlan
	tp       *responses.TreatmentPlan

	treatmentsAdded  []*common.Treatment
	noteUpdated      string
	regimenPlanAdded *common.RegimenPlan
	tpIDSubmitted    int64
	ftpPicked        *responses.FavoriteTreatmentPlan
}

func (m *mockDoctorCLI) SetToken(token string) {}
func (m *mockDoctorCLI) Auth(email, password string) (*doctor.AuthenticationResponse, error) {
	return &doctor.AuthenticationResponse{}, nil
}
func (m *mockDoctorCLI) ListFavoriteTreatmentPlans() ([]*responses.PathwayFTPGroup, error) {
	return m.ftps, nil
}
func (m *mockDoctorCLI) ListFavoriteTreatmentPlansForTag(pathwayTag string) ([]*responses.FavoriteTreatmentPlan, error) {
	return m.ftpsList, nil
}
func (m *mockDoctorCLI) ReviewVisit(patientVisitID int64) (*patient_file.VisitReviewResponse, error) {
	return nil, nil
}
func (m *mockDoctorCLI) PickTreatmentPlanForVisit(visitID int64, ftp *responses.FavoriteTreatmentPlan) (*responses.TreatmentPlan, error) {
	m.ftpPicked = ftp
	return m.tp, nil
}
func (m *mockDoctorCLI) SubmitTreatmentPlan(treatmentPlanID int64) error {
	m.tpIDSubmitted = treatmentPlanID
	return nil
}
func (m *mockDoctorCLI) CreateRegimenPlan(regimen *common.RegimenPlan) (*common.RegimenPlan, error) {
	m.regimenPlanAdded = regimen
	return nil, nil
}
func (m *mockDoctorCLI) AddTreatmentsToTreatmentPlan(treatments []*common.Treatment, tpID int64) (*doctor_treatment_plan.GetTreatmentsResponse, error) {
	m.treatmentsAdded = treatments
	return nil, nil
}
func (m *mockDoctorCLI) UpdateTreatmentPlanNote(treatmentPlanID int64, note string) error {
	m.noteUpdated = note
	return nil
}

// This test is to ensure that the automatic TP creation doesn't get in the
// way of patient visits submitted with non demo domain email addresses
func TestAutomaticTP_NotForNonDemoDomains(t *testing.T) {
	mDataAPI := &mockDataAPI_DemoListener{
		patient: &common.Patient{
			Email: "test@notpatient.com",
		},
	}

	test.OK(t, automaticTPPatient(&cost.VisitChargedEvent{}, mDataAPI, nil))
}

// This test is to ensure that the automatic TP creation works as expected
// when there is no doctor picked by the patient for @patient.com or @usertesting.com
// In this case the default doctor does not have an FTP associated with the pathway either
func TestAutomaticTP_DefaultDoctor_NoFTP(t *testing.T) {
	var doctorEmailLookup string
	mDataAPI := &mockDataAPI_DemoListener{
		patient: &common.Patient{
			Email: "test@patient.com",
		},
		visit: &common.PatientVisit{
			PathwayTag: api.AcnePathwayTag,
		},
		careTeamMemberError: api.ErrNotFound("care_team_member"),
		doctorLookupByEmail: func(email string) (*common.Doctor, error) {
			doctorEmailLookup = email
			if email != "default@doctor.com" {
				return nil, api.ErrNotFound("doctor")
			}
			return &common.Doctor{}, nil
		},
	}

	mDoctorCLI := &mockDoctorCLI{
		tp: &responses.TreatmentPlan{
			ID: encoding.NewObjectID(124),
		},
	}

	test.OK(t, automaticTPPatient(&cost.VisitChargedEvent{}, mDataAPI, mDoctorCLI))

	// test to ensure that the FTP used was the stockFTP
	stockFTP := favoriteTreatmentPlans["doxy_and_tretinoin"]
	test.Equals(t, stockFTP.TreatmentList.Treatments, mDoctorCLI.treatmentsAdded)
	test.Equals(t, stockFTP.RegimenPlan.Sections, mDoctorCLI.regimenPlanAdded.Sections)
	test.Equals(t, stockFTP.Note, mDoctorCLI.noteUpdated)
	test.Equals(t, mDoctorCLI.tp.ID.Int64(), mDoctorCLI.tpIDSubmitted)

	// test to ensure that the default doctor was used
	test.Equals(t, "default@doctor.com", doctorEmailLookup)
}

// This test is to ensure that the TP pertaining to the pathway
// is picked up if one exists for the automatic creation of TP
func TestAutomaticTP_DefaultDoctor_FTPForPathway(t *testing.T) {
	pathwayTag := "test"
	var doctorEmailLookup string
	mDataAPI := &mockDataAPI_DemoListener{
		patient: &common.Patient{
			Email: "test@patient.com",
		},
		visit: &common.PatientVisit{
			PathwayTag: pathwayTag,
		},
		careTeamMemberError: api.ErrNotFound("care_team_member"),
		doctorLookupByEmail: func(email string) (*common.Doctor, error) {
			doctorEmailLookup = email
			if email != "default@doctor.com" {
				return nil, api.ErrNotFound("doctor")
			}
			return &common.Doctor{}, nil
		},
	}

	mDoctorCLI := &mockDoctorCLI{
		tp: &responses.TreatmentPlan{
			ID: encoding.NewObjectID(124),
		},
		ftpsList: []*responses.FavoriteTreatmentPlan{
			{
				Name: "ftp1",
			},
			{
				Name: "ftp2",
			},
		},
	}

	test.OK(t, automaticTPPatient(&cost.VisitChargedEvent{}, mDataAPI, mDoctorCLI))

	// test to ensure that the FTP used was the one specific to the pathway
	ftp := mDoctorCLI.ftpsList[0]
	test.Equals(t, mDoctorCLI.tp.ID.Int64(), mDoctorCLI.tpIDSubmitted)
	test.Equals(t, ftp, mDoctorCLI.ftpPicked)
	test.Equals(t, true, mDoctorCLI.regimenPlanAdded == nil)
	test.Equals(t, true, mDoctorCLI.treatmentsAdded == nil)
	test.Equals(t, "", mDoctorCLI.noteUpdated)

	// test to ensure that the default doctor was used
	test.Equals(t, "default@doctor.com", doctorEmailLookup)
}

// This test is to ensure that if no doctor is picked, but if the patient
// maps to the doctor account based on the name, then that particular doctor is picked
func TestAutomaticTP_PairDoctorBasedOnName(t *testing.T) {
	testAutomaticTP_PairDoctorBasedOnName(t, "kunal@patient.com", "kunal@doctor.com")
	testAutomaticTP_PairDoctorBasedOnName(t, "kunal+1@patient.com", "kunal@doctor.com")
	testAutomaticTP_PairDoctorBasedOnName(t, "kunal+100@patient.com", "kunal@doctor.com")
	testAutomaticTP_PairDoctorBasedOnName(t, "kunal+1@usertesting.com", "kunal@doctor.com")
}

// This test is to ensure that if a doctor is picked by the patient, then we use that doctor's account
// to create an TP and send back to patient
func TestAutomaticTP_DoctorPicked_NoFTP(t *testing.T) {
	pathwayTag := "test"
	var doctorIDLookedup int64
	mDataAPI := &mockDataAPI_DemoListener{
		patient: &common.Patient{
			Email: "test@patient.com",
		},
		visit: &common.PatientVisit{
			PathwayTag: pathwayTag,
		},
		careTeamMember: &common.CareProviderAssignment{
			ProviderID:   1224,
			ProviderRole: api.RoleDoctor,
		},
		doctorLookupByID: func(id int64) (*common.Doctor, error) {
			doctorIDLookedup = id
			return &common.Doctor{
				DoctorID: encoding.NewObjectID(id),
				Email:    "doctorPicked@test.com",
			}, nil
		},
	}

	mDoctorCLI := &mockDoctorCLI{
		tp: &responses.TreatmentPlan{
			ID: encoding.NewObjectID(124),
		},
	}

	test.OK(t, automaticTPPatient(&cost.VisitChargedEvent{}, mDataAPI, mDoctorCLI))

	// test to ensure that the doctor picked was the one part of the case
	test.Equals(t, doctorIDLookedup, mDataAPI.careTeamMember.ProviderID)

	// test to ensure that the FTP used was the stockFTP
	stockFTP := favoriteTreatmentPlans["doxy_and_tretinoin"]
	test.Equals(t, stockFTP.TreatmentList.Treatments, mDoctorCLI.treatmentsAdded)
	test.Equals(t, stockFTP.RegimenPlan.Sections, mDoctorCLI.regimenPlanAdded.Sections)
	test.Equals(t, stockFTP.Note, mDoctorCLI.noteUpdated)
	test.Equals(t, mDoctorCLI.tp.ID.Int64(), mDoctorCLI.tpIDSubmitted)
}

// This test is to ensure that if a doctor is picked by the patient, then we use the doctor's account
// to create a TP and also use any FTP associated with the pathway
func TestAutomaticTP_DoctorPicked_FTPForPathway(t *testing.T) {
	pathwayTag := "test"
	var doctorIDLookedup int64
	mDataAPI := &mockDataAPI_DemoListener{
		patient: &common.Patient{
			Email: "test@patient.com",
		},
		visit: &common.PatientVisit{
			PathwayTag: pathwayTag,
		},
		careTeamMember: &common.CareProviderAssignment{
			ProviderID:   1224,
			ProviderRole: api.RoleDoctor,
		},
		doctorLookupByID: func(id int64) (*common.Doctor, error) {
			doctorIDLookedup = id
			return &common.Doctor{
				DoctorID: encoding.NewObjectID(id),
				Email:    "doctorPicked@test.com",
			}, nil
		},
	}

	mDoctorCLI := &mockDoctorCLI{
		tp: &responses.TreatmentPlan{
			ID: encoding.NewObjectID(124),
		},
		ftpsList: []*responses.FavoriteTreatmentPlan{
			{
				Name: "ftp1",
			},
			{
				Name: "ftp2",
			},
		},
	}

	test.OK(t, automaticTPPatient(&cost.VisitChargedEvent{}, mDataAPI, mDoctorCLI))

	// test to ensure that the doctor picked was the one part of the case
	test.Equals(t, doctorIDLookedup, mDataAPI.careTeamMember.ProviderID)

	// test to ensure that the FTP specific to the pathway was used
	ftp := mDoctorCLI.ftpsList[0]
	test.Equals(t, mDoctorCLI.tp.ID.Int64(), mDoctorCLI.tpIDSubmitted)
	test.Equals(t, ftp, mDoctorCLI.ftpPicked)
	test.Equals(t, true, mDoctorCLI.regimenPlanAdded == nil)
	test.Equals(t, true, mDoctorCLI.treatmentsAdded == nil)
	test.Equals(t, "", mDoctorCLI.noteUpdated)
}

func testAutomaticTP_PairDoctorBasedOnName(t *testing.T, patientEmail, doctorEmail string) {
	pathwayTag := "test"
	var doctorEmailLookup string
	mDataAPI := &mockDataAPI_DemoListener{
		patient: &common.Patient{
			Email: patientEmail,
		},
		visit: &common.PatientVisit{
			PathwayTag: pathwayTag,
		},
		careTeamMemberError: api.ErrNotFound("care_team_member"),
		doctorLookupByEmail: func(email string) (*common.Doctor, error) {
			doctorEmailLookup = email
			return &common.Doctor{}, nil
		},
	}

	mDoctorCLI := &mockDoctorCLI{
		tp: &responses.TreatmentPlan{
			ID: encoding.NewObjectID(124),
		},
	}

	test.OK(t, automaticTPPatient(&cost.VisitChargedEvent{}, mDataAPI, mDoctorCLI))

	// test to ensure that the doctor picked was kunal@doctor.com
	test.Equals(t, doctorEmail, doctorEmailLookup)

	// test to ensure that the FTP used was the stockFTP
	stockFTP := favoriteTreatmentPlans["doxy_and_tretinoin"]
	test.Equals(t, stockFTP.TreatmentList.Treatments, mDoctorCLI.treatmentsAdded)
	test.Equals(t, stockFTP.RegimenPlan.Sections, mDoctorCLI.regimenPlanAdded.Sections)
	test.Equals(t, stockFTP.Note, mDoctorCLI.noteUpdated)
	test.Equals(t, mDoctorCLI.tp.ID.Int64(), mDoctorCLI.tpIDSubmitted)
}
