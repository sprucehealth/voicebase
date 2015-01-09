package doctor_treatment_plan

import (
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

// populateTreatmentPlan populates the appropriate treatmentplan section
func populateTreatmentPlan(tp *common.TreatmentPlan, doctorID int64, dataAPI api.DataAPI, sections Sections) error {
	var err error

	if sections&TreatmentsSection != 0 {
		tp.TreatmentList = &common.TreatmentList{}
		tp.TreatmentList.Treatments, err = dataAPI.GetTreatmentsBasedOnTreatmentPlanID(tp.ID.Int64())
		if err != nil {
			return fmt.Errorf("Unable to get treatments for treatment plan: %s", err)
		}
		if err := indicateExistenceOfRXGuidesForTreatments(dataAPI, tp.TreatmentList); err != nil {
			golog.Errorf(err.Error())
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
	}

	return err
}

func setCommittedStateForEachSection(tp *common.TreatmentPlan) {
	// FIXME: The committed/uncommitted status has only been left in here for backwards compatability.
	// We will need this until the client stops relying on the status for the treatments and instructions
	// sections. Default to UNCOMMITTED so that the client is inclined to resubmit the sections.
	if tp.TreatmentList != nil {
		tp.TreatmentList.Status = api.STATUS_UNCOMMITTED
	}

	if tp.RegimenPlan != nil {
		tp.RegimenPlan.Status = api.STATUS_UNCOMMITTED
	}
}

// copyContentSourceIntoTreatmentPlan copies the content of the content source (based on type) into the
// treatment plan
func copyContentSourceIntoTreatmentPlan(tp *common.TreatmentPlan, dataAPI api.DataAPI, doctorID int64) error {
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
		copyTreatmentsIntoTreatmentPlan(prevTP.TreatmentList.Treatments, tp)
		copyRegimenSectionsIntoTreatmentPlan(prevTP.RegimenPlan.Sections, tp)
		tp.Note = VersionedTreatmentPlanNote
		tp.ScheduledMessages = copyScheduledMessages(tp.ID.Int64(), prevTP.ScheduledMessages)
		tp.ResourceGuides = prevTP.ResourceGuides

	case common.TPContentSourceTypeFTP:
		ftp, err := dataAPI.GetFavoriteTreatmentPlan(tp.ContentSource.ID.Int64())
		if err != nil {
			return err
		}
		copyTreatmentsIntoTreatmentPlan(ftp.TreatmentList.Treatments, tp)
		copyRegimenSectionsIntoTreatmentPlan(ftp.RegimenPlan.Sections, tp)
		tp.Note = ftp.Note
		tp.ScheduledMessages = copyScheduledMessages(tp.ID.Int64(), ftp.ScheduledMessages)
		tp.ResourceGuides = ftp.ResourceGuides
	}
	return nil
}

func copyRegimenSectionsIntoTreatmentPlan(sourceRegimenSections []*common.RegimenSection, treatmentPlan *common.TreatmentPlan) {
	treatmentPlan.RegimenPlan = &common.RegimenPlan{
		Sections: make([]*common.RegimenSection, len(sourceRegimenSections)),
	}
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

func copyTreatmentsIntoTreatmentPlan(sourceTreatments []*common.Treatment, treatmentPlan *common.TreatmentPlan) {
	treatmentPlan.TreatmentList = &common.TreatmentList{
		Treatments: make([]*common.Treatment, len(sourceTreatments)),
	}
	for i, treatment := range sourceTreatments {
		treatmentPlan.TreatmentList.Treatments[i] = &common.Treatment{
			DrugDBIDs:               treatment.DrugDBIDs,
			DrugInternalName:        treatment.DrugInternalName,
			DrugName:                treatment.DrugName,
			DrugForm:                treatment.DrugForm,
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
		sm[i] = msg
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
