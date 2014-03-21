package common

import (
	"carefront/libs/pharmacy"
	"time"
)

type PhoneInformation struct {
	Phone     string `json:"phone,omitempty"`
	PhoneType string `json:"phone_type,omitempty"`
}

type Patient struct {
	PatientId         *ObjectId              `json:"id,omitempty"`
	IsUnlinked        bool                   `json:"is_unlinked,omitempty"`
	FirstName         string                 `json:"first_name,omitempty"`
	LastName          string                 `json:"last_name,omiempty"`
	MiddleName        string                 `json:"middle_name,omitempty"`
	Suffix            string                 `json:"suffix,omitempty"`
	Prefix            string                 `json:"prefix,omitempty"`
	Dob               time.Time              `json:"dob,omitempty"`
	Email             string                 `json:"email,omitempty"`
	Gender            string                 `json:"gender,omitempty"`
	ZipCode           string                 `json:"zip_code,omitempty"`
	City              string                 `json:"city,omitempty"`
	State             string                 `json:"state,omitempty"`
	PhoneNumbers      []*PhoneInformation    `json:"phone_numbers,omitempty"`
	Status            string                 `json:"-"`
	AccountId         *ObjectId              `json:"-"`
	ERxPatientId      *ObjectId              `json:"-"`
	PaymentCustomerId string                 `json:"-"`
	Pharmacy          *pharmacy.PharmacyData `json:"pharmacy,omitempty"`
	PatientAddress    *Address               `json:"address,omitempty"`
}

type Card struct {
	Id             *ObjectId `json:"id,omitempty"`
	ThirdPartyId   string    `json:"third_party_id"`
	Fingerprint    string    `json:"fingerprint"`
	Token          string    `json:"token,omitempty"`
	Type           string    `json:"type"`
	ExpMonth       int64     `json:"exp_month"`
	ExpYear        int64     `json:"exp_year"`
	Last4          int64     `json:"last4,string"`
	Label          string    `json:"label,omitempty"`
	BillingAddress *Address  `json:"address,omitempty"`
	IsDefault      bool      `json:"is_default,omitempty"`
}

type Doctor struct {
	DoctorId            *ObjectId `json:"id,omitempty"`
	FirstName           string    `json:"first_name,omitempty"`
	LastName            string    `json:"last_name,omitempty"`
	Dob                 time.Time `json:"-"`
	Gender              string    `json:"-"`
	Status              string    `json:"-"`
	AccountId           *ObjectId `json:"-"`
	CellPhone           string    `json:"phone"`
	ThumbnailUrl        string    `json:"thumbnail_url,omitempty"`
	DoseSpotClinicianId int64     `json:"-"`
}

type PatientVisit struct {
	PatientVisitId    *ObjectId `json:"patient_visit_id,omitempty"`
	PatientId         *ObjectId `json:"patient_id,omitempty"`
	CreationDate      time.Time `json:"creation_date,omitempty"`
	SubmittedDate     time.Time `json:"submitted_date,omitempty"`
	ClosedDate        time.Time `json:"closed_date,omitempty"`
	HealthConditionId *ObjectId `json:"health_condition_id,omitempty"`
	Status            string    `json:"status,omitempty"`
	LayoutVersionId   *ObjectId `json:"layout_version_id,omitempty"`
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
	AnswerIntakeId    *ObjectId       `json:"answer_id,omitempty"`
	QuestionId        *ObjectId       `json:"-"`
	RoleId            *ObjectId       `json:"-"`
	Role              string          `json:"-"`
	ContextId         *ObjectId       `json:"-"`
	ParentQuestionId  *ObjectId       `json:"-"`
	ParentAnswerId    *ObjectId       `json:"-"`
	PotentialAnswerId *ObjectId       `json:"potential_answer_id,omitempty"`
	PotentialAnswer   string          `json:"potential_answer,omitempty"`
	AnswerSummary     string          `json:"potential_answer_summary,omitempty"`
	LayoutVersionId   *ObjectId       `json:"-"`
	SubAnswers        []*AnswerIntake `json:"answers,omitempty"`
	AnswerText        string          `json:"answer_text,omitempty"`
	ObjectUrl         string          `json:"object_url,omitempty"`
	StorageBucket     string          `json:"-"`
	StorageKey        string          `json:"-"`
	StorageRegion     string          `json:"-"`
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
	Id               *ObjectId         `json:"treatment_plan_id,omitempty"`
	PatientId        *ObjectId         `json:"patient_id,omitempty"`
	PatientInfo      *Patient          `json:"patient,omitempty"`
	PatientVisitId   *ObjectId         `json:"patient_visit_id,omitempty"`
	Status           string            `json:"status,omitempty"`
	CreationDate     *time.Time        `json:"creation_date,omitempty"`
	SentDate         *time.Time        `json:"sent_date,omitempty"`
	Treatments       []*Treatment      `json:"treatments,omitempty"`
	Title            string            `json:"title,omitempty"`
	DiagnosisSummary *DiagnosisSummary `json:"diagnosis_summary,omitempty"`
	RegimenPlan      *RegimenPlan      `json:"regimen_plan,omitempty"`
	Advice           *Advice           `json:"advice,omitempty"`
	Followup         *FollowUp         `json:"follow_up,omitempty"`
}

