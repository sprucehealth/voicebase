package common

import (
	"carefront/libs/pharmacy"
	"time"
)

type Patient struct {
	PatientId      int64                  `json:"id,omitempty,string"`
	FirstName      string                 `json:"first_name,omitempty"`
	LastName       string                 `json:"last_name,omiempty"`
	Dob            time.Time              `json:"dob,omitempty"`
	Gender         string                 `json:"gender,omitempty"`
	ZipCode        string                 `json:"zip_code,omitempty"`
	City           string                 `json:"city,omitempty"`
	State          string                 `json:"state,omitempty"`
	Phone          string                 `json:"phone,omitempty"`
	PhoneType      string                 `json:"-"`
	Status         string                 `json:"-"`
	AccountId      int64                  `json:"-"`
	ERxPatientId   int64                  `json:"-"`
	Pharmacy       *pharmacy.PharmacyData `json:"pharmacy,omitempty"`
	PatientAddress *Address               `json:"address,omitempty"`
}

type Doctor struct {
	DoctorId     int64     `json:"id,string,omitempty"`
	FirstName    string    `json:"first_name,omitempty"`
	LastName     string    `json:"last_name,omitempty"`
	Dob          time.Time `json:"-"`
	Gender       string    `json:"-"`
	Status       string    `json:"-"`
	AccountId    int64     `json:"-"`
	CellPhone    string    `json:"phone"`
	ThumbnailUrl string    `json:"thumbnail_url,omitempty"`
}

type PatientVisit struct {
	PatientVisitId    int64     `json:"patient_visit_id,string,omitempty"`
	PatientId         int64     `json:"patient_id,string,omitempty"`
	CreationDate      time.Time `json:"creation_date,omitempty"`
	SubmittedDate     time.Time `json:"submitted_date,omitempty"`
	ClosedDate        time.Time `json:"closed_date,omitempty"`
	HealthConditionId int64     `json:"health_condition_id,omitempty,string"`
	Status            string    `json:"status,omitempty"`
	LayoutVersionId   int64     `json:"layout_version_id,omitempty,string"`
}

type Address struct {
	AddressLine1 string `json:"address_line_1"`
	AddressLine2 string `json:"address_line_2,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state"`
	ZipCode      string `json:"zip_code"`
}

type AnswerIntake struct {
	AnswerIntakeId    int64           `json:"answer_id,string,omitempty"`
	QuestionId        int64           `json:"-"`
	RoleId            int64           `json:"-"`
	Role              string          `json:"-"`
	ContextId         int64           `json:"-"`
	ParentQuestionId  int64           `json:"-"`
	ParentAnswerId    int64           `json:"-"`
	PotentialAnswerId int64           `json:"potential_answer_id,string,omitempty"`
	PotentialAnswer   string          `json:"potential_answer,omitempty"`
	AnswerSummary     string          `json:"potential_answer_summary,omitempty"`
	LayoutVersionId   int64           `json:"-"`
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
	Id               int64             `json:"treatment_plan_id,string,omitempty"`
	PatientId        int64             `json:"patient_id,string,omitempty"`
	PatientInfo      *Patient          `json:"patient,omitempty"`
	PatientVisitId   int64             `json:"patient_visit_id,string,omitempty"`
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

type Treatment struct {
	Id                        int64                    `json:"treatment_id,string,omitempty"`
	DoctorFavoriteTreatmentId int64                    `json:"dr_favorite_treatment_id,string,omitempty"`
	PrescriptionId            int64                    `json:"erx_id,string,omitempty"`
	ErxMedicationId           int64                    `json:"-"`
	PrescriptionStatus        string                   `json:"erx_status,omitempty"`
	PharmacyLocalId           int64                    `json:"-"`
	StatusDetails             string                   `json:"erx_status_details,omitempty"`
	TreatmentPlanId           int64                    `json:"treatment_plan_id,string,omitempty"`
	PatientVisitId            int64                    `json:"patient_visit_id,string,omitempty"`
	PatientId                 int64                    `json:"-"`
	DrugDBIds                 map[string]string        `json:"drug_db_ids,omitempty"`
	DrugInternalName          string                   `json:"drug_internal_name,omitempty"`
	DrugName                  string                   `json:"drug_name"`
	DrugRoute                 string                   `json:"drug_route,omitempty"`
	DrugForm                  string                   `json:"drug_form,omitempty"`
	DosageStrength            string                   `json:"dosage_strength,omitempty"`
	DispenseValue             int64                    `json:"dispense_value,string,omitempty"`
	DispenseUnitId            int64                    `json:"dispense_unit_id,string,omitempty"`
	DispenseUnitDescription   string                   `json:"dispense_unit_description,omitempty"`
	NumberRefills             int64                    `json:"refills,string,omitempty"`
	SubstitutionsAllowed      bool                     `json:"substitutions_allowed,omitempty"`
	DaysSupply                int64                    `json:"days_supply,string,omitempty"`
	PharmacyNotes             string                   `json:"pharmacy_notes,omitempty"`
	PatientInstructions       string                   `json:"patient_instructions,omitempty"`
	CreationDate              *time.Time               `json:"creation_date,omitempty"`
	TransmissionErrorDate     *time.Time               `json:"error_date,omitempty"`
	ErxSentDate               *time.Time               `json:"erx_sent_date,omitempty"`
	Status                    string                   `json:"-"`
	OTC                       bool                     `json:"otc,omitempty"`
	SupplementalInstructions  []*DoctorInstructionItem `json:"supplemental_instructions,omitempty"`
}

type DoctorFavoriteTreatment struct {
	Id                 int64      `json:"id,string"`
	Name               string     `json:"name"`
	FavoritedTreatment *Treatment `json:"treatment"`
	Status             string     `json:"-"`
}

const (
	STATE_ADDED    = "added"
	STATE_MODIFIED = "modified"
	STATE_DELETED  = "deleted"
)

type DoctorInstructionItem struct {
	Id       int64  `json:"id,string"`
	Text     string `json:"text"`
	Selected bool   `json:"selected,omitempty"`
	State    string `json:"state,omitempty"`
	Status   string `json:"-"`
}

type RegimenSection struct {
	RegimenName  string                   `json:"regimen_name"`
	RegimenSteps []*DoctorInstructionItem `json:"regimen_steps"`
}

type RegimenPlan struct {
	TreatmentPlanId int64                    `json:"treatment_plan_id,string,omitempty"`
	PatientVisitId  int64                    `json:"patient_visit_id,string,omitempty"`
	RegimenSections []*RegimenSection        `json:"regimen_sections"`
	AllRegimenSteps []*DoctorInstructionItem `json:"all_regimen_steps,omitempty"`
	Title           string                   `json:"title,omitempty"`
}

type FollowUp struct {
	TreatmentPlanId int64     `json:"treatment_plan_id,string,omitempty"`
	FollowUpValue   int64     `json:"follow_up_value,string,omitempty"`
	FollowUpUnit    string    `json:"follow_up_unit,omitempty"`
	FollowUpTime    time.Time `json:"follow_up_time,omitempty"`
	Title           string    `json:"title,omitempty"`
}

type Advice struct {
	AllAdvicePoints      []*DoctorInstructionItem `json:"all_advice_points,omitempty"`
	SelectedAdvicePoints []*DoctorInstructionItem `json:"selected_advice_points,omitempty"`
	PatientVisitId       int64                    `json:"patient_visit_id,string,omitempty"`
	TreatmentPlanId      int64                    `json:"treatment_plan_id,string,omitempty"`
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
