package doctor_queue

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/patient_file"
	"github.com/sprucehealth/backend/patient_visit"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
)

var (
	defaultMinutesThreshold = time.Duration(15)

	// ExpireDuration is the maximum time between actions on the patient case that the doctor
	// has to maintain their claim on the case.
	ExpireDuration = defaultMinutesThreshold * time.Minute

	// GracePeriod is to ensure that any pending/ongoing requests
	// have ample time to complete before yanking access from
	// doctors who's claim on the case has expired
	GracePeriod = 5 * time.Minute

	// timePeriodBetweenChecks is the frequency with which the checker runs
	timePeriodBetweenChecks = 5 * time.Minute
)

func initJumpBallCaseQueueListeners(dataAPI api.DataAPI, analyticsLogger analytics.Logger,
	dispatcher *dispatch.Dispatcher, statsRegistry metrics.Registry, jbcqMinutesThreshold int) {
	if jbcqMinutesThreshold > 0 {
		ExpireDuration = time.Duration(jbcqMinutesThreshold) * time.Minute
	}

	tempClaimSucess := metrics.NewCounter()
	tempClaimFailure := metrics.NewCounter()
	permanentClaimSuccess := metrics.NewCounter()
	permanentClaimFailure := metrics.NewCounter()
	claimExtensionSucess := metrics.NewCounter()
	claimExtensionFailure := metrics.NewCounter()

	statsRegistry.Add("temp_claim/success", tempClaimSucess)
	statsRegistry.Add("temp_claim/failure", tempClaimFailure)
	statsRegistry.Add("perm_claim/success", permanentClaimSuccess)
	statsRegistry.Add("perm_claim/failure", permanentClaimFailure)
	statsRegistry.Add("claim_extension/success", claimExtensionSucess)
	statsRegistry.Add("claim_extension/failure", claimExtensionFailure)

	// Grant temporary access to the patient case for an unclaimed case to the doctor requesting access to the case
	dispatcher.Subscribe(func(ev *patient_file.PatientVisitOpenedEvent) error {
		// nothing to do if it wasn't a doctor that opened the patient file
		if ev.Role != api.RoleDoctor {
			return nil
		} else if ev.PatientVisit.IsFollowup {
			return nil
		}

		// nothing to do if the case has already claimed
		patientCase, err := dataAPI.GetPatientCaseFromID(ev.PatientVisit.PatientCaseID.Int64())
		if err != nil {
			tempClaimFailure.Inc(1)
			return err
		} else if patientCase.Claimed {
			return nil
		}

		// check if the case has been temporarily claimed
		tempClaimedItem, err := dataAPI.GetTempClaimedCaseInQueue(ev.PatientVisit.PatientCaseID.Int64())
		if !api.IsErrNotFound(err) && err != nil {
			tempClaimFailure.Inc(1)
			return err
		} else if tempClaimedItem != nil {
			// nothing to do if the case is currently claimed
			return nil
		}

		if err := dataAPI.TemporarilyClaimCaseAndAssignDoctorToCaseAndPatient(ev.DoctorID, patientCase, ExpireDuration); err != nil {
			tempClaimFailure.Inc(1)
			golog.Errorf("Unable to temporarily assign the patient visit to the doctor: %s", err)
			return err
		}
		tempClaimSucess.Inc(1)

		return nil
	})

	// Extend the doctor's claim on the patient case if the doctor modifies the diagnosis associated with the case
	dispatcher.Subscribe(func(ev *patient_visit.DiagnosisModifiedEvent) error {
		patientCase, err := dataAPI.GetPatientCaseFromPatientVisitID(ev.PatientVisitID)
		if err != nil {
			golog.Errorf("Unable to get patiente case from patient visit: %s", err)
			return err
		}

		if !patientCase.Claimed {
			if err := dataAPI.ExtendClaimForDoctor(ev.DoctorID, patientCase.PatientID.Int64(), patientCase.ID.Int64(), ExpireDuration); err != nil {
				golog.Errorf("Unable to extend the claim on the case for the doctor: %s", err)
				claimExtensionFailure.Inc(1)
				return err
			}
			claimExtensionSucess.Inc(1)
		}
		return nil
	})

	// extend the doctor's claim on the patient case if the doctor modifies any aspect of the treatment plan
	dispatcher.Subscribe(func(ev *doctor_treatment_plan.TreatmentPlanUpdatedEvent) error {
		return extendClaimOnTreatmentPlanModification(
			ev.TreatmentPlanID,
			ev.DoctorID,
			dataAPI,
			analyticsLogger,
			claimExtensionSucess,
			claimExtensionFailure)
	})

	// If the doctor successfully submits a treatment plan for an unclaimed case, the case is then considered
	// claimed by the doctor and the doctor is assigned to the case and made part of the patient's care team
	dispatcher.Subscribe(func(ev *doctor_treatment_plan.TreatmentPlanSubmittedEvent) error {
		return permanentlyAssignDoctorToCaseAndPatient(ev.VisitID, ev.TreatmentPlan.DoctorID.Int64(), dataAPI,
			analyticsLogger, permanentClaimSuccess, permanentClaimFailure)

	})

	// If the doctor marks a case unsuitable for spruce, it is also considered claimed by the doctor
	// with the doctor permanently being assigned to the case and patient
	dispatcher.Subscribe(func(ev *patient_visit.PatientVisitMarkedUnsuitableEvent) error {
		return permanentlyAssignDoctorToCaseAndPatient(ev.PatientVisitID, ev.DoctorID, dataAPI,
			analyticsLogger, permanentClaimSuccess, permanentClaimFailure)
	})

	// If the doctor sends a message to the patient for an unclaimed case, then the case
	// should get permanently assigned to the doctor and the patient visit put into the doctor's inbox
	// for the doctor to come back to.
	dispatcher.Subscribe(func(ev *messages.PostEvent) error {
		if ev.Case.Claimed {
			return nil
		}

		if ev.Person.RoleType == api.RoleDoctor {

			tempClaimedItem, err := dataAPI.GetTempClaimedCaseInQueue(ev.Case.ID.Int64())
			if api.IsErrNotFound(err) {
				// nothing to do if case is not temporarily claimed
				return nil
			} else if err != nil {
				golog.Errorf("Unable to get temporarily claimed item in queue: %s", err)
				return err
			}

			if patient, err := dataAPI.Patient(ev.Case.PatientID.Int64(), true); err != nil {
				golog.Errorf("Unable to load patient: %s", err.Error())
			} else if err := dataAPI.UpdateDoctorQueue([]*api.DoctorQueueUpdate{
				{
					Action: api.DQActionInsert,
					QueueItem: &api.DoctorQueueItem{
						DoctorID:         ev.Person.Doctor.DoctorID.Int64(),
						PatientID:        ev.Case.PatientID.Int64(),
						ItemID:           tempClaimedItem.ItemID,
						Status:           api.DQItemStatusOngoing,
						EventType:        api.DQEventTypePatientVisit,
						Description:      fmt.Sprintf("Continue reviewing visit with %s %s", patient.FirstName, patient.LastName),
						ShortDescription: "New visit",
						ActionURL:        app_url.ViewPatientVisitInfoAction(ev.Case.PatientID.Int64(), tempClaimedItem.ItemID, ev.Case.ID.Int64()),
						Tags:             tempClaimedItem.Tags,
					},
				},
			}); err != nil {
				golog.Errorf("Unable to insert item into the doctor queue: %s", err)
				return err
			}

			if err := dataAPI.TransitionToPermanentAssignmentOfDoctorToCaseAndPatient(ev.Person.Doctor.DoctorID.Int64(), ev.Case); err != nil {
				golog.Errorf("Unable to permanently assign doctor to case and patient: %s", err)
				permanentClaimFailure.Inc(1)
				return err
			}
			permanentClaimSuccess.Inc(1)
		}
		return nil
	})

	dispatcher.Subscribe(func(ev *messages.CaseAssignEvent) error {
		if ev.Case.Claimed {
			return nil
		}

		// permanently assign the case to the doctor if it was the doctor that assigned the case to the MA
		if ev.Person.RoleType == api.RoleDoctor {
			tempClaimedItem, err := dataAPI.GetTempClaimedCaseInQueue(ev.Case.ID.Int64())
			if api.IsErrNotFound(err) {
				// nothing to do if case is not temporarily claimed
				return nil
			} else if err != nil {
				golog.Errorf("Unable to get temporarily claimed case from unclaimed queue: %s", err)
				permanentClaimFailure.Inc(1)
				return err
			}

			if patient, err := dataAPI.Patient(ev.Case.PatientID.Int64(), true); err != nil {
				golog.Errorf("Unable to load patient: %s", err.Error())
			} else if err := dataAPI.UpdateDoctorQueue([]*api.DoctorQueueUpdate{
				{
					Action: api.DQActionInsert,
					QueueItem: &api.DoctorQueueItem{
						DoctorID:         ev.Person.RoleID,
						PatientID:        ev.Case.PatientID.Int64(),
						ItemID:           tempClaimedItem.ItemID,
						Status:           api.DQItemStatusOngoing,
						EventType:        api.DQEventTypePatientVisit,
						Description:      fmt.Sprintf("Continue reviewing visit with %s %s", patient.FirstName, patient.LastName),
						ShortDescription: fmt.Sprintf("New visit"),
						ActionURL:        app_url.ViewPatientVisitInfoAction(ev.Case.PatientID.Int64(), tempClaimedItem.ItemID, ev.Case.ID.Int64()),
						Tags:             tempClaimedItem.Tags,
					},
				},
			}); err != nil {
				golog.Errorf("Unable to insert item into the doctor queue: %s", err)
				return err
			}

			if err := dataAPI.TransitionToPermanentAssignmentOfDoctorToCaseAndPatient(ev.Person.RoleID, ev.Case); err != nil {
				golog.Errorf("Unable to transition to permanent assignment of case to doctor: %s", err)
				permanentClaimFailure.Inc(1)
				return err
			}

			permanentClaimSuccess.Inc(1)
		}
		return nil
	})
}

