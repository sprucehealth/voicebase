package doctor_treatment_plan

import (
	"fmt"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/messages"
)

type TreatmentPlan struct {
	ID                encoding.ObjectID                  `json:"id,omitempty"`
	DoctorID          encoding.ObjectID                  `json:"doctor_id,omitempty"`
	PatientCaseID     encoding.ObjectID                  `json:"case_id"`
	PatientID         int64                              `json:"patient_id,omitempty,string"`
	Status            common.TreatmentPlanStatus         `json:"status,omitempty"`
	CreationDate      time.Time                          `json:"creation_date"`
	SentDate          *time.Time                         `json:"sent_date,omitempty"`
	TreatmentList     *common.TreatmentList              `json:"treatment_list"`
	RegimenPlan       *common.RegimenPlan                `json:"regimen_plan,omitempty"`
	Parent            *common.TreatmentPlanParent        `json:"parent,omitempty"`
	ContentSource     *common.TreatmentPlanContentSource `json:"content_source,omitempty"`
	Note              string                             `json:"note,omitempty"`
	ScheduledMessages []*ScheduledMessage                `json:"scheduled_messages"`
	ResourceGuides    []*ResourceGuide                   `json:"resource_guides,omitempty"`
}

type ResourceGuide struct {
	ID        int64  `json:"id,string"`
	SectionID int64  `json:"section_id,string"`
	Title     string `json:"title"`
	PhotoURL  string `json:"photo_url"`
}

type ScheduledMessage struct {
	ID            int64                  `json:"id,string"`
	Title         *string                `json:"title"`
	ScheduledDays int                    `json:"scheduled_days"`
	ScheduledFor  *time.Time             `json:"scheduled_for"`
	Message       string                 `json:"message"`
	Attachments   []*messages.Attachment `json:"attachments"`
}

type FavoriteTreatmentPlan struct {
	ID                encoding.ObjectID     `json:"id"`
	PathwayTag        string                `json:"pathway_id,string"`
	Name              string                `json:"name"`
	ModifiedDate      time.Time             `json:"modified_date,omitempty"`
	DoctorID          int64                 `json:"-"`
	RegimenPlan       *common.RegimenPlan   `json:"regimen_plan,omitempty"`
	TreatmentList     *common.TreatmentList `json:"treatment_list,omitempty"`
	Note              string                `json:"note"`
	ScheduledMessages []*ScheduledMessage   `json:"scheduled_messages"`
	ResourceGuides    []*ResourceGuide      `json:"resource_guides,omitempty"`
}

func (tp *TreatmentPlan) IsActive() bool {
	switch tp.Status {
	case common.TPStatusActive, common.TPStatusSubmitted, common.TPStatusRXStarted:
		return true
	}
	return false
}

