package common

import (
	"carefront/libs/pharmacy"
	"time"
)

type Patient struct {
	PatientId int64                  `json:"id,omitempty,string"`
	FirstName string                 `json:"first_name,omitempty"`
	LastName  string                 `json:"last_name,omiempty"`
	Dob       time.Time              `json:"dob,omitempty"`
	Gender    string                 `json:"gender,omitempty"`
	ZipCode   string                 `json:"zip_code,omitempty"`
	City      string                 `json:"city,omitempty"`
	State     string                 `json:"state,omitempty"`
	Phone     string                 `json:"phone,omitempty"`
	Status    string                 `json:"-"`
	AccountId int64                  `json:"-"`
	Pharmacy  *pharmacy.PharmacyData `json:"pharmacy,omitempty"`
}

type Doctor struct {
	DoctorId  int64
	FirstName string
	LastName  string
	Dob       time.Time
	Gender    string
	Status    string
	AccountId int64
	CellPhone string
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

type AnswerIntake struct {
	AnswerIntakeId    int64           `json:"answer_id,string,omitempty"`
	QuestionId        int64           `json:"-"`
	RoleId            int64           `json:"-"`
	Role              string          `json:"-"`
	PatientVisitId    int64           `json:"-"`
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
	Id             int64        `json:"treatment_plan_id,string,omitempty"`
	PatientVisitId int64        `json:"patient_visit_id,string,omitempty"`
	Status         string       `json:"status,omitempty"`
	CreationDate   time.Time    `json:"creation_date,omitempty"`
	Treatments     []*Treatment `json:"treatments,omitempty"`
	Title          string       `json:"title,omitempty"`
}

type Treatment struct {
	Id                       int64                    `json:"treatment_id,string,omitempty"`
	TreatmentPlanId          int64                    `json:"treatment_plan_id,string,omitempty"`
	PatientVisitId           int64                    `json:"patient_visit_id,string,omitempty"`
	DrugDBIds                map[string]string        `json:"drug_db_ids,omitempty"`
	DrugInternalName         string                   `json:"drug_internal_name,omitempty"`
	DrugName                 string                   `json:"drug_name"`
	DrugRoute                string                   `json:"drug_route,omitempty"`
	DrugForm                 string                   `json:"drug_form,omitempty"`
	DosageStrength           string                   `json:"dosage_strength,omitempty"`
	DispenseValue            int64                    `json:"dispense_value,string,omitempty"`
	DispenseUnitId           int64                    `json:"dispense_unit_id,string,omitempty"`
	DispenseUnitDescription  string                   `json:"dispense_unit_description,omitempty"`
	NumberRefills            int64                    `json:"refills,string,omitempty"`
	SubstitutionsAllowed     bool                     `json:"substitutions_allowed,omitempty"`
	DaysSupply               int64                    `json:"days_supply,string,omitempty"`
	PharmacyNotes            string                   `json:"pharmacy_notes,omitempty"`
	PatientInstructions      string                   `json:"patient_instructions,omitempty"`
	CreationDate             time.Time                `json:"creation_date,omitempty"`
	Status                   string                   `json:"-"`
	OTC                      bool                     `json:"otc,omitempty"`
	SupplementalInstructions []*DoctorInstructionItem `json:"supplemental_instructions,omitempty"`
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
	PatientVisitId  int64                    `json:"patient_visit_id,string,omitempty"`
	RegimenSections []*RegimenSection        `json:"regimen_sections"`
	AllRegimenSteps []*DoctorInstructionItem `json:"all_regimen_steps,omitempty"`
	Title           string                   `json:"title,omitempty"`
}

type FollowUp struct {
	PatientVisitId int64     `json:"patient_visit_id,string,omitempty"`
	FollowUpValue  int64     `json:"follow_up_value,string,omitempty"`
	FollowUpUnit   string    `json:"follow_up_unit,omitempty"`
	FollowUpTime   time.Time `json:"follow_up_time,omitempty"`
	Title          string    `json:"title,omitempty"`
}

type Advice struct {
	AllAdvicePoints      []*DoctorInstructionItem `json:"all_advice_points,omitempty"`
	SelectedAdvicePoints []*DoctorInstructionItem `json:"selected_advice_points,omitempty"`
	PatientVisitId       int64                    `json:"patient_visit_id,string,omitempty"`
	Title                string                   `json:"title,omitempty"`
}

type DiagnosisSummary struct {
	Type    string `json:"type"`
	Summary string `json:"text"`
	Title   string `json:"title,omitempty"`
}
