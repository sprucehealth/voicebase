package common

import (
	"carefront/app_url"
	"carefront/encoding"
	"carefront/libs/pharmacy"
	"time"
)

const (
	AttachmentTypePhoto         = "photo"
	AttachmentTypeTreatmentPlan = "treatment_plan"
)

const (
	ClaimerTypeConversationMessage = "conversation_message"
	ClaimerTypePhotoIntakeSection  = "patient_intake_photo_section"
)

type PhoneInformation struct {
	Phone     string `json:"phone,omitempty"`
	PhoneType string `json:"phone_type,omitempty"`
}

type Patient struct {
	PatientId         encoding.ObjectId      `json:"id,omitempty"`
	IsUnlinked        bool                   `json:"is_unlinked,omitempty"`
	FirstName         string                 `json:"first_name,omitempty"`
	LastName          string                 `json:"last_name,omiempty"`
	MiddleName        string                 `json:"middle_name,omitempty"`
	Suffix            string                 `json:"suffix,omitempty"`
	Prefix            string                 `json:"prefix,omitempty"`
	Dob               encoding.Dob           `json:"dob,omitempty"`
	Email             string                 `json:"email,omitempty"`
	Gender            string                 `json:"gender,omitempty"`
	ZipCode           string                 `json:"zip_code,omitempty"`
	CityFromZipCode   string                 `json:"-"`
	StateFromZipCode  string                 `json:"-"`
	PhoneNumbers      []*PhoneInformation    `json:"phone_numbers,omitempty"`
	Status            string                 `json:"-"`
	AccountId         encoding.ObjectId      `json:"account_id,omitempty"`
	ERxPatientId      encoding.ObjectId      `json:"-"`
	PaymentCustomerId string                 `json:"-"`
	Pharmacy          *pharmacy.PharmacyData `json:"pharmacy,omitempty"`
	PatientAddress    *Address               `json:"address,omitempty"`
	PersonId          int64                  `json:"person_id"`
	PromptStatus      PushPromptStatus       `json:"prompt_status"`
}

type Card struct {
	Id             encoding.ObjectId `json:"id,omitempty"`
	ThirdPartyId   string            `json:"third_party_id"`
	Fingerprint    string            `json:"fingerprint"`
	Token          string            `json:"token,omitempty"`
	Type           string            `json:"type"`
	ExpMonth       int64             `json:"exp_month"`
	ExpYear        int64             `json:"exp_year"`
	Last4          int64             `json:"last4,string"`
	Label          string            `json:"label,omitempty"`
	BillingAddress *Address          `json:"address,omitempty"`
	IsDefault      bool              `json:"is_default,omitempty"`
	CreationDate   time.Time         `json:"creation_date"`
}

type Doctor struct {
	DoctorId            encoding.ObjectId    `json:"id,omitempty"`
	FirstName           string               `json:"first_name,omitempty"`
	LastName            string               `json:"last_name,omitempty"`
	MiddleName          string               `json:"middle_name,omitempty"`
	Prefix              string               `json:"prefix,omitempty"`
	Suffix              string               `json:"suffix,omitempty"`
	Dob                 encoding.Dob         `json:"-"`
	Gender              string               `json:"-"`
	Status              string               `json:"-"`
	AccountId           encoding.ObjectId    `json:"-"`
	CellPhone           string               `json:"phone"`
	LargeThumbnailUrl   *app_url.SpruceAsset `json:"large_thumbnail_url,omitempty"`
	SmallThumbnailUrl   *app_url.SpruceAsset `json:"small_thumbnail_url,omitempty"`
	DoseSpotClinicianId int64                `json:"-"`
	DoctorAddress       *Address             `json:"address,omitempty"`
	PersonId            int64                `json:"person_id"`
	PromptStatus        PushPromptStatus     `json:"prompt_status"`
}

