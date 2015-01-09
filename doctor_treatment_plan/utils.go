package doctor_treatment_plan

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	question_acne_diagnosis = "q_acne_diagnosis"
	question_acne_severity  = "q_acne_severity"
	question_acne_type      = "q_acne_type"
	question_rosacea_type   = "q_acne_rosacea_type"
)

const VersionedTreatmentPlanNote = `Here is your revised treatment plan.

P.S. Please remember to consult the attached 'Prescription Guide' for additional information regarding the medication I've prescribed for you, including usage tips, warnings, and common side effects.`

func fillInTreatmentPlan(tp *common.TreatmentPlan, doctorID int64, dataAPI api.DataAPI, sections Sections) error {
	var err error

	if sections&TreatmentsSection != 0 {
		tp.TreatmentList = &common.TreatmentList{}
		tp.TreatmentList.Treatments, err = dataAPI.GetTreatmentsBasedOnTreatmentPlanID(tp.ID.Int64())
		if err != nil {
			return fmt.Errorf("Unable to get treatments for treatment plan: %s", err)
		}
	}

	if sections&RegimenSection != 0 {
		tp.RegimenPlan, err = dataAPI.GetRegimenPlanForTreatmentPlan(tp.ID.Int64())
		if err != nil {
			return fmt.Errorf("Unable to get regimen plan for treatment plan: %s", err)
		}
	}

	if sections&NoteSection != 0 {
		tp.Note, err = dataAPI.GetTreatmentPlanNote(tp.ID.Int64())
		if err != nil && err != api.NoRowsError {
			return fmt.Errorf("Unable to get note for treatment plan: %s", err)
		}
	}

	if sections&ScheduledMessagesSection != 0 {
		tp.ScheduledMessages, err = dataAPI.ListTreatmentPlanScheduledMessages(tp.ID.Int64())
		if err != nil {
			return fmt.Errorf("Unable to get scheduled messages for treatment plan: %s", err.Error())
		}
	}

	if sections&ResourceGuidesSection != 0 {
		tp.ResourceGuides, err = dataAPI.ListTreatmentPlanResourceGuides(tp.ID.Int64())
		if err != nil {
			return fmt.Errorf("Unable to get resource guides for treatment plan: %s", err.Error())
		}
	}

	// only populate the draft state if we are dealing with a draft treatment plan and the same doctor
	// that owns it is requesting the treatment plan (so that they can edit it)
	if tp.DoctorID.Int64() == doctorID && tp.InDraftMode() {
		if sections&RegimenSection != 0 {
			tp.RegimenPlan.AllSteps, err = dataAPI.GetRegimenStepsForDoctor(tp.DoctorID.Int64())
			if err != nil {
				return err
			}
		}

		setCommittedStateForEachSection(tp)

		if err := populateContentSourceIntoTreatmentPlan(tp, dataAPI, doctorID, sections); err == api.NoRowsError {
			return errors.New("No treatment plan found")
		} else if err != nil {
			return err
		}

		if sections&TreatmentsSection != 0 {
			if err := indicateExistenceOfRXGuidesForTreatments(dataAPI, tp.TreatmentList); err != nil {
				golog.Errorf(err.Error())
			}
		}
	}
	return err
}

func setCommittedStateForEachSection(tp *common.TreatmentPlan) {
	// depending on which sections have data in them, mark them to be committed or uncommitted
	// note that we intentionally treat a section with no data to be in the UNCOMMITTED state so as
	// to ensure that the doctor actually wanted to leave a particular section blank

	if tp.TreatmentList != nil {
		if len(tp.TreatmentList.Treatments) > 0 {
			tp.TreatmentList.Status = api.STATUS_COMMITTED
		} else {
			tp.TreatmentList.Status = api.STATUS_UNCOMMITTED
		}
	}

	if tp.RegimenPlan != nil {
		if len(tp.RegimenPlan.Sections) > 0 {
			tp.RegimenPlan.Status = api.STATUS_COMMITTED
		} else {
			tp.RegimenPlan.Status = api.STATUS_UNCOMMITTED
		}
	}
}

