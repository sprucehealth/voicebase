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

type selfReportedMedicationSearchResultItem struct {
	DisplayName       string
	LexiCompDrugId    string
	LexiCompSynonymId int
}

type selfReportedMedicationSearchResponse struct {
	XMLName xml.Name     `xml:"SelfReportedMedicationSearchResult"`
	SSO     singleSignOn `xml:"SingleSignOn"`
	Result
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
	XMLName xml.Name     `xml:"MedicationSelectMessageResult"`
	SSO     singleSignOn `xml:"SingleSignOn"`
	Result
	DoseFormDescription     string `xml:"DoseFormDescription"`
	RouteDescription        string `xml:"RouteDescription"`
	StrengthDescription     string `xml:"StrengthDescription"`
	DispenseUnitId          int    `xml:"DispenseUnitId"`
	DispenseUnitDescription string `xml:"DispenseUnitDescription"`
	GenericProductName      string `xml:"GenericProductName"`
	LexiGenProductId        int    `xml:"LexiGenProductId"`
	LexiDrugSynId           int    `xml:"LexiDrugSynId"`
	LexiSynonymTypeId       int    `xml:"LexiSynonymTypeId"`
	MatchedDrugName         string `xml:"MatchedDrugName"`
	RXCUI                   string `xml:"RXCUI"`
	TermType                string `xml:"TermType"`
	OTC                     bool   `xml:"OTC"`
	RepresentativeNDC       string `xml:"RepresentativeNDC"`
	Schedule                string `xml:"Schedule"`
}

type patientStartPrescribingRequest struct {
	XMLName               xml.Name                    `xml:"http://www.dosespot.com/API/11/ PatientStartPrescribingMessage"`
	SSO                   singleSignOn                `xml:"SingleSignOn"`
	Patient               *patient                    `xml:"Patient"`
	AddFavoritePharmacies []*patientPharmacySelection `xml:"AddFavoritePharmacies>AddPatientPharmacy"`
	AddPrescriptions      []*prescription             `xml:"AddPrescriptions>Prescription"`
}

type patientStartPrescribingResponse struct {
	XMLName        xml.Name         `xml:"http://www.dosespot.com/API/11/ PatientStartPrescribingMessageResult"`
	SSO            singleSignOn     `xml:"SingleSignOn"`
	PatientUpdates []*patientUpdate `xml:"PatientUpdates>PatientUpdate"`
	Result
}

type patientUpdate struct {
	Patient     *patient      `xml:"Patient"`
	Medications []*medication `xml:"Medications>Medication"`
}

type prescription struct {
	Medication *medication `xml:"Medication"`
}

type medication struct {
	DisplayName            string `xml:"DisplayName"`
	Strength               string `xml:"Strength"`
	DoseSpotPrescriptionId int    `xml:"PrescriptionId"`
	LexiGenProductId       int    `xml:"LexiGenProductId"`
	LexiDrugSynId          int    `xml:"LexiDrugSynId"`
	LexiSynonymTypeId      int    `xml:"LexiSynonymTypeId"`
	Refills                int    `xml:"Refills"`
	DaysSupply             int    `xml:"DaysSupply"`
	Dispense               string `xml:"Dispense"`
	DispenseUnitId         int    `xml:"DispenseUnitId"`
	Instructions           string `xml:"Instructions"`
	PharmacyId             int    `xml:"PharmacyId"`
	PharmacyNotes          string `xml:"PharmacyNotes"`
	NoSubstitutions        bool   `xml:"NoSubstitutions"`
	RxReferenceNumber      string `xml:"RxReferenceNumber"`
	PrescriptionStatus     string `xml:"PrescriptionStatus,omitempty"`
	MedicationId           int64  `xml:"MedicationId,omitempty"`
}

// Need to treat the date object for date of birth as a special case
// because the date format returned from dosespot does not match the format
// layout that the built in datetime object is unmarshalled into
type specialDateTime struct {
	DateTime            time.Time
	DateTimeElementName string
}

func (c specialDateTime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var dateStr string
	err := d.DecodeElement(&dateStr, &start)
	if err != nil {
		return err
	}
	c.DateTime, err = time.Parse(time.RFC3339, dateStr+"Z")
	return err
}

func (c specialDateTime) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = c.DateTimeElementName
	err := e.EncodeElement(c.DateTime, start)
	return err
}

