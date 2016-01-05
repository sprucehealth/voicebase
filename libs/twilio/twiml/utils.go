package twiml

import (
	"encoding/xml"
	"fmt"
)

func validateMethod(method string) error {
	switch method {
	case "", "POST", "GET":
	default:
		return fmt.Errorf("invalid http method '%s'. Only valid options are GET/POST.", method)
	}
	return nil
}

type StatusCallbackEvent int

const (
	SCInitiated StatusCallbackEvent = 1 << iota
	SCRinging
	SCAnswered
	SCCompleted
	SCNone StatusCallbackEvent = 0
)

func (s StatusCallbackEvent) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	if s == SCNone {
		return xml.Attr{}, nil
	}

	buffer := make([]byte, 0, (9 * 4))
	var added bool
	if s&SCInitiated == SCInitiated {
		buffer = appendEvent(buffer, added, "initiated")
		added = true
	}
	if s&SCRinging == SCRinging {
		buffer = appendEvent(buffer, added, "ringing")
		added = true
	}
	if s&SCAnswered == SCAnswered {
		buffer = appendEvent(buffer, added, "answered")
		added = true
	}
	if s&SCCompleted == SCCompleted {
		buffer = appendEvent(buffer, added, "completed")
		added = true
	}

	return xml.Attr{
		Name:  name,
		Value: string(buffer),
	}, nil
}

func appendEvent(buffer []byte, added bool, event string) []byte {
	if added {
		buffer = append(buffer, ' ')
	}
	return append(buffer, []byte(event)...)
}
