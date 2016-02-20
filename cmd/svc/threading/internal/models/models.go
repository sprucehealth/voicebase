package models

import (
	"time"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/svc/threading"
)

// ThreadID is the ID for a Thread
type ThreadID struct{ model.ObjectID }

func NewThreadID() (ThreadID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return ThreadID{}, errors.Trace(err)
	}
	return ThreadID{
		model.ObjectID{
			Prefix:  threading.ThreadIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

func ParseThreadID(s string) (ThreadID, error) {
	t := EmptyThreadID()
	err := t.UnmarshalText([]byte(s))
	return t, errors.Trace(err)
}

func EmptyThreadID() ThreadID {
	return ThreadID{
		model.ObjectID{
			Prefix:  threading.ThreadIDPrefix,
			IsValid: false,
		},
	}
}

// ThreadItemID is the ID for a ThreadItem
type ThreadItemID struct{ model.ObjectID }

func NewThreadItemID() (ThreadItemID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return ThreadItemID{}, errors.Trace(err)
	}
	return ThreadItemID{
		model.ObjectID{
			Prefix:  threading.ThreadItemIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

func ParseThreadItemID(s string) (ThreadItemID, error) {
	t := EmptyThreadItemID()
	err := t.UnmarshalText([]byte(s))
	return t, errors.Trace(err)
}

func EmptyThreadItemID() ThreadItemID {
	return ThreadItemID{
		model.ObjectID{
			Prefix:  threading.ThreadItemIDPrefix,
			IsValid: false,
		},
	}
}

// SavedQueryID is the ID for a SavedQuery
type SavedQueryID struct{ model.ObjectID }

func NewSavedQueryID() (SavedQueryID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return SavedQueryID{}, errors.Trace(err)
	}
	return SavedQueryID{
		model.ObjectID{
			Prefix:  threading.SavedQueryIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

func ParseSavedQueryID(s string) (SavedQueryID, error) {
	t := EmptySavedQueryID()
	err := t.UnmarshalText([]byte(s))
	return t, errors.Trace(err)
}

func EmptySavedQueryID() SavedQueryID {
	return SavedQueryID{
		model.ObjectID{
			Prefix:  threading.SavedQueryIDPrefix,
			IsValid: false,
		},
	}
}

// ItemType is an enum of possible thread item types
type ItemType string

const (
	// ItemTypeMessage is a message item which is the only concrete type. Every other item type is an event.
	ItemTypeMessage ItemType = "MESSAGE"
)

// ThreadEvent is an enum of possible thread event types
type ThreadEvent string

const (
	// ThreadEventDelete signifies that a thread was deleted
	ThreadEventDelete ThreadEvent = "DELETE"
)

// Thread is a thread of conversation and the parent of thread items.
type Thread struct {
	ID                           ThreadID
	OrganizationID               string
	PrimaryEntityID              string
	LastMessageTimestamp         time.Time
	LastExternalMessageTimestamp time.Time
	LastMessageSummary           string
	LastExternalMessageSummary   string
	LastPrimaryEntityEndpoints   EndpointList
	Created                      time.Time
}

// ThreadMember links an entity to a thread.
type ThreadMember struct {
	ThreadID   ThreadID
	EntityID   string
	Following  bool
	Joined     time.Time
	LastViewed *time.Time
}

// ThreadItem is an item within a thread. It can be a message or an event modifying a message.
type ThreadItem struct {
	ID            ThreadItemID
	ThreadID      ThreadID
	Created       time.Time
	ActorEntityID string
	Internal      bool
	Type          ItemType
	Data          interface{}
}

// ThreadItemViewDetails is the view details associated with a thread item
type ThreadItemViewDetails struct {
	ThreadItemID  ThreadItemID
	ActorEntityID string
	ViewTime      *time.Time
}

// SavedQuery is a saved thread query.
type SavedQuery struct {
	ID             SavedQueryID
	OrganizationID string
	EntityID       string
	Query          []byte // TODO
	Created        time.Time
	Modified       time.Time
}
