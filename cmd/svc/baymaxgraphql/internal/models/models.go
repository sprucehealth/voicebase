package models

import (
	"github.com/sprucehealth/backend/svc/directory"
	"sync"
)

/*
Notes about GraphQL package compatibility:
- can't use custom types for things like `type someEnum string` so just use `string`
*/

const (
	OrganizationIDType     = "organization"
	SavedThreadQueryIDType = "saved_thread_query"
	ThreadIDType           = "thread"
)

const (
	MessageStatusNormal  = "NORMAL"
	MessageStatusDeleted = "DELETED"
)

const (
	ContactTypeApp   = "APP"
	ContactTypePhone = "PHONE"
	ContactTypeEmail = "EMAIL"
)

const (
	EndpointChannelApp   = "APP"
	EndpointChannelSMS   = "SMS"
	EndpointChannelVoice = "VOICE"
	EndpointChannelEmail = "EMAIL"
)

type Me struct {
	Account             *Account `json:"account"`
	ClientEncryptionKey string   `json:"clientEncryptionKey"`
}

type Account struct {
	ID string `json:"id"`
}

type Entity struct {
	ID            string         `json:"id"`
	IsEditable    bool           `json:"isEditable"`
	FirstName     string         `json:"firstName"`
	MiddleInitial string         `json:"middleInitial"`
	LastName      string         `json:"lastName"`
	GroupName     string         `json:"groupName"`
	DisplayName   string         `json:"displayName"`
	ShortTitle    string         `json:"shortTitle"`
	LongTitle     string         `json:"longTitle"`
	Note          string         `json:"note"`
	Contacts      []*ContactInfo `json:"contacts"`
	IsInternal    bool           `json:"isInternal"`

	Avatar *Image
}

type ContactInfo struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Value        string `json:"value"`
	DisplayValue string `json:"displayValue"`
	Provisioned  bool   `json:"provisioned"`
	Label        string `json:"label"`
}

type Endpoint struct {
	Channel      string `json:"channel"`
	ID           string `json:"id"`
	DisplayValue string `json:"displayValue"`
}

const (
	EntityRef = "entity"
)

type Reference struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type Message struct {
	ThreadItemID    string
	SummaryMarkup   string        `json:"summaryMarkup"`
	TextMarkup      string        `json:"textMarkup"`
	Status          string        `json:"status"`
	Source          *Endpoint     `json:"source"`
	Destinations    []*Endpoint   `json:"destinations,omitempty"`
	Attachments     []*Attachment `json:"attachments,omitempty"`
	EditorEntityID  string        `json:"editorEntityID,omitempty"`
	EditedTimestamp uint64        `json:"editedTimestamp,omitempty"`
	Refs            []*Reference  `json:"refs,omitempty"`
}

type Attachment struct {
	Title string      `json:"title"`
	URL   string      `json:"url"`
	Data  interface{} `json:"data"`
}

type ImageAttachment struct {
	Mimetype string `json:"mimetype"`
	URL      string `json:"url"`
	Image    *Image `json:"image"`
}

type Image struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type AudioAttachment struct {
	Mimetype          string  `json:"mimetype"`
	URL               string  `json:"url"`
	DurationInSeconds float64 `json:"durationInSeconds"`
}

type Thread struct {
	ID                         string `json:"id"`
	OrganizationID             string `json:"organizationID"`
	PrimaryEntityID            string `json:"primaryEntityID"`
	Title                      string `json:"title"`
	Subtitle                   string `json:"subtitle"`
	LastMessageTimestamp       uint64 `json:"lastMessageTimestamp"`
	Unread                     bool   `json:"unread"`
	AllowInternalMessages      bool   `json:"allowInternalMessages"`
	IsDeletable                bool   `json:"isDeletable"`
	LastPrimaryEntityEndpoints []*Endpoint
	EmptyStateTextMarkup       string `json:"emptyStateTextMarkup,omitempty"`
	MessageCount               int    `json:"messageCount"`

	Mu            sync.RWMutex
	PrimaryEntity *directory.Entity
}

type ThreadItem struct {
	ID             string      `json:"id"`
	UUID           string      `json:"uuid,omitempty"`
	Timestamp      uint64      `json:"timestamp"`
	ActorEntityID  string      `json:"actorEntityID"`
	Internal       bool        `json:"internal"`
	Type           string      `json:"type"`
	Data           interface{} `json:"data"`
	OrganizationID string      `json:"organizationID"`
	ThreadID       string      `json:"threadID"`
}

type ThreadItemViewDetails struct {
	ThreadItemID  string `json:"threadItemID"`
	ActorEntityID string `json:"actorEntityID"`
	ViewTime      uint64 `json:"viewTime"`
}

type SerializedEntityContact struct {
	SerializedContact string `json:"serializedContact"`
}

type SavedThreadQuery struct {
	ID             string `json:"id"`
	OrganizationID string `json:"id"`
}

type Organization struct {
	ID       string         `json:"id"`
	Entity   *Entity        `json:"entity"`
	Name     string         `json:"name"`
	Contacts []*ContactInfo `json:"contacts"`
}

type Subdomain struct {
	Available bool `json:"available"`
}

// settings

type StringListSetting struct {
	Key         string                  `json:"key"`
	Subkey      string                  `json:"subkey,omitempty"`
	Title       string                  `json:"title"`
	Description string                  `json:"description"`
	Value       *StringListSettingValue `json:"value"`
}

type BooleanSetting struct {
	Key         string               `json:"key"`
	Subkey      string               `json:"subkey,omitempty"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Value       *BooleanSettingValue `json:"value"`
}

type SelectableItem struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	AllowFreeText bool   `json:"allowFreeText"`
}

type SelectSetting struct {
	Key         string                  `json:"key"`
	Subkey      string                  `json:"subkey,omitempty"`
	Title       string                  `json:"title"`
	Description string                  `json:"description"`
	Options     []*SelectableItem       `json:"options"`
	Value       *SelectableSettingValue `json:"value"`
}

// setting values

type StringListSettingValue struct {
	Values []string `json:"list"`
	Key    string   `json:"key"`
	Subkey string   `json:"subkey,omitempty"`
}

type BooleanSettingValue struct {
	Value  bool   `json:"set"`
	Key    string `json:"key"`
	Subkey string `json:"subkey,omitempty"`
}

type SelectableItemValue struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}
type SelectableSettingValue struct {
	Items  []*SelectableItemValue `json:"items"`
	Key    string                 `json:"key"`
	Subkey string                 `json:"subkey,omitempty"`
}

// force upgrade status

type ForceUpgradeStatus struct {
	URL         string `json:"url"`
	Upgrade     bool   `json:"upgrade"`
	UserMessage string `json:"userMessage"`
}
