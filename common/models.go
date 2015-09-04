package common

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/pharmacy"
)

const (
	// AttachmentTypeAudio describes the communication attachment type as being audio
	AttachmentTypeAudio = "audio"

	// AttachmentTypeFollowupVisit describes the communication attachment type as being a follow up visit
	AttachmentTypeFollowupVisit = "followup_visit"

	// AttachmentTypePhoto describes the communication attachment type as being a photo
	AttachmentTypePhoto = "photo"

	// AttachmentTypeResourceGuide describes the communication attachment type as being a resource guide
	AttachmentTypeResourceGuide = "resource_guide"

	// AttachmentTypeTreatmentPlan describes the communication attachment type as being a treatment plan
	AttachmentTypeTreatmentPlan = "treatment_plan"

	// AttachmentTypeVisit describes the communication attachment type as being an individual visit
	AttachmentTypeVisit = "visit"
)

const (
	// ClaimerTypeConversationMessage is used to represent a media object claim by a case message
	ClaimerTypeConversationMessage = "conversation_message"
	// ClaimerTypePhotoIntakeSection is used to represent a media object claim by a photo intake section
	ClaimerTypePhotoIntakeSection = "patient_intake_photo_section"
	// ClaimerTypeTreatmentPlanScheduledMessage is used to represent a media object claim by a media
	// attachment in a scheduled message in a treatment plan
	ClaimerTypeTreatmentPlanScheduledMessage = "tp_scheduled_message"
	// ClaimerTypeFavoriteTreatmentPlanScheduledMessage is used to represent a media object claim
	// by a media attached in a scheduled message part of a favorite treatment plan
	ClaimerTypeFavoriteTreatmentPlanScheduledMessage = "ftp_scheduled_message"
	// ClaimerTypeParentalConsentProof is used to represent a media object claim
	// by a photoID attached as part of parental consent proof
	ClaimerTypeParentalConsentProof = "parental_consent_proof"
)

// PhoneNumberType preresents the type of phone number associated with the account
type PhoneNumberType string

const (
	// PNTCell represents a cell phone number
	PNTCell PhoneNumberType = "CELL"

	// PNTWork represents a work phone number
	PNTWork PhoneNumberType = "WORK"

	// PNTHome represents a home phone number
	PNTHome PhoneNumberType = "HOME"

	// PNTEmpty represents an empty phone number type
	PNTEmpty PhoneNumberType = ""
)

// ParsePhoneNumberType returns the PhoneNumberType the maps to the provided string
func ParsePhoneNumberType(s string) (PhoneNumberType, error) {
	switch t := PhoneNumberType(strings.ToUpper(s)); t {
	case PNTCell, PNTWork, PNTHome, PNTEmpty:
		return t, nil
	}
	return PhoneNumberType(""), fmt.Errorf("Unkown phone number type: %s", s)
}

func (t PhoneNumberType) String() string {
	return string(t)
}

// Scan allows for scanning of PhoneNumberType from a database conforming to the sql.Scanner interface
func (t *PhoneNumberType) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParsePhoneNumberType(ts)
	case []byte:
		*t, err = ParsePhoneNumberType(string(ts))
	}
	return err
}

// PhoneNumber represents a phone number mapped to an account
type PhoneNumber struct {
	Phone    Phone           `json:"phone,omitempty"`
	Type     PhoneNumberType `json:"phone_type,omitempty"`
	Status   string          `json:"status"`
	Verified bool            `json:"verified"`
}

// NewPatientID returns a new PatientID using the provided value. If id is 0
// then the returned PatiendID is tagged as invalid.
func NewPatientID(id uint64) PatientID {
	return PatientID{
		ObjectID: encoding.ObjectID{
			Uint64Value: id,
			IsValid:     id != 0,
		},
	}
}

