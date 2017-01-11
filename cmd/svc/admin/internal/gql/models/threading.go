package models

import (
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/threading"
)

// SavedMessage represents a threading service saved message
type SavedMessage struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	OrganizationID  string   `json:"organizationID"`
	CreatorEntityID string   `json:"creatorEntityID"`
	OwnerEntityID   string   `json:"ownerEntityID"`
	Internal        bool     `json:"internal"`
	Created         uint64   `json:"created"`
	Message         *Message `json:"message"`
}

// TransformSavedMessagesToModel transforms an internal saved messages into something understood by graphql
func TransformSavedMessagesToModel(sms []*threading.SavedMessage) []*SavedMessage {
	rsms := make([]*SavedMessage, len(sms))
	for i, sm := range sms {
		rsms[i] = TransformSavedMessageToModel(sm)
	}
	return rsms
}

// TransformSavedMessageToModel transforms an internal saved message into something understood by graphql
func TransformSavedMessageToModel(sm *threading.SavedMessage) *SavedMessage {
	if sm == nil {
		return nil
	}
	rsm := &SavedMessage{
		ID:              sm.ID,
		Title:           sm.Title,
		OrganizationID:  sm.OrganizationID,
		CreatorEntityID: sm.CreatorEntityID,
		OwnerEntityID:   sm.OwnerEntityID,
		Internal:        sm.Internal,
		Created:         sm.Created,
	}
	switch c := sm.Content.(type) {
	case *threading.SavedMessage_Message:
		rsm.Message = TransformMessageToModel(c.Message)
	default:
		golog.Errorf("Unable to parse saved message content of unknown type %+v", sm.Content)
	}
	return rsm
}

// TriggeredMessage represents a threading service triggered message
type TriggeredMessage struct {
	ID                   string                  `json:"id"`
	OrganizationEntityID string                  `json:"organizationEntityID"`
	ActorEntityID        string                  `json:"actorEntityID"`
	Key                  string                  `json:"key"`
	SubKey               string                  `json:"subkey"`
	Enabled              bool                    `json:"enabled"`
	Created              uint64                  `json:"created"`
	Items                []*TriggeredMessageItem `json:"items"`
}

// TransformTriggeredMessagesToModel transforms an internal triggered messages into something understood by graphql
func TransformTriggeredMessagesToModel(tms []*threading.TriggeredMessage) []*TriggeredMessage {
	rtms := make([]*TriggeredMessage, len(tms))
	for i, tm := range tms {
		rtms[i] = TransformTriggeredMessageToModel(tm)
	}
	return rtms
}

// TransformTriggeredMessageToModel transforms an internal triggered message into something understood by graphql
func TransformTriggeredMessageToModel(tm *threading.TriggeredMessage) *TriggeredMessage {
	if tm == nil {
		return nil
	}
	return &TriggeredMessage{
		ID:                   tm.ID,
		OrganizationEntityID: tm.OrganizationEntityID,
		ActorEntityID:        tm.ActorEntityID,
		Key:                  tm.Key.Key.String(),
		SubKey:               tm.Key.Subkey,
		Enabled:              tm.Enabled,
		Items:                TransformTriggeredMessageItemsToModel(tm.Items),
		Created:              tm.Created,
	}
}

// TriggeredMessageItem represents a threading service triggered message item
type TriggeredMessageItem struct {
	ID                 string   `json:"id"`
	TriggeredMessageID string   `json:"triggeredMessageID"`
	ActorEntityID      string   `json:"actorEntityID"`
	Internal           bool     `json:"internal"`
	Ordinal            int64    `json:"ordinal"`
	Content            string   `json:"content"`
	Created            uint64   `json:"created"`
	Message            *Message `json:"message"`
}

// TransformTriggeredMessageItemsToModel transforms an internal triggered messages items into something understood by graphql
func TransformTriggeredMessageItemsToModel(tmis []*threading.TriggeredMessageItem) []*TriggeredMessageItem {
	rtmis := make([]*TriggeredMessageItem, len(tmis))
	for i, tmi := range tmis {
		rtmis[i] = TransformTriggeredMessageItemToModel(tmi)
	}
	return rtmis
}

// TransformTriggeredMessageItemToModel transforms an internal triggered message item into something understood by graphql
func TransformTriggeredMessageItemToModel(tmi *threading.TriggeredMessageItem) *TriggeredMessageItem {
	if tmi == nil {
		return nil
	}
	rtmi := &TriggeredMessageItem{
		ID:                 tmi.ID,
		TriggeredMessageID: tmi.TriggeredMessageID,
		ActorEntityID:      tmi.ActorEntityID,
		Internal:           tmi.Internal,
		Ordinal:            tmi.Ordinal,
		Created:            tmi.Created,
	}
	switch c := tmi.Content.(type) {
	case *threading.TriggeredMessageItem_Message:
		rtmi.Message = TransformMessageToModel(c.Message)
	default:
		golog.Errorf("Unable to parse triggered message content of unknown type %+v", tmi.Content)
	}
	return rtmi
}

