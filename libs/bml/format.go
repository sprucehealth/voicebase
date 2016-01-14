package bml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
)

var reReplacement = regexp.MustCompile(`%.`)

// PlainTexter is implemented by elements that can convert themselves to plain text
type PlainTexter interface {
	PlainText() (string, error)
}

// BML is a slice of BML formattable elements
type BML []interface{}

// Validate returns an error iff any element of the BML is invalid.
func (bml BML) Validate() error {
	for _, v := range bml {
		switch vt := v.(type) {
		case int, int64, uint64, string:
		case Validator:
			if err := vt.Validate(); err != nil {
				return err
			}
		case fmt.Stringer:
		default:
			return fmt.Errorf("bml: unsupported type %T", v)
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

// PlainText returns a plain text version of the BML with all formatting removed.
func (bml BML) PlainText() (string, error) {
	if err := bml.Validate(); err != nil {
		return "", err
	}
	var buf []byte
	for _, b := range bml {
		switch v := b.(type) {
		case int:
			buf = strconv.AppendInt(buf, int64(v), 10)
		case int64:
			buf = strconv.AppendInt(buf, v, 10)
		case uint64:
			buf = strconv.AppendUint(buf, v, 10)
		case string:
			buf = append(buf, v...)
		case PlainTexter:
			s, err := v.PlainText()
			if err != nil {
				return "", err
			}
			buf = append(buf, s...)
		default:
			return "", fmt.Errorf("bml: unsupported type for plain text %T", b)
		}
	}
	return string(buf), nil
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
	case Validator:
		if err := vt.Validate(); err != nil {
			return err
		}
		return e.Encode(v)
	case fmt.Stringer:
		return e.EncodeToken(xml.CharData(vt.String()))
	}
	return fmt.Errorf("bml: unsupported type %T", v)
}
