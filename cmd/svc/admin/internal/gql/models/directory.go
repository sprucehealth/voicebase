package models

import "github.com/sprucehealth/backend/svc/directory"

// Me represents the active caller
type Me struct{}

// Entity represents the various aspects of an entity in the baymax system
type Entity struct {
	ID            string     `json:"id"`
	AccountID     string     `json:"accountID"`
	Type          string     `json:"type"`
	Status        string     `json:"status"`
	FirstName     string     `json:"firstName"`
	MiddleInitial string     `json:"middleInitial"`
	LastName      string     `json:"lastName"`
	GroupName     string     `json:"groupName"`
	DisplayName   string     `json:"displayName"`
	ShortTitle    string     `json:"shortTitle"`
	LongTitle     string     `json:"longTitle"`
	Gender        string     `json:"gender"`
	DOB           *DOB       `json:"dob"`
	Note          string     `json:"note"`
	Contacts      []*Contact `json:"contacts"`
	ExternalIDs   []string   `json:"external_ids"`
	CreatedAt     int        `json:"created_at"`
	ModifiedAt    int        `json:"modified_at"`
	Members       []*Entity  `json:"members"`
	Memberships   []*Entity  `json:"memberships"`
}

// DOB TODO: Consider merging this with the baymax equivalents
type DOB struct {
	Month int `json:"month"`
	Day   int `json:"day"`
	Year  int `json:"year"`
}

// Contact TODO: Consider merging this with the baymax equivalents
type Contact struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Value       string `json:"value"`
	Provisioned bool   `json:"provisioned"`
	Label       string `json:"label"`
}

// TransformEntitiesToModels transforms the internal directory entities into something understood by graphql
func TransformEntitiesToModels(es []*directory.Entity) []*Entity {
	mes := make([]*Entity, len(es))
	for i, e := range es {
		mes[i] = TransformEntityToModel(e)
	}
	return mes
}

// TransformEntityToModel transforms the internal directory entity into something understood by graphql
func TransformEntityToModel(e *directory.Entity) *Entity {
	var dob *DOB
	if e.Info.DOB != nil {
		dob = &DOB{
			Day:   int(e.Info.DOB.Day),
			Month: int(e.Info.DOB.Month),
			Year:  int(e.Info.DOB.Year),
		}
	}
	var members []*Entity
	if len(e.Members) != 0 {
		members = TransformEntitiesToModels(e.Members)
	}
	var memberships []*Entity
	if len(e.Memberships) != 0 {
		memberships = TransformEntitiesToModels(e.Memberships)
	}
	externalIDs := make([]string, len(e.ExternalIDs))
	for i, eid := range e.ExternalIDs {
		externalIDs[i] = eid
	}
	var contacts []*Contact
	if len(e.Contacts) != 0 {
		contacts = TransformContactsToModel(e.Contacts)
	}
	return &Entity{
		ID:            e.ID,
		AccountID:     e.AccountID,
		Type:          e.Type.String(),
		Status:        e.Status.String(),
		FirstName:     e.Info.FirstName,
		MiddleInitial: e.Info.MiddleInitial,
		LastName:      e.Info.LastName,
		GroupName:     e.Info.GroupName,
		DisplayName:   e.Info.DisplayName,
		ShortTitle:    e.Info.ShortTitle,
		LongTitle:     e.Info.LongTitle,
		Gender:        e.Info.Gender.String(),
		DOB:           dob,
		Note:          e.Info.Note,
		Contacts:      contacts,
		ExternalIDs:   externalIDs,
		CreatedAt:     int(e.CreatedTimestamp),
		ModifiedAt:    int(e.LastModifiedTimestamp),
		Members:       members,
		Memberships:   memberships,
	}
}

// TransformContactsToModel transforms the internal directory entity contact into something understood by graphql
func TransformContactsToModel(cs []*directory.Contact) []*Contact {
	mcs := make([]*Contact, len(cs))
	for i, c := range cs {
		mcs[i] = TransformContactToModel(c)
	}
	return mcs
}

// TransformContactToModel transforms the internal directory entity contact into something understood by graphql
func TransformContactToModel(c *directory.Contact) *Contact {
	return &Contact{
		ID:          c.ID,
		Type:        c.ContactType.String(),
		Value:       c.Value,
		Provisioned: c.Provisioned,
		Label:       c.Label,
	}
}