// ParsePatientID parses a string version of a patient ID. If the string
// does not represent a valid ID then an error is returned and the PatientID
// is returned with IsValid == false.
func ParsePatientID(id string) (PatientID, error) {
	i, err := strconv.ParseUint(id, 10, 64)
	return PatientID{
		ObjectID: encoding.ObjectID{
			Uint64Value: i,
			IsValid:     err == nil,
		},
	}, err
}

// PatientID is the ID for a patient object
type PatientID struct {
	encoding.ObjectID
}

// String implements fmt.Stringer
func (id PatientID) String() string {
	return strconv.FormatUint(id.Uint64(), 10)
}

type Patient struct {
	ID                 PatientID              `json:"id,omitempty"`
	IsUnlinked         bool                   `json:"is_unlinked,omitempty"`
	FirstName          string                 `json:"first_name,omitempty"`
	LastName           string                 `json:"last_name,omiempty"`
	MiddleName         string                 `json:"middle_name,omitempty"`
	Suffix             string                 `json:"suffix,omitempty"`
	Prefix             string                 `json:"prefix,omitempty"`
	DOB                encoding.Date          `json:"dob,omitempty"`
	Email              string                 `json:"email,omitempty"`
	Gender             string                 `json:"gender,omitempty"`
	ZipCode            string                 `json:"zip_code,omitempty"`
	CityFromZipCode    string                 `json:"-"`
	StateFromZipCode   string                 `json:"state_code,omitempty"`
	PhoneNumbers       []*PhoneNumber         `json:"phone_numbers,omitempty"`
	Status             string                 `json:"-"`
	AccountID          encoding.ObjectID      `json:"account_id,omitempty"`
	ERxPatientID       encoding.ObjectID      `json:"erx_patient_id"`
	PaymentCustomerID  string                 `json:"-"`
	Pharmacy           *pharmacy.PharmacyData `json:"pharmacy,omitempty"`
	PatientAddress     *Address               `json:"address,omitempty"`
	PersonID           int64                  `json:"person_id"`
	PromptStatus       PushPromptStatus       `json:"prompt_status"`
	Training           bool                   `json:"is_training"`
	HasParentalConsent bool                   `json:"has_parental_consent"`
}

// IsUnder18 indicates whether or not patient is under 18 years of age.
func (p *Patient) IsUnder18() bool {
	return p.DOB.Age() < 18
}

type PCP struct {
	PatientID     PatientID `json:"-"`
	PhysicianName string    `json:"physician_full_name"`
	PhoneNumber   string    `json:"phone_number"`
	PracticeName  string    `json:"practice_name,omitempty"`
	Email         string    `json:"email,omitempty"`
	FaxNumber     string    `json:"fax_number,omitempty"`
}

func (p PCP) IsZero() bool {
	return p.PhysicianName == "" && p.PhoneNumber == "" && p.PracticeName == "" && p.Email == "" && p.FaxNumber == ""
}

type EmergencyContact struct {
	ID           int64     `json:"id,string"`
	PatientID    PatientID `json:"-"`
	FullName     string    `json:"full_name"`
	PhoneNumber  string    `json:"phone_number"`
	Relationship string    `json:"relationship"`
}

type Card struct {
	ID             encoding.ObjectID `json:"id,omitempty"`
	ThirdPartyID   string            `json:"third_party_id"`
	Fingerprint    string            `json:"fingerprint"`
	Token          string            `json:"token,omitempty"`
	Type           string            `json:"type"`
	ExpMonth       int64             `json:"exp_month"`
	ExpYear        int64             `json:"exp_year"`
	Last4          string            `json:"last4"`
	Label          string            `json:"label,omitempty"`
	BillingAddress *Address          `json:"address,omitempty"`
	IsDefault      bool              `json:"is_default,omitempty"`
	CreationDate   time.Time         `json:"creation_date"`
	ApplePay       bool              `json:"apple_pay"`
}

type Alert struct {
	ID           int64
	VisitID      int64
	QuestionID   *int64
	Message      string
	CreationDate time.Time
}

