package bml

import (
	"errors"
	"reflect"
	"strings"
)

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

// Validate implements the Validator interface
func (r *Ref) Validate() error {
	// TODO: this should happen during unmarshal but for now it's fine here since
	// it adds less complexity and validate is always called before marshal.
	r.Type = RefType(strings.ToLower(string(r.Type)))
	if r.ID == "" {
		return errors.New("bml: Ref requires ID")
	}
	if r.Type == "" {
		return errors.New("bml: Ref requires Type")
	}
	if r.Text == "" {
		return errors.New("bml: Ref requires Text")
	}
	return nil
}
