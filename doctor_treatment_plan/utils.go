package doctor_treatment_plan

import (
	"errors"
	"fmt"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
)

const (
	question_acne_diagnosis = "q_acne_diagnosis"
	question_acne_severity  = "q_acne_severity"
	question_acne_type      = "q_acne_type"
	question_rosacea_type   = "q_acne_rosacea_type"
)

const VersionedTreatmentPlanNote = "Here is your revised treatment plan."

func fillInTreatmentPlan(tp *common.TreatmentPlan, doctorID int64, dataAPI api.DataAPI) error {
	var err error

	tp.TreatmentList = &common.TreatmentList{}
	tp.TreatmentList.Treatments, err = dataAPI.GetTreatmentsBasedOnTreatmentPlanId(tp.Id.Int64())
	if err != nil {
		return fmt.Errorf("Unable to get treatments for treatment plan: %s", err)
	}

	tp.RegimenPlan, err = dataAPI.GetRegimenPlanForTreatmentPlan(tp.Id.Int64())
	if err != nil {
		return fmt.Errorf("Unable to get regimen plan for treatment plan: %s", err)
	}

	tp.Note, err = dataAPI.GetTreatmentPlanNote(tp.Id.Int64())
	if err != nil && err != api.NoRowsError {
		return fmt.Errorf("Unable to get note for treatment plan: %s", err)
	}

	// only populate the draft state if we are dealing with a draft treatment plan and the same doctor
	// that owns it is requesting the treatment plan (so that they can edit it)
	if tp.DoctorId.Int64() == doctorID && tp.InDraftMode() {
		tp.RegimenPlan.AllSteps, err = dataAPI.GetRegimenStepsForDoctor(tp.DoctorId.Int64())
		if err != nil {
			return err
		}

		setCommittedStateForEachSection(tp)

		if err := populateContentSourceIntoTreatmentPlan(tp, dataAPI, doctorID); err == api.NoRowsError {
			return errors.New("No treatment plan found")
		} else if err != nil {
			return err
		}
	}
	return err
}

func setCommittedStateForEachSection(drTreatmentPlan *common.TreatmentPlan) {
	// depending on which sections have data in them, mark them to be committed or uncommitted
	// note that we intentionally treat a section with no data to be in the UNCOMMITTED state so as
	// to ensure that the doctor actually wanted to leave a particular section blank

	if len(drTreatmentPlan.TreatmentList.Treatments) > 0 {
		drTreatmentPlan.TreatmentList.Status = api.STATUS_COMMITTED
	} else {
		drTreatmentPlan.TreatmentList.Status = api.STATUS_UNCOMMITTED
	}

	if len(drTreatmentPlan.RegimenPlan.Sections) > 0 {
		drTreatmentPlan.RegimenPlan.Status = api.STATUS_COMMITTED
	} else {
		drTreatmentPlan.RegimenPlan.Status = api.STATUS_UNCOMMITTED
	}
}

func populateContentSourceIntoTreatmentPlan(tp *common.TreatmentPlan, dataAPI api.DataAPI, doctorID int64) error {
	// only continue if the content source of the treatment plan is a favorite treatment plan
	if tp.ContentSource == nil {
		return nil
	}

	switch tp.ContentSource.Type {
	case common.TPContentSourceTypeTreatmentPlan:
		prevTP, err := dataAPI.GetTreatmentPlan(tp.ContentSource.ID.Int64(), doctorID)
		if err != nil {
			return err
		}

		if len(tp.TreatmentList.Treatments) == 0 {
			fillTreatmentsIntoTreatmentPlan(prevTP.TreatmentList.Treatments, tp)
		}

		if len(tp.RegimenPlan.Sections) == 0 {
			fillRegimenSectionsIntoTreatmentPlan(prevTP.RegimenPlan.Sections, tp)
		}

		if tp.Note == "" {
			tp.Note = VersionedTreatmentPlanNote
		}
	case common.TPContentSourceTypeFTP:
		ftp, err := dataAPI.GetFavoriteTreatmentPlan(tp.ContentSource.ID.Int64())
		if err != nil {
			return err
		}

		// The assumption here is that all components of a treatment plan that are already populated
		// match the items in the favorite treatment plan, if there exists a mapping to indicate that this
		// treatment plan must be filled in from a favorite treatment plan. The reason that we don't just write over
		// the items that do already belong in the treatment plan is to maintain the ids of the items that have been committed
		// to the database as part of the treatment plan.

		// populate treatments
		if len(tp.TreatmentList.Treatments) == 0 {
			fillTreatmentsIntoTreatmentPlan(ftp.TreatmentList.Treatments, tp)
		}

		// populate regimen plan
		if len(tp.RegimenPlan.Sections) == 0 {
			fillRegimenSectionsIntoTreatmentPlan(ftp.RegimenPlan.Sections, tp)
		}

		if tp.Note == "" {
			tp.Note = ftp.Note
		}
	}

	return nil

}

