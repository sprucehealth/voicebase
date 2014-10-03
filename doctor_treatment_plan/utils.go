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

func fillInTreatmentPlan(drTreatmentPlan *common.DoctorTreatmentPlan, doctorId int64, dataApi api.DataAPI) error {
	var err error

	drTreatmentPlan.TreatmentList = &common.TreatmentList{}
	drTreatmentPlan.TreatmentList.Treatments, err = dataApi.GetTreatmentsBasedOnTreatmentPlanId(drTreatmentPlan.Id.Int64())
	if err != nil {
		return fmt.Errorf("Unable to get treatments for treatment plan: %s", err)
	}

	drTreatmentPlan.RegimenPlan = &common.RegimenPlan{}
	drTreatmentPlan.RegimenPlan, err = dataApi.GetRegimenPlanForTreatmentPlan(drTreatmentPlan.Id.Int64())
	if err != nil {
		return fmt.Errorf("Unable to get regimen plan for treatment plan: %s", err)
	}

	drTreatmentPlan.Advice = &common.Advice{}
	drTreatmentPlan.Advice.SelectedAdvicePoints, err = dataApi.GetAdvicePointsForTreatmentPlan(drTreatmentPlan.Id.Int64())
	if err != nil {
		return fmt.Errorf("Unable to get advice points for treatment plan")
	}

	// only populate the draft state if we are dealing with a draft treatment plan and the same doctor
	// that owns it is requesting the treatment plan (so that they can edit it)
	if drTreatmentPlan.DoctorId.Int64() == doctorId && drTreatmentPlan.InDraftMode() {
		drTreatmentPlan.RegimenPlan.AllRegimenSteps, err = dataApi.GetRegimenStepsForDoctor(drTreatmentPlan.DoctorId.Int64())
		if err != nil {
			return err
		}

		drTreatmentPlan.Advice.AllAdvicePoints, err = dataApi.GetAdvicePointsForDoctor(drTreatmentPlan.DoctorId.Int64())
		if err != nil {
			return err
		}

		setCommittedStateForEachSection(drTreatmentPlan)

		if err := populateContentSourceIntoTreatmentPlan(drTreatmentPlan, dataApi, doctorId); err == api.NoRowsError {
			return errors.New("No treatment plan found")
		} else if err != nil {
			return err
		}
	}
	return err
}

func setCommittedStateForEachSection(drTreatmentPlan *common.DoctorTreatmentPlan) {
	// depending on which sections have data in them, mark them to be committed or uncommitted
	// note that we intentionally treat a section with no data to be in the UNCOMMITTED state so as
	// to ensure that the doctor actually wanted to leave a particular section blank

	if len(drTreatmentPlan.TreatmentList.Treatments) > 0 {
		drTreatmentPlan.TreatmentList.Status = api.STATUS_COMMITTED
	} else {
		drTreatmentPlan.TreatmentList.Status = api.STATUS_UNCOMMITTED
	}

	if len(drTreatmentPlan.RegimenPlan.RegimenSections) > 0 {
		drTreatmentPlan.RegimenPlan.Status = api.STATUS_COMMITTED
	} else {
		drTreatmentPlan.RegimenPlan.Status = api.STATUS_UNCOMMITTED
	}

	if len(drTreatmentPlan.Advice.SelectedAdvicePoints) > 0 {
		drTreatmentPlan.Advice.Status = api.STATUS_COMMITTED
	} else {
		drTreatmentPlan.Advice.Status = api.STATUS_UNCOMMITTED
	}

}

