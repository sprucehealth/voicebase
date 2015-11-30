package models

import (
	"time"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/idgen"
)

const (
	savedQueryIDPrefix = "sq_"
	threadIDPrefix     = "t_"
	threadItemIDPrefix = "ti_"
)

// ThreadID is the ID for a Thread
type ThreadID struct{ objectID }

func NewThreadID() (ThreadID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return ThreadID{}, errors.Trace(err)
	}
	return ThreadID{
		objectID{
			prefix:  threadIDPrefix,
			value:   id,
			isValid: true,
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
		objectID{
			prefix:  threadIDPrefix,
			isValid: false,
		},
	}
}

// ThreadItemID is the ID for a ThreadItem
type ThreadItemID struct{ objectID }

func NewThreadItemID() (ThreadItemID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return ThreadItemID{}, errors.Trace(err)
	}
	return ThreadItemID{
		objectID{
			prefix:  threadItemIDPrefix,
			value:   id,
			isValid: true,
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
		objectID{
			prefix:  threadItemIDPrefix,
			isValid: false,
		},
	}
}

// SavedQueryID is the ID for a SavedQuery
type SavedQueryID struct{ objectID }

func NewSavedQueryID() (SavedQueryID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return SavedQueryID{}, errors.Trace(err)
	}
	return SavedQueryID{
		objectID{
			prefix:  savedQueryIDPrefix,
			value:   id,
			isValid: true,
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
		objectID{
			prefix:  savedQueryIDPrefix,
			isValid: false,
		},
	}
}

type ItemType string

const (
	ItemTypeMessage ItemType = "MESSAGE"
)

type Thread struct {
	ID              ThreadID
	OrganizationID  string
	PrimaryEntityID string
}

type ThreadMember struct {
	ThreadID  ThreadID
	EntityID  string
	Following bool
	Joined    time.Time
}

type ThreadItem struct {
	ID            ThreadItemID
	ThreadID      ThreadID
	Created       time.Time
	ActorEntityID string
	Internal      bool
	Type          ItemType
	Data          interface{}
}

type SavedQuery struct {
	ID             SavedQueryID
	OrganizationID string
	EntityID       string
	Query          []byte // TODO
	Created        time.Time
	Modified       time.Time
}