func fillRegimenSectionsIntoTreatmentPlan(sourceRegimenSections []*common.RegimenSection, treatmentPlan *common.TreatmentPlan) {
	treatmentPlan.RegimenPlan.Sections = make([]*common.RegimenSection, len(sourceRegimenSections))

	for i, regimenSection := range sourceRegimenSections {
		treatmentPlan.RegimenPlan.Sections[i] = &common.RegimenSection{
			Name:  regimenSection.Name,
			Steps: make([]*common.DoctorInstructionItem, len(regimenSection.Steps)),
		}

		for j, regimenStep := range regimenSection.Steps {
			treatmentPlan.RegimenPlan.Sections[i].Steps[j] = &common.DoctorInstructionItem{
				ParentID: regimenStep.ParentID,
				Text:     regimenStep.Text,
			}
		}
	}
}

func fillTreatmentsIntoTreatmentPlan(sourceTreatments []*common.Treatment, treatmentPlan *common.TreatmentPlan) {
	treatmentPlan.TreatmentList.Treatments = make([]*common.Treatment, len(sourceTreatments))
	for i, treatment := range sourceTreatments {
		treatmentPlan.TreatmentList.Treatments[i] = &common.Treatment{
			DrugDBIds:               treatment.DrugDBIds,
			DrugInternalName:        treatment.DrugInternalName,
			DrugName:                treatment.DrugName,
			DrugRoute:               treatment.DrugRoute,
			DosageStrength:          treatment.DosageStrength,
			DispenseValue:           treatment.DispenseValue,
			DispenseUnitId:          treatment.DispenseUnitId,
			DispenseUnitDescription: treatment.DispenseUnitDescription,
			NumberRefills:           treatment.NumberRefills,
			SubstitutionsAllowed:    treatment.SubstitutionsAllowed,
			DaysSupply:              treatment.DaysSupply,
			PharmacyNotes:           treatment.PharmacyNotes,
			PatientInstructions:     treatment.PatientInstructions,
			CreationDate:            treatment.CreationDate,
			OTC:                     treatment.OTC,
			IsControlledSubstance:    treatment.IsControlledSubstance,
			SupplementalInstructions: treatment.SupplementalInstructions,
		}
	}
}

func sendCaseMessageAndPublishTPActivatedEvent(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, treatmentPlan *common.TreatmentPlan,
	doctor *common.Doctor, message string) error {
	// only send a case message if one has not already been sent for this particular
	// treatment plan for this particular case
	caseMessage, err := dataAPI.CaseMessageForAttachment(common.AttachmentTypeTreatmentPlan,
		treatmentPlan.Id.Int64(), doctor.PersonId, treatmentPlan.PatientCaseId.Int64())
	if err != api.NoRowsError && err != nil {
		return err
	} else if err == api.NoRowsError {
		caseMessage = &common.CaseMessage{
			CaseID:   treatmentPlan.PatientCaseId.Int64(),
			PersonID: doctor.PersonId,
			Body:     message,
			Attachments: []*common.CaseMessageAttachment{
				&common.CaseMessageAttachment{
					ItemType: common.AttachmentTypeTreatmentPlan,
					ItemID:   treatmentPlan.Id.Int64(),
				},
			},
		}
		if _, err := dataAPI.CreateCaseMessage(caseMessage); err != nil {
			return err
		}
	}

	patientVisitID, err := dataAPI.GetPatientVisitIdFromTreatmentPlanId(treatmentPlan.Id.Int64())
	if err != nil {
		return err
	}

	// Publish event that treamtent plan was created
	dispatcher.Publish(&TreatmentPlanActivatedEvent{
		PatientId:     treatmentPlan.PatientId,
		DoctorId:      doctor.DoctorId.Int64(),
		VisitId:       patientVisitID,
		TreatmentPlan: treatmentPlan,
		Message:       caseMessage,
	})

	return nil
}
