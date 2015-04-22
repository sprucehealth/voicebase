package mandrill

import "fmt"

type Message struct {
	HTML                    string              `json:"html,omitempty"`
	Text                    string              `json:"text, omitempty"`
	Subject                 string              `json:"subject,omitempty"`
	FromEmail               string              `json:"from_email,omitempty"`
	FromName                string              `json:"from_name,omitempty"`
	To                      []*Recipient        `json:"to"`
	Headers                 map[string]string   `json:"headers,omitempty"`
	Important               bool                `json:"important"`
	TrackClicks             *bool               `json:"track_clicks,omitempty"`
	TrackOpens              *bool               `json:"track_opens,omitempty"`
	AutoHTML                *bool               `json:"auto_html,omitempty"`
	AutoText                *bool               `json:"auto_text,omitempty"`
	InlineCSS               *bool               `json:"inline_css,omitempty"`
	URLStripQs              *bool               `json:"url_strip_qs,omitempty"`
	PreserveRecipients      *bool               `json:"preserve_recipients,omitempty"`
	ViewContentLink         *bool               `json:"view_content_link,omitempty"`
	BCCAddress              string              `json:"bcc_address,omitempty"`
	TrackingDomain          string              `json:"tracking_domain,omitempty"`
	SigningDomain           string              `json:"signing_domain,omitempty"`
	ReturnPathDomain        string              `json:"return_path_domain,omitempty"`
	Merge                   bool                `json:"merge"`
	MergeLanguage           string              `json:"merge_language,omitempty"`
	GlobalMergeVars         []Var               `json:"global_merge_vars,omitempty"`
	MergeVars               []MergeVar          `json:"merge_vars,omitempty"`
	Tags                    []string            `json:"tags,omitempty"`
	Subaccount              string              `json:"subaccount,omitempty"`
	GoogleAnalyticsDomains  []string            `json:"google_analytics_domains,omitempty"`
	GoogleAnalyticsCampaign string              `json:"google_analytics_campaign,omitempty"`
	Metadata                map[string]string   `json:"metadata,omitempty"`
	RecipientMetadata       []RecipientMetadata `json:"recipient_metadata,omitempty"`
	Attachments             []*Attachment       `json:"attachments,omitempty"`
	Images                  []*Attachment       `json:"images,omitempty"`
}

type Attachment struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content []byte `json:"content"`
}

type Var struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type MergeVar struct {
	Rcpt string `json:"rcpt"`
	Vars []Var  `json:"vars"`
}

type Recipient struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Type  string `json:"type,omitempty"`
}

type RecipientMetadata struct {
	Rcpt   string            `json:"rcpt"`
	Values map[string]string `json:"values"`
}

// Responses

type SendMessageResponse struct {
	ID           string `json:"_id"`
	Status       string `json:"status"`
	Email        string `json:"email"`
	RejectReason string `json:"reject_reason"`
}

type Error struct {
	Status  string `json:"status"`
	Code    int    `json:"code"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("mandrill: api error %s: %s", e.Name, e.Message)
}
