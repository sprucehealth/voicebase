package bml

import (
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestParse(t *testing.T) {
	bml, err := Parse("who <ref id=\"e_1\" type=\"Entity\">Dr. Dribbles</ref> called <ref id=\"e_2\" type=\"Entity\">Mr. Jones</ref>,\n&lt;answered&gt;")
	test.OK(t, err)
	test.Equals(t, BML{
		"who ",
		&Ref{ID: "e_1", Type: "entity", Text: "Dr. Dribbles"},
		" called ",
		&Ref{ID: "e_2", Type: "entity", Text: "Mr. Jones"},
		",\n<answered>",
	}, bml)
}

func TestParseFail(t *testing.T) {
	_, err := Parse("who <ref>Dr. Dribbles</ref> called <ref id=\"e_2\" type=\"Entity\">Mr. Jones</ref>,\n&lt;answered&gt;")
	test.Equals(t, "bml: Ref requires ID", err.Error())
}
