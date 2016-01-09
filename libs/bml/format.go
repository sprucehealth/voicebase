package bml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
)

var reReplacement = regexp.MustCompile(`%.`)

// BML is a slice of BML formattable elements
type BML []interface{}

// Validate returns an error iff any element of the BML is invalid.
func (bml BML) Validate() error {
	for _, v := range bml {
		switch v.(type) {
		case int, int64, uint64, string:
		default:
			if vd, ok := v.(Validator); ok {
				if err := vd.Validate(); err != nil {
					return err
				}
			} else if _, ok := v.(fmt.Stringer); !ok {
				return fmt.Errorf("bml: unsupported type %T", v)
			}
		}
	}
	return nil
}

// Format returns a formatted BML string by encoding and concatenating the provided parts
func (bml BML) Format() (string, error) {
	b := &bytes.Buffer{}
	e := xml.NewEncoder(b)
	for _, p := range bml {
		if err := encodeValue(e, p); err != nil {
			return "", err
		}
	}
	if err := e.Flush(); err != nil {
		return "", err
	}
	return b.String(), nil
}

// Sprintf formats according to a format specifier and returns the resulting BML string.
// NOTE: it does not currently handle advanced formatting specifiers. Only single letter
// specifiers such as %s are supported and they're all treated the same.
func Sprintf(format string, a ...interface{}) (string, error) {
	return Parsef(format, a...).Format()
}

// Parsef is similar to Sprintf but instead of encoding as a string it returns a
// slice of the BML elements. The returns BML is not validated.
func Parsef(format string, a ...interface{}) BML {
	var bml BML
	ixs := reReplacement.FindAllStringIndex(format, -1)
	k := 0
	for i, ix := range ixs {
		if s := format[k:ix[0]]; s != "" {
			bml = append(bml, s)
		}
		k = ix[1]
		switch format[ix[0]+1] {
		case '%':
			bml = append(bml, "%")
		default:
			// TODO: currently ignoring the type of the replacement and not supporting advanced formatting
			bml = append(bml, a[i])
		}
	}
	if s := format[k:]; s != "" {
		bml = append(bml, s)
	}
	return bml
}

func encodeValue(e *xml.Encoder, v interface{}) error {
	switch vt := v.(type) {
	case int:
		return e.EncodeToken(xml.CharData(strconv.Itoa(vt)))
	case int64:
		return e.EncodeToken(xml.CharData(strconv.FormatInt(vt, 10)))
	case uint64:
		return e.EncodeToken(xml.CharData(strconv.FormatUint(vt, 10)))
	case string:
		if vt == "" {
			return nil
		}
		return e.EncodeToken(xml.CharData(vt))
	}
	if vd, ok := v.(Validator); ok {
		if err := vd.Validate(); err != nil {
			return err
		}
		return e.Encode(v)
	}
	if s, ok := v.(fmt.Stringer); ok {
		return e.EncodeToken(xml.CharData(s.String()))
	}
	return fmt.Errorf("bml: unsupported type %T", v)
}