// Message represents a threading service message
type Message struct {
	Text         string        `json:"text"`
	Title        string        `json:"title"`
	Summary      string        `json:"summary"`
	Attachments  []*Attachment `json:"attachments"`
	Source       *Endpoint     `json:"source"`
	Destinations []*Endpoint   `json:"destinations"`
	TextRefs     []*Reference  `json:"textRefs"`
}

// TransformMessageToModel transforms an internal triggered message into something understood by graphql
func TransformMessageToModel(m *threading.Message) *Message {
	if m == nil {
		return nil
	}
	rm := &Message{
		Text:         m.Text,
		Title:        m.Title,
		Summary:      m.Summary,
		Attachments:  TransformAttachmentsToModel(m.Attachments),
		Source:       TransformEndpointToModel(m.Source),
		Destinations: TransformEndpointsToModel(m.Destinations),
		TextRefs:     TransformReferencesToModel(m.TextRefs),
	}
	return rm
}

// Attachment represents a threading service attachment
type Attachment struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	UserTitle string `json:"userTitle"`
	ContentID string `json:"contentID"`
}

// TransformAttachmentsToModel transforms an internal attachment into something understood by graphql
func TransformAttachmentsToModel(as []*threading.Attachment) []*Attachment {
	ras := make([]*Attachment, len(as))
	for i, a := range as {
		ras[i] = TransformAttachmentToModel(a)
	}
	return ras
}

// TransformAttachmentToModel transforms an internal attachment into something understood by graphql
func TransformAttachmentToModel(a *threading.Attachment) *Attachment {
	if a == nil {
		return nil
	}
	return &Attachment{
		Title:     a.Title,
		URL:       a.URL,
		UserTitle: a.UserTitle,
		ContentID: a.ContentID,
	}
}

// Endpoint represents a threading service endpoint
type Endpoint struct {
	ID      string `json:"id"`
	Channel string `json:"channel"`
}

// TransformEndpointsToModel transforms an internal endpoint into something understood by graphql
func TransformEndpointsToModel(es []*threading.Endpoint) []*Endpoint {
	res := make([]*Endpoint, len(es))
	for i, e := range es {
		res[i] = TransformEndpointToModel(e)
	}
	return res
}

// TransformEndpointToModel transforms an internal endpoint into something understood by graphql
func TransformEndpointToModel(e *threading.Endpoint) *Endpoint {
	if e == nil {
		return nil
	}
	return &Endpoint{
		ID:      e.ID,
		Channel: e.Channel.String(),
	}
}

// Reference represents a threading service reference
type Reference struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// TransformReferencesToModel transforms an internal reference into something understood by graphql
func TransformReferencesToModel(rs []*threading.Reference) []*Reference {
	rrs := make([]*Reference, len(rs))
	for i, r := range rs {
		rrs[i] = TransformReferenceToModel(r)
	}
	return rrs
}

// TransformReferenceToModel transforms an internal reference into something understood by graphql
func TransformReferenceToModel(r *threading.Reference) *Reference {
	if r == nil {
		return nil
	}
	return &Reference{
		ID:   r.ID,
		Type: r.Type.String(),
	}
}

// SavedThreadQuery is a saved thread query of the threading service
type SavedThreadQuery struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	Query                string `json:"query"`
	ShortTitle           string `json:"shortTitle"`
	LongTitle            string `json:"longTitle"`
	Description          string `json:"description"`
	Unread               int    `json:"unread"`
	Total                int    `json:"total"`
	Ordinal              int    `json:"ordinal"`
	NotificationsEnabled bool   `json:"notificationsEnabled"`
	Hidden               bool   `json:"hidden"`
	Template             bool   `json:"template"`
	DefaultTemplate      bool   `json:"defaultTemplate"`
}

// TransformSavedThreadQueriesToModel transforms a set of internal saved thread queries into something understood by graphql
func TransformSavedThreadQueriesToModel(sqs []*threading.SavedQuery) ([]*SavedThreadQuery, error) {
	rsqs := make([]*SavedThreadQuery, len(sqs))
	for i, sq := range sqs {
		var err error
		rsqs[i], err = TransformSavedThreadQueryToModel(sq)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	return rsqs, nil
}

// TransformSavedThreadQueryToModel transforms an internal saved thread query into something understood by graphql
func TransformSavedThreadQueryToModel(sq *threading.SavedQuery) (*SavedThreadQuery, error) {
	query, err := threading.FormatQuery(sq.Query)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &SavedThreadQuery{
		ID:                   sq.ID,
		Type:                 sq.Type.String(),
		Query:                query,
		ShortTitle:           sq.ShortTitle,
		LongTitle:            sq.LongTitle,
		Description:          sq.Description,
		Unread:               int(sq.Unread),
		Total:                int(sq.Total),
		Ordinal:              int(sq.Ordinal),
		NotificationsEnabled: sq.NotificationsEnabled,
		Hidden:               sq.Hidden,
		Template:             sq.Template,
		DefaultTemplate:      sq.DefaultTemplate,
	}, nil
}
