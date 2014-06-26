package erx

import (
	"encoding/xml"
	"github.com/sprucehealth/backend/encoding"
	"time"
)

const (
	LexiGenProductId  = "lexi_gen_product_id"
	LexiDrugSynId     = "lexi_drug_syn_id"
	LexiSynonymTypeId = "lexi_synonym_type_id"
	NDC               = "ndc"
)

type singleSignOn struct {
	ClinicId     int64  `xml:"SingleSignOnClinicId"`
	Code         string `xml:"SingleSignOnCode"`
	UserId       int64  `xml:"SingleSignOnUserId"`
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
	LexiCompSynonymId int64
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
	DispenseUnitId          int64  `xml:"DispenseUnitId"`
	DispenseUnitDescription string `xml:"DispenseUnitDescription"`
	GenericProductName      string `xml:"GenericProductName"`
	LexiGenProductId        int64  `xml:"LexiGenProductId"`
	LexiDrugSynId           int64  `xml:"LexiDrugSynId"`
	LexiSynonymTypeId       int64  `xml:"LexiSynonymTypeId"`
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
	Pharmacies  []*pharmacy   `xml:"Pharmacies>Pharmacy"`
}

type prescription struct {
	Medication *medication `xml:"Medication"`
}

type medication struct {
	DisplayName             string             `xml:"DisplayName"`
	DrugName                string             `xml:"DrugName,omitempty"`
	Strength                string             `xml:"Strength"`
	Route                   string             `xml:"Route"`
	DoseSpotPrescriptionId  int64              `xml:"PrescriptionId"`
	LexiGenProductId        int64              `xml:"LexiGenProductId"`
	LexiDrugSynId           int64              `xml:"LexiDrugSynId"`
	LexiSynonymTypeId       int64              `xml:"LexiSynonymTypeId"`
	NDC                     string             `xml:"NDC"`
	RepresentativeNDC       string             `xml:"RepresentativeNDC"`
	Refills                 encoding.NullInt64 `xml:"Refills"`
	DaysSupply              encoding.NullInt64 `xml:"DaysSupply,omitempty"`
	Dispense                string             `xml:"Dispense"`
	DispenseUnitId          int64              `xml:"DispenseUnitId"`
	DispenseUnitDescription string             `xml:"DispenseUnit"`
	Instructions            string             `xml:"Instructions"`
	PharmacyId              int64              `xml:"PharmacyId"`
	PharmacyNotes           string             `xml:"PharmacyNotes"`
	NoSubstitutions         bool               `xml:"NoSubstitutions"`
	RxReferenceNumber       string             `xml:"RxReferenceNumber"`
	PrescriptionStatus      string             `xml:"PrescriptionStatus,omitempty"`
	Status                  string             `xml:"Status,omitempty"`
	DatePrescribed          *specialDateTime   `xml:"DatePrescribed,omitempty"`
	LastDateFilled          *specialDateTime   `xml:"LastDateFilled,omitempty"`
	DateWritten             *specialDateTime   `xml:"DateWritten,omitempty"`
	ClinicianId             int64              `xml:"ClinicianId"`
	ClinicId                int64              `xml:"ClinicId"`
	MedicationId            int64              `xml:"MedicationId,omitempty"`
	Schedule                string             `xml:"Schedule"`
}

// Need to treat the date object for date of birth as a special case
// because the date format returned from dosespot does not match the format
// layout that the built in datetime object is unmarshalled into
type specialDateTime struct {
	DateTime            time.Time
	DateTimeElementName string
}

func (c *specialDateTime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var dateStr string
	// nothing to do if the value is indicated to be nil via the attribute
	// form of element would be: <elementName xsi:nil="true" />
	if len(start.Attr) > 0 {
		if start.Attr[0].Name.Local == "nil" && start.Attr[0].Value == "true" {
			// still decoding to consume the element in the xml document
			d.DecodeElement(&dateStr, &start)
			return nil
		}
	}

	err := d.DecodeElement(&dateStr, &start)
	if err != nil {
		return err
	}
	c.DateTime, err = time.Parse(time.RFC3339, dateStr+"Z")
	return err
}

func (c *specialDateTime) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = c.DateTimeElementName
	err := e.EncodeElement(c.DateTime, start)
	return err
}

