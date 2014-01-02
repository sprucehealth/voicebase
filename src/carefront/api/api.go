package api

import (
	"carefront/common"
	"errors"
	"time"
)

const (
	EN_LANGUAGE_ID           = 1
	DOCTOR_ROLE              = "DOCTOR"
	PATIENT_ROLE             = "PATIENT"
	REVIEW_PURPOSE           = "REVIEW"
	CONDITION_INTAKE_PURPOSE = "CONDITION_INTAKE"
	DIAGNOSE_PURPOSE         = "DIAGNOSE"
	FOLLOW_UP_WEEK           = "week"
	FOLLOW_UP_DAY            = "day"
	FOLLOW_UP_MONTH          = "month"
)

var (
	NoRowsError                  = errors.New("No rows exist")
	NoElligibileProviderInState  = errors.New("There are no providers elligible in the state the patient resides")
	NoRegimenPlanForPatientVisit = errors.New("There is no regimen plan for patient visit")
)

type PotentialAnswerInfo struct {
	PotentialAnswerId int64
	AnswerType        string
	Answer            string
	AnswerSummary     string
	AnswerTag         string
	Ordering          int64
}

type PatientAPI interface {
	GetPatientFromId(patientId int64) (patient *common.Patient, err error)
	RegisterPatient(accountId int64, firstName, lastName, gender, zipCode string, dob time.Time) (int64, error)
	CreateNewPatientVisit(patientId, healthConditionId, layoutVersionId int64) (int64, error)
	GetPatientIdFromAccountId(accountId int64) (int64, error)
	CreateCareTeamForPatient(patientId int64) error
	GetCareTeamForPatient(patientId int64) (careTeam *common.PatientCareProviderGroup, err error)
	CheckCareProvidingElligibility(shortState string, healthConditionId int64) (isElligible bool, err error)
}

type DoctorAPI interface {
	RegisterDoctor(accountId int64, firstName, lastName, gender string, dob time.Time) (int64, error)
	GetDoctorFromId(doctorId int64) (doctor *common.Doctor, err error)
	GetDoctorIdFromAccountId(accountId int64) (int64, error)
	GetRegimenStepsForDoctor(doctorId int64) (regimenSteps []*common.DoctorInstructionItem, err error)
	AddRegimenStepForDoctor(regimenStep *common.DoctorInstructionItem, doctorId int64) error
	UpdateRegimenStepForDoctor(regimenStep *common.DoctorInstructionItem, doctorId int64) error
	MarkRegimenStepToBeDeleted(regimenStep *common.DoctorInstructionItem, doctorId int64) error
	CreateRegimenPlanForPatientVisit(regimenPlan *common.RegimenPlan) error
	GetRegimenPlanForPatientVisit(patientVisitId int64) (regimenPlan *common.RegimenPlan, err error)

	GetAdvicePointsForDoctor(doctorId int64) (advicePoints []*common.DoctorInstructionItem, err error)
	GetAdvicePointsForPatientVisit(patientVisitId int64) (advicePoints []*common.DoctorInstructionItem, err error)
	CreateAdviceForPatientVisit(advicePoints []*common.DoctorInstructionItem, patientVisitId int64) error
	AddOrUpdateAdvicePointForDoctor(advicePoint *common.DoctorInstructionItem, doctorId int64) error
	MarkAdvicePointToBeDeleted(advicePoint *common.DoctorInstructionItem, doctorId int64) error
}

type PatientVisitAPI interface {
	GetActivePatientVisitIdForHealthCondition(patientId, healthConditionId int64) (int64, error)
	GetPatientIdFromPatientVisitId(patientVisitId int64) (int64, error)
	GetLatestSubmittedPatientVisit() (*common.PatientVisit, error)
	GetPatientVisitFromId(patientVisitId int64) (patientVisit *common.PatientVisit, err error)
	SubmitPatientVisitWithId(patientVisitId int64) error
	UpdateFollowUpTimeForPatientVisit(patientVisitId, doctorId, followUpValue int64, followUpUnit string) error
	GetFollowUpTimeForPatientVisit(patientVisitId int64) (followupTime time.Time, followUpValue int64, followUpUnit string, err error)
}

type PatientIntakeAPI interface {
	GetPatientAnswersForQuestionsInGlobalSections(questionIds []int64, patientId int64) (patientAnswers map[int64][]*common.AnswerIntake, err error)
}

type IntakeAPI interface {
	GetAnswersForQuestionsInPatientVisit(role string, questionIds []int64, roleId int64, patientVisitId int64) (answerIntakes map[int64][]*common.AnswerIntake, err error)
	StoreAnswersForQuestion(role string, roleId, patientVisitId, layoutVersionId int64, answersToStorePerQuestion map[int64][]*common.AnswerIntake) (err error)
	CreatePhotoAnswerForQuestionRecord(role string, roleId, questionId, patientVisitId, potentialAnswerId, layoutVersionId int64) (patientInfoIntakeId int64, err error)
	UpdatePhotoAnswerRecordWithObjectStorageId(patientInfoIntakeId, objectStorageId int64) error
	MakeCurrentPhotoAnswerInactive(role string, roleId, questionId, patientVisitId, potentialAnswerId, layoutVersionId int64) error
}

