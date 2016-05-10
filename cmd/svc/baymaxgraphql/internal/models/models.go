package models

import (
	"sync"

	"github.com/sprucehealth/backend/svc/directory"
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
	Account             Account `json:"account"`
	ClientEncryptionKey string  `json:"clientEncryptionKey"`
}

type AccountType string

const (
	AccountTypePatient  AccountType = "PATIENT"
	AccountTypeProvider AccountType = "PROVIDER"
)

type Account interface {
	// This method is unfortunatly named, but don't want to cover the exported ID
	GetID() string
	Type() AccountType
}

// ProviderAccount represents the information associated with a provider's account
type ProviderAccount struct {
	ID string `json:"id"`
}

func (a *ProviderAccount) GetID() string {
	return a.ID
}

func (a *ProviderAccount) Type() AccountType {
	return AccountTypeProvider
}

// PatientAccount represents the information associated with a patient's account
type PatientAccount struct {
	ID string `json:"id"`
}

func (a *PatientAccount) GetID() string {
	return a.ID
}

func (a *PatientAccount) Type() AccountType {
	return AccountTypePatient
}

type DOB struct {
	Month int `json:"month"`
	Day   int `json:"day"`
	Year  int `json:"year"`
}

type Entity struct {
	ID                    string         `json:"id"`
	IsEditable            bool           `json:"isEditable"`
	FirstName             string         `json:"firstName"`
	MiddleInitial         string         `json:"middleInitial"`
	LastName              string         `json:"lastName"`
	GroupName             string         `json:"groupName"`
	DisplayName           string         `json:"displayName"`
	ShortTitle            string         `json:"shortTitle"`
	LongTitle             string         `json:"longTitle"`
	Gender                string         `json:"gender"`
	DOB                   *DOB           `json:"dob"`
	Note                  string         `json:"note"`
	Contacts              []*ContactInfo `json:"contacts"`
	IsInternal            bool           `json:"isInternal"`
	LastModifiedTimestamp uint64         `json:"lastModifiedTimestamp"`
	HasAccount            bool           `json:"hasAccount"`
	AllowEdit             bool           `json:"allowEdit"`
	HasPendingInvite      bool           `json:"hasPendingInvite"`
	Avatar                *Image
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

type VerifiedEntityInfo struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
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

type BannerButtonAttachment struct {
	Title   string `json:"title"`
	CTAText string `json:"ctaText"`
	IconURL string `json:"iconURL"`
	TapURL  string `json:"tapURL"`
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

const (
	ThreadTypeUnknown        = "UNKNOWN" // TODO: remove this once old threads are migrated
	ThreadTypeExternal       = "EXTERNAL"
	ThreadTypeTeam           = "TEAM"
	ThreadTypeSetup          = "SETUP"
	ThreadTypeSupport        = "SUPPORT"
	ThreadTypeLegacyTeam     = "LEGACY_TEAM"
	ThreadTypeSecureExternal = "SECURE_EXTERNAL"
)

const (
	ThreadTypeIndicatorNone  = "NONE"
	ThreadTypeIndicatorLock  = "LOCK"
	ThreadTypeIndicatorGroup = "GROUP"
)

type Thread struct {
	ID                         string `json:"id"`
	OrganizationID             string `json:"organizationID"`
	PrimaryEntityID            string `json:"primaryEntityID"`
	Title                      string `json:"title"`
	Subtitle                   string `json:"subtitle"`
	LastMessageTimestamp       uint64 `json:"lastMessageTimestamp"`
	Unread                     bool   `json:"unread"`
	UnreadReference            bool   `json:"unreadReference"`
	AllowAddMembers            bool   `json:"allowAddMembers"`
	AllowCarePlanAttachments   bool   `json:"allowCarePlanAttachments"`
	AllowDelete                bool   `json:"allowDelete"`
	AllowEmailAttachment       bool   `json:"allowEmailAttachments"`
	AllowExternalDelivery      bool   `json:"allowExternalDelivery"`
	AllowInternalMessages      bool   `json:"allowInternalMessages"`
	AllowLeave                 bool   `json:"allowLeave"`
	AllowMentions              bool   `json:"allowMentions"`
	AllowRemoveMembers         bool   `json:"allowRemoveMembers"`
	AllowSMSAttachments        bool   `json:"allowSMSAttachments"`
	AllowUpdateTitle           bool   `json:"allowUpdateTitle"`
	AllowVisitAttachment       bool   `json:"allowVisitAttachments"`
	LastPrimaryEntityEndpoints []*Endpoint
	EmptyStateTextMarkup       string `json:"emptyStateTextMarkup,omitempty"`
	MessageCount               int    `json:"messageCount"`
	Type                       string `json:"type"`
	TypeIndicator              string `json:"typeIndicator"`

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
	OrganizationID string `json:"organizationID"`
}

type Organization struct {
	ID                     string         `json:"id"`
	Entity                 *Entity        `json:"entity"`
	Name                   string         `json:"name"`
	Contacts               []*ContactInfo `json:"contacts"`
	AllowTeamConversations bool           `json:"allowTeamConversations"`
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
	Key                     string                  `json:"key"`
	Subkey                  string                  `json:"subkey,omitempty"`
	Title                   string                  `json:"title"`
	Description             string                  `json:"description"`
	Options                 []*SelectableItem       `json:"options"`
	Value                   *SelectableSettingValue `json:"value"`
	AllowsMultipleSelection bool                    `json:"allowsMultipleSelection"`
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

// visits

type VisitCategory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type VisitLayout struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type VisitLayoutVersion struct {
	ID            string `json:"id"`
	SAMLLayout    string `json:"samlLayout"`
	LayoutPreview string `json:"layoutPreview"`
}

type Visit struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	CanReview           bool   `json:"canReview"`
	CanModify           bool   `json:"canModify"`
	LayoutContainer     string `json:"layoutContainer"`
	LayoutContainerType string `json:"layoutContainerType"`

	EntityID string `json:"-"`
}

type CarePlan struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	Treatments         []*CarePlanTreatment   `json:"treatments"`
	Instructions       []*CarePlanInstruction `json:"instructions"`
	CreatedTimestamp   uint64                 `json:"createdTimestamp"`
	Submitted          bool                   `json:"submitted"`
	SubmittedTimestamp uint64                 `json:"submittedTimestamp,omitempty"`
	ParentID           string                 `json:"parentID,omitempty"`
	CreatorID          string                 `json:"creatorID,omitempty"`
}

type CarePlanTreatment struct {
	EPrescribe           bool   `json:"ePrescribe"`
	Name                 string `json:"name"`
	Form                 string `json:"form"`
	Route                string `json:"route"`
	Availability         string `json:"availability"`
	Dosage               string `json:"dosage"`
	DispenseType         string `json:"dispenseType"`
	DispenseNumber       int    `json:"dispenseNumber"`
	Refills              int    `json:"refills"`
	SubstitutionsAllowed bool   `json:"substitutionsAllowed"`
	DaysSupply           int    `json:"daysSupply"`
	Sig                  string `json:"sig"`
	PharmacyInstructions string `json:"pharmacyInstructions"`
}

type CarePlanInstruction struct {
	Title string   `json:"title"`
	Steps []string `json:"steps"`
}

const (
	TreatmentAvailabilityUnknown = "UNKNOWN"
	TreatmentAvailabilityOTC     = "OTC"
	TreatmentAvailabilityRx      = "RX"
)

type Pharmacy struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Address         *Address `json:"address"`
	PhoneNumber     string   `json:"phoneNumber"`
	Retail          bool     `json:"retail"`
	TwentyFourHours bool     `json:"twentyFourHours"`
	Specialty       bool     `json:"specialty"`
	MailOrder       bool     `json:"mailOrder"`
}

type Address struct {
	Address1  string `json:"address1"`
	Address2  string `json:"address2"`
	City      string `json:"city"`
	StateCode string `json:"stateCode"`
	Country   string `json:"country"`
	ZipCode   string `json:"zipCode"`
}

type Medication struct {
	ID      string              `json:"id"`
	Name    string              `json:"name"`
	Route   string              `json:"route"`
	Form    string              `json:"form"`
	Dosages []*MedicationDosage `json:"dosages"`
}

type MedicationDosage struct {
	Dosage       string `json:"dosage"`
	DispenseType string `json:"dispenseType"`
	OTC          bool   `json:"otc"`
}
