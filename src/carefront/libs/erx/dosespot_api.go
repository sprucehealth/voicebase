package erx

import (
	"encoding/xml"
)

const (
	LexiGenProductId  = "lexi_gen_product_id"
	LexiDrugSynId     = "lexi_drug_syn_id"
	LexiSynonymTypeId = "lexi_synonym_type_id"
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

type medicationStrengthSearchRequest struct {
	XMLName        xml.Name     `xml:"http://www.dosespot.com/API/11/ MedicationStrengthSearchMessage"`
	SSO            singleSignOn `xml:"SingleSignOn"`
	MedicationName string       `xml:"SearchString"`
}

type medicationStrengthSearchResponse struct {
	XMLName          xml.Name     `xml:"MedicationStrengthSearchMessageResult"`
	SSO              singleSignOn `xml:"SingleSignOn"`
	DisplayStrengths []string     `xml:"DisplayStrength>string"`
}

type medicationSelectRequest struct {
	XMLName            xml.Name     `xml:"http://www.dosespot.com/API/11/ MedicationSelectMessage"`
	SSO                singleSignOn `xml:"SingleSignOn"`
	MedicationName     string       `xml:"MedicationWithDoseFormRoute"`
	MedicationStrength string       `xml:"MedicationStrength"`
}

type medicationSelectResponse struct {
	XMLName                 xml.Name     `xml:"MedicationSelectMessageResult"`
	SSO                     singleSignOn `xml:"SingleSignOn"`
	ResultCode              string       `xml:"Result>ResultCode"`
	ResultDescription       string       `xml:"Result>Description"`
	DoseFormDescription     string       `xml:"DoseFormDescription"`
	RouteDescription        string       `xml:"RouteDescription"`
	StrengthDescription     string       `xml:"StrengthDescription"`
	DispenseUnitId          int          `xml:"DispenseUnitId"`
	DispenseUnitDescription string       `xml:"DispenseUnitDescription"`
	GenericProductName      string       `xml:"GenericProductName"`
	LexiGenProductId        int          `xml:"LexiGenProductId"`
	LexiDrugSynId           int          `xml:"LexiDrugSynId"`
	LexiSynonymTypeId       int          `xml:"LexiSynonymTypeId"`
	MatchedDrugName         string       `xml:"MatchedDrugName"`
	RXCUI                   string       `xml:"RXCUI"`
	TermType                string       `xml:"TermType"`
	OTC                     bool         `xml:"OTC"`
	RepresentativeNDC       string       `xml:"RepresentativeNDC"`
	Schedule                string       `xml:"Schedule"`
}