const (
	PVStatusOpen      = "OPEN"
	PVStatusSubmitted = "SUBMITTED"
	PVStatusReviewing = "REVIEWING"
	PVStatusTriaged   = "TRIAGED"
	PVStatusTreated   = "TREATED"
)

type PatientVisit struct {
	PatientVisitId    encoding.ObjectId `json:"patient_visit_id,omitempty"`
	PatientCaseId     encoding.ObjectId `json:"case_id"`
	PatientId         encoding.ObjectId `json:"patient_id,omitempty"`
	CreationDate      time.Time         `json:"creation_date,omitempty"`
	SubmittedDate     time.Time         `json:"submitted_date,omitempty"`
	ClosedDate        time.Time         `json:"closed_date,omitempty"`
	HealthConditionId encoding.ObjectId `json:"health_condition_id,omitempty"`
	Status            string            `json:"status,omitempty"`
	LayoutVersionId   encoding.ObjectId `json:"layout_version_id,omitempty"`
}

const (
	PCStatusUnclaimed   = "UNCLAIMED"
	PCStatusTempClaimed = "TEMP_CLAIMED"
	PCStatusClaimed     = "CLAIMED"
)

type PatientCase struct {
	Id                encoding.ObjectId `json:"case_id"`
	PatientId         encoding.ObjectId `json:"patient_id"`
	HealthConditionId encoding.ObjectId `json:"health_condition_id"`
	CreationDate      time.Time         `json:"creation_date"`
	Status            string            `json:"status"`
}

type Address struct {
	Id           int64  `json:"-"`
	AddressLine1 string `json:"address_line_1"`
	AddressLine2 string `json:"address_line_2,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state"`
	ZipCode      string `json:"zip_code"`
	Country      string `json:"country"`
}

type CareProviderAssignment struct {
	ProviderRole string
	ProviderId   int64
	PatientId    int64
	Status       string
	CreationDate time.Time
}

type PatientCareTeam struct {
	Assignments []*CareProviderAssignment
}

type TreatmentPlan struct {
	Id            encoding.ObjectId `json:"treatment_plan_id,omitempty"`
	PatientCaseId encoding.ObjectId `json:"case_id"`
	PatientId     encoding.ObjectId `json:"patient_id,omitempty"`
	PatientInfo   *Patient          `json:"patient,omitempty"`
	Status        string            `json:"status,omitempty"`
	CreationDate  *time.Time        `json:"creation_date,omitempty"`
	SentDate      *time.Time        `json:"sent_date,omitempty"`
	TreatmentList *TreatmentList    `json:"treatment_list,omitempty"`
	Title         string            `json:"title,omitempty"`
	RegimenPlan   *RegimenPlan      `json:"regimen_plan,omitempty"`
	Advice        *Advice           `json:"advice,omitempty"`
	Followup      *FollowUp         `json:"follow_up,omitempty"`
}

type FavoriteTreatmentPlan struct {
	Id            encoding.ObjectId `json:"id"`
	Name          string            `json:"name"`
	ModifiedDate  time.Time         `json:"modified_date,omitempty"`
	DoctorId      int64             `json:"-"`
	RegimenPlan   *RegimenPlan      `json:"regimen_plan,omitempty"`
	TreatmentList *TreatmentList    `json:"treatment_list,omitempty"`
	Advice        *Advice           `json:"advice,omitempty"`
}

func (f *FavoriteTreatmentPlan) EqualsDoctorTreatmentPlan(treatmentPlan *DoctorTreatmentPlan) bool {
	if f == nil || treatmentPlan == nil {
		return false
	}

	if !f.TreatmentList.Equals(treatmentPlan.TreatmentList) {
		return false
	}

	if !f.RegimenPlan.Equals(treatmentPlan.RegimenPlan) {
		return false
	}

	if !f.Advice.Equals(treatmentPlan.Advice) {
		return false
	}

	return true
}

