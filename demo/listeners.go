package demo

import (
	"fmt"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/patient_visit"
)

func InitListeners(dataAPI api.DataAPI, domain string) {

	// only setup the listeners in non-production environments
	if environment.IsProd() {
		return
	}

	// On Visit submission, automatically submit a treamtent plan for patients
	// created under certain demo domains
	dispatch.Default.Subscribe(func(ev *patient_visit.VisitChargedEvent) error {
		go func() {

			patient, err := dataAPI.GetPatientFromId(ev.PatientID)
			if err != nil {
				golog.Errorf("Unable to get patient from id: %s", err)
				return
			}

			demoDomain := onDemoDomain(patient.Email)
			// nothing to do with visit if we are not on a demo domain
			if demoDomain == "" {
				return
			}

			// sleep to wait for a bit before sending treatment plan to patient
			time.Sleep(15 * time.Second)

			favoriteTreatmentPlan, ok := favoriteTreatmentPlans["doxy_and_tretinoin"]
			if !ok {
				golog.Errorf("Unable to find the favorite treatment plan with which to create the treatment plan")
				return
			}

			// Identify doctor
			doctor, err := pickDoctorBasedOnPatientEmail(patient.Email, demoDomain, dataAPI)
			if err == api.NoRowsError {
				golog.Errorf("No default doctor identified for domain so not sending automated treatment plan.")
				return
			} else if err != nil {
				golog.Errorf("Unable to identify doctor based on patient email: %s", err)
				return
			}

			// login as doctor to get token
			token, _, err := loginAsDoctor(doctor.Email, "12345", domain)
			if err != nil {
				golog.Errorf("Unable to login as doctor: %s", err)
				return
			}

			authHeader := "token " + token

			// Get doctor to start reviewing the case
			if err := reviewPatientVisit(ev.VisitID, authHeader, domain); err != nil {
				golog.Errorf("Unable to review patient visit: %s", err)
				return
			}

			// Get doctor to pick a treatment plan
			tpResponse, err := pickTreatmentPlan(ev.VisitID, authHeader, domain)
			if err != nil {
				golog.Errorf("Unable to pick treatment plan for visit: %s", err)
				return
			}

			// Get doctor to add regimen steps
			regimenSteps, err := dataAPI.GetRegimenStepsForDoctor(doctor.DoctorId.Int64())
			if err != nil {
				golog.Errorf("unable to get regimen steps for doctor: %s", err)
				return
			}

			_, err = addRegimenToTreatmentPlan(&common.RegimenPlan{
				AllRegimenSteps: regimenSteps,
				RegimenSections: favoriteTreatmentPlan.RegimenPlan.RegimenSections,
				TreatmentPlanId: tpResponse.TreatmentPlan.Id,
			}, authHeader, domain)
			if err != nil {
				golog.Errorf("Unable to add regimen to treatment plan: %s", err)
				return
			}

			// Get doctor to add treatments
			if err := addTreatmentsToTreatmentPlan(favoriteTreatmentPlan.TreatmentList.Treatments, tpResponse.TreatmentPlan.Id.Int64(), authHeader, domain); err != nil {
				golog.Errorf("Unable to add treatments to treatment plan: %s", err)
				return
			}

			// Submit treatment plan back to patient
			message := fmt.Sprintf(messageForTreatmentPlan, patient.FirstName, doctor.LastName)
			if err := submitTreatmentPlan(tpResponse.TreatmentPlan.Id.Int64(), message, authHeader, domain); err != nil {
				golog.Errorf("Unable to submit treatment plan: %s", err)
				return
			}
		}()

		return nil
	})

}

var demoDomains = []string{"patient.com", "usertesting.com"}

func onDemoDomain(email string) string {
	for _, domain := range demoDomains {
		if strings.HasSuffix(email, domain) {
			return domain
		}
	}

	return ""
}

func pickDoctorBasedOnPatientEmail(email, domain string, dataAPI api.DataAPI) (*common.Doctor, error) {
	// identify the username from the email address. the username
	// can be of the form username@domain or username+N@domain
	var username string
	index := strings.IndexRune(email, '+')
	if index == -1 {
		username = email[:strings.IndexRune(email, '@')]
	} else {
		username = email[:index]
	}

	// check if doctor account exists with specified email
	doctor, err := dataAPI.GetDoctorWithEmail(username + "@doctor.com")
	if err == api.NoRowsError {
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
