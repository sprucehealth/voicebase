package twiml

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
)

const (
	DoNotRecord       = "do-not-record"
	RecordFromAnswer  = "record-from-answer"
	RecordFromRinging = "record-from-ringing"
	TrimSilence       = "trim-silence"
	DoNotTrim         = "do-not-trim"
)

type Validator interface {
	Validate() error
}

type Response struct {
	XMLName xml.Name      `xml:"Response"`
	Verbs   []interface{} `xml:""`
}

func NewResponse(verbs ...interface{}) *Response {
	return &Response{
		Verbs: verbs,
	}
}

func (t *Response) GenerateTwiML() (string, error) {

	for _, v := range t.Verbs {
		if va, ok := v.(Validator); ok {
			if err := va.Validate(); err != nil {
				return "", err
			}
		}

		switch s := v.(type) {
		case *Dial, *Gather, *Enqueue, *Hangup, *Leave, *Pause, *Play, *Record, *Redirect, *Reject, *Say, *SMS:
		default:
			return "", fmt.Errorf("invalid verb '%T'", s)
		}
	}

	data, err := xml.Marshal(t)
	if err != nil {
		return "", err
	}

	return xml.Header + string(data), nil
}

func (t *Response) WriteResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "text/xml")
	str, err := t.GenerateTwiML()
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(str))
	return err
}

type Dial struct {
	XMLName            xml.Name `xml:"Dial"`
	Action             string   `xml:"action,attr,omitempty"`
	Method             string   `xml:"method,attr,omitempty"`
	TimeoutInSeconds   uint     `xml:"timeout,attr,omitempty"`
	HangupOnStar       bool     `xml:"hangupOnStar,attr,omitempty"`
	TimeLimitInSeconds uint     `xml:"timeLimit,attr,omitempty"`
	CallerID           string   `xml:"callerId,attr,omitempty"`
	Record             string   `xml:"record,attr,omitempty"`
	Trim               string   `xml:"trim,attr,omitempty"`
	Nouns              []interface{}
	PlainText          string `xml:",chardata"`
}

func (d *Dial) Validate() error {
	_, err := url.Parse(d.Action)
	if err != nil {
		return fmt.Errorf("action is not a valid url: %s", err.Error())
	}

	if d.Method != "GET" && d.Method != "POST" && d.Method != "" {
		return fmt.Errorf("method can only be GET or POST")
	}

	switch d.Record {
	case "", DoNotRecord, RecordFromAnswer, RecordFromRinging:
	default:
		return fmt.Errorf("invalid value for record: %s", d.Record)
	}

	switch d.Trim {
	case TrimSilence, DoNotTrim, "":
	default:
		return fmt.Errorf("invalid value for trim: %s", d.Trim)
	}

	for _, n := range d.Nouns {

		switch t := n.(type) {
		default:
			return fmt.Errorf("invalid verb: '%T'", t)
		case *Client, *Conference, *Number, *Queue, *SIP:
		}

		if va, ok := n.(Validator); ok {
			if err := va.Validate(); err != nil {
				return err
			}
		}

	}

	return nil
}

type Number struct {
	XMLName              xml.Name            `xml:"Number"`
	Text                 string              `xml:",chardata"`
	SendDigits           string              `xml:"sendDigits,attr,omitempty"`
	URL                  string              `xml:"url,attr,omitempty"`
	Method               string              `xml:"method,attr,omitempty"`
	StatusCallbackEvent  StatusCallbackEvent `xml:"statusCallbackEvent,attr,omitempty"`
	StatusCallback       string              `xml:"statusCallback,attr,omitempty"`
	StatusCallbackMethod string              `xml:"statusCallbackMethod,attr,omitempty"`
}

func (n *Number) Validate() error {
	if n.URL != "" {
		_, err := url.Parse(n.URL)
		if err != nil {
			return fmt.Errorf("invalid url: %s", err.Error())
		}
	}

	if n.StatusCallback != "" {
		_, err := url.Parse(n.StatusCallback)
		if err != nil {
			return fmt.Errorf("invalid status callback url: %s", err.Error())
		}
	}

	if err := validateMethod(n.StatusCallbackMethod); err != nil {
		return err
	}

	return nil
}

type Client struct {
	XMLName              xml.Name            `xml:"Client"`
	URL                  string              `xml:"url,attr,omitempty"`
	Method               string              `xml:"method,attr,omitempty"`
	StatusCallbackEvent  StatusCallbackEvent `xml:"statusCallbackEvent,attr,omitempty"`
	StatusCallback       string              `xml:"statusCallback,attr,omitempty"`
	StatusCallbackMethod string              `xml:"statusCallbackMethod,attr,omitempty"`
	Text                 string              `xml:",innerxml"`
}

