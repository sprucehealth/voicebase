package main

/*
Notes about GraphQL package compatibility:
- can't use custom types for things like `type someEnum string` so just use `string`
*/

const (
	organizationIDType     = "organization"
	savedThreadQueryIDType = "saved_thread_query"
	threadIDType           = "thread"
)

const (
	messageStatusNormal  = "NORMAL"
	messageStatusDeleted = "DELETED"
)

const (
	contactTypeApp   = "APP"
	contactTypePhone = "PHONE"
	contactTypeEmail = "EMAIL"
)

const (
	endpointChannelApp   = "APP"
	endpointChannelSMS   = "SMS"
	endpointChannelVoice = "VOICE"
	endpointChannelEmail = "EMAIL"
)

type me struct {
	Account             *account `json:"account"`
	ClientEncryptionKey string   `json:"clientEncryptionKey"`
}

type account struct {
	ID string `json:"id"`
	// Entity        *Entity         `json:"entity"`
	// Organizations []*Organization `json:"organizations"`
}

type entity struct {
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
	Contacts      []*contactInfo `json:"contacts"`
	// TODO avatar(width: Int = 120, height: Int = 120, crop: Boolean = true): Image
}

type contactInfo struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Value        string `json:"value"`
	DisplayValue string `json:"displayValue"`
	Provisioned  bool   `json:"provisioned"`
	Label        string `json:"label"`
}

type endpoint struct {
	Channel      string `json:"channel"`
	ID           string `json:"id"`
	DisplayValue string `json:"displayValue"`
}

const (
	entityRef = "entity"
)

type reference struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type message struct {
	ThreadItemID    string
	TitleMarkup     string        `json:"titleMarkup"`
	TextMarkup      string        `json:"textMarkup"`
	Status          string        `json:"status"`
	Source          *endpoint     `json:"source"`
	Destinations    []*endpoint   `json:"destinations,omitempty"`
	Attachments     []*attachment `json:"attachments,omitempty"`
	EditorEntityID  string        `json:"editorEntityID,omitempty"`
	EditedTimestamp uint64        `json:"editedTimestamp,omitempty"`
	Refs            []*reference  `json:"refs,omitempty"`
}

type attachment struct {
	Title string      `json:"title"`
	URL   string      `json:"url"`
	Data  interface{} `json:"data"`
}

type imageAttachment struct {
	Mimetype string `json:"mimetype"`
	URL      string `json:"url"`
	Image    *image `json:"image"`
}

type image struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type audioAttachment struct {
	Mimetype          string  `json:"mimetype"`
	URL               string  `json:"url"`
	DurationInSeconds float64 `json:"durationInSeconds"`
}

type thread struct {
	ID                         string `json:"id"`
	OrganizationID             string `json:"organizationID"`
	PrimaryEntityID            string `json:"primaryEntityID"`
	Title                      string `json:"title"`
	Subtitle                   string `json:"subtitle"`
	LastMessageTimestamp       uint64 `json:"lastMessageTimestamp"`
	Unread                     bool   `json:"unread"`
	AllowInternalMessages      bool   `json:"allowInternalMessages"`
	LastPrimaryEntityEndpoints []*endpoint
}

type threadItem struct {
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

type threadItemViewDetails struct {
	ThreadItemID  string `json:"threadItemID"`
	ActorEntityID string `json:"actorEntityID"`
	ViewTime      uint64 `json:"viewTime"`
}

type serializedEntityContact struct {
	SerializedContact string `json:"serializedContact"`
}

type savedThreadQuery struct {
	ID             string `json:"id"`
	OrganizationID string `json:"id"`
}

type organization struct {
	ID       string         `json:"id"`
	Entity   *entity        `json:"entity"`
	Name     string         `json:"name"`
	Contacts []*contactInfo `json:"contacts"`
}

type subdomain struct {
	Available bool `json:"available"`
}

// settings

type stringListSetting struct {
	Key         string                  `json:"key"`
	Subkey      string                  `json:"subkey,omitempty"`
	Title       string                  `json:"title"`
	Description string                  `json:"description"`
	Value       *stringListSettingValue `json:"value"`
}

type booleanSetting struct {
	Key         string               `json:"key"`
	Subkey      string               `json:"subkey,omitempty"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Value       *booleanSettingValue `json:"value"`
}

type selectableItem struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	AllowFreeText bool   `json:"allowFreeText"`
}

type selectSetting struct {
	Key         string                  `json:"key"`
	Subkey      string                  `json:"subkey,omitempty"`
	Title       string                  `json:"title"`
	Description string                  `json:"description"`
	Options     []*selectableItem       `json:"options"`
	Value       *selectableSettingValue `json:"value"`
}

// setting values

type stringListSettingValue struct {
	Values []string `json:"list"`
	Key    string   `json:"key"`
	Subkey string   `json:"subkey,omitempty"`
}

type booleanSettingValue struct {
	Value  bool   `json:"set"`
	Key    string `json:"key"`
	Subkey string `json:"subkey,omitempty"`
}

type selectableItemValue struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}
type selectableSettingValue struct {
	Items  []*selectableItemValue `json:"items"`
	Key    string                 `json:"key"`
	Subkey string                 `json:"subkey,omitempty"`
}

// force upgrade status

type forceUpgradeStatus struct {
	URL         string `json:"url"`
	Upgrade     bool   `json:"upgrade"`
	UserMessage string `json:"userMessage"`
}
