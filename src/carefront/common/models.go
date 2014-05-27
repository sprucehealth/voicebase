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
	PhoneNumbers      []*PhoneInformation    `json:"phone_numbers,omitempty"`
	Status            string                 `json:"-"`
	AccountId         encoding.ObjectId      `json:"account_id,omitempty"`
	ERxPatientId      encoding.ObjectId      `json:"-"`
	PaymentCustomerId string                 `json:"-"`
	Pharmacy          *pharmacy.PharmacyData `json:"pharmacy,omitempty"`
	PatientAddress    *Address               `json:"address,omitempty"`
	PersonId          int64                  `json:"person_id"`
	PromptStatus      PatientPromptStatus    `json:"prompt_status"`
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
}

type PatientVisit struct {
	PatientVisitId    encoding.ObjectId `json:"patient_visit_id,omitempty"`
	PatientId         encoding.ObjectId `json:"patient_id,omitempty"`
	CreationDate      time.Time         `json:"creation_date,omitempty"`
	SubmittedDate     time.Time         `json:"submitted_date,omitempty"`
	ClosedDate        time.Time         `json:"closed_date,omitempty"`
	HealthConditionId encoding.ObjectId `json:"health_condition_id,omitempty"`
	Status            string            `json:"status,omitempty"`
	LayoutVersionId   encoding.ObjectId `json:"layout_version_id,omitempty"`
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

type AnswerIntake struct {
	AnswerIntakeId    encoding.ObjectId `json:"answer_id,omitempty"`
	QuestionId        encoding.ObjectId `json:"-"`
	RoleId            encoding.ObjectId `json:"-"`
	Role              string            `json:"-"`
	ContextId         encoding.ObjectId `json:"-"`
	ParentQuestionId  encoding.ObjectId `json:"-"`
	ParentAnswerId    encoding.ObjectId `json:"-"`
	PotentialAnswerId encoding.ObjectId `json:"potential_answer_id,omitempty"`
	PotentialAnswer   string            `json:"potential_answer,omitempty"`
	AnswerSummary     string            `json:"potential_answer_summary,omitempty"`
	LayoutVersionId   encoding.ObjectId `json:"-"`
	SubAnswers        []*AnswerIntake   `json:"answers,omitempty"`
	AnswerText        string            `json:"answer_text,omitempty"`
	ObjectUrl         string            `json:"object_url,omitempty"`
	StorageBucket     string            `json:"-"`
	StorageKey        string            `json:"-"`
	StorageRegion     string            `json:"-"`
	ToAlert           bool              `json:"-"`
}

type PatientCareProviderAssignment struct {
	Id           int64
	ProviderRole string
	ProviderId   int64
	Status       string
}

type PatientCareProviderGroup struct {
	Id           int64
	PatientId    int64
	CreationDate time.Time
	ModifiedDate time.Time
	Status       string
	Assignments  []*PatientCareProviderAssignment
}

type TreatmentPlan struct {
	Id               encoding.ObjectId `json:"treatment_plan_id,omitempty"`
	PatientId        encoding.ObjectId `json:"patient_id,omitempty"`
	PatientInfo      *Patient          `json:"patient,omitempty"`
	PatientVisitId   encoding.ObjectId `json:"patient_visit_id,omitempty"`
	Status           string            `json:"status,omitempty"`
	CreationDate     *time.Time        `json:"creation_date,omitempty"`
	SentDate         *time.Time        `json:"sent_date,omitempty"`
	TreatmentList    *TreatmentList    `json:"treatment_list,omitempty"`
	Title            string            `json:"title,omitempty"`
	DiagnosisSummary *DiagnosisSummary `json:"diagnosis_summary,omitempty"`
	RegimenPlan      *RegimenPlan      `json:"regimen_plan,omitempty"`
	Advice           *Advice           `json:"advice,omitempty"`
	Followup         *FollowUp         `json:"follow_up,omitempty"`
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
	Id                              encoding.ObjectId `json:"id,omitempty"`
	PatientVisitId                  encoding.ObjectId `json:"patient_visit_id,omitempty"`
	DoctorFavoriteTreatmentPlanId   encoding.ObjectId `json:"dr_favorite_treatment_plan_id"`
	DoctorFavoriteTreatmentPlanName string            `json:"dr_favorite_treatment_plan_name,omitempty"`
	TreatmentList                   *TreatmentList    `json:"treatment_list"`
	RegimenPlan                     *RegimenPlan      `json:"regimen_plan,omitempty"`
	Advice                          *Advice           `json:"advice,omitempty"`
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

	return d.Text == other.Text && d.ParentId.Int64() == other.ParentId.Int64()
}

type RegimenSection struct {
	RegimenName  string                   `json:"regimen_name"`
	RegimenSteps []*DoctorInstructionItem `json:"regimen_steps"`
}

type RegimenPlan struct {
	TreatmentPlanId encoding.ObjectId        `json:"treatment_plan_id,omitempty"`
	PatientVisitId  encoding.ObjectId        `json:"patient_visit_id,omitempty"`
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
	PatientVisitId       encoding.ObjectId        `json:"patient_visit_id,omitempty"`
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

type DiagnosisSummary struct {
	Type            string `json:"type"`
	Summary         string `json:"text"`
	Title           string `json:"title,omitempty"`
	UpdatedByDoctor bool   `json:"-"`
}

type QuestionInfo struct {
	Id                 int64
	QuestionTag        string
	Title              string
	Type               string
	Summary            string
	SubText            string
	ParentQuestionId   int64
	AdditionalFields   map[string]string
	FormattedFieldTags string
	Required           bool
	ToAlert            bool
	AlertFormattedText string
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

type Conversation struct {
	Id                int64
	Time              time.Time
	Title             string
	TopicId           int64
	MessageCount      int
	CreatorId         int64
	OwnerId           int64
	LastParticipantId int64
	LastMessageTime   time.Time
	Unread            bool

	Messages     []*ConversationMessage
	Participants map[int64]*Person
}

type ConversationTopic struct {
	Id      int64
	Title   string
	Ordinal int
	Active  bool
}

type ConversationAttachment struct {
	Id       int64
	ItemType string
	ItemId   int64
}

type ConversationMessage struct {
	Id             int64
	ConversationId int64
	Time           time.Time
	FromId         int64
	Body           string
	Attachments    []*ConversationAttachment
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
