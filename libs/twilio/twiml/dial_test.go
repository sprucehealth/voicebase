package twiml

import (
	"encoding/xml"
	"testing"
)

func TestStatusCallbackEvent_Attr(t *testing.T) {

	name := xml.Name{
		Space: "",
		Local: "sup",
	}

	compare(t, SCInitiated, "initiated", name)
	compare(t, SCRinging|SCAnswered, "ringing answered", name)
	compare(t, SCAnswered|SCRinging, "ringing answered", name)
	compare(t, SCNone, "", xml.Name{})
	compare(t, SCInitiated|SCRinging|SCAnswered|SCCompleted, "initiated ringing answered completed", name)
}

func compare(t *testing.T, sc StatusCallbackEvent, expected string, name xml.Name) {
	attr, err := sc.MarshalXMLAttr(name)
	if err != nil {
		t.Fatal(err)
	}
	if attr.Value != expected {
		t.Fatalf("Expected %s but got %s", expected, attr.Value)
	}
	if attr.Name.Local != name.Local {
		t.Fatalf("Expected %s but got %s", attr.Name.Local, name.Local)
	}
	if attr.Name.Space != name.Space {
		t.Fatalf("Expected %s but got %s", attr.Name.Space, name.Space)
	}
}
