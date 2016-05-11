package bml

import (
	"bytes"
	"encoding/xml"
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestRef(t *testing.T) {
	b := &bytes.Buffer{}
	e := xml.NewEncoder(b)
	r := &Ref{
		ID:   "e_1",
		Type: "Entity",
		Text: "Dr. <D>ribbles",
	}
	test.OK(t, e.EncodeElement(r, xml.StartElement{Name: xml.Name{Local: "ref"}}))
	test.Equals(t, `<ref id="e_1" type="Entity">Dr. &lt;D&gt;ribbles</ref>`, b.String())

	b.Reset()
	r = &Ref{
		ID:   "v_1",
		Type: AttachmentRef,
		Text: "S<u>p",
		URL:  "https://test.com/visit/visitID",
	}
	test.OK(t, e.EncodeElement(r, xml.StartElement{Name: xml.Name{Local: "ref"}}))
	test.Equals(t, `<ref id="v_1" type="thread_attachment" url="https://test.com/visit/visitID">S&lt;u&gt;p</ref>`, b.String())
}

func TestAnchor(t *testing.T) {
	b := &bytes.Buffer{}
	e := xml.NewEncoder(b)
	r := &Anchor{
		HREF: "https://www.google.com/",
		Text: "Dr. <D>ribbles",
	}
	test.OK(t, e.EncodeElement(r, xml.StartElement{Name: xml.Name{Local: "a"}}))
	test.Equals(t, `<a href="https://www.google.com/">Dr. &lt;D&gt;ribbles</a>`, b.String())
}