func (n *Client) Validate() error {
	if n.URL != "" {
		_, err := url.Parse(n.URL)
		if err != nil {
			return fmt.Errorf("invalid url: %s", err.Error())
		}
	}

	if n.StatusCallback != "" {
		_, err := url.Parse(n.StatusCallback)
		if err != nil {
			return fmt.Errorf("invalid status callback url: %s", err.Error())
		}
	}

	if err := validateMethod(n.StatusCallbackMethod); err != nil {
		return err
	}

	return nil
}

type Queue struct {
	XMLName             xml.Name `xml:"Queue"`
	URL                 string   `xml:"url,attr,omitempty"`
	Method              string   `xml:"method,attr,omitempty"`
	ReservationSID      string   `xml:"reservationSid,attr,omitempty"`
	PostWorkActivitySID string   `xml:"postWorkActivitySid,attr,omitempty"`
	Text                string   `xml:",innerxml"`
}

func (n *Queue) Validate() error {
	if n.URL != "" {
		_, err := url.Parse(n.URL)
		if err != nil {
			return fmt.Errorf("invalid url: %s", err.Error())
		}
	}

	return nil
}

type Conference struct {
	XMLName                xml.Name `xml:"Conference"`
	Muted                  bool     `xml:"muted,attr,omitempty"`
	Beep                   bool     `xml:"beep,attr,omitempty"`
	StartConferenceOnEnter bool     `xml:"startConferenceOnEnter,attr,omitempty"`
	EndConferenceOnExit    bool     `xml:"endConferenceOnExit,attr,omitempty"`
	WaitURL                string   `xml:"waitUrl,attr,omitempty"`
	WaitMethod             string   `xml:"waitMethod,attr,omitempty"`
	MaxParticipants        uint     `xml:"maxParticipants,attr,omitempty"`
	Record                 string   `xml:"record,attr,omitempty"`
	Trim                   string   `xml:"trim,attr,omitempty"`
	EventCallbackURL       string   `xml:"eventCallbackUrl,attr,omitempty"`
	Text                   string   `xml:",innerxml"`
}

func (c *Conference) Validate() error {
	if c.WaitURL != "" {
		_, err := url.Parse(c.WaitURL)
		if err != nil {
			return err
		}
	}
	if c.EventCallbackURL != "" {
		_, err := url.Parse(c.EventCallbackURL)
		if err != nil {
			return err
		}
	}
	if c.MaxParticipants >= 250 {
		return fmt.Errorf("Cannot have more than 250 max participants.")
	}
	if err := validateMethod(c.WaitMethod); err != nil {
		return err
	}
	return nil
}

type SIP struct {
	XMLName              xml.Name            `xml:"Sip"`
	URL                  string              `xml:"url,attr,omitempty"`
	Method               string              `xml:"method,attr,omitempty"`
	StatusCallbackEvent  StatusCallbackEvent `xml:"statusCallbackEvent,attr,omitempty"`
	StatusCallback       string              `xml:"statusCallback,attr,omitempty"`
	StatusCallbackMethod string              `xml:"statusCallbackMethod,attr,omitempty"`
}

func (s *SIP) Validate() error {
	if s.URL != "" {
		_, err := url.Parse(s.URL)
		if err != nil {
			return err
		}
	}
	if s.StatusCallback != "" {
		_, err := url.Parse(s.StatusCallback)
		if err != nil {
			return err
		}
	}
	if err := validateMethod(s.StatusCallbackMethod); err != nil {
		return err
	}
	return nil
}

type Enqueue struct {
	XMLName       xml.Name `xml:"Enqueue"`
	Action        string   `xml:"action,attr,omitempty"`
	Method        string   `xml:"method,attr,omitempty"`
	WaitURL       string   `xml:"waitUrl,attr,omitempty"`
	WaitURLMethod string   `xml:"waiturlMethod,attr,omitempty"`
	WorkflowSID   string   `xml:"workflowSid,attr,omitempty"`
	Text          string   `xml:",innerxml"`
}

func (e *Enqueue) Validate() error {
	if e.WaitURL != "" {
		_, err := url.Parse(e.WaitURL)
		if err != nil {
			return err
		}
	}

	if e.Action != "" {
		_, err := url.Parse(e.Action)
		if err != nil {
			return err
		}
	}

	if err := validateMethod(e.WaitURLMethod); err != nil {
		return err
	}

	return nil
}

type Gather struct {
	XMLName          xml.Name `xml:"Gather"`
	Action           string   `xml:"action,attr,omitempty"`
	Method           string   `xml:"method,attr,omitempty"`
	TimeoutInSeconds uint     `xml:"timeout,attr,omitempty"`
	FinishOnKey      string   `xml:"finishOnKey,attr,omitempty"`
	NumDigits        uint     `xml:"numDigits,attr,omitempty"`
	Verbs            []interface{}
}

