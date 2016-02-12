package bml

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

// ErrValidation is returned for errors during validation
type ErrValidation struct {
	Element string
	Reason  string
}

func (e ErrValidation) Error() string {
	return fmt.Sprintf("bml: invalid %s: %s", e.Element, e.Reason)
}

var elementTypes = map[string]reflect.Type{
	"ref": reflect.TypeOf(Ref{}),
}

// Validator is implemented by elements to provide validation of requirements
type Validator interface {
	Validate() error
}

// RefType is a type of a reference element
type RefType string

const (
	// EntityRef is an entity as owned by the directory service
	EntityRef RefType = "entity"
)

// Ref is a reference to a node type (object with an ID)
type Ref struct {
	XMLName string  `xml:"ref"`
	ID      string  `xml:"id,attr"`
	Type    RefType `xml:"type,attr"`
	Text    string  `xml:",chardata"`
}

// Anchor tag links to a URL
type Anchor struct {
	XMLName string `xml:"a"`
	HREF    string `xml:"href,attr"`
	Text    string `xml:",chardata"`
}

// Validate implements the Validator interface
func (r *Ref) Validate() error {
	// TODO: this should happen during unmarshal but for now it's fine here since
	// it adds less complexity and validate is always called before marshal.
	r.Type = RefType(strings.ToLower(string(r.Type)))
	if r.ID == "" {
		return ErrValidation{Element: "ref", Reason: "id required"}
	}
	if r.Type == "" {
		return ErrValidation{Element: "ref", Reason: "type required"}
	}
	if r.Text == "" {
		return ErrValidation{Element: "ref", Reason: "text required"}
	}
	return nil
}

// PlainText implements the PlainTexter interface
func (r *Ref) PlainText() (string, error) {
	return r.Text, nil
}

// Validate implements the Validator interface
func (r *Anchor) Validate() error {
	if r.HREF == "" {
		return ErrValidation{Element: "a", Reason: "href required"}
	}
	u, err := url.Parse(r.HREF)
	if err != nil {
		return ErrValidation{Element: "a", Reason: "href is not a valid URL"}
	}
	// TODO: this should happen during unmarshal but for now it's fine here since
	// it adds less complexity and validate is always called before marshal.
	r.HREF = u.String()
	return nil
}

// PlainText implements the PlainTexter interface
func (r *Anchor) PlainText() (string, error) {
	return r.Text, nil
}
