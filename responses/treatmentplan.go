package responses

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/messages"
)

type TreatmentPlan struct {
	ID                     encoding.ObjectID           `json:"id,omitempty"`
	DoctorID               encoding.ObjectID           `json:"doctor_id,omitempty"`
	PatientCaseID          encoding.ObjectID           `json:"case_id"`
	PatientID              int64                       `json:"patient_id,string,omitempty"`
	Status                 common.TreatmentPlanStatus  `json:"status,omitempty"`
	CreationEpoch          int64                       `json:"creation_epoch,string"`
	DeprecatedCreationDate time.Time                   `json:"creation_date"`
	SentDate               *time.Time                  `json:"sent_date,omitempty"`
	TreatmentList          *common.TreatmentList       `json:"treatment_list"`
	RegimenPlan            *common.RegimenPlan         `json:"regimen_plan,omitempty"`
	Parent                 *TreatmentPlanParent        `json:"parent,omitempty"`
	ContentSource          *TreatmentPlanContentSource `json:"content_source,omitempty"`
	Note                   string                      `json:"note,omitempty"`
	ScheduledMessages      []*ScheduledMessage         `json:"scheduled_messages"`
	ResourceGuides         []*ResourceGuide            `json:"resource_guides,omitempty"`
}

func NewTreatmentPlan(tp *common.TreatmentPlan) *TreatmentPlan {
	return &TreatmentPlan{
		ID:                     tp.ID,
		PatientCaseID:          tp.PatientCaseID,
		Status:                 tp.Status,
		DoctorID:               tp.DoctorID,
		DeprecatedCreationDate: tp.CreationDate,
		CreationEpoch:          tp.CreationDate.Unix(),
	}
}

type TreatmentPlanParent struct {
	ID                     int64     `json:"parent_id,string"`
	Type                   string    `json:"parent_type"`
	DeprecatedCreationDate time.Time `json:"creation_date"`
	CreationEpoch          int64     `json:"creation_epoch"`
}

func NewTreatmentPlanParent(tpp *common.TreatmentPlanParent) *TreatmentPlanParent {
	return &TreatmentPlanParent{
		ID:   tpp.ParentID.Int64(),
		Type: tpp.ParentType,
		DeprecatedCreationDate: tpp.CreationDate,
		CreationEpoch:          tpp.CreationDate.Unix(),
	}
}

type TreatmentPlanContentSource struct {
	ID       int64  `json:"content_source_id,string"`
	Type     string `json:"content_source_type"`
	Deviated bool   `json:"has_deviated"`
}

type ResourceGuide struct {
	ID        int64  `json:"id,string"`
	SectionID int64  `json:"section_id,string"`
	Title     string `json:"title"`
	PhotoURL  string `json:"photo_url"`
}

type ScheduledMessage struct {
	ID                     int64                  `json:"id,string"`
	Title                  *string                `json:"title"`
	ScheduledDays          int                    `json:"scheduled_days"`
	DeprecatedScheduledFor *time.Time             `json:"scheduled_for"`
	ScheduledForEpoch      int64                  `json:"scheduled_for_epoch"`
	Message                string                 `json:"message"`
	Attachments            []*messages.Attachment `json:"attachments"`
}

type FavoriteTreatmentPlan struct {
	ID                     encoding.ObjectID     `json:"id"`
	PathwayTag             string                `json:"pathway_id"`
	Name                   string                `json:"name"`
	DeprecatedModifiedDate time.Time             `json:"modified_date,omitempty"`
	ModifiedEpoch          int64                 `json:"modified_epoch,omitempty"`
	DoctorID               int64                 `json:"-"`
	RegimenPlan            *common.RegimenPlan   `json:"regimen_plan,omitempty"`
	TreatmentList          *common.TreatmentList `json:"treatment_list,omitempty"`
	Note                   string                `json:"note"`
	ScheduledMessages      []*ScheduledMessage   `json:"scheduled_messages"`
	ResourceGuides         []*ResourceGuide      `json:"resource_guides,omitempty"`
}

type FavoriteTreatmentPlanMembership struct {
	ID                      int64  `json:"id,string"`
	FavoriteTreatmentPlanID int64  `json:"favorite_treatment_plan_id,string"`
	DoctorID                int64  `json:"doctor_id,string"`
	FirstName               string `json:"first_name"`
	LastName                string `json:"last_name"`
	PathwayID               int64  `json:"pathway_id,string"`
	PathwayName             string `json:"pathway_name"`
	PathwayTag              string `json:"pathway_tag"`
}

