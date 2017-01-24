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

// SavedMessageID is the ID for a SavedMessage
type SavedMessageID struct{ model.ObjectID }

func NewSavedMessageID() (SavedMessageID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return SavedMessageID{}, errors.Trace(err)
	}
	return SavedMessageID{
		model.ObjectID{
			Prefix:  threading.SavedMessageIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

func ParseSavedMessageID(s string) (SavedMessageID, error) {
	t := EmptySavedMessageID()
	err := t.UnmarshalText([]byte(s))
	return t, errors.Trace(err)
}

func EmptySavedMessageID() SavedMessageID {
	return SavedMessageID{
		model.ObjectID{
			Prefix:  threading.SavedMessageIDPrefix,
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

const (
	// ItemTypeMessage is a message item which is the only concrete type. Every other item type is an event.
	ItemTypeMessage = "MESSAGE"
	// ItemTypeMessageUpdate is an update to an existing message
	ItemTypeMessageUpdate = "MESSAGE_UPDATE"
	// ItemTypeMessageDelete is a message deletion event
	ItemTypeMessageDelete = "MESSAGE_DELETE"
)

// ItemValue is the interface for a thread item value
type ItemValue interface {
	Marshal() ([]byte, error)
}

// ItemTypeForValue returns the ItemType for a given value object
func ItemTypeForValue(v ItemValue) (string, error) {
	switch v.(type) {
	case *Message:
		return ItemTypeMessage, nil
	case *MessageUpdate:
		return ItemTypeMessageUpdate, nil
	case *MessageDelete:
		return ItemTypeMessageDelete, nil
	}
	return "INVALID", errors.Errorf("invalid item value type %T", v)
}

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
	// ThreadTypeLegacyTeam is a thread that represents the legacy internal thread
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
		return errors.Errorf("unsupported type for ThreadType: %T", src)
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
	return errors.Errorf("unknown thread type '%s'", string(tt))
}

func (tt ThreadType) String() string {
	return string(tt)
}

// SavedQueryType is an enum of possible saved query types
type SavedQueryType string

const (
	// SavedQueryTypeNormal is
	SavedQueryTypeNormal SavedQueryType = "NORMAL"
	// SavedQueryTypeNotifications is
	SavedQueryTypeNotifications SavedQueryType = "NOTIFICATIONS"
)

// Scan implements sql.Scanner and expects src to be nil or of type string or []byte
func (t *SavedQueryType) Scan(src interface{}) error {
	if src == nil {
		*t = SavedQueryType("")
		return nil
	}
	var typ string
	switch v := src.(type) {
	case []byte:
		typ = string(v)
	case string:
		typ = v
	default:
		return errors.Errorf("unsupported type for SavedQueryType: %T", src)
	}
	*t = SavedQueryType(strings.ToUpper(typ))
	return errors.Trace(t.Validate())
}

// Value implements sql/driver.Valuer
func (t SavedQueryType) Value() (driver.Value, error) {
	return strings.ToUpper(string(t)), errors.Trace(t.Validate())
}

// Validate returns nil iff the value of the type is valid
func (t SavedQueryType) Validate() error {
	switch t {
	case SavedQueryTypeNormal, SavedQueryTypeNotifications:
		return nil
	}
	return errors.Errorf("unknown saved query type '%s'", string(t))
}

func (t SavedQueryType) String() string {
	return string(t)
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
		return errors.Errorf("unsupported type for ThreadType: %T", src)
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
	return errors.Errorf("unknown thread origin '%s'", string(to))
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
	Deleted                      bool
	Tags                         []Tag
}

type Tag struct {
	Name   string
	Hidden bool
}

type TagsByName []Tag

func (ts TagsByName) Len() int           { return len(ts) }
func (ts TagsByName) Swap(a, b int)      { ts[a], ts[b] = ts[b], ts[a] }
func (ts TagsByName) Less(a, b int) bool { return ts[a].Name < ts[b].Name }

// ThreadIDs is a convenience method for retrieving ID's from a list
// Note: This could be made more generic using reflection but don't want the performance cost
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
	Following        bool
	Joined           time.Time
	LastViewed       *time.Time
	LastUnreadNotify *time.Time
	LastReferenced   *time.Time
}

// ThreadItem is an item within a thread. It can be a message or an event modifying a message.
type ThreadItem struct {
	ID            ThreadItemID
	Deleted       bool
	ThreadID      ThreadID
	Created       time.Time
	Modified      time.Time
	ActorEntityID string
	Internal      bool
	Data          ItemValue
}

// ThreadItemViewDetails is the view details associated with a thread item
type ThreadItemViewDetails struct {
	ThreadItemID  ThreadItemID
	ActorEntityID string
	ViewTime      *time.Time
}

// SavedMessage is a message template
type SavedMessage struct {
	ID              SavedMessageID
	Title           string
	OrganizationID  string
	CreatorEntityID string
	OwnerEntityID   string
	Internal        bool
	Content         ItemValue
	Created         time.Time
	Modified        time.Time
}

// DefaultSavedQueries is the default set of queries that gets created for every organization
// unless an organization has a particular template of saved thread queries to be created.
var DefaultSavedQueries = []*SavedQuery{
	{
		Template:             true,
		Type:                 SavedQueryTypeNormal,
		ShortTitle:           "All",
		LongTitle:            "All Conversations",
		Description:          "Any new activity in any conversation",
		Ordinal:              1000,
		NotificationsEnabled: false,
		Query:                &Query{},
	},
	{
		Template:             true,
		Type:                 SavedQueryTypeNormal,
		ShortTitle:           "Patient",
		LongTitle:            "All Patient Conversations",
		Description:          "Any new activity in a patient conversation",
		Ordinal:              2000,
		NotificationsEnabled: true,
		Query:                &Query{Expressions: []*Expr{{Value: &Expr_ThreadType_{ThreadType: EXPR_THREAD_TYPE_PATIENT}}}},
	},
	{
		Template:             true,
		Type:                 SavedQueryTypeNormal,
		ShortTitle:           "Team",
		LongTitle:            "Team Conversations",
		Description:          "New messages in team conversations",
		Ordinal:              3000,
		NotificationsEnabled: true,
		Query:                &Query{Expressions: []*Expr{{Value: &Expr_ThreadType_{ThreadType: EXPR_THREAD_TYPE_TEAM}}}},
	},
	{
		Template:             true,
		Type:                 SavedQueryTypeNormal,
		ShortTitle:           "@Pages",
		LongTitle:            "@ Pages",
		Description:          "When you're @ paged in a message",
		Ordinal:              4000,
		NotificationsEnabled: true,
		Query:                &Query{Expressions: []*Expr{{Value: &Expr_Flag_{Flag: EXPR_FLAG_UNREAD_REFERENCE}}}},
	},
	{
		Template:             true,
		Type:                 SavedQueryTypeNormal,
		ShortTitle:           "Following",
		LongTitle:            "Patient Conversations You Follow",
		Description:          "New activity in patient conversations you are currently following",
		Ordinal:              5000,
		NotificationsEnabled: true,
		Query:                &Query{Expressions: []*Expr{{Value: &Expr_Flag_{Flag: EXPR_FLAG_FOLLOWING}}}},
	},
	{
		Template:             true,
		Type:                 SavedQueryTypeNormal,
		ShortTitle:           "Support",
		LongTitle:            "Spruce Support",
		Description:          "New messages in the Spruce Support conversation",
		Query:                &Query{Expressions: []*Expr{{Value: &Expr_ThreadType_{ThreadType: EXPR_THREAD_TYPE_SUPPORT}}}},
		Ordinal:              6000,
		NotificationsEnabled: true,
		Hidden:               true,
	},
	{
		Template:             true,
		Type:                 SavedQueryTypeNotifications,
		ShortTitle:           "Notifications",
		LongTitle:            "Notifications",
		Description:          "Hidden query to populate an accurate count of notifications",
		Query:                &Query{},
		Ordinal:              1000000000,
		NotificationsEnabled: false,
		Hidden:               true,
	},
}

// SavedQuery is a saved thread query.
type SavedQuery struct {
	ID                   SavedQueryID
	Ordinal              int
	ShortTitle           string
	LongTitle            string
	Description          string
	EntityID             string
	Query                *Query
	Unread               int
	Total                int
	Hidden               bool
	NotificationsEnabled bool
	Type                 SavedQueryType
	Created              time.Time
	Modified             time.Time
	Template             bool
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

// NewScheduledMessageID returns a new ScheduledMessageID.
func NewScheduledMessageID() (ScheduledMessageID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return ScheduledMessageID{}, errors.Trace(err)
	}
	return ScheduledMessageID{
		model.ObjectID{
			Prefix:  threading.ScheduledMessageIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyScheduledMessageID returns an empty initialized ID
func EmptyScheduledMessageID() ScheduledMessageID {
	return ScheduledMessageID{
		model.ObjectID{
			Prefix:  threading.ScheduledMessageIDPrefix,
			IsValid: false,
		},
	}
}

// ParseScheduledMessageID transforms an ScheduledMessageID from it's string representation into the actual ID value
func ParseScheduledMessageID(s string) (ScheduledMessageID, error) {
	id := EmptyScheduledMessageID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// ScheduledMessageID is the ID for a ScheduledMessageID object
type ScheduledMessageID struct {
	model.ObjectID
}

// ScheduledMessageStatus represents the type associated with the status column of the scheduled_message table
type ScheduledMessageStatus string

const (
	// ScheduledMessageStatusPending represents the PENDING state of the status field on a scheduled_message record
	ScheduledMessageStatusPending ScheduledMessageStatus = "PENDING"
	// ScheduledMessageStatusSent represents the SENT state of the status field on a scheduled_message record
	ScheduledMessageStatusSent ScheduledMessageStatus = "SENT"
	// ScheduledMessageStatusDeleted represents the DELETED state of the status field on a scheduled_message record
	ScheduledMessageStatusDeleted ScheduledMessageStatus = "DELETED"
)

// ParseScheduledMessageStatus converts a string into the correcponding enum value
func ParseScheduledMessageStatus(s string) (ScheduledMessageStatus, error) {
	switch t := ScheduledMessageStatus(strings.ToUpper(s)); t {
	case ScheduledMessageStatusPending, ScheduledMessageStatusSent, ScheduledMessageStatusDeleted:
		return t, nil
	}
	return ScheduledMessageStatus(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t ScheduledMessageStatus) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t ScheduledMessageStatus) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of ScheduledMessageStatus from a database conforming to the sql.Scanner interface
func (t *ScheduledMessageStatus) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseScheduledMessageStatus(ts)
	case []byte:
		*t, err = ParseScheduledMessageStatus(string(ts))
	}
	return errors.Trace(err)
}

// ScheduledMessage represents a scheduled_message record
type ScheduledMessage struct {
	ScheduledFor     time.Time
	SentAt           *time.Time
	Created          time.Time
	Modified         time.Time
	ActorEntityID    string
	ThreadID         ThreadID
	Internal         bool
	Data             ItemValue
	Status           ScheduledMessageStatus
	ID               ScheduledMessageID
	SentThreadItemID ThreadItemID
}

// ScheduledMessageUpdate represents the mutable aspects of a scheduled_message record
type ScheduledMessageUpdate struct {
	SentAt           *time.Time
	Status           *ScheduledMessageStatus
	SentThreadItemID *ThreadItemID
}

// TriggeredMessageIDPrefix represents the string that is attached to the beginning of these identifiers
const TriggeredMessageIDPrefix = "trm_"

// NewTriggeredMessageID returns a new TriggeredMessageID.
func NewTriggeredMessageID() (TriggeredMessageID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return TriggeredMessageID{}, errors.Trace(err)
	}
	return TriggeredMessageID{
		model.ObjectID{
			Prefix:  TriggeredMessageIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyTriggeredMessageID returns an empty initialized ID
func EmptyTriggeredMessageID() TriggeredMessageID {
	return TriggeredMessageID{
		model.ObjectID{
			Prefix:  TriggeredMessageIDPrefix,
			IsValid: false,
		},
	}
}

// ParseTriggeredMessageID transforms an TriggeredMessageID from it's string representation into the actual ID value
func ParseTriggeredMessageID(s string) (TriggeredMessageID, error) {
	id := EmptyTriggeredMessageID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// TriggeredMessageID is the ID for a TriggeredMessageID object
type TriggeredMessageID struct {
	model.ObjectID
}

// TriggeredMessageItemIDPrefix represents the string that is attached to the beginning of these identifiers
const TriggeredMessageItemIDPrefix = "trmi_"

// NewTriggeredMessageItemID returns a new TriggeredMessageItemID.
func NewTriggeredMessageItemID() (TriggeredMessageItemID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return TriggeredMessageItemID{}, errors.Trace(err)
	}
	return TriggeredMessageItemID{
		model.ObjectID{
			Prefix:  TriggeredMessageItemIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyTriggeredMessageItemID returns an empty initialized ID
func EmptyTriggeredMessageItemID() TriggeredMessageItemID {
	return TriggeredMessageItemID{
		model.ObjectID{
			Prefix:  TriggeredMessageItemIDPrefix,
			IsValid: false,
		},
	}
}

// ParseTriggeredMessageItemID transforms an TriggeredMessageItemID from it's string representation into the actual ID value
func ParseTriggeredMessageItemID(s string) (TriggeredMessageItemID, error) {
	id := EmptyTriggeredMessageItemID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// TriggeredMessageItemID is the ID for a TriggeredMessageItemID object
type TriggeredMessageItemID struct {
	model.ObjectID
}

// TriggeredMessageItem represents a triggered_message_item record
type TriggeredMessageItem struct {
	ID                 TriggeredMessageItemID
	TriggeredMessageID TriggeredMessageID
	Ordinal            int64
	ActorEntityID      string
	Internal           bool
	Type               string
	Data               ItemValue
	Created            time.Time
	Modified           time.Time
}

// TriggeredMessage represents a triggered_message record
type TriggeredMessage struct {
	ID                   TriggeredMessageID
	ActorEntityID        string
	OrganizationEntityID string
	TriggerKey           string
	TriggerSubkey        string
	Enabled              bool
	Created              time.Time
	Modified             time.Time
}

// TriggeredMessageUpdate represents the mutable aspects of a triggered_message record
type TriggeredMessageUpdate struct {
	Enabled *bool
}

const (
	TriggeredMessageKeyNewPatient  = "NEW_PATIENT"
	TriggeredMessageKeyAwayMessage = "AWAY_MESSAGE"
)

// BatchJobIDPrefix represents the string that is attached to the beginning of these identifiers
const BatchJobIDPrefix = "batchJob_"

// NewBatchJobID returns a new BatchJobID.
func NewBatchJobID() (BatchJobID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return BatchJobID{}, errors.Trace(err)
	}
	return BatchJobID{
		model.ObjectID{
			Prefix:  BatchJobIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyBatchJobID returns an empty initialized ID
func EmptyBatchJobID() BatchJobID {
	return BatchJobID{
		model.ObjectID{
			Prefix:  BatchJobIDPrefix,
			IsValid: false,
		},
	}
}

// ParseBatchJobID transforms an BatchJobID from it's string representation into the actual ID value
func ParseBatchJobID(s string) (BatchJobID, error) {
	id := EmptyBatchJobID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// BatchJobID is the ID for a BatchJobID object
type BatchJobID struct {
	model.ObjectID
}

// BatchTaskIDPrefix represents the string that is attached to the beginning of these identifiers
const BatchTaskIDPrefix = "batchTask_"

// NewBatchTaskID returns a new BatchTaskID.
func NewBatchTaskID() (BatchTaskID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return BatchTaskID{}, errors.Trace(err)
	}
	return BatchTaskID{
		model.ObjectID{
			Prefix:  BatchTaskIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyBatchTaskID returns an empty initialized ID
func EmptyBatchTaskID() BatchTaskID {
	return BatchTaskID{
		model.ObjectID{
			Prefix:  BatchTaskIDPrefix,
			IsValid: false,
		},
	}
}

// ParseBatchTaskID transforms an BatchTaskID from it's string representation into the actual ID value
func ParseBatchTaskID(s string) (BatchTaskID, error) {
	id := EmptyBatchTaskID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// BatchTaskID is the ID for a BatchTaskID object
type BatchTaskID struct {
	model.ObjectID
}

// BatchJobStatus represents the type associated with the status column of the batch_job table
type BatchJobStatus string

const (
	// BatchJobStatusPending represents the PENDING state of the status field on a batch_job record
	BatchJobStatusPending BatchJobStatus = "PENDING"
	// BatchJobStatusComplete represents the COMPLETE state of the status field on a batch_job record
	BatchJobStatusComplete BatchJobStatus = "COMPLETE"
)

// ParseBatchJobStatus converts a string into the correcponding enum value
func ParseBatchJobStatus(s string) (BatchJobStatus, error) {
	switch t := BatchJobStatus(strings.ToUpper(s)); t {
	case BatchJobStatusPending, BatchJobStatusComplete:
		return t, nil
	}
	return BatchJobStatus(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t BatchJobStatus) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t BatchJobStatus) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of BatchJobStatus from a database conforming to the sql.Scanner interface
func (t *BatchJobStatus) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseBatchJobStatus(ts)
	case []byte:
		*t, err = ParseBatchJobStatus(string(ts))
	}
	return errors.Trace(err)
}

// BatchJobType represents the type associated with the type column of the batch_job table
type BatchJobType string

const (
	// BatchJobTypeBatchPostMessages represents the BATCH_POST_MESSAGES state of the type field on a batch_job record
	BatchJobTypeBatchPostMessages BatchJobType = "BATCH_POST_MESSAGES"
)

// ParseBatchJobType converts a string into the correcponding enum value
func ParseBatchJobType(s string) (BatchJobType, error) {
	switch t := BatchJobType(strings.ToUpper(s)); t {
	case BatchJobTypeBatchPostMessages:
		return t, nil
	}
	return BatchJobType(""), errors.Trace(fmt.Errorf("Unknown type:%s", s))
}

func (t BatchJobType) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t BatchJobType) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of BatchJobType from a database conforming to the sql.Scanner interface
func (t *BatchJobType) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseBatchJobType(ts)
	case []byte:
		*t, err = ParseBatchJobType(string(ts))
	}
	return errors.Trace(err)
}

// BatchTaskType represents the type associated with the type column of the batch_task table
type BatchTaskType string

const (
	// BatchTaskTypePostMessages represents the POST_MESSAGES state of the type field on a batch_task record
	BatchTaskTypePostMessages BatchTaskType = "POST_MESSAGES"
)

// ParseBatchTaskType converts a string into the correcponding enum value
func ParseBatchTaskType(s string) (BatchTaskType, error) {
	switch t := BatchTaskType(strings.ToUpper(s)); t {
	case BatchTaskTypePostMessages:
		return t, nil
	}
	return BatchTaskType(""), errors.Trace(fmt.Errorf("Unknown type:%s", s))
}

func (t BatchTaskType) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t BatchTaskType) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of BatchTaskType from a database conforming to the sql.Scanner interface
func (t *BatchTaskType) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseBatchTaskType(ts)
	case []byte:
		*t, err = ParseBatchTaskType(string(ts))
	}
	return errors.Trace(err)
}

// BatchTaskStatus represents the type associated with the status column of the batch_task table
type BatchTaskStatus string

const (
	// BatchTaskStatusPending represents the PENDING state of the status field on a batch_task record
	BatchTaskStatusPending BatchTaskStatus = "PENDING"
	// BatchTaskStatusComplete represents the COMPLETE state of the status field on a batch_task record
	BatchTaskStatusComplete BatchTaskStatus = "COMPLETE"
	// BatchTaskStatusError represents the ERROR state of the status field on a batch_task record
	BatchTaskStatusError BatchTaskStatus = "ERROR"
)

// ParseBatchTaskStatus converts a string into the correcponding enum value
func ParseBatchTaskStatus(s string) (BatchTaskStatus, error) {
	switch t := BatchTaskStatus(strings.ToUpper(s)); t {
	case BatchTaskStatusPending, BatchTaskStatusComplete, BatchTaskStatusError:
		return t, nil
	}
	return BatchTaskStatus(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t BatchTaskStatus) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t BatchTaskStatus) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of BatchTaskStatus from a database conforming to the sql.Scanner interface
func (t *BatchTaskStatus) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseBatchTaskStatus(ts)
	case []byte:
		*t, err = ParseBatchTaskStatus(string(ts))
	}
	return errors.Trace(err)
}

// BatchTask represents a batch_task record
type BatchTask struct {
	Status         BatchTaskStatus
	Data           []byte
	Error          string
	Completed      *time.Time
	Created        time.Time
	Modified       time.Time
	ID             BatchTaskID
	BatchJobID     BatchJobID
	Type           BatchTaskType
	AvailableAfter time.Time
}

// BatchTaskUpdate represents the mutable aspects of a batch_task record
type BatchTaskUpdate struct {
	Status         *BatchTaskStatus
	Error          *string
	Completed      *time.Time
	AvailableAfter *time.Time
}

// BatchJob represents a batch_job record
type BatchJob struct {
	Completed        *time.Time
	ID               BatchJobID
	Status           BatchJobStatus
	TasksRequested   uint64
	TasksErrored     uint64
	Type             BatchJobType
	TasksCompleted   uint64
	Created          time.Time
	Modified         time.Time
	RequestingEntity string
}

// BatchJobUpdate represents the mutable aspects of a batch_job record
type BatchJobUpdate struct {
	TasksCompleted *uint64
	Status         *BatchJobStatus
	TasksRequested *uint64
	TasksErrored   *uint64
	Completed      *time.Time
}
