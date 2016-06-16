package icd10

import (
	"encoding/xml"
	"strings"
)

type Diagnosis struct {
	Code              Code         `xml:"name"`
	Description       string       `xml:"desc"`
	Includes          []string     `xml:"includes>note"`
	InclusionTerms    []string     `xml:"inclusionTerm>note"`
	Excludes1         []string     `xml:"excludes1>note"`
	Excludes2         []string     `xml:"excludes2>note"`
	UseAdditionalCode []string     `xml:"useAdditionalCode>note"`
	CodeFirst         []string     `xml:"codeFirst>note"`
	SeventhCharDef    []*Extension `xml:"sevenChrDef>extension"`
	Subcategories     []*Diagnosis `xml:"diag"`
	Billable          bool         `xml:"-"`
}

type Extension struct {
	XMLName   xml.Name `xml:"extension"`
	Character string   `xml:"char,attr"`
	Value     string   `xml:",chardata"`
}

type DB interface {
	SetDiagnoses(diagnoses map[string]*Diagnosis) error
	Connect(host, username, name, password string, port int) error
	Close() error
}

// Code represents a unique diagnosis code. This doesn't necessarily have to be
// from the ICD10 database only.
type Code string

// Key is a globally unique key to identify each diagnosis code by.
func (c Code) Key() string {
	return "diag_" + c.normalize()
}

func (c Code) String() string {
	return string(c)
}

func (c Code) normalize() string {
	return strings.Replace(strings.ToLower(string(c)), ".", "", -1)
}