func populateContentSourceIntoTreatmentPlan(tp *common.TreatmentPlan, dataAPI api.DataAPI, doctorID int64, sections Sections) error {
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

		if sections&TreatmentsSection != 0 && len(tp.TreatmentList.Treatments) == 0 {
			fillTreatmentsIntoTreatmentPlan(prevTP.TreatmentList.Treatments, tp)
		}

		if sections&RegimenSection != 0 && len(tp.RegimenPlan.Sections) == 0 {
			fillRegimenSectionsIntoTreatmentPlan(prevTP.RegimenPlan.Sections, tp)
		}

		if sections&NoteSection != 0 && tp.Note == "" {
			tp.Note = VersionedTreatmentPlanNote
		}

		if sections&ScheduledMessagesSection != 0 && len(tp.ScheduledMessages) == 0 {
			msgs, err := dataAPI.ListTreatmentPlanScheduledMessages(tp.ID.Int64())
			if err != nil {
				return err
			}
			tp.ScheduledMessages = copyScheduledMessages(tp.ID.Int64(), msgs)
		}

		if sections&ResourceGuidesSection != 0 && len(tp.ResourceGuides) == 0 {
			tp.ResourceGuides, err = dataAPI.ListTreatmentPlanResourceGuides(prevTP.ID.Int64())
			if err != nil {
				return err
			}
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
		if sections&TreatmentsSection != 0 && len(tp.TreatmentList.Treatments) == 0 {
			fillTreatmentsIntoTreatmentPlan(ftp.TreatmentList.Treatments, tp)
		}

		// populate regimen plan
		if sections&RegimenSection != 0 && len(tp.RegimenPlan.Sections) == 0 {
			fillRegimenSectionsIntoTreatmentPlan(ftp.RegimenPlan.Sections, tp)
		}

		if sections&NoteSection != 0 && tp.Note == "" {
			tp.Note = ftp.Note
		}

		if sections&ScheduledMessagesSection != 0 && len(tp.ScheduledMessages) == 0 {
			tp.ScheduledMessages = copyScheduledMessages(tp.ID.Int64(), ftp.ScheduledMessages)
		}

		if sections&ResourceGuidesSection != 0 && len(tp.ResourceGuides) == 0 {
			tp.ResourceGuides = ftp.ResourceGuides
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
			DrugDBIDs:               treatment.DrugDBIDs,
			DrugInternalName:        treatment.DrugInternalName,
			DrugName:                treatment.DrugName,
			DrugRoute:               treatment.DrugRoute,
			DosageStrength:          treatment.DosageStrength,
			DispenseValue:           treatment.DispenseValue,
			DispenseUnitID:          treatment.DispenseUnitID,
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

func indicateExistenceOfRXGuidesForTreatments(dataAPI api.DataAPI, treatmentList *common.TreatmentList) error {
	if treatmentList == nil || len(treatmentList.Treatments) == 0 {
		return nil
	}

	drugQueries := make([]*api.DrugDetailsQuery, len(treatmentList.Treatments))
	for i, t := range treatmentList.Treatments {
		drugQueries[i] = &api.DrugDetailsQuery{
			NDC:         t.DrugDBIDs[erx.NDC],
			GenericName: t.GenericDrugName,
			Route:       t.DrugRoute,
			Form:        t.DrugForm,
		}
	}
	drugDetails, err := dataAPI.MultiQueryDrugDetailIDs(drugQueries)
	if err != nil {
		return err
	}

	for i, t := range treatmentList.Treatments {
		t.HasRxGuide = drugDetails[i] != 0
	}

	return nil
}

func sendCaseMessageAndPublishTPActivatedEvent(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, treatmentPlan *common.TreatmentPlan,
	doctor *common.Doctor, message string) error {
	// only send a case message if one has not already been sent for this particular
	// treatment plan for this particular case
	caseMessage, err := dataAPI.CaseMessageForAttachment(common.AttachmentTypeTreatmentPlan,
		treatmentPlan.ID.Int64(), doctor.PersonID, treatmentPlan.PatientCaseID.Int64())
	if err != api.NoRowsError && err != nil {
		return err
	} else if err == api.NoRowsError {
		caseMessage = &common.CaseMessage{
			CaseID:   treatmentPlan.PatientCaseID.Int64(),
			PersonID: doctor.PersonID,
			Body:     message,
			Attachments: []*common.CaseMessageAttachment{
				&common.CaseMessageAttachment{
					ItemType: common.AttachmentTypeTreatmentPlan,
					ItemID:   treatmentPlan.ID.Int64(),
				},
			},
		}
		if _, err := dataAPI.CreateCaseMessage(caseMessage); err != nil {
			return err
		}
	}

	patientVisitID, err := dataAPI.GetPatientVisitIDFromTreatmentPlanID(treatmentPlan.ID.Int64())
	if err != nil {
		return err
	}

	// Publish event that treamtent plan was created
	dispatcher.Publish(&TreatmentPlanActivatedEvent{
		PatientID:     treatmentPlan.PatientID,
		DoctorID:      doctor.DoctorID.Int64(),
		VisitID:       patientVisitID,
		TreatmentPlan: treatmentPlan,
		Message:       caseMessage,
	})

	return nil
}

func copyScheduledMessages(tpID int64, msgs []*common.TreatmentPlanScheduledMessage) []*common.TreatmentPlanScheduledMessage {
	sm := make([]*common.TreatmentPlanScheduledMessage, len(msgs))
	for i, m := range msgs {
		msg := &common.TreatmentPlanScheduledMessage{
			Message:         m.Message,
			ScheduledDays:   m.ScheduledDays,
			TreatmentPlanID: tpID,
			Attachments:     make([]*common.CaseMessageAttachment, len(m.Attachments)),
		}
		for j, a := range m.Attachments {
			msg.Attachments[j] = &common.CaseMessageAttachment{
				ItemID:   a.ItemID,
				ItemType: a.ItemType,
				MimeType: a.MimeType,
				Title:    a.Title,
			}
		}
		sm[i] = m
	}
	return sm
}

// Sections is a bitmap representing a set of treatment plan sections
type Sections int

const (
	TreatmentsSection Sections = 1 << iota
	RegimenSection
	NoteSection
	ScheduledMessagesSection
	ResourceGuidesSection
	AllSections  Sections = (1 << iota) - 1
	NoSections   Sections = 0
	sectionCount          = iota
)

var sectionNames = map[string]Sections{
	"note":               NoteSection,
	"regimen":            RegimenSection,
	"treatments":         TreatmentsSection,
	"scheduled_messages": ScheduledMessagesSection,
	"resource_guides":    ResourceGuidesSection,
}

func (s Sections) String() string {
	if s == 0 {
		// Use an explicit 'none' token instead of an empty string to differentiate
		// between unspecified vs empty set
		return "none"
	}
	if s&AllSections == AllSections {
		return "all"
	}
	names := make([]string, 0, sectionCount)
	for n, b := range sectionNames {
		if s&b != 0 {
			names = append(names, n)
		}
	}
	return strings.Join(names, ",")
}

func parseSections(sec string) Sections {
	if len(sec) == 0 {
		return AllSections
	}
	var sections Sections
	sec = strings.ToLower(sec)
	for len(sec) != 0 {
		i := strings.IndexByte(sec, ',')
		name := sec
		if i >= 0 {
			name = sec[:i]
			sec = sec[i+1:]
		} else {
			sec = sec[:0]
		}
		if name == "all" {
			sections = AllSections
			break
		}
		sections |= sectionNames[name]
	}
	return sections
}
