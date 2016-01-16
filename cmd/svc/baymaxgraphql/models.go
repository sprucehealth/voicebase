package main

/*
Notes about GraphQL package compatibility:
- can't use custom types for things like `type someEnum string` so just use `string`
*/

const (
	accountIDType          = "account"
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

type account struct {
	ID string `json:"id"`
	// Entity        *Entity         `json:"entity"`
	// Organizations []*Organization `json:"organizations"`
}

type entity struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Contacts []*contactInfo `json:"contacts"`
	// TODO avatar(width: Int = 120, height: Int = 120, crop: Boolean = true): Image
}

type contactInfo struct {
	Type        string `json:"type"`
	Value       string `json:"value"`
	Provisioned bool   `json:"provisioned"`
}

type endpoint struct {
	Channel string `json:"channel"`
	ID      string `json:"id"`
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
	Title           string        `json:"title"`
	Text            string        `json:"text"`
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
	Mimetype          string `json:"mimetype"`
	URL               string `json:"url"`
	DurationInSeconds int    `json:"durationInSeconds"`
}

type thread struct {
	ID                   string `json:"id"`
	OrganizationID       string `json:"organizationID"`
	PrimaryEntityID      string `json:"primaryEntityID"`
	Title                string `json:"title"`
	Subtitle             string `json:"subtitle"`
	LastMessageTimestamp uint64 `json:"lastMessageTimestamp"`
	Unread               bool   `json:"unread"`
}

type threadItem struct {
	ID            string      `json:"id"`
	UUID          string      `json:"uuid"`
	Timestamp     uint64      `json:"timestamp"`
	ActorEntityID string      `json:"actorEntityID"`
	Internal      bool        `json:"internal"`
	Type          string      `json:"type"`
	Data          interface{} `json:"data"`
}

type threadItemViewDetails struct {
	ThreadItemID  string `json:"threadItemID"`
	ActorEntityID string `json:"actorEntityID"`
	ViewTime      uint64 `json:"viewTime"`
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