func populateContentSourceIntoTreatmentPlan(treatmentPlan *common.DoctorTreatmentPlan, dataApi api.DataAPI, doctorId int64) error {
	// only continue if the content source of the treaetment plan is a favorite treatment plan
	if treatmentPlan.ContentSource == nil {
		return nil
	}

	switch treatmentPlan.ContentSource.ContentSourceType {
	case common.TPContentSourceTypeTreatmentPlan:
		previousTreatmentPlan, err := dataApi.GetTreatmentPlan(treatmentPlan.ContentSource.ContentSourceId.Int64(), doctorId)
		if err != nil {
			return err
		}

		if len(treatmentPlan.TreatmentList.Treatments) == 0 {
			fillTreatmentsIntoTreatmentPlan(previousTreatmentPlan.TreatmentList.Treatments, treatmentPlan)
		}

		if len(treatmentPlan.RegimenPlan.RegimenSections) == 0 {
			fillRegimenSectionsIntoTreatmentPlan(previousTreatmentPlan.RegimenPlan.RegimenSections, treatmentPlan)
		}

		if len(treatmentPlan.Advice.SelectedAdvicePoints) == 0 {
			fillAdvicePointsIntoTreatmentPlan(previousTreatmentPlan.Advice.SelectedAdvicePoints, treatmentPlan)
		}

	case common.TPContentSourceTypeFTP:
		favoriteTreatmentPlanId := treatmentPlan.ContentSource.ContentSourceId.Int64()

		favoriteTreatmentPlan, err := dataApi.GetFavoriteTreatmentPlan(favoriteTreatmentPlanId)
		if err != nil {
			return err
		}

		// The assumption here is that all components of a treatment plan that are already populated
		// match the items in the favorite treatment plan, if there exists a mapping to indicate that this
		// treatment plan must be filled in from a favorite treatment plan. The reason that we don't just write over
		// the items that do already belong in the treatment plan is to maintain the ids of the items that have been committed
		// to the database as part of the treatment plan.

		// populate treatments
		if len(treatmentPlan.TreatmentList.Treatments) == 0 {
			fillTreatmentsIntoTreatmentPlan(favoriteTreatmentPlan.TreatmentList.Treatments, treatmentPlan)
		}

		// populate regimen plan
		if len(treatmentPlan.RegimenPlan.RegimenSections) == 0 {
			fillRegimenSectionsIntoTreatmentPlan(favoriteTreatmentPlan.RegimenPlan.RegimenSections, treatmentPlan)
		}

		// populate advice
		if len(treatmentPlan.Advice.SelectedAdvicePoints) == 0 {
			fillAdvicePointsIntoTreatmentPlan(favoriteTreatmentPlan.Advice.SelectedAdvicePoints, treatmentPlan)
		}
	}

	return nil

}

func fillAdvicePointsIntoTreatmentPlan(sourceAdvicePoints []*common.DoctorInstructionItem, treatmentPlan *common.DoctorTreatmentPlan) {
	treatmentPlan.Advice.SelectedAdvicePoints = make([]*common.DoctorInstructionItem, len(sourceAdvicePoints))
	for i, advicePoint := range sourceAdvicePoints {
		treatmentPlan.Advice.SelectedAdvicePoints[i] = &common.DoctorInstructionItem{
			ParentId: advicePoint.ParentId,
			Text:     advicePoint.Text,
		}
	}
}

func fillRegimenSectionsIntoTreatmentPlan(sourceRegimenSections []*common.RegimenSection, treatmentPlan *common.DoctorTreatmentPlan) {
	treatmentPlan.RegimenPlan.RegimenSections = make([]*common.RegimenSection, len(sourceRegimenSections))

	for i, regimenSection := range sourceRegimenSections {
		treatmentPlan.RegimenPlan.RegimenSections[i] = &common.RegimenSection{
			RegimenName:  regimenSection.RegimenName,
			RegimenSteps: make([]*common.DoctorInstructionItem, len(regimenSection.RegimenSteps)),
		}

		for j, regimenStep := range regimenSection.RegimenSteps {
			treatmentPlan.RegimenPlan.RegimenSections[i].RegimenSteps[j] = &common.DoctorInstructionItem{
				ParentId: regimenStep.ParentId,
				Text:     regimenStep.Text,
			}
		}
	}
}

func fillTreatmentsIntoTreatmentPlan(sourceTreatments []*common.Treatment, treatmentPlan *common.DoctorTreatmentPlan) {
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

func sendCaseMessageAndPublishTPActivatedEvent(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, treatmentPlan *common.DoctorTreatmentPlan,
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