type RefillRequestItem struct {
	Id                               int64          `json:"id,string"`
	RxRequestQueueItemId             int64          `json:"-"`
	ReferenceNumber                  string         `json:"-"`
	PharmacyRxReferenceNumber        string         `json:"-"`
	RequestedDrugDescription         string         `json:"requested_drug_name"`
	RequestedRefillAmount            string         `json:"requested_refill"`
	ApprovedRefillAmount             int64          `json:"approved_refill,string,omitempty"`
	RequestedDispense                string         `json:"requested_dispense_value"`
	RequestedDispenseUnitDescription string         `json:"requested_dispense_unit_description,omitempty"`
	ErxPatientId                     int64          `json:"-"`
	PatientAddedForRequest           bool           `json:"-"`
	RequestDateStamp                 time.Time      `json:"requested_date"`
	ClinicianId                      int64          `json:"-"`
	Patient                          *Patient       `json:"patient,omitempty"`
	RequestedPrescription            *Treatment     `json:"requested_prescription,omitempty"`
	DispensedPrescription            *Treatment     `json:"dispensed_prescription"`
	Doctor                           *Doctor        `json:"-"`
	TreatmentPlanId                  int64          `json:"treatment_plan_id,omitempty"`
	RxHistory                        []*StatusEvent `json:"erx_history,omitempty"`
	Status                           string         `json:"status,omitempty"`
	Comments                         string         `json:"comments,omitempty"`
	DenialReason                     string         `json:"denial_reason,omitempty"`
}

type DoctorTreatmentTemplate struct {
	Id        *ObjectId  `json:"id,omitempty"`
	Name      string     `json:"name"`
	Treatment *Treatment `json:"treatment"`
	Status    string     `json:"-"`
}

const (
	STATE_ADDED    = "added"
	STATE_MODIFIED = "modified"
	STATE_DELETED  = "deleted"
)

type DoctorInstructionItem struct {
	Id       *ObjectId `json:"id,omitempty"`
	Text     string    `json:"text"`
	Selected bool      `json:"selected,omitempty"`
	State    string    `json:"state,omitempty"`
	Status   string    `json:"-"`
}

type RegimenSection struct {
	RegimenName  string                   `json:"regimen_name"`
	RegimenSteps []*DoctorInstructionItem `json:"regimen_steps"`
}

type RegimenPlan struct {
	TreatmentPlanId *ObjectId                `json:"treatment_plan_id,omitempty"`
	PatientVisitId  *ObjectId                `json:"patient_visit_id,omitempty"`
	RegimenSections []*RegimenSection        `json:"regimen_sections"`
	AllRegimenSteps []*DoctorInstructionItem `json:"all_regimen_steps,omitempty"`
	Title           string                   `json:"title,omitempty"`
}

type FollowUp struct {
	TreatmentPlanId *ObjectId `json:"treatment_plan_id,omitempty"`
	FollowUpValue   int64     `json:"follow_up_value,string, omitempty"`
	FollowUpUnit    string    `json:"follow_up_unit,omitempty"`
	FollowUpTime    time.Time `json:"follow_up_time,omitempty"`
	Title           string    `json:"title,omitempty"`
}

type Advice struct {
	AllAdvicePoints      []*DoctorInstructionItem `json:"all_advice_points,omitempty"`
	SelectedAdvicePoints []*DoctorInstructionItem `json:"selected_advice_points,omitempty"`
	PatientVisitId       *ObjectId                `json:"patient_visit_id,omitempty"`
	TreatmentPlanId      *ObjectId                `json:"treatment_plan_id,omitempty"`
	Title                string                   `json:"title,omitempty"`
}

type DiagnosisSummary struct {
	Type    string `json:"type"`
	Summary string `json:"text"`
	Title   string `json:"title,omitempty"`
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
}

type StatusEvent struct {
	TreatmentId          int64     `json:"-"`
	PrescriptionId       int64     `json:"erx_id,string,omitempty"`
	Status               string    `json:"erx_status,omitempty"`
	StatusTimestamp      time.Time `json:"erx_status_timestamp,omitempty"`
	ReportedTimestamp    time.Time `json:"reported_timestamp,omitempty"`
	StatusDetails        string    `json:"erx_status_details,omitempty"`
	ErxRefillRequestId   int64     `json:"-"`
	RxRequestQueueItemId int64     `json:"-"`
}
