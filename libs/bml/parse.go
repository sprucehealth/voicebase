package bml

import (
	"encoding/xml"
	"fmt"
	"io"
	"reflect"
	"strings"
)

func Parse(s string) (BML, error) {
	d := xml.NewDecoder(strings.NewReader(s))
	var bml BML
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		switch v := t.(type) {
		case xml.CharData:
			bml = append(bml, string(v))
		case xml.StartElement:
			name := v.Name.Local
			et := elementTypes[name]
			if et == nil {
				return nil, fmt.Errorf("bml: unsupported element %s", name)
			}
			nv := reflect.New(et).Interface()
			if err := d.DecodeElement(nv, &v); err != nil {
				return nil, fmt.Errorf("bml: failed to decode %s: %s", name, err)
			}
			if vd, ok := nv.(Validator); ok {
				if err := vd.Validate(); err != nil {
					return nil, err
				}
			}
			bml = append(bml, nv)
		default:
			return nil, fmt.Errorf("bml: unsupported token %T", t)
		}
	}
	return bml, nil
}
