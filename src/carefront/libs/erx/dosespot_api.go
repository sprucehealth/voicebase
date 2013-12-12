package erx

import (
	"encoding/xml"
)

const (
	doseSpotAPIEndPoint         = "http://www.dosespot.com/API/11/"
	doseSpotSOAPEndPoint        = "http://i.dosespot.com/api/11/apifull.asmx"
	medicationQuickSearchAction = "MedicationQuickSearchMessage"
)

type singleSignOn struct {
	ClinicId     string `xml:"SingleSignOnClinicId"`
	Code         string `xml:"SingleSignOnCode"`
	UserId       string `xml:"SingleSignOnUserId"`
	UserIdVerify string `xml:"SingleSignOnUserIdVerify"`
	PhraseLength int64  `xml:"SingleSignOnPhraseLength"`
}

type medicationQuickSearchMessage struct {
	XMLName      xml.Name     `xml:"MedicationQuickSearchMessage"`
	SSO          singleSignOn `xml:"SingleSignOn"`
	APIEndPoint  string       `xml:"xmlns,attr"`
	SearchString string
}

type medicationQuickSearchResult struct {
	XMLName      xml.Name     `xml:"MedicationQuickSearchMessageResult"`
	SSO          singleSignOn `xml:"SingleSignOn"`
	DisplayNames []string     `xml:"DisplayNames>string"`
}

func newMedicationQuickSearchMessage() *medicationQuickSearchMessage {
	m := &medicationQuickSearchMessage{}
	m.APIEndPoint = doseSpotAPIEndPoint
	return m
}

func (m *medicationQuickSearchMessage) GetSoapAction() string {
	return doseSpotAPIEndPoint + medicationQuickSearchAction
}

func (m *medicationQuickSearchMessage) GetSoapAPIEndPoint() string {
	return doseSpotSOAPEndPoint
}
