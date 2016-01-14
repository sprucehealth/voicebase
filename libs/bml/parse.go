package bml

import (
	"encoding/xml"
	"fmt"
	"io"
	"reflect"
	"strings"
)

// ErrParseFailure is returned for errors during parsing. Reason provides context for
// the cause of the failure without leaking internal details. Err, if not nil, is the
// underlying error.
type ErrParseFailure struct {
	Offset int64
	Reason string
	Err    error
}

func (e ErrParseFailure) Error() string {
	var es string
	if e.Err != nil {
		es = ": " + e.Err.Error()
	}
	return fmt.Sprintf("bml: parsing failed at pos %d: %s%s", e.Offset, e.Reason, es)
}

// Parse parses a BML string into its parts.
func Parse(s string) (BML, error) {
	d := xml.NewDecoder(strings.NewReader(s))
	var bml BML
	for {
		offset := d.InputOffset()
		t, err := d.Token()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, ErrParseFailure{Offset: offset, Reason: "invalid XML", Err: err}
		}
		switch v := t.(type) {
		case xml.CharData:
			bml = append(bml, string(v))
		case xml.StartElement:
			name := v.Name.Local
			et := elementTypes[name]
			if et == nil {
				return nil, ErrParseFailure{Offset: offset, Reason: "unsupported tag " + name}
			}
			nv := reflect.New(et).Interface()
			if err := d.DecodeElement(nv, &v); err != nil {
				return nil, ErrParseFailure{Offset: offset, Reason: "failed to decode element " + name, Err: err}
			}
			if vd, ok := nv.(Validator); ok {
				if err := vd.Validate(); err != nil {
					if err, ok := err.(ErrValidation); ok {
						return nil, ErrParseFailure{Offset: offset, Reason: fmt.Sprintf("invalid %s: %s", err.Element, err.Reason)}
					}
					return nil, ErrParseFailure{Offset: offset, Reason: "bad tag", Err: err}
				}
			}
			bml = append(bml, nv)
		default:
			return nil, ErrParseFailure{Offset: offset, Reason: "unsupported xml", Err: fmt.Errorf("unsupported token %T", t)}
		}
	}
	return bml, nil
}