type patient struct {
	PatientId            int64           `xml:"PatientId,omitempty"`
	Prefix               string          `xml:"Prefix"`
	FirstName            string          `xml:"FirstName"`
	MiddleName           string          `xml:"MiddleName"`
	LastName             string          `xml:"LastName"`
	Suffix               string          `xml:"Suffix"`
	DateOfBirth          specialDateTime `xml:"DateOfBirth"`
	Gender               string          `xml:"Gender"`
	Email                string          `xml:"Email"`
	Address1             string          `xml:"Address1"`
	Address2             string          `xml:"Address2"`
	City                 string          `xml:"City"`
	State                string          `xml:"State"`
	ZipCode              string          `xml:"ZipCode"`
	PrimaryPhone         string          `xml:"PrimaryPhone"`
	PrimaryPhoneType     string          `xml:"PrimaryPhoneType"`
	PhoneAdditional1     string          `xml:"PhoneAdditional1"`
	PhoneAdditionalType1 string          `xml:"PhoneAdditionalType1"`
	PhoneAdditional2     string          `xml:"PhoneAdditional2"`
	PhoneAdditionalType2 string          `xml:"PhoneAdditionalType2"`
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
	PharmacyId int64 `xml:"PharmacyId"`
	IsPrimary  bool  `xml:"IsPrimary"`
}

type sendMultiplePrescriptionsRequest struct {
	XMLName         xml.Name     `xml:"http://www.dosespot.com/API/11/ SendMultiplePrescriptionsRequest"`
	SSO             singleSignOn `xml:"SingleSignOn"`
	PatientId       int64        `xml:"PatientId"`
	PrescriptionIds []int64      `xml:"PrescriptionIDs>int"`
}

type sendMultiplePrescriptionsResponse struct {
	XMLName                 xml.Name                  `xml:"http://www.dosespot.com/API/11/ SendMultiplePrescriptionsResult"`
	SSO                     singleSignOn              `xml:"SingleSignOn"`
	SendPrescriptionResults []*sendPrescriptionResult `xml:"Prescriptions>SendPrescriptionResult"`
	Result
}

type sendPrescriptionResult struct {
	PrescriptionId int64 `xml:"PrescriptionID"`
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
	Medication                  *medication      `xml:"Medication"`
	ErrorDateTimeStamp          *specialDateTime `xml:"ErrorDateTimeStamp"`
	ErrorDetails                string           `xml:"ErrorDetails"`
	RelatedRxRequestQueueItemID int64            `xml:"RelatedRxRequestQueueItemID"`
}

type getTransmissionErrorDetailsResponse struct {
	XMLName xml.Name     `xml:"http://www.dosespot.com/API/11/ GetTransmissionErrorsDetailsResult"`
	SSO     singleSignOn `xml:"SingleSignOn"`
	Result
	TransmissionErrors []*transmissionErrorDetailsItem `xml:"TransmissionErrors>TransmissionErrorDetails"`
}

type getRefillRequestsTransmissionErrorsMessageRequest struct {
	XMLName     xml.Name     `xml:"http://www.dosespot.com/API/11/ GetRefillRequestsTransmissionErrorsMessageRequest"`
	SSO         singleSignOn `xml:"SingleSignOn"`
	ClinicianId int64        `xml:"ClinicianId"`
}

type getRefillRequestsTransmissionErrorsResult struct {
	XMLName                          xml.Name                                    `xml:"http://www.dosespot.com/API/11/ GetRefillRequestsTransmissionErrorsResult"`
	SSO                              singleSignOn                                `xml:"SingleSignOn"`
	RefillRequestsTransmissionErrors []*refillRequestTransmissionErrorResultItem `xml:"RefillRequestsTransmissionErrors>RefillRequestsTransmissionError"`
}