// Doctor represents a care provider which can be either a Doctor or Care Coordinator.
type Doctor struct {
	ID               encoding.ObjectID `json:"id,omitempty"`
	FirstName        string            `json:"first_name,omitempty"`
	LastName         string            `json:"last_name,omitempty"`
	MiddleName       string            `json:"middle_name,omitempty"`
	Prefix           string            `json:"prefix,omitempty"`
	Suffix           string            `json:"suffix,omitempty"`
	ShortTitle       string            `json:"short_title,omitempty"`
	LongTitle        string            `json:"long_title,omitempty"`
	ShortDisplayName string            `json:"short_display_name,omitempty"`
	LongDisplayName  string            `json:"long_display_name,omitempty"`
	DOB              encoding.Date     `json:"-"`
	Email            string            `json:"email"`
	Gender           string            `json:"-"`
	Status           string            `json:"-"`
	AccountID        encoding.ObjectID `json:"account_id"`
	CellPhone        Phone             `json:"phone"`
	LargeThumbnailID string            `json:"-"`
	SmallThumbnailID string            `json:"-"`
	HeroImageID      string            `json:"-"`
	PersonID         int64             `json:"person_id"`
	PromptStatus     PushPromptStatus  `json:"prompt_status"`
	// Doctor specific
	DoseSpotClinicianID int64    `json:"prescriber_id,omitempty"`
	Address             *Address `json:"address,omitempty"`
	NPI                 string   `json:"npi,omitempty"`
	DEA                 string   `json:"dea,omitempty"`
	// Care coordinator specific
	IsCC        bool `json:"is_ma"`
	IsPrimaryCC bool `json:"is_primary_cc"`
}

type State struct {
	Name         string
	Abbreviation string
	Country      string
}

type Address struct {
	ID           int64  `json:"-"`
	AddressLine1 string `json:"address_line_1"`
	AddressLine2 string `json:"address_line_2,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state"`
	ZipCode      string `json:"zip_code"`
	Country      string `json:"country"`
}

func (a *Address) Validate() error {
	if strings.TrimSpace(a.AddressLine1) == "" {
		return errors.New("AddressLine1 required")
	}
	if strings.TrimSpace(a.City) == "" {
		return errors.New("City required")
	}
	if strings.TrimSpace(a.State) == "" {
		return errors.New("State required")
	}
	if strings.TrimSpace(a.ZipCode) == "" {
		return errors.New("ZipCode required")
	}
	return nil
}

type CareProviderAssignment struct {
	ProviderRole     string     `json:"provider_role"`
	ProviderID       int64      `json:"provider_id"`
	FirstName        string     `json:"first_name,omitempty"`
	LastName         string     `json:"last_name,omitempty"`
	ShortTitle       string     `json:"short_title,omitempty"`
	LongTitle        string     `json:"long_title,omitempty"`
	ShortDisplayName string     `json:"short_display_name,omitempty"`
	LongDisplayName  string     `json:"long_display_name,omitempty"`
	SmallThumbnailID string     `json:"-"`
	LargeThumbnailID string     `json:"-"`
	PatientID        PatientID  `json:"-"`
	PathwayTag       string     `json:"-"`
	Status           string     `json:"-"`
	CreationDate     time.Time  `json:"assignment_date"`
	Expires          *time.Time `json:"-"`
}

type PatientCareTeam struct {
	Assignments []*CareProviderAssignment
}

type TreatmentPlanStatus string

const (
	TPStatusDraft     TreatmentPlanStatus = "DRAFT"
	TPStatusSubmitted TreatmentPlanStatus = "SUBMITTED"
	TPStatusActive    TreatmentPlanStatus = "ACTIVE"
	TPStatusInactive  TreatmentPlanStatus = "INACTIVE"
	TPStatusRXStarted TreatmentPlanStatus = "RX_STARTED"
)