func permanentlyAssignDoctorToCaseAndPatient(patientVisitID, doctorID int64, dataAPI api.DataAPI,
	analyticsLogger analytics.Logger, permClaimSuccess, permClaimFailure *metrics.Counter) error {
	patientCase, err := dataAPI.GetPatientCaseFromPatientVisitID(patientVisitID)
	if err != nil {
		return err
	}

	if !patientCase.Claimed {
		if err := dataAPI.TransitionToPermanentAssignmentOfDoctorToCaseAndPatient(doctorID, patientCase); err != nil {
			golog.Errorf("Unable to permanently assign doctor to case and patient: %s", err)
			permClaimFailure.Inc(1)
			return err
		}

		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:     "jbcq_perm_assign",
				Timestamp: analytics.Time(time.Now()),
				DoctorID:  doctorID,
				CaseID:    patientCase.ID.Int64(),
			},
		})

		permClaimSuccess.Inc(1)
	}

	return nil
}

func extendClaimOnTreatmentPlanModification(treatmentPlanID, doctorID int64, dataAPI api.DataAPI, analyticsLogger analytics.Logger, claimExtensionSucess, claimExtensionFailure *metrics.Counter) error {
	patientCase, err := dataAPI.GetPatientCaseFromTreatmentPlanID(treatmentPlanID)
	if err != nil {
		golog.Errorf("Unable to get patient case from treatment plan id: %s", err)
		return err
	}

	if !patientCase.Claimed {
		if err := dataAPI.ExtendClaimForDoctor(doctorID, patientCase.PatientID.Int64(), patientCase.ID.Int64(), ExpireDuration); err != nil {
			golog.Errorf("Unable to extend claim on the case for the doctor: %s", err)
			claimExtensionFailure.Inc(1)
			return err
		}

		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Timestamp:       analytics.Time(time.Now()),
				Event:           "jbcq_claim_extend",
				DoctorID:        doctorID,
				CaseID:          patientCase.ID.Int64(),
				TreatmentPlanID: treatmentPlanID,
			},
		})
		claimExtensionSucess.Inc(1)
	}

	return nil
}
