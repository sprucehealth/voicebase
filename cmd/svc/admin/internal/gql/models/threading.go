package models

import "github.com/sprucehealth/backend/svc/threading"

// SavedMessage represents a threading service saved message
type SavedMessage struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	OrganizationID  string `json:"organizationID"`
	CreatorEntityID string `json:"creatorEntityID"`
	OwnerEntityID   string `json:"ownerEntityID"`
	Internal        bool   `json:"internal"`
	Created         uint64 `json:"created"`
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
	return &SavedMessage{
		ID:              sm.ID,
		Title:           sm.Title,
		OrganizationID:  sm.OrganizationID,
		CreatorEntityID: sm.CreatorEntityID,
		OwnerEntityID:   sm.OwnerEntityID,
		Internal:        sm.Internal,
		Created:         sm.Created,
	}
}

// TriggeredMessage represents a threading service triggered message
type TriggeredMessage struct {
	ID                   string `json:"id"`
	OrganizationEntityID string `json:"organizationEntityID"`
	ActorEntityID        string `json:"actorEntityID"`
	Key                  string `json:"key"`
	SubKey               string `json:"subkey"`
	Enabled              bool   `json:"enabled"`
	Created              uint64 `json:"created"`
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
	return &TriggeredMessage{
		ID:                   tm.ID,
		OrganizationEntityID: tm.OrganizationEntityID,
		ActorEntityID:        tm.ActorEntityID,
		Key:                  tm.Key.Key.String(),
		SubKey:               tm.Key.Subkey,
		Enabled:              tm.Enabled,
		Created:              tm.Created,
	}
}
