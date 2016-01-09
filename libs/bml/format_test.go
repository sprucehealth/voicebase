package bml

import (
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestFormat(t *testing.T) {
	s, err := BML{
		&Ref{ID: "e_1", Type: "Entity", Text: "Dr. Dribbles"},
		" called ",
		&Ref{ID: "e_2", Type: "Entity", Text: "Mr. Jones"},
		",\n<answered>",
	}.Format()
	test.OK(t, err)
	test.Equals(t, "<ref id=\"e_1\" type=\"entity\">Dr. Dribbles</ref> called <ref id=\"e_2\" type=\"entity\">Mr. Jones</ref>,\n&lt;answered&gt;", s)
}

func TestSprintf(t *testing.T) {
	s, err := Sprintf("who %v called %v,\n<answered>",
		&Ref{ID: "e_1", Type: "Entity", Text: "Dr. Dribbles"},
		&Ref{ID: "e_2", Type: "Entity", Text: "Mr. Jones"})
	test.OK(t, err)
	test.Equals(t, "who <ref id=\"e_1\" type=\"entity\">Dr. Dribbles</ref> called <ref id=\"e_2\" type=\"entity\">Mr. Jones</ref>,\n&lt;answered&gt;", s)
}

func TestSprintfFail(t *testing.T) {
	_, err := Sprintf("who %v called %v,\n<answered>",
		&Ref{ID: "e_1", Type: "Entity", Text: "Dr. Dribbles"},
		&Ref{ID: "e_2", Text: "Mr. Jones"})
	test.Equals(t, "bml: Ref requires Type", err.Error())
}