func ParseTreatmentPlanStatus(s string) (TreatmentPlanStatus, error) {
	switch t := TreatmentPlanStatus(s); t {
	case TPStatusDraft, TPStatusSubmitted, TPStatusActive, TPStatusInactive, TPStatusRXStarted:
		return t, nil
	}
	return TreatmentPlanStatus(""), fmt.Errorf("Unkown treatment plan status: %s", s)
}

func (t TreatmentPlanStatus) String() string {
	return string(t)
}

func (t *TreatmentPlanStatus) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseTreatmentPlanStatus(ts)
	case []byte:
		*t, err = ParseTreatmentPlanStatus(string(ts))
	}
	return err
}

type FavoriteTreatmentPlan struct {
	ID                encoding.ObjectID                `json:"id"`
	Name              string                           `json:"name"`
	ModifiedDate      time.Time                        `json:"modified_date,omitempty"`
	CreatorID         *int64                           `json:"-"`
	ParentID          *int64                           `json:"-"`
	RegimenPlan       *RegimenPlan                     `json:"regimen_plan,omitempty"`
	TreatmentList     *TreatmentList                   `json:"treatment_list,omitempty"`
	Note              string                           `json:"note"`
	ScheduledMessages []*TreatmentPlanScheduledMessage `json:"scheduled_messages"`
	ResourceGuides    []*ResourceGuide                 `json:"resource_guides,omitempty"`
	Lifecycle         string                           `json:"lifecycle"`
}

type FavoriteTreatmentPlanByName []*FavoriteTreatmentPlan

func (s FavoriteTreatmentPlanByName) Len() int {
	return len(s)
}

func (s FavoriteTreatmentPlanByName) Less(i, j int) bool {
	return strings.ToLower(s[i].Name) < strings.ToLower(s[j].Name)
}