type patient struct {
	PatientId        int             `xml:"PatientId,omitempty"`
	Prefix           string          `xml:"Prefix"`
	FirstName        string          `xml:"FirstName"`
	MiddleName       string          `xml:"MiddleName"`
	LastName         string          `xml:"LastName"`
	Suffix           string          `xml:"Suffix"`
	DateOfBirth      specialDateTime `xml:"DateOfBirth"`
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

type pharmacy struct {
	PharmacyId          int64  `xml:"PharmacyId"`
	StoreName           string `xml:"StoreName"`
	Address1            string `xml:"Address1"`
	Address2            string `xml:"Address2"`
	City                string `xml:"City"`
	State               string `xml:"State"`
	ZipCode             string `xml:"ZipCode"`
	PrimaryPhone        string `xml:"PrimaryPhone"`
	PrimaryPhoneType    string `xml:"PrimaryPhoneType"`
	PrimaryFax          string `xml:"PrimaryFax"`
	PharmacySpecialties string `xml:"PharmacySpecialties"`
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
	SendPrescriptionResults []*sendPrescriptionResult `xml:"Prescriptions>SendPrescriptionResult"`
	Result
}

type sendPrescriptionResult struct {
	PrescriptionId int `xml:"PrescriptionID"`
	Result
}

type pharmacySearchRequest struct {
	XMLName                 xml.Name     `xml:"http://www.dosespot.com/API/11/ PharmacySearchMessageDetailed"`
	SSO                     singleSignOn `xml:"SingleSignOn"`
	PharmacyNameSearch      string       `xml:"PharmacyNameSearch,omitempty"`
	PharmacyCity            string       `xml:"PharmacyCity,omitempty"`
	PharmacyStateTwoLetters string       `xml:"PharmacyStateTwoLetters,omitempty"`
	PharmacyZipCode         string       `xml:"PharmacyZipCode,omitempty"`
	PharmacyTypes           []string     `xml:"PharmacySpecialties>PharmacySpecialtyTypes,omitempty"`
}

type pharmacySearchResult struct {
	XMLName    xml.Name     `xml:"http://www.dosespot.com/API/11/ PharmacySearchMessageDetailedResult"`
	SSO        singleSignOn `xml:"SingleSignOn"`
	Pharmacies []*pharmacy  `xml:"Pharmacies>PharmacyDetailed"`
	Result
}

type getPrescriptionLogDetailsRequest struct {
	XMLName        xml.Name     `xml:"http://www.dosespot.com/API/11/ GetPrescriptionLogDetailsRequest"`
	SSO            singleSignOn `xml:"SingleSignOn"`
	PrescriptionId int64        `xml:"PrescriptionID"`
}

type getPrescriptionLogDetailsResult struct {
	XMLName xml.Name               `xml:"http://www.dosespot.com/API/11/ GetPrescriptionLogDetailsResult"`
	SSO     singleSignOn           `xml:"SingleSignOn"`
	Log     []*prescriptionLogInfo `xml:"Log>PrescriptionLogInfo"`
	Result
}

type getTransmissionErrorDetailsRequest struct {
	XMLName xml.Name     `xml:"http://www.dosespot.com/API/11/ GetTransmissionErrorsRequest"`
	SSO     singleSignOn `xml:"SingleSignOn"`
}

type transmissionErrorDetailsItem struct {
	Medication   *medication `xml:"Medication"`
	ErrorDetails string      `xml:"ErrorDetails"`
}

type getTransmissionErrorDetailsResponse struct {
	XMLName xml.Name     `xml:"http://www.dosespot.com/API/11/ GetTransmissionErrorsDetailsResult"`
	SSO     singleSignOn `xml:"SingleSignOn"`
	Result
	TransmissionErrors []*transmissionErrorDetailsItem `xml:"TransmissionErrors>TransmissionErrorDetails"`
}

type prescriptionLogInfo struct {
	Status         string           `xml:"Status"`
	DateTimeStamp  *specialDateTime `xml:"DateTimeStamp"`
	AdditionalInfo string           `xml:"AdditionalInfo"`
}

type Result struct {
	ResultCode        string `xml:"Result>ResultCode"`
	ResultDescription string `xml:"Result>ResultDescription"`
}

type getMedicationListRequest struct {
	XMLName   xml.Name     `xml:"http://www.dosespot.com/API/11/ GetMedicationListRequest"`
	SSO       singleSignOn `xml:"SingleSignOn"`
	PatientId int64        `xml:"PatientId"`
	Sources   []string     `xml:"Sources>MedicationSourceType"`
	Status    []string     `xml:"Status>MedicationStatusType"`
}

type getMedicationListResult struct {
	XMLName xml.Name `xml:"http://www.dosespot.com/API/11/ GetMedicationListResult"`
	Result
	Medications []*medication `xml:"Medications>MedicationListItem"`
}
