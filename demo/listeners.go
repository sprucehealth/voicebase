package demo

import (
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiclient"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost"
	"github.com/sprucehealth/backend/doctor"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/patient_file"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/www/dronboard"
)

type doctorCLI interface {
	SetToken(token string)
	Auth(email, password string) (*doctor.AuthenticationResponse, error)
	ListFavoriteTreatmentPlans() ([]*responses.PathwayFTPGroup, error)
	ReviewVisit(patientVisitID int64) (*patient_file.VisitReviewResponse, error)
	PickTreatmentPlanForVisit(visitID int64, ftp *responses.FavoriteTreatmentPlan) (*responses.TreatmentPlan, error)
	SubmitTreatmentPlan(treatmentPlanID int64) error
	CreateRegimenPlan(regimen *common.RegimenPlan) (*common.RegimenPlan, error)
	AddTreatmentsToTreatmentPlan(treatments []*common.Treatment, tpID int64) (*doctor_treatment_plan.GetTreatmentsResponse, error)
	ListFavoriteTreatmentPlansForTag(pathwayTag string) ([]*responses.FavoriteTreatmentPlan, error)
	UpdateTreatmentPlanNote(treatmentPlanID int64, note string) error
}

func InitListeners(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, domain string, doseSpotClinicianID int64) {

	// only setup the listeners in non-production environments
	if environment.IsProd() {
		return
	}

	// On Visit submission, automatically submit a treamtent plan for patients
	// created under certain demo domains
	dispatcher.SubscribeAsync(func(ev *cost.VisitChargedEvent) error {
		cli := &apiclient.DoctorClient{
			Config: apiclient.Config{
				BaseURL:    LocalServerURL,
				HostHeader: domain,
			},
		}

		return automaticTPPatient(ev, dataAPI, cli)
	})

	dispatcher.Subscribe(func(ev *dronboard.DoctorRegisteredEvent) error {
		// update the doctor credentials to assign a default dosespot clinicianID
		// which can be used to treat cases
		if err := dataAPI.UpdateDoctor(ev.DoctorID, &api.DoctorUpdate{DosespotClinicianID: &doseSpotClinicianID}); err != nil {
			golog.Errorf("Unable to set a default dosespot clinicianID for the doctor: %s", err)
			return err
		}

		// TODO: don't assume acne
		if err := dataAPI.ClaimTrainingSet(ev.DoctorID, api.AcnePathwayTag); err != nil {
			golog.Errorf("Unable to claim training set for doctor: %s", err)
			return err
		}
		return nil
	})
}

func automaticTPPatient(ev *cost.VisitChargedEvent, dataAPI api.DataAPI, cli doctorCLI) error {
	patient, err := dataAPI.GetPatientFromID(ev.PatientID)
	if err != nil {
		golog.Errorf("Unable to get patient from id: %s", err)
		return nil
	}

	demoDomain := onDemoDomain(patient.Email)
	// nothing to do with visit if we are not on a demo domain
	if demoDomain == "" {
		return nil
	}

	if !environment.IsTest() {
		// sleep to wait for a bit before sending treatment plan to patient
		time.Sleep(15 * time.Second)
	}

	// default FTP to use if no pathway specific ones are available
	stockFTP, ok := favoriteTreatmentPlans["doxy_and_tretinoin"]
	if !ok {
		golog.Errorf("Unable to find the favorite treatment plan with which to create the treatment plan")
		return nil
	}

	// Identify doctor
	doctor, err := pickDoctor(patient, ev, demoDomain, dataAPI)
	if api.IsErrNotFound(err) {
		golog.Errorf("No default doctor identified for domain so not sending automated treatment plan.")
		return nil
	} else if err != nil {
		golog.Errorf("Unable to identify doctor based on patient email: %s", err)
		return nil
	}

	// login as doctor to get token
	res, err := cli.Auth(doctor.Email, "12345")
	if err != nil {
		golog.Errorf("Unable to login as doctor: %s", err)
		return nil
	}
	cli.SetToken(res.Token)

	visit, err := dataAPI.GetPatientVisitFromID(ev.VisitID)
	if err != nil {
		golog.Errorf(("Unable to get visit from id: %s"), err)
		return nil
	}

	ftps, err := cli.ListFavoriteTreatmentPlansForTag(visit.PathwayTag)
	if err != nil {
		golog.Errorf("Unable to get ftps for doctor: %s", err)
		return nil
	}

	var ftp *responses.FavoriteTreatmentPlan
	if len(ftps) > 0 {
		ftp = ftps[0]
	}

	// Get doctor to start reviewing the case
	_, err = cli.ReviewVisit(ev.VisitID)
	if err != nil {
		golog.Errorf("Unable to review patient visit: %s", err)
		return nil
	}

	// Get doctor to pick a treatment plan
	tp, err := cli.PickTreatmentPlanForVisit(ev.VisitID, ftp)
	if err != nil {
		golog.Errorf("Unable to pick treatment plan for visit: %s", err)
		return nil
	}

	// manually add parts of the TP only if an FTP doesn't exist
	// for the doctor
	if ftp == nil {
		_, err = cli.CreateRegimenPlan(&common.RegimenPlan{
			TreatmentPlanID: tp.ID,
			Sections:        stockFTP.RegimenPlan.Sections,
		})
		if err != nil {
			golog.Errorf("Unable to add regimen to treatment plan: %s", err)
			return nil
		}

		// Get doctor to add treatments
		_, err = cli.AddTreatmentsToTreatmentPlan(stockFTP.TreatmentList.Treatments, tp.ID.Int64())
		if err != nil {
			golog.Errorf("Unable to add treatments to treatment plan: %s", err)
			return nil
		}

		if err := cli.UpdateTreatmentPlanNote(tp.ID.Int64(), stockFTP.Note); err != nil {
			golog.Errorf("Unable to set treatment plan note: %s", err.Error())
			return nil
		}
	}

	if err := cli.SubmitTreatmentPlan(tp.ID.Int64()); err != nil {
		golog.Errorf("Unable to submit treatment plan: %s", err.Error())
		return nil
	}

	return nil
}

var demoDomains = []string{"@patient.com", "@usertesting.com"}

func onDemoDomain(email string) string {
	for _, domain := range demoDomains {
		if strings.HasSuffix(email, domain) {
			return domain
		}
	}

	return ""
}

func pickDoctor(patient *common.Patient, ev *cost.VisitChargedEvent, domain string, dataAPI api.DataAPI) (*common.Doctor, error) {

	// check if the case already has a doctor assigned, if so then pick that doctor
	member, err := dataAPI.GetActiveCareTeamMemberForCase(api.RoleDoctor, ev.PatientCaseID)
	if !api.IsErrNotFound(err) && err != nil {
		return nil, err
	}

	// return the doctor if an active doctor is found on the case
	if member != nil {
		doctor, err := dataAPI.GetDoctorFromID(member.ProviderID)
		if err != nil {
			return nil, err
		}

		return doctor, err
	}

	// identify the username from the email address. the username
	// can be of the form username@domain or username+N@domain
	var username string
	email := patient.Email
	index := strings.IndexRune(email, '+')
	if index == -1 {
		username = email[:strings.IndexRune(email, '@')]
	} else {
		username = email[:index]
	}

	// check if doctor account exists with specified email
	doctor, err := dataAPI.GetDoctorWithEmail(username + "@doctor.com")
	if api.IsErrNotFound(err) {
		// if not then find a default doctor on the same domain
		doctor, err = dataAPI.GetDoctorWithEmail("default@doctor.com")
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return doctor, nil
}
