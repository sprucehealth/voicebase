package erx

import (
	"encoding/xml"
	"time"
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

type patientStartPrescribingRequest struct {
	XMLName               xml.Name                    `xml:"http://www.dosespot.com/API/11/ PatientStartPrescribingMessage"`
	SSO                   singleSignOn                `xml:"SingleSignOn"`
	Patient               *patient                    `xml:"Patient"`
	AddFavoritePharmacies []*patientPharmacySelection `xml:"AddFavoritePharmacies>AddPatientPharmacy"`
	AddPrescriptions      []*prescription             `xml:"AddPrescriptions>Prescription"`
}

type patientStartPrescribingResponse struct {
	XMLName           xml.Name         `xml:"http://www.dosespot.com/API/11/ PatientStartPrescribingMessageResult"`
	SSO               singleSignOn     `xml:"SingleSignOn"`
	PatientUpdates    []*patientUpdate `xml:"PatientUpdates>PatientUpdate"`
	ResultCode        string           `xml:"Result>ResultCode"`
	ResultDescription string           `xml:"Result>ResultDescription"`
}

type patientUpdate struct {
	Patient     *patient      `xml:"Patient"`
	Medications []*medication `xml:"Medications>Medication"`
}

type prescription struct {
	Medication *medication `xml:"Medication"`
}

type medication struct {
	XMLName                xml.Name `xml:"Medication"`
	DoseSpotPrescriptionId int      `xml:"PrescriptionId"`
	LexiGenProductId       int      `xml:"LexiGenProductId"`
	LexiDrugSynId          int      `xml:"LexiDrugSynId"`
	LexiSynonymTypeId      int      `xml:"LexiSynonymTypeId"`
	Refills                int      `xml:"Refills"`
	DaysSupply             int      `xml:"DaysSupply"`
	Dispense               string   `xml:"Dispense"`
	DispenseUnitId         int      `xml:"DispenseUnitId"`
	Instructions           string   `xml:"Instructions"`
	PharmacyId             int      `xml:"PharmacyId"`
	PharmacyNotes          string   `xml:"PharmacyNotes"`
	NoSubstitutions        bool     `xml:"NoSubstitutions"`
	RxReferenceNumber      string   `xml:"RxReferenceNumber"`
}

// Need to treat the date object for date of birth as a special case
// because the date format returned from dosespot does not match the format
// layout that the built in datetime object is unmarshalled into
type DateOfBirthType struct {
	DateOfBirth time.Time `xml:"DateOfBirth"`
}

func (c DateOfBirthType) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var dateStr string
	err := d.DecodeElement(&dateStr, &start)
	if err != nil {
		return err
	}
	c.DateOfBirth, err = time.Parse(time.RFC3339, dateStr+"Z")
	return err
}

func (c DateOfBirthType) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	err := e.EncodeElement(c.DateOfBirth, start)
	return err
}

type patient struct {
	PatientId        int             `xml:"PatientId,omitempty"`
	Prefix           string          `xml:"Prefix"`
	FirstName        string          `xml:"FirstName"`
	MiddleName       string          `xml:"MiddleName"`
	LastName         string          `xml:"LastName"`
	Suffix           string          `xml:"Suffix"`
	DateOfBirth      DateOfBirthType `xml:"DateOfBirth"`
	Gender           string          `xml:"Gender"`
	Email            string          `xml:"Email"`
	Address1         string          `xml:"Address1"`
	Address2         string          `xml:"Address2"`
	City             string          `xml:"City"`
	State            string          `xml:"State"`
	ZipCode          string          `xml:"ZipCode"`
	PrimaryPhone     string          `xml:"PrimaryPhone"`
	PrimaryPhoneType string          `xml:"PrimaryPhoneType"`
}

type patientPharmacySelection struct {
	PharmacyId int  `xml:"PharmacyId"`
	IsPrimary  bool `xml:"IsPrimary"`
}

type sendMultiplePrescriptionsRequest struct {
	XMLName         xml.Name     `xml:"http://www.dosespot.com/API/11/ SendMultiplePrescriptionsRequest"`
	SSO             singleSignOn `xml:"SingleSignOn"`
	PatientId       int          `xml:"PatientId"`
	PrescriptionIds []int        `xml:"PrescriptionIDs>int"`
}

type sendMultiplePrescriptionsResponse struct {
	XMLName                 xml.Name                  `xml:"http://www.dosespot.com/API/11/ SendMultiplePrescriptionsResult"`
	SSO                     singleSignOn              `xml:"SingleSignOn"`
	ResultCode              string                    `xml:"Result>ResultCode"`
	ResultDescription       string                    `xml:"Result>ResultDescription"`
	SendPrescriptionResults []*sendPrescriptionResult `xml:"Prescriptions>SendPrescriptionResult"`
}

type sendPrescriptionResult struct {
	PrescriptionId    int    `xml:"PrescriptionID"`
	ResultCode        string `xml:"Result>ResultCode"`
	ResultDescription string `xml:"Result>ResultDescription"`
}