type mediaLookup interface {
	GetPersonIDByRole(role string, doctorID int64) (int64, error)
	GetMedia(id int64) (*common.Media, error)
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

func TransformTPToResponse(
	mLookup mediaLookup,
	mediaStore *media.Store,
	mediaExpirationDuration time.Duration,
	tp *common.TreatmentPlan,
) (*TreatmentPlan, error) {

	if tp == nil {
		return nil, nil
	}
	tpRes := &TreatmentPlan{
		ID:                     tp.ID,
		DoctorID:               tp.DoctorID,
		PatientCaseID:          tp.PatientCaseID,
		PatientID:              tp.PatientID,
		Status:                 tp.Status,
		DeprecatedCreationDate: tp.CreationDate,
		CreationEpoch:          tp.CreationDate.Unix(),
		SentDate:               tp.SentDate,
		TreatmentList:          tp.TreatmentList,
		RegimenPlan:            tp.RegimenPlan,
		Note:                   tp.Note,
		ScheduledMessages:      make([]*ScheduledMessage, len(tp.ScheduledMessages)),
		ResourceGuides:         make([]*ResourceGuide, len(tp.ResourceGuides)),
	}

	if tp.Parent != nil {
		tpRes.Parent = &TreatmentPlanParent{
			ID:   tp.Parent.ParentID.Int64(),
			Type: tp.Parent.ParentType,
			DeprecatedCreationDate: tp.Parent.CreationDate,
			CreationEpoch:          tp.Parent.CreationDate.Unix(),
		}
	}

	if tp.ContentSource != nil {
		tpRes.ContentSource = &TreatmentPlanContentSource{
			ID:       tp.ContentSource.ID.Int64(),
			Type:     tp.ContentSource.Type,
			Deviated: tp.ContentSource.HasDeviated,
		}
	}

	var sentTime time.Time
	if tp.SentDate != nil {
		sentTime = *tp.SentDate
	} else {
		sentTime = time.Now().UTC()
	}
	var err error
	for i, sm := range tp.ScheduledMessages {
		tpRes.ScheduledMessages[i], err = TransformScheduledMessageToResponse(mLookup, mediaStore, sm, sentTime, mediaExpirationDuration)
		if err != nil {
			return nil, err
		}
	}

	for i, g := range tp.ResourceGuides {
		tpRes.ResourceGuides[i] = transformResourceGuideToResponse(g)
	}

	return tpRes, nil
}

func TransformTPFromResponse(mLookup mediaLookup, tp *TreatmentPlan, doctorID int64, role string) (*common.TreatmentPlan, error) {
	if tp == nil {
		return nil, nil
	}
	tp2 := &common.TreatmentPlan{
		ID:                tp.ID,
		DoctorID:          tp.DoctorID,
		PatientCaseID:     tp.PatientCaseID,
		PatientID:         tp.PatientID,
		Status:            tp.Status,
		CreationDate:      tp.DeprecatedCreationDate,
		SentDate:          tp.SentDate,
		TreatmentList:     tp.TreatmentList,
		RegimenPlan:       tp.RegimenPlan,
		Note:              tp.Note,
		ScheduledMessages: make([]*common.TreatmentPlanScheduledMessage, len(tp.ScheduledMessages)),
		ResourceGuides:    make([]*common.ResourceGuide, len(tp.ResourceGuides)),
	}

	if tp.Parent != nil {
		tp2.Parent = &common.TreatmentPlanParent{
			ParentID:     encoding.NewObjectID(tp.Parent.ID),
			ParentType:   tp.Parent.Type,
			CreationDate: tp.Parent.DeprecatedCreationDate,
		}
	}

	if tp.ContentSource != nil {
		tp2.ContentSource = &common.TreatmentPlanContentSource{
			ID:          encoding.NewObjectID(tp.ContentSource.ID),
			Type:        tp.ContentSource.Type,
			HasDeviated: tp.ContentSource.Deviated,
		}
	}

	var err error
	for i, sm := range tp.ScheduledMessages {
		tp2.ScheduledMessages[i], err = TransformScheduledMessageFromResponse(mLookup, sm, tp2.ID.Int64(), doctorID, role)
		if err != nil {
			return nil, err
		}
	}

	for i, g := range tp.ResourceGuides {
		tp2.ResourceGuides[i] = transformResourceGuideFromResponse(g)
	}

	return tp2, nil
}

func TransformFTPToResponse(
	mLookup mediaLookup,
	mediaStore *media.Store,
	mediaExpirationDuration time.Duration,
	ftp *common.FavoriteTreatmentPlan,
	pathwayTag string,
) (*FavoriteTreatmentPlan, error) {
	if ftp == nil {
		return nil, nil
	}
	ftpRes := &FavoriteTreatmentPlan{
		ID:         ftp.ID,
		PathwayTag: pathwayTag,
		Name:       ftp.Name,
		DeprecatedModifiedDate: ftp.ModifiedDate,
		ModifiedEpoch:          ftp.ModifiedDate.Unix(),
		RegimenPlan:            ftp.RegimenPlan,
		TreatmentList:          ftp.TreatmentList,
		Note:                   ftp.Note,
		ScheduledMessages:      make([]*ScheduledMessage, len(ftp.ScheduledMessages)),
		ResourceGuides:         make([]*ResourceGuide, len(ftp.ResourceGuides)),
	}

	now := time.Now().UTC()
	var err error
	for i, sm := range ftp.ScheduledMessages {
		ftpRes.ScheduledMessages[i], err = TransformScheduledMessageToResponse(mLookup, mediaStore, sm, now, mediaExpirationDuration)
		if err != nil {
			return nil, err
		}
	}

	for i, g := range ftp.ResourceGuides {
		ftpRes.ResourceGuides[i] = transformResourceGuideToResponse(g)
	}

	return ftpRes, nil
}

func TransformFTPFromResponse(mLookup mediaLookup, ftp *FavoriteTreatmentPlan, doctorID int64, role string) (*common.FavoriteTreatmentPlan, error) {
	if ftp == nil {
		return nil, nil
	}
	ftp2 := &common.FavoriteTreatmentPlan{
		ID:                ftp.ID,
		Name:              ftp.Name,
		ModifiedDate:      ftp.DeprecatedModifiedDate,
		RegimenPlan:       ftp.RegimenPlan,
		TreatmentList:     ftp.TreatmentList,
		Note:              ftp.Note,
		ScheduledMessages: make([]*common.TreatmentPlanScheduledMessage, len(ftp.ScheduledMessages)),
		ResourceGuides:    make([]*common.ResourceGuide, len(ftp.ResourceGuides)),
	}

	var err error
	for i, sm := range ftp.ScheduledMessages {
		ftp2.ScheduledMessages[i], err = TransformScheduledMessageFromResponse(mLookup, sm, ftp2.ID.Int64(), doctorID, role)
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

func TransformScheduledMessageFromResponse(mLookup mediaLookup, msg *ScheduledMessage, tpID, doctorID int64, role string) (*common.TreatmentPlanScheduledMessage, error) {
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
				personID, err = mLookup.GetPersonIDByRole(role, doctorID)
				if err != nil {
					return nil, err
				}
			}

			// Make sure media is uploaded by the same person and is unclaimed
			media, err := mLookup.GetMedia(a.ID)
			if err != nil {
				return nil, err
			}
			if media.UploaderID != personID {
				return nil, errors.New("invalid attached media")
			}
		case common.AttachmentTypeFollowupVisit:
		default:
			return nil, fmt.Errorf("attachment type %s not allowed in scheduled message", att.ItemType)
		}

		if att.Title == "" {
			att.Title = messages.AttachmentTitle(att.ItemType)
		}
	}
	return m, nil
}

func TransformScheduledMessageToResponse(
	mLookup mediaLookup,
	mediaStore *media.Store,
	m *common.TreatmentPlanScheduledMessage,
	sentTime time.Time,
	mediaExpirationDuration time.Duration,
) (*ScheduledMessage, error) {

	scheduledFor := sentTime.Add(24 * time.Hour * time.Duration(m.ScheduledDays))
	msg := &ScheduledMessage{
		ID:                     m.ID,
		ScheduledDays:          m.ScheduledDays,
		DeprecatedScheduledFor: &scheduledFor,
		ScheduledForEpoch:      scheduledFor.Unix(),
		Message:                m.Message,
		Attachments:            make([]*messages.Attachment, len(m.Attachments)),
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
			var err error
			att.URL, err = mediaStore.SignedURL(a.ItemID, mediaExpirationDuration)
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
	days := m.DeprecatedScheduledFor.Sub(time.Now()) / (time.Hour * 24)
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
