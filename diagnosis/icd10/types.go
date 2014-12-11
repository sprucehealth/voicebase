package icd10

import "encoding/xml"

type Diagnosis struct {
	Code              string       `xml:"name"`
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
