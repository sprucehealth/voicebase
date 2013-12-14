package erx

import (
	"encoding/xml"
)

type singleSignOn struct {
	ClinicId     string `xml:"SingleSignOnClinicId"`
	Code         string `xml:"SingleSignOnCode"`
	UserId       string `xml:"SingleSignOnUserId"`
	UserIdVerify string `xml:"SingleSignOnUserIdVerify"`
	PhraseLength int64  `xml:"SingleSignOnPhraseLength"`
}

type medicationQuickSearchRequest struct {
	XMLName      xml.Name     `xml:"http://www.dosespot.com/API/11/ MedicationQuickSearchMessage"`
	SSO          singleSignOn `xml:"SingleSignOn"`
	SearchString string
}

type medicationQuickSearchResponse struct {
	XMLName      xml.Name     `xml:"MedicationQuickSearchMessageResult"`
	SSO          singleSignOn `xml:"SingleSignOn"`
	DisplayNames []string     `xml:"DisplayNames>string"`
}

type selfReportedMedicationSearchRequest struct {
	XMLName    xml.Name     `xml:"http://www.dosespot.com/API/11/ SelfReportedMedicationSearchRequest"`
	SSO        singleSignOn `xml:"SingleSignOn"`
	SearchTerm string
}

type doseSpotResult struct {
	ResultCode        string
	ResultDescription string
}

type selfReportedMedicationSearchResultItem struct {
	DisplayName       string
	LexiCompDrugId    string
	LexiCompSynonymId int
}

type selfReportedMedicationSearchResponse struct {
	XMLName       xml.Name     `xml:"SelfReportedMedicationSearchResult"`
	SSO           singleSignOn `xml:"SingleSignOn"`
	Result        doseSpotResult
	SearchResults []*selfReportedMedicationSearchResultItem `xml:"SearchResults>SelfReportedMedicationSearchResult"`
}