type DoctorTreatmentPlan struct {
	Id            encoding.ObjectId           `json:"id,omitempty"`
	DoctorId      encoding.ObjectId           `json:"doctor_id,omitempty"`
	CreationDate  time.Time                   `json:"creation_date"`
	TreatmentList *TreatmentList              `json:"treatment_list"`
	RegimenPlan   *RegimenPlan                `json:"regimen_plan,omitempty"`
	Advice        *Advice                     `json:"advice,omitempty"`
	Status        string                      `json:"status,omitempty"`
	Parent        *TreatmentPlanParent        `json:"parent,omitempty"`
	PatientId     int64                       `json:"-"`
	PatientCaseId encoding.ObjectId           `json:"patient_case_id"`
	ContentSource *TreatmentPlanContentSource `json:"content_source,omitempty"`
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
	ParentId     encoding.ObjectId `json:"parent_id"`
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
	ContentSourceId   encoding.ObjectId `json:"content_source_id"`
	ContentSourceType string            `json:"content_source_type"`
	HasDeviated       bool              `json:"has_deviated"`
}

func (d *DoctorTreatmentPlan) Equals(other *DoctorTreatmentPlan) bool {
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

	if !d.Advice.Equals(other.Advice) {
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
	Id                        int64             `json:"id,string"`
	RxRequestQueueItemId      int64             `json:"-"`
	ReferenceNumber           string            `json:"-"`
	PharmacyRxReferenceNumber string            `json:"-"`
	ApprovedRefillAmount      int64             `json:"approved_refill,string,omitempty"`
	ErxPatientId              int64             `json:"-"`
	PrescriptionId            int64             `json:"-"`
	PatientAddedForRequest    bool              `json:"-"`
	RequestDateStamp          time.Time         `json:"requested_date"`
	ClinicianId               int64             `json:"-"`
	Patient                   *Patient          `json:"patient,omitempty"`
	RequestedPrescription     *Treatment        `json:"requested_prescription,omitempty"`
	DispensedPrescription     *Treatment        `json:"dispensed_prescription"`
	Doctor                    *Doctor           `json:"-"`
	TreatmentPlanId           encoding.ObjectId `json:"treatment_plan_id,string,omitempty"`
	RxHistory                 []StatusEvent     `json:"refill_rx_history,omitempty"`
	Comments                  string            `json:"comments,omitempty"`
	DenialReason              string            `json:"denial_reason,omitempty"`
}

type DoctorTreatmentTemplate struct {
	Id        encoding.ObjectId `json:"id,omitempty"`
	Name      string            `json:"name"`
	Treatment *Treatment        `json:"treatment"`
	Status    string            `json:"-"`
}

const (
	STATE_ADDED    = "added"
	STATE_MODIFIED = "modified"
	STATE_DELETED  = "deleted"
)

type DoctorInstructionItem struct {
	Id       encoding.ObjectId `json:"id,omitempty"`
	ParentId encoding.ObjectId `json:"parent_id,omitempty"`
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
	RegimenName  string                   `json:"regimen_name"`
	RegimenSteps []*DoctorInstructionItem `json:"regimen_steps"`
}

type RegimenPlan struct {
	TreatmentPlanId encoding.ObjectId        `json:"treatment_plan_id,omitempty"`
	RegimenSections []*RegimenSection        `json:"regimen_sections"`
	AllRegimenSteps []*DoctorInstructionItem `json:"all_regimen_steps,omitempty"`
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
		if regimenSection.RegimenName != otherRegimenSections[i].RegimenName {
			return false
		}

		if len(regimenSection.RegimenSteps) != len(otherRegimenSections[i].RegimenSteps) {
			return false
		}

		for j, regimenStep := range regimenSection.RegimenSteps {
			if !regimenStep.Equals(otherRegimenSections[i].RegimenSteps[j]) {
				return false
			}
		}
	}

	return true
}