func (s FavoriteTreatmentPlanByName) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (f *FavoriteTreatmentPlan) EqualsTreatmentPlan(tp *TreatmentPlan) bool {
	if f == nil || tp == nil {
		return false
	}

	if !f.TreatmentList.Equals(tp.TreatmentList) {
		return false
	}

	if !f.RegimenPlan.Equals(tp.RegimenPlan) {
		return false
	}

	if f.Note != tp.Note {
		return false
	}

	if len(f.ScheduledMessages) != len(tp.ScheduledMessages) {
		return false
	}

	for _, sm1 := range f.ScheduledMessages {
		matched := false
		for _, sm2 := range tp.ScheduledMessages {
			if sm1.Equal(sm2) {
				matched = true
			}
		}
		if !matched {
			return false
		}
	}

	if len(f.ResourceGuides) != len(tp.ResourceGuides) {
		return false
	}

	for _, g1 := range f.ResourceGuides {
		found := false
		for _, g2 := range tp.ResourceGuides {
			if g1.ID == g2.ID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (f *FavoriteTreatmentPlan) Validate() error {
	if f == nil {
		return errors.New("Favorite treatment plan not provided")
	}

	if f.Name == "" {
		return errors.New("A favorite treatment plan requires a name")
	}

	// ensure that favorite treatment plan has at least one of treatments, regimen, or note
	if (f.TreatmentList == nil || len(f.TreatmentList.Treatments) == 0) &&
		(f.RegimenPlan == nil || len(f.RegimenPlan.Sections) == 0) &&
		f.Note == "" && len(f.ScheduledMessages) == 0 && len(f.ResourceGuides) == 0 {
		return errors.New("A favorite treatment plan must have at least one of: treatments, regimen, note, scheduled messages, or resource guides")
	}

	return nil
}

type TreatmentPlan struct {
	ID                encoding.ObjectID                `json:"id,omitempty"`
	DoctorID          encoding.ObjectID                `json:"doctor_id,omitempty"`
	PatientCaseID     encoding.ObjectID                `json:"case_id"`
	PatientID         PatientID                        `json:"patient_id,omitempty"`
	Status            TreatmentPlanStatus              `json:"status,omitempty"`
	CreationDate      time.Time                        `json:"creation_date"`
	SentDate          *time.Time                       `json:"sent_date,omitempty"`
	TreatmentList     *TreatmentList                   `json:"treatment_list"`
	RegimenPlan       *RegimenPlan                     `json:"regimen_plan,omitempty"`
	Parent            *TreatmentPlanParent             `json:"parent,omitempty"`
	ContentSource     *TreatmentPlanContentSource      `json:"content_source,omitempty"`
	Note              string                           `json:"note,omitempty"`
	ScheduledMessages []*TreatmentPlanScheduledMessage `json:"scheduled_messages"`
	ResourceGuides    []*ResourceGuide                 `json:"resource_guides,omitempty"`
	PatientViewed     bool                             `json:"-"`
}

func (d *TreatmentPlan) IsReadyForPatient() bool {
	switch d.Status {
	case TPStatusActive, TPStatusInactive:
		return true
	}

	return false
}

func (d *TreatmentPlan) IsActive() bool {
	switch d.Status {
	case TPStatusActive, TPStatusSubmitted, TPStatusRXStarted:
		return true
	}
	return false
}

func ActiveTreatmentPlanStates() []TreatmentPlanStatus {
	return []TreatmentPlanStatus{TPStatusActive, TPStatusSubmitted, TPStatusRXStarted}
}

func InactiveTreatmentPlanStates() []TreatmentPlanStatus {
	return []TreatmentPlanStatus{TPStatusInactive}
}

func (d *TreatmentPlan) InDraftMode() bool {
	return d.Status == TPStatusDraft
}

const (
	TPParentTypeTreatmentPlan        = "TREATMENT_PLAN"
	TPParentTypePatientVisit         = "PATIENT_VISIT"
	TPContentSourceTypeFTP           = "FAVORITE_TREATMENT_PLAN"
	TPContentSourceTypeTreatmentPlan = "TREATMENT_PLAN"
)

// TreatmentPlanParent keeps track of the parent (either patient visit or previous treatment plan)
// so that we know how the treatment plan came into existence
type TreatmentPlanParent struct {
	ParentID     encoding.ObjectID `json:"parent_id"`
	ParentType   string            `json:"parent_type"`
	CreationDate time.Time         `json:"parent_creation_date"`
}

// TreatmentPlanContentSource keeps track of the source of the content
// for the treatment plan, given that doctor can start fresh with an empty treatment plan,
// from a previous treatment plan or from a favorite treatment plan when generating one for a patient
// Note that we indicate that the doctor started with an empty treatment plan by having nil for the
// content source in the treatment plan object.
// We also keep track of whether or not the treatment plan has deviated from the content source via the
// has_deviated flag
type TreatmentPlanContentSource struct {
	ID          encoding.ObjectID `json:"content_source_id"`
	Type        string            `json:"content_source_type"`
	HasDeviated bool              `json:"has_deviated"`
}

func (d *TreatmentPlan) Equals(other *TreatmentPlan) bool {
	if d == nil && other == nil {
		return true
	} else if d == nil || other == nil {
		return false
	}

	if !d.TreatmentList.Equals(other.TreatmentList) {
		return false
	}

	if !d.RegimenPlan.Equals(other.RegimenPlan) {
		return false
	}

	return true
}

type TreatmentList struct {
	Treatments []*Treatment `json:"treatments,omitempty"`
	Status     string       `json:"status,omitempty"`
}

func (t *TreatmentList) Equals(other *TreatmentList) bool {
	if t == nil || other == nil {
		return false
	}

	if len(t.Treatments) != len(other.Treatments) {
		return false
	}

	for i, treatment := range t.Treatments {
		if !treatment.Equals(other.Treatments[i]) {
			return false
		}
	}

	return true
}

type RefillRequestItem struct {
	ID                        int64             `json:"id,string"`
	RxRequestQueueItemID      int64             `json:"-"`
	ReferenceNumber           string            `json:"-"`
	PharmacyRxReferenceNumber string            `json:"-"`
	ApprovedRefillAmount      int64             `json:"approved_refill,string,omitempty"`
	ErxPatientID              int64             `json:"-"`
	PrescriptionID            int64             `json:"-"`
	PatientAddedForRequest    bool              `json:"-"`
	RequestDateStamp          time.Time         `json:"requested_date"`
	ClinicianID               int64             `json:"-"`
	Patient                   *Patient          `json:"patient,omitempty"`
	RequestedRefillAmount     string            `json:"requested_refill_amount,omitempty"`
	RequestedPrescription     *Treatment        `json:"requested_prescription,omitempty"`
	DispensedPrescription     *Treatment        `json:"dispensed_prescription"`
	Doctor                    *Doctor           `json:"-"`
	TreatmentPlanID           encoding.ObjectID `json:"treatment_plan_id,string,omitempty"`
	RxHistory                 []StatusEvent     `json:"refill_rx_history,omitempty"`
	Comments                  string            `json:"comments,omitempty"`
	DenialReason              string            `json:"denial_reason,omitempty"`
}

type DoctorTreatmentTemplate struct {
	ID        encoding.ObjectID `json:"id,omitempty"`
	Name      string            `json:"name"`
	Treatment *Treatment        `json:"treatment"`
	Status    string            `json:"-"`
}

const (
	StateAdded    = "added"
	StateDeleted  = "deleted"
	StateModified = "modified"
)

type DoctorInstructionItem struct {
	ID       encoding.ObjectID `json:"id,omitempty"`
	ParentID encoding.ObjectID `json:"parent_id,omitempty"`
	Text     string            `json:"text"`
	Selected bool              `json:"selected,omitempty"`
	State    string            `json:"state,omitempty"`
	Status   string            `json:"-"`
}

func (d *DoctorInstructionItem) Equals(other *DoctorInstructionItem) bool {
	if d == nil || other == nil {
		return false
	}

	return d.Text == other.Text
}

type RegimenSection struct {
	ID    encoding.ObjectID        `json:"id,omitempty"`
	Name  string                   `json:"regimen_name"`
	Steps []*DoctorInstructionItem `json:"regimen_steps"`
}

type RegimenPlan struct {
	TreatmentPlanID encoding.ObjectID        `json:"treatment_plan_id,omitempty"`
	Sections        []*RegimenSection        `json:"regimen_sections"`
	AllSteps        []*DoctorInstructionItem `json:"all_regimen_steps,omitempty"`
	Title           string                   `json:"title,omitempty"`
	Status          string                   `json:"status,omitempty"`
}

func (r *RegimenPlan) Equals(other *RegimenPlan) bool {
	if r == nil && other == nil {
		return true
	} else if r == nil || other == nil {
		return false
	}

	// only compare regimen sections with atleast one step in them, because
	// the client currently sends regimen sections with no steps in them
	// making it harder to truly compare the contents of two regimen plans.
	rRegimenSections := getRegimenSectionsWithAtleastOneStep(r)
	otherRegimenSections := getRegimenSectionsWithAtleastOneStep(other)

	if len(rRegimenSections) != len(otherRegimenSections) {
		return false
	}

	// the ordering of the regimen sections and its steps have to be
	// exactly the same for the regimen plan to be considered equal
	for i, regimenSection := range rRegimenSections {
		if regimenSection.Name != otherRegimenSections[i].Name {
			return false
		}

		if len(regimenSection.Steps) != len(otherRegimenSections[i].Steps) {
			return false
		}

		for j, regimenStep := range regimenSection.Steps {
			if !regimenStep.Equals(otherRegimenSections[i].Steps[j]) {
				return false
			}
		}
	}

	return true
}

func getRegimenSectionsWithAtleastOneStep(r *RegimenPlan) []*RegimenSection {
	regimenSections := make([]*RegimenSection, 0, len(r.Sections))
	for _, regimenSection := range r.Sections {
		if len(regimenSection.Steps) > 0 {
			regimenSections = append(regimenSections, regimenSection)
		}
	}
	return regimenSections
}

type StatusEvent struct {
	ItemID            int64     `json:"-"`
	PrescriptionID    int64     `json:"-"`
	Status            string    `json:"status,omitempty"`
	InternalStatus    string    `json:"-"`
	StatusTimestamp   time.Time `json:"status_timestamp,omitempty"`
	ReportedTimestamp time.Time `json:"-"`
	StatusDetails     string    `json:"status_details,omitempty"`
}

type DrugDetails struct {
	ID                int64
	Name              string
	NDC               string
	GenericName       string
	Route             string
	Form              string
	ImageURL          string
	OtherNames        string
	Description       string
	Tips              []string
	Warnings          []string
	CommonSideEffects []string
}

type Notification struct {
	ID              int64
	UID             string // Unique ID scoped to the patient.
	Timestamp       time.Time
	Expires         *time.Time
	Dismissible     bool
	DismissOnAction bool
	Priority        int
	Data            Typed
}

type HealthLogItem struct {
	ID        int64
	PatientID PatientID
	UID       string // Unique ID scoped to the patient.
	Timestamp time.Time
	Data      Typed
}

type Media struct {
	ID         int64
	Uploaded   time.Time
	UploaderID int64
	URL        string
	Mimetype   string
}

type Person struct {
	ID       int64
	RoleType string
	RoleID   int64

	Patient *Patient
	Doctor  *Doctor
}

type CommunicationPreference struct {
	CommunicationType
	ID           int64
	AccountID    int64
	CreationDate time.Time
	Status       string
}

type SnoozeConfig struct {
	AccountID int64
	StartHour int
	NumHours  int
}

type PushConfigData struct {
	ID              int64
	AccountID       int64
	DeviceToken     string
	PushEndpoint    string
	Platform        Platform
	PlatformVersion string
	AppType         string
	AppEnvironment  string
	AppVersion      string
	DeviceModel     string
	Device          string
	DeviceID        string
	CreationDate    time.Time
}

type ResourceGuideSection struct {
	ID      int64  `json:"id,string"`
	Ordinal int    `json:"ordinal"`
	Title   string `json:"title"`
}

type ResourceGuide struct {
	ID        int64       `json:"id,string"`
	SectionID int64       `json:"section_id,string"`
	Ordinal   int         `json:"ordinal"`
	Title     string      `json:"title"`
	PhotoURL  string      `json:"photo_url"`
	Layout    interface{} `json:"layout"`
	Active    bool        `json:"active"`
	Tag       string      `json:"tag"`
}

type Account struct {
	ID               int64     `json:"id,string"`
	Role             string    `json:"role"`
	Email            string    `json:"email"`
	Registered       time.Time `json:"registered"`
	TwoFactorEnabled bool      `json:"two_factor_enabled"`
	AccountCode      *uint64
}

type AccountDevice struct {
	AccountID    int64     `json:"account_id,string"`
	DeviceID     string    `json:"device_id"`
	Verified     bool      `json:"verified"`
	VerifiedTime time.Time `json:"verified_time"`
	Created      time.Time `json:"created"`
}

type MedicalLicense struct {
	ID         int64                `json:"id,string"`
	DoctorID   int64                `json:"doctor_id,string"`
	State      string               `json:"state"`
	Number     string               `json:"number"`
	Expiration *encoding.Date       `json:"expiration,omitempty"`
	Status     MedicalLicenseStatus `json:"status"`
}

type BankAccount struct {
	ID                int64
	AccountID         int64
	StripeRecipientID string
	Created           time.Time
	Default           bool
	Verified          bool
	VerifyAmount1     int
	VerifyAmount2     int
	VerifyTransfer1ID string
	VerifyTransfer2ID string
	VerifyExpires     time.Time
}

type DoctorSearchResult struct {
	DoctorID  int64  `json:"doctor_id,string"`
	AccountID int64  `json:"account_id,string"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

type CareProviderProfile struct {
	AccountID           int64     `json:"account_id,string"`
	FullName            string    `json:"full_name"`
	WhySpruce           string    `json:"why_spruce"`
	Qualifications      string    `json:"qualifications"`
	UndergraduateSchool string    `json:"undergraduate_school"`
	GraduateSchool      string    `json:"graduate_school"`
	MedicalSchool       string    `json:"medical_school"`
	Residency           string    `json:"residency"`
	Fellowship          string    `json:"fellowship"`
	Experience          string    `json:"experience"`
	Created             time.Time `json:"created"`
	Modified            time.Time `json:"modified"`
}

type MedicalRecord struct {
	ID         int64               `json:"id,string"`
	PatientID  PatientID           `json:"patient_id"`
	Status     MedicalRecordStatus `json:"status"`
	Error      string              `json:"error,omitempty"`
	StorageURL string              `json:"storage_url"`
	Requested  time.Time           `json:"requested"`
	Completed  *time.Time          `json:"completed,omitempty"`
}

type AnalyticsReport struct {
	ID             int64     `json:"id,string"`
	OwnerAccountID int64     `json:"owner_account_id,string"`
	Name           string    `json:"name"`
	Query          string    `json:"query"`
	Presentation   string    `json:"presentation"`
	Created        time.Time `json:"created"`
	Modified       time.Time `json:"modified"`
}

type AccountGroup struct {
	ID          int64    `json:"id,string"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions,omitempty"`
}

type PatientCaseFeedItem struct {
	DoctorID         int64     `json:"doctor_id,string"`
	PatientID        PatientID `json:"patient_id"`
	PatientFirstName string    `json:"patient_first_name"`
	PatientLastName  string    `json:"patient_last_name"`
	CaseID           int64     `json:"case_id,string"`
	PathwayTag       string    `json:"pathway_id"`
	PathwayName      string    `json:"pathway_name,string"`
	LastVisitID      int64     `json:"last_visit_id,string"`
	LastVisitTime    time.Time `json:"last_visit_time"`
	LastVisitDoctor  string    `json:"last_visit_doctor"`
}

type VersionedQuestion struct {
	ID                 int64
	QuestionTag        string
	ParentQuestionID   *int64
	Required           bool
	FormattedFieldTags string
	ToAlert            bool
	TextHasTokens      bool
	LanguageID         int64
	Version            int64
	QuestionText       string
	SubtextText        string
	SummaryText        string
	AlertText          string
	QuestionType       string
}

type VersionedAnswer struct {
	ID                int64
	AnswerTag         string
	ToAlert           bool
	Ordering          int64
	QuestionID        int64
	LanguageID        int64
	AnswerText        string
	AnswerSummaryText string
	AnswerType        string
	Status            string
	ClientData        []byte
}

type VersionedAdditionalQuestionField struct {
	ID         int64
	QuestionID int64
	JSON       []byte
	LanguageID int64
}

type VersionedPhotoSlot struct {
	ID         int64
	QuestionID int64
	Required   bool
	Status     string
	Ordering   int64
	LanguageID int64
	Name       string
	Type       string
	ClientData []byte
}

func (vps VersionedPhotoSlot) String() string {
	return fmt.Sprintf("{ID: %v, QuestionID: %v, Required: %v, Status: %v, Ordering: %v, LanguageID: %v, Name: %v, Type %v, ClientData: %v}", vps.ID, vps.QuestionID, vps.Required, vps.Status, vps.Ordering, vps.LanguageID, vps.Name, vps.Type, vps.ClientData)
}

type FTPMembership struct {
	ID                   int64
	DoctorFavoritePlanID int64
	DoctorID             int64
	ClinicalPathwayID    int64
	CreatorID            *int64
}
