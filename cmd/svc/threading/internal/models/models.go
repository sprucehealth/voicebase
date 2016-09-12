package models

import (
	"database/sql/driver"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/libs/textutil"
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

func ThreadIDsToInterfaces(ids []ThreadID) []interface{} {
	ifs := make([]interface{}, len(ids))
	for i, id := range ids {
		ifs[i] = id
	}
	return ifs
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

type threadIDSlice []ThreadID

func (t threadIDSlice) Len() int {
	return len([]ThreadID(t))
}

func (t threadIDSlice) Less(i, j int) bool {
	ts := []ThreadID(t)
	return ts[i].Val < ts[j].Val
}

func (t threadIDSlice) Swap(i, j int) {
	ts := []ThreadID(t)
	jv := ts[j]
	ts[j] = ts[i]
	ts[i] = jv
}

// SortThreadID sorts the list of thread ids in ascending order
func SortThreadID(ids []ThreadID) {
	sort.Sort(threadIDSlice(ids))
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

// ThreadType is an enum of possible thread types
type ThreadType string

const (
	// ThreadTypeUnknown is temporary until all threads are migrated
	ThreadTypeUnknown ThreadType = ""
	// ThreadTypeExternal is a thread with with an external entity (e.g. patient)
	ThreadTypeExternal ThreadType = "EXTERNAL"
	// ThreadTypeTeam is an internal org thread between team mebers
	ThreadTypeTeam ThreadType = "TEAM"
	// ThreadTypeSetup is a scripted setup assistant thread
	ThreadTypeSetup ThreadType = "SETUP"
	// ThreadTypeSupport is a thread linked to the spruce support org
	ThreadTypeSupport ThreadType = "SUPPORT"
	// ThreadTypeLegacyInternal is a thread that represents the legacy internal thread
	// visible to all members of the org for internal communication.
	ThreadTypeLegacyTeam ThreadType = "LEGACY_TEAM"
	// ThreadTypeSecureExternal is a thread with with an external entity (e.g. patient) limited to secure in app communication
	ThreadTypeSecureExternal ThreadType = "SECURE_EXTERNAL"
)

// Scan implements sql.Scanner and expects src to be nil or of type string or []byte
func (tt *ThreadType) Scan(src interface{}) error {
	if src == nil {
		*tt = ThreadTypeUnknown
		return nil
	}
	var typ string
	switch v := src.(type) {
	case []byte:
		typ = string(v)
	case string:
		typ = v
	default:
		return errors.Trace(fmt.Errorf("unsupported type for ThreadType: %T", src))
	}
	*tt = ThreadType(strings.ToUpper(typ))
	return errors.Trace(tt.Validate())
}

// Value implements sql/driver.Valuer
func (tt ThreadType) Value() (driver.Value, error) {
	return strings.ToUpper(string(tt)), errors.Trace(tt.Validate())
}

// Validate returns nil iff the value of the type is valid
func (tt ThreadType) Validate() error {
	switch tt {
	case ThreadTypeUnknown, ThreadTypeExternal, ThreadTypeTeam, ThreadTypeSetup, ThreadTypeSupport, ThreadTypeLegacyTeam, ThreadTypeSecureExternal:
		return nil
	}
	return errors.Trace(fmt.Errorf("unknown thread type '%s'", string(tt)))
}

func (tt ThreadType) String() string {
	return string(tt)
}

// ThreadOrigin is an enum of possible thread origins
type ThreadOrigin string

const (
	// ThreadOriginUnknown is an unknown thread origin
	ThreadOriginUnknown ThreadOrigin = ""
	// ThreadOriginPatientInvite is a thread created from a patient invite
	ThreadOriginPatientInvite ThreadOrigin = "PATIENT_INVITE"
	// ThreadOriginOrganizationCode is a thread created from an organization code
	ThreadOriginOrganizationCode ThreadOrigin = "ORGANIZATION_CODE"
	// ThreadOriginPatientSync is a thread created from an external system sync
	ThreadOriginPatientSync ThreadOrigin = "SYNC"
)

// Scan implements sql.Scanner and expects src to be nil or of type string or []byte
func (to *ThreadOrigin) Scan(src interface{}) error {
	if src == nil {
		*to = ThreadOriginUnknown
		return nil
	}
	var typ string
	switch v := src.(type) {
	case []byte:
		typ = string(v)
	case string:
		typ = v
	default:
		return errors.Trace(fmt.Errorf("unsupported type for ThreadType: %T", src))
	}
	*to = ThreadOrigin(strings.ToUpper(typ))
	return errors.Trace(to.Validate())
}

// Value implements sql/driver.Valuer
func (to ThreadOrigin) Value() (driver.Value, error) {
	return strings.ToUpper(string(to)), errors.Trace(to.Validate())
}

// Validate returns nil iff the value of the type is valid
func (to ThreadOrigin) Validate() error {
	switch to {
	case ThreadOriginUnknown, ThreadOriginPatientInvite, ThreadOriginOrganizationCode, ThreadOriginPatientSync:
		return nil
	}
	return errors.Trace(fmt.Errorf("unknown thread origin '%s'", string(to)))
}

func (to ThreadOrigin) String() string {
	return string(to)
}

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
	MessageCount                 int
	SystemTitle                  string
	UserTitle                    string
	Type                         ThreadType
	Origin                       ThreadOrigin
}

// ThreadIDs is a convenience method for retrieving ID's from a list
// Note: This could be made more gneeric using reflection but don't want the performance cost
func ThreadIDs(ts []*Thread) []ThreadID {
	ids := make([]ThreadID, len(ts))
	for i, t := range ts {
		ids[i] = t.ID
	}
	return ids
}

// ThreadEntity links an entity to a thread.
type ThreadEntity struct {
	ThreadID         ThreadID
	EntityID         string
	Member           bool
	Joined           time.Time
	LastViewed       *time.Time
	LastUnreadNotify *time.Time
	LastReferenced   *time.Time
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
	ID       SavedQueryID
	Ordinal  int
	Title    string
	EntityID string
	Query    *Query
	Unread   int
	Total    int
	Created  time.Time
	Modified time.Time
}

// SetupThreadState is the state of a setup thread
type SetupThreadState struct {
	ThreadID ThreadID
	Step     int
}

// SummaryFromText returns a summary appropriate plaintext given BML markup.
func SummaryFromText(textMarkup string) (string, error) {
	textBML, err := bml.Parse(textMarkup)
	if err != nil {
		return "", errors.Trace(err)
	}
	plainText, err := textBML.PlainText()
	if err != nil {
		// Shouldn't fail here since the parsing should have done validation
		return "", errors.Trace(err)
	}
	plainText = strings.Replace(plainText, "\n", " ", -1)
	plainText = strings.Replace(plainText, "  ", " ", -1)
	pt := textutil.TruncateUTF8(plainText, 1000)
	if pt != plainText {
		pt += "…"
	}
	return pt, nil
}