func getRegimenSectionsWithAtleastOneStep(r *RegimenPlan) []*RegimenSection {

	regimenSections := make([]*RegimenSection, 0, len(r.RegimenSections))
	for _, regimenSection := range r.RegimenSections {
		if len(regimenSection.RegimenSteps) > 0 {
			regimenSections = append(regimenSections, regimenSection)
		}
	}
	return regimenSections
}

type FollowUp struct {
	TreatmentPlanId encoding.ObjectId `json:"treatment_plan_id,omitempty"`
	FollowUpValue   int64             `json:"follow_up_value,string, omitempty"`
	FollowUpUnit    string            `json:"follow_up_unit,omitempty"`
	FollowUpTime    time.Time         `json:"follow_up_time,omitempty"`
	Title           string            `json:"title,omitempty"`
	Status          string            `json:"omitempty"`
}

type Advice struct {
	AllAdvicePoints      []*DoctorInstructionItem `json:"all_advice_points,omitempty"`
	SelectedAdvicePoints []*DoctorInstructionItem `json:"selected_advice_points,omitempty"`
	TreatmentPlanId      encoding.ObjectId        `json:"treatment_plan_id,omitempty"`
	Title                string                   `json:"title,omitempty"`
	Status               string                   `json:"status,omitempty"`
}

func (a *Advice) Equals(other *Advice) bool {
	if a == nil && other == nil {
		return true
	} else if a == nil || other == nil {
		return false
	}

	if len(a.SelectedAdvicePoints) != len(other.SelectedAdvicePoints) {
		return false
	}

	for i, advicePoint := range a.SelectedAdvicePoints {
		if !advicePoint.Equals(other.SelectedAdvicePoints[i]) {
			return false
		}
	}

	return true
}

type StatusEvent struct {
	ItemId            int64     `json:"-"`
	PrescriptionId    int64     `json:"-"`
	Status            string    `json:"status,omitempty"`
	InternalStatus    string    `json:"-"`
	StatusTimestamp   time.Time `json:"status_timestamp,omitempty"`
	ReportedTimestamp time.Time `json:"-"`
	StatusDetails     string    `json:"status_details,omitempty"`
}

type DrugDetails struct {
	Name           string
	NDC            string
	ImageURL       string
	Alternative    string
	Description    string
	Warnings       []string
	Precautions    []string
	HowToUse       []string
	SideEffects    []string
	AdverseEffects []string
}

type Notification struct {
	Id              int64
	UID             string // Unique ID scoped to the patient.
	Timestamp       time.Time
	Expires         *time.Time
	Dismissible     bool
	DismissOnAction bool
	Priority        int
	Data            Typed
}

type HealthLogItem struct {
	Id        int64
	PatientId int64
	UID       string // Unique ID scoped to the patient.
	Timestamp time.Time
	Data      Typed
}

type Photo struct {
	Id          int64
	Uploaded    time.Time
	UploaderId  int64
	URL         string
	Mimetype    string
	ClaimerType string
	ClaimerId   int64
}

type Person struct {
	Id       int64
	RoleType string
	RoleId   int64

	Patient *Patient
	Doctor  *Doctor
}

type CaseMessage struct {
	ID          int64
	CaseID      int64
	PersonID    int64
	Time        time.Time
	Body        string
	Attachments []*CaseMessageAttachment
}

type CaseMessageAttachment struct {
	ID       int64
	ItemType string
	ItemID   int64
}

type CaseMessageParticipant struct {
	CaseID   int64
	Unread   bool
	LastRead time.Time
	Person   *Person
}

type CommunicationPreference struct {
	CommunicationType
	Id           int64
	AccountId    int64
	CreationDate time.Time
	Status       string
}

type PushConfigData struct {
	Id           int64
	AccountId    int64
	DeviceToken  string
	PushEndpoint string
	Platform
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
	Id      int64
	Ordinal int
	Title   string
}

type ResourceGuide struct {
	Id        int64
	SectionId int64
	Ordinal   int
	Title     string
	PhotoURL  string
	Layout    interface{}
}

type Account struct {
	ID   int64
	Role string
}