func (g *Gather) Validate() error {
	if g.Action != "" {
		_, err := url.Parse(g.Action)
		return err
	}
	if err := validateMethod(g.Method); err != nil {
		return err
	}

	for _, v := range g.Verbs {
		if va, ok := v.(Validator); ok {
			if err := va.Validate(); err != nil {
				return err
			}
		}

		switch t := v.(type) {
		case Say, Play, Pause:
		default:
			return fmt.Errorf("invalid verb '%T", t)
		}
	}

	return nil
}

type Hangup struct {
	XMLName xml.Name `xml:"Hangup"`
}

type Leave struct {
	XMLName xml.Name `xml:"Leave"`
}

type Pause struct {
	XMLName xml.Name `xml:"Pause"`
	Length  uint     `xml:"length,attr,omitempty"`
}

type Play struct {
	XMLName xml.Name `xml:"Play"`
	Loop    uint     `xml:"loop,attr,omitempty"`
	Digits  string   `xml:"digits,attr,omitempty"`
	Text    string   `xml:",innerxml"`
}

type Record struct {
	XMLName            xml.Name `xml:"Record"`
	Action             string   `xml:"action,attr,omitempty"`
	Method             string   `xml:"method,attr,omitempty"`
	TimeoutInSeconds   uint     `xml:"timeout,attr,omitempty"`
	FinishOnKey        string   `xml:"finishOnKey,attr,omitempty"`
	MaxLength          uint     `xml:"maxLength,attr,omitempty"`
	Transcribe         bool     `xml:"transcribe,attr,omitempty"`
	TranscribeCallback string   `xml:"transcribeCallback,attr,omitempty"`
	PlayBeep           bool     `xml:"playBeep,attr,omitempty"`
	Trim               string   `xml:"trim,attr,omitempty"`
}

func (r *Record) Validate() error {
	if r.Action != "" {
		_, err := url.Parse(r.Action)
		if err != nil {
			return err
		}
	}
	if r.TranscribeCallback != "" {
		_, err := url.Parse(r.TranscribeCallback)
		if err != nil {
			return err
		}
	}
	return nil
}

type Redirect struct {
	XMLName xml.Name `xml:"Redirect"`
	Method  string   `xml:"method,attr,omitempty"`
	Text    string   `xml:",chardata"`
}

func (r *Redirect) Validate() error {
	return validateMethod(r.Method)
}

type Reject struct {
	XMLName xml.Name `xml:"Reject"`
	Reason  string   `xml:"reason"`
}

func (r *Reject) Validate() error {
	switch r.Reason {
	case "", "rejected", "busy":
	default:
		return fmt.Errorf("invalid reason %s", r.Reason)
	}
	return nil
}

type Say struct {
	XMLName  xml.Name `xml:"Say"`
	Voice    string   `xml:"voice,attr,omitempty"`
	Loop     uint     `xml:"loop,attr,omitempty"`
	Language string   `xml:"language,attr,omitempty"`
	Text     string   `xml:",chardata"`
}

func (s *Say) Validate() error {
	switch s.Voice {
	case "":
	case "man", "woman":
		switch s.Language {
		case "", "en", "en-gb", "es", "fr", "de", "it":
		default:
			return fmt.Errorf("invalid language setting for man or woman: %s", s.Language)
		}
	case "alice":
		switch s.Language {
		case "", "da-DK", "de-DE", "en-AU", "en-CA", "en-GB", "en-IN", "en-US", "ca-ES", "es-ES", "es-MX", "fi-FI", "fr-CA", "fr-FR", "it-IT", "ja-JP", "ko-KR", "nb-NO", "nl-NL", "pl-PL", "pt-BR", "pt-PT", "ru-RU", "sv-SE", "zh-CN", "zh-HK", "zh-TW":
		default:
			return fmt.Errorf("invalid language setting for alice: %s", s.Language)
		}
	default:
		return fmt.Errorf("invalid voice option %s", s.Voice)
	}
	return nil
}

type SMS struct {
	XMLName        xml.Name `xml:"Sms"`
	To             string   `xml:"to,attr,omitempty"`
	From           string   `xml:"from,attr,omitempty"`
	Action         string   `xml:"action,attr,omitempty"`
	Method         string   `xml:"method,attr,omitempty"`
	StatusCallback string   `xml:"statusCallback,attr,omitempty"`
	Text           string   `xml:",chardata"`
}

func (s *SMS) Validate() error {
	if err := validateMethod(s.Method); err != nil {
		return err
	}
	if s.StatusCallback != "" {
		_, err := url.Parse(s.StatusCallback)
		if err != nil {
			return err
		}
	}
	if s.Action != "" {
		_, err := url.Parse(s.Action)
		if err != nil {
			return err
		}
	}
	return nil
}