type IntakeLayoutAPI interface {
	GetQuestionType(questionId int64) (questionType string, err error)
	GetActiveLayoutInfoForHealthCondition(healthConditionTag, role, purpose string) (bucket, key, region string, err error)
	GetStorageInfoOfCurrentActivePatientLayout(languageId, healthConditionId int64) (bucket, key, region string, layoutVersionId int64, err error)
	GetStorageInfoOfCurrentActiveDoctorLayout(healthConditionId int64) (bucket, storage, region string, layoutVersionId int64, err error)
	GetStorageInfoOfActiveDoctorDiagnosisLayout(healthConditionId int64) (bucket, storage, region string, layoutVersionId int64, err error)
	GetLayoutVersionIdForPatientVisit(patientVisitId int64) (layoutVersionId int64, err error)
	GetStorageInfoForClientLayout(layoutVersionId, languageId int64) (bucket, key, region string, err error)
	MarkNewLayoutVersionAsCreating(objectId int64, syntaxVersion int64, healthConditionId int64, role, purpose, comment string) (int64, error)
	MarkNewPatientLayoutVersionAsCreating(objectId int64, languageId int64, layoutVersionId int64, healthConditionId int64) (int64, error)
	UpdatePatientActiveLayouts(layoutId int64, clientLayoutIds []int64, healthConditionId int64) error
	MarkNewDoctorLayoutAsCreating(objectId int64, layoutVersionId int64, healthConditionId int64) (int64, error)
	UpdateDoctorActiveLayouts(layoutId, doctorLayoutId, healthConditionId int64, purpose string) error
	GetGlobalSectionIds() (globalSectionIds []int64, err error)
	GetSectionIdsForHealthCondition(healthConditionId int64) (sectionIds []int64, err error)
	GetHealthConditionInfo(healthConditionTag string) (int64, error)
	GetSectionInfo(sectionTag string, languageId int64) (id int64, title string, err error)
	GetQuestionInfo(questionTag string, languageId int64) (id int64, questionTitle string, questionType string, questionSummary string, questionSubText string, parentQuestionId int64, additionalFields map[string]string, err error)
	GetAnswerInfo(questionId int64, languageId int64) (answerInfos []PotentialAnswerInfo, err error)
	GetTipSectionInfo(tipSectionTag string, languageId int64) (id int64, tipSectionTitle string, tipSectionSubtext string, err error)
	GetTipInfo(tipTag string, languageId int64) (id int64, tip string, err error)
	GetSupportedLanguages() (languagesSupported []string, languagesSupportedIds []int64, err error)
}

type ERxAPI interface {
	GetMedicationDispenseUnits(languageId int64) (dispenseUnitIds []int64, dispenseUnits []string, err error)
	AddTreatmentsForPatientVisit(treatments []*common.Treatment) error
	GetTreatmentPlanForPatientVisit(patientVisitId int64) (treatmentPlan *common.TreatmentPlan, err error)
	GetDrugInstructionsForDoctor(drugName, drugForm, drugRoute string, doctorId int64) (drugInstructions []*common.DoctorInstructionItem, err error)
	AddOrUpdateDrugInstructionForDoctor(drugName, drugForm, drugRoute string, drugInstructionToAdd *common.DoctorInstructionItem, doctorId int64) error
	DeleteDrugInstructionForDoctor(drugInstructionToDelete *common.DoctorInstructionItem, doctorId int64) error
	AddDrugInstructionsToTreatment(drugName, drugForm, drugRoute string, drugInstructions []*common.DoctorInstructionItem, treatmentId int64, doctorId int64) error
}

type ObjectStorageAPI interface {
	CreateNewUploadCloudObjectRecord(bucket, key, region string) (int64, error)
	UpdateCloudObjectRecordToSayCompleted(id int64) error
}

type DataAPI interface {
	PatientAPI
	DoctorAPI
	PatientIntakeAPI
	PatientVisitAPI
	IntakeLayoutAPI
	ObjectStorageAPI
	IntakeAPI
	ERxAPI
}

type CloudStorageAPI interface {
	GetObjectAtLocation(bucket, key, region string) (rawData []byte, err error)
	GetSignedUrlForObjectAtLocation(bucket, key, region string, duration time.Time) (url string, err error)
	DeleteObjectAtLocation(bucket, key, region string) error
	PutObjectToLocation(bucket, key, region, contentType string, rawData []byte, duration time.Time, dataApi DataAPI) (int64, string, error)
}

type Layout interface {
	VerifyAndUploadIncomingLayout(rawLayout []byte, healthConditionTag string) error
}

type ACLAPI interface {
	IsAuthorizedForCase(accountId, caseId int64) (bool, error)
}