func (f *FavoriteTreatmentPlan) EqualsTreatmentPlan(tp *TreatmentPlan) bool {
	if f == nil || tp == nil {
		return false
	}

	if !f.TreatmentList.Equals(tp.TreatmentList) {
		return false
	}

	if !f.RegimenPlan.Equals(tp.RegimenPlan) {
		return false
	}

	if f.Note != tp.Note {
		return false
	}

	if len(f.ScheduledMessages) != len(tp.ScheduledMessages) {
		return false
	}

	for _, sm1 := range f.ScheduledMessages {
		matched := false
		for _, sm2 := range tp.ScheduledMessages {
			if sm1.Equal(sm2) {
				matched = true
			}
		}
		if !matched {
			return false
		}
	}

	if len(f.ResourceGuides) != len(tp.ResourceGuides) {
		return false
	}

	for _, g1 := range f.ResourceGuides {
		found := false
		for _, g2 := range tp.ResourceGuides {
			if g1.ID == g2.ID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (m *ScheduledMessage) Equal(to *ScheduledMessage) bool {
	if m.Message != to.Message {
		return false
	}
	if m.ScheduledDays != to.ScheduledDays {
		return false
	}
	if len(m.Attachments) != len(to.Attachments) {
		return false
	}

	for _, a1 := range m.Attachments {
		matched := false
		for _, a2 := range to.Attachments {
			if a1.Type == a2.Type && a1.ID == a2.ID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

func TransformTPToResponse(dataAPI api.DataAPI, mediaStore storage.Store, tp *common.TreatmentPlan) (*TreatmentPlan, error) {
	if tp == nil {
		return nil, nil
	}
	tpRes := &TreatmentPlan{
		ID:                tp.ID,
		DoctorID:          tp.DoctorID,
		PatientCaseID:     tp.PatientCaseID,
		PatientID:         tp.PatientID,
		Status:            tp.Status,
		CreationDate:      tp.CreationDate,
		SentDate:          tp.SentDate,
		TreatmentList:     tp.TreatmentList,
		RegimenPlan:       tp.RegimenPlan,
		Parent:            tp.Parent,
		ContentSource:     tp.ContentSource,
		Note:              tp.Note,
		ScheduledMessages: make([]*ScheduledMessage, len(tp.ScheduledMessages)),
		ResourceGuides:    make([]*ResourceGuide, len(tp.ResourceGuides)),
	}

	var sentTime time.Time
	if tp.SentDate != nil {
		sentTime = *tp.SentDate
	} else {
		sentTime = time.Now().UTC()
	}
	var err error
	for i, sm := range tp.ScheduledMessages {
		tpRes.ScheduledMessages[i], err = transformScheduledMessageToResponse(dataAPI, mediaStore, sm, sentTime)
		if err != nil {
			return nil, err
		}
	}

	for i, g := range tp.ResourceGuides {
		tpRes.ResourceGuides[i] = transformResourceGuideToResponse(g)
	}

	return tpRes, nil
}

func TransformTPFromResponse(dataAPI api.DataAPI, tp *TreatmentPlan, doctorID int64, role string) (*common.TreatmentPlan, error) {
	if tp == nil {
		return nil, nil
	}
	tp2 := &common.TreatmentPlan{
		ID:                tp.ID,
		DoctorID:          tp.DoctorID,
		PatientCaseID:     tp.PatientCaseID,
		PatientID:         tp.PatientID,
		Status:            tp.Status,
		CreationDate:      tp.CreationDate,
		SentDate:          tp.SentDate,
		TreatmentList:     tp.TreatmentList,
		RegimenPlan:       tp.RegimenPlan,
		Parent:            tp.Parent,
		ContentSource:     tp.ContentSource,
		Note:              tp.Note,
		ScheduledMessages: make([]*common.TreatmentPlanScheduledMessage, len(tp.ScheduledMessages)),
		ResourceGuides:    make([]*common.ResourceGuide, len(tp.ResourceGuides)),
	}

	var err error
	for i, sm := range tp.ScheduledMessages {
		tp2.ScheduledMessages[i], err = transformScheduledMessageFromResponse(dataAPI, sm, tp2.ID.Int64(), doctorID, role)
		if err != nil {
			return nil, err
		}
	}

	for i, g := range tp.ResourceGuides {
		tp2.ResourceGuides[i] = transformResourceGuideFromResponse(g)
	}

	return tp2, nil
}

func TransformFTPToResponse(dataAPI api.DataAPI, mediaStore storage.Store, ftp *common.FavoriteTreatmentPlan) (*FavoriteTreatmentPlan, error) {
	if ftp == nil {
		return nil, nil
	}
	ftpRes := &FavoriteTreatmentPlan{
		ID:                ftp.ID,
		PathwayTag:        ftp.PathwayTag,
		Name:              ftp.Name,
		ModifiedDate:      ftp.ModifiedDate,
		DoctorID:          ftp.DoctorID,
		RegimenPlan:       ftp.RegimenPlan,
		TreatmentList:     ftp.TreatmentList,
		Note:              ftp.Note,
		ScheduledMessages: make([]*ScheduledMessage, len(ftp.ScheduledMessages)),
		ResourceGuides:    make([]*ResourceGuide, len(ftp.ResourceGuides)),
	}

	now := time.Now().UTC()
	var err error
	for i, sm := range ftp.ScheduledMessages {
		ftpRes.ScheduledMessages[i], err = transformScheduledMessageToResponse(dataAPI, mediaStore, sm, now)
		if err != nil {
			return nil, err
		}
	}

	for i, g := range ftp.ResourceGuides {
		ftpRes.ResourceGuides[i] = transformResourceGuideToResponse(g)
	}

	return ftpRes, nil
}

func TransformFTPFromResponse(dataAPI api.DataAPI, ftp *FavoriteTreatmentPlan, doctorID int64, role string) (*common.FavoriteTreatmentPlan, error) {
	if ftp == nil {
		return nil, nil
	}
	ftp2 := &common.FavoriteTreatmentPlan{
		ID:                ftp.ID,
		PathwayTag:        ftp.PathwayTag,
		Name:              ftp.Name,
		ModifiedDate:      ftp.ModifiedDate,
		DoctorID:          ftp.DoctorID,
		RegimenPlan:       ftp.RegimenPlan,
		TreatmentList:     ftp.TreatmentList,
		Note:              ftp.Note,
		ScheduledMessages: make([]*common.TreatmentPlanScheduledMessage, len(ftp.ScheduledMessages)),
		ResourceGuides:    make([]*common.ResourceGuide, len(ftp.ResourceGuides)),
	}

	// TODO: for now assume Acne
	if ftp2.PathwayTag == "" {
		ftp2.PathwayTag = api.AcnePathwayTag
	}

	var err error
	for i, sm := range ftp.ScheduledMessages {
		ftp2.ScheduledMessages[i], err = transformScheduledMessageFromResponse(dataAPI, sm, ftp2.ID.Int64(), doctorID, role)
		if err != nil {
			return nil, err
		}
	}

	for i, g := range ftp.ResourceGuides {
		ftp2.ResourceGuides[i] = transformResourceGuideFromResponse(g)
	}

	return ftp2, nil
}
func transformResourceGuideToResponse(g *common.ResourceGuide) *ResourceGuide {
	return &ResourceGuide{
		ID:        g.ID,
		SectionID: g.SectionID,
		Title:     g.Title,
		PhotoURL:  g.PhotoURL,
	}
}

func transformResourceGuideFromResponse(g *ResourceGuide) *common.ResourceGuide {
	return &common.ResourceGuide{
		ID:        g.ID,
		SectionID: g.SectionID,
		Title:     g.Title,
		PhotoURL:  g.PhotoURL,
	}
}

func transformScheduledMessageFromResponse(dataAPI api.DataAPI, msg *ScheduledMessage, tpID, doctorID int64, role string) (*common.TreatmentPlanScheduledMessage, error) {
	m := &common.TreatmentPlanScheduledMessage{
		TreatmentPlanID: tpID,
		ScheduledDays:   msg.ScheduledDays,
		Attachments:     make([]*common.CaseMessageAttachment, len(msg.Attachments)),
		Message:         msg.Message,
	}
	var personID int64
	for i, a := range msg.Attachments {
		att := &common.CaseMessageAttachment{
			ItemID:   a.ID,
			ItemType: a.Type,
			MimeType: a.MimeType,
			Title:    a.Title,
		}
		if idx := strings.IndexByte(att.ItemType, ':'); idx >= 0 {
			att.ItemType = att.ItemType[idx+1:]
		}
		m.Attachments[i] = att

		switch att.ItemType {
		case common.AttachmentTypePhoto, common.AttachmentTypeAudio:
			// Delayed querying of person ID (only needed when checking media)
			if personID == 0 {
				var err error
				personID, err = dataAPI.GetPersonIDByRole(role, doctorID)
				if err != nil {
					return nil, err
				}
			}

			// Make sure media is uploaded by the same person and is unclaimed
			media, err := dataAPI.GetMedia(a.ID)
			if err != nil {
				return nil, err
			}
			if media.UploaderID != personID {
				return nil, apiservice.NewValidationError("invalid attached media")
			}
		case common.AttachmentTypeFollowupVisit:
		default:
			return nil, apiservice.NewValidationError("attachment type " + att.ItemType + " not allowed in scheduled message")
		}

		if att.Title == "" {
			att.Title = messages.AttachmentTitle(att.ItemType)
		}
	}
	return m, nil
}

func transformScheduledMessageToResponse(dataAPI api.DataAPI, mediaStore storage.Store, m *common.TreatmentPlanScheduledMessage, sentTime time.Time) (*ScheduledMessage, error) {
	scheduledFor := sentTime.Add(24 * time.Hour * time.Duration(m.ScheduledDays))
	msg := &ScheduledMessage{
		ID:            m.ID,
		ScheduledDays: m.ScheduledDays,
		ScheduledFor:  &scheduledFor,
		Message:       m.Message,
		Attachments:   make([]*messages.Attachment, len(m.Attachments)),
	}
	for j, a := range m.Attachments {
		att := &messages.Attachment{
			ID:       a.ItemID,
			Type:     messages.AttachmentTypePrefix + a.ItemType,
			Title:    a.Title,
			MimeType: a.MimeType,
		}
		msg.Attachments[j] = att

		switch a.ItemType {
		case common.AttachmentTypePhoto, common.AttachmentTypeAudio:
			media, err := dataAPI.GetMedia(a.ItemID)
			if err != nil {
				return nil, err
			}

			att.URL, err = mediaStore.GetSignedURL(media.URL, time.Now().Add(scheduledMessageMediaExpirationDuration))
			if err != nil {
				return nil, err
			}
		}
	}
	title := titleForScheduledMessage(msg)
	msg.Title = &title
	return msg, nil
}

func titleForScheduledMessage(m *ScheduledMessage) string {
	isFollowUp := false
	for _, a := range m.Attachments {
		if a.Type == messages.AttachmentTypePrefix+common.AttachmentTypeFollowupVisit {
			isFollowUp = true
			break
		}
	}

	var humanTime string
	days := m.ScheduledFor.Sub(time.Now()) / (time.Hour * 24)
	if days <= 1 {
		humanTime = "1 day"
	} else if days < 7 {
		humanTime = fmt.Sprintf("%d days", days)
	} else if days < 14 {
		humanTime = "1 week"
	} else {
		humanTime = fmt.Sprintf("%d weeks", days/7)
	}

	if isFollowUp {
		return "Message & Follow-Up Visit in " + humanTime
	}
	return "Message in " + humanTime
}