type refillRequestTransmissionErrorResultItem struct {
	ClinicianId            int64 `xml:"ClinicianId"`
	RefillRequestsCount    int64 `xml:"RefillRequestsCount"`
	TransactionErrorsCount int64 `xml:"TransactionErrorsCount"`
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

type ignoreAlertRequest struct {
	XMLName        xml.Name     `xml:"http://www.dosespot.com/API/11/ IgnoreAlertRequest"`
	SSO            singleSignOn `xml:"SingleSignOn"`
	PrescriptionId int64        `xml:"PrescriptionId"`
}

type ignoreAlertResponse struct {
	XMLName xml.Name `xml:"http://www.dosespot.com/API/11/ IgnoreAlertResult"`
	Result
}

type clinician struct {
	ClinicianId      int64            `xml:"ClinicianId"`
	Prefix           string           `xml:"Prefix"`
	FirstName        string           `xml:"FirstName"`
	MiddleName       string           `xml:"MiddleName"`
	LastName         string           `xml:"LastName"`
	Suffix           string           `xml:"Suffix"`
	DateOfBirth      *specialDateTime `xml:"SpecialDateTime"`
	Gender           string           `xml:"Gender"`
	Email            string           `xml:"Email"`
	Address1         string           `xml:"Address1"`
	Address2         string           `xml:"Address2"`
	City             string           `xml:"City"`
	State            string           `xml:"State"`
	ZipCode          string           `xml:"ZipCode"`
	PrimaryPhone     string           `xml:"PrimaryPhone"`
	PrimaryPhoneType string           `xml:"PrimaryPhoneType"`
	PrimaryFax       string           `xml:"PrimaryFax"`
	DEANumber        string           `xml:"DEANumber"`
	NPINumber        string           `xml:"NPINumber"`
}

type refillRequestQueueItem struct {
	RxRequestQueueItemId      int64            `xml:"RxRequestQueueItemID"`
	ReferenceNumber           string           `xml:"ReferenceNumber"`
	PharmacyRxReferenceNumber string           `xml:"PharmacyRxReferenceNumber"`
	RequestedDrugDescription  string           `xml:"RequestedDrugDescription"`
	RequestedRefillAmount     string           `xml:"RequestedRefillAmount"`
	RequestedDispense         string           `xml:"RequestedDispense"`
	PatientId                 int64            `xml:"PatientID"`
	PatientAddedForRequest    bool             `xml:"PatientAddedForRequest"`
	RequestDateStamp          *specialDateTime `xml:"CreatedDateStamp"`
	Clinician                 *clinician       `xml:"Clinician"`
	RequestedPrescription     *medication      `xml:"RequestedPrescription"`
	DispensedPrescription     *medication      `xml:"DispensedPrescription"`
}

type getMedicationRefillRequestQueueForClinicRequest struct {
	XMLName xml.Name     `xml:"http://www.dosespot.com/API/11/ GetMedicationRefillRequestQueueRequestForClinic"`
	SSO     singleSignOn `xml:"SingleSignOn"`
}

type getMedicationRefillRequestQueueForClinicResult struct {
	XMLName xml.Name     `xml:"http://www.dosespot.com/API/11/ GetMedicationRefillRequestQueueForClinicResult"`
	SSO     singleSignOn `xml:"SingleSignOn"`
	Result
	RefillRequestQueue []*refillRequestQueueItem `xml:"List>RxRequestQueueItem"`
}

type getPatientDetailRequest struct {
	XMLName   xml.Name     `xml:"http://www.dosespot.com/API/11/ GetPatientDetailRequest"`
	SSO       singleSignOn `xml:"SingleSignOn"`
	PatientId int64        `xml:"PatientId"`
}

type getPatientDetailResult struct {
	XMLName        xml.Name         `xml:"http://www.dosespot.com/API/11/ GetPatientDetailResult"`
	PatientUpdates []*patientUpdate `xml:"PatientUpdates>PatientUpdate"`
	Result
}

type pharmacyDetailsRequest struct {
	XMLName    xml.Name     `xml:"http://www.dosespot.com/API/11/ PharmacyValidateMessage"`
	SSO        singleSignOn `xml:"SingleSignOn"`
	PharmacyId int64        `xml:"PharmacyId"`
}

type pharmacyDetailsResult struct {
	XMLName xml.Name `xml:"http://www.dosespot.com/API/11/ PharmacyValidateMessageResult"`
	Result
	PharmacyDetails *pharmacy `xml:"Pharmacy"`
}

type approveRefillRequest struct {
	XMLName              xml.Name     `xml:"http://www.dosespot.com/API/11/ ApproveRefillRequest"`
	SSO                  singleSignOn `xml:"SingleSignOn"`
	RxRequestQueueItemId int64        `xml:"RxRequestQueueItemID"`
	Refills              int64        `xml:"Refills"`
	Comments             string       `xml:"Note"`
}

type approveRefillResponse struct {
	XMLName xml.Name `xml:"http://www.dosespot.com/API/11/ ApproveRefillResult"`
	Result
	PatientId      int64 `xml:"PatientID"`
	PrescriptionId int64 `xml:"PrescriptionId"`
}

type denyRefillRequest struct {
	XMLName              xml.Name     `xml:"http://www.dosespot.com/API/11/ DenyRefillRequest"`
	SSO                  singleSignOn `xml:"SingleSignOn"`
	RxRequestQueueItemId int64        `xml:"RxRequestQueueItemID"`
	DenialReason         string       `xml:"DenialReason"`
	Comments             string       `xml:"Note"`
}

type denyRefillResponse struct {
	XMLName xml.Name     `xml:"http://www.dosespot.com/API/11/ DenyRefillResult"`
	SSO     singleSignOn `xml:"SingleSignOn"`
	Result
	PatientId      int64 `xml:"PatientID"`
	PrescriptionId int64 `xml:"PrescriptionId"`
}
