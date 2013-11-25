package api

import (
	"carefront/common"
	"errors"
	"time"
)

const (
	EN_LANGUAGE_ID = 1
	DOCTOR_ROLE    = "DOCTOR"
	PATIENT_ROLE   = "PATIENT"
)

var ErrLoginFailed = errors.New("api: login failed")
var ErrSignupFailedUserExists = errors.New("api: signup failed because user exists")

type Auth interface {
	Signup(login, password string) (token string, accountId int64, err error)
	Login(login, password string) (token string, accountId int64, err error)
	Logout(token string) error
	ValidateToken(token string) (valid bool, accountId int64, err error)
}

type PotentialAnswerInfo struct {
	PotentialAnswerId int64
	AnswerType        string
	Answer            string
	AnswerTag         string
	Ordering          int64
}

type PatientAPI interface {
	GetPatientFromId(patientId int64) (patient *common.Patient, err error)
	RegisterPatient(accountId int64, firstName, lastName, gender, zipCode string, dob time.Time) (int64, error)
	CreateNewPatientVisit(patientId, healthConditionId, layoutVersionId int64) (int64, error)
	GetPatientIdFromAccountId(accountId int64) (int64, error)
}

type PatientVisitAPI interface {
	GetActivePatientVisitIdForHealthCondition(patientId, healthConditionId int64) (int64, error)
	GetPatientIdFromPatientVisitId(patientVisitId int64) (int64, error)
	GetPatientVisitFromId(patientVisitId int64) (patientVisit *common.PatientVisit, err error)
}

type PatientIntakeAPI interface {
	StoreAnswersForQuestion(questionId, patientId, patientVisitId, layoutVersionId int64, answersToStore []*common.PatientAnswer) (err error)
	CreatePhotoAnswerForQuestionRecord(patientId, questionId, patientVisitId, potentialAnswerId, layoutVersionId int64) (patientInfoIntakeId int64, err error)
	UpdatePhotoAnswerRecordWithObjectStorageId(patientInfoIntakeId, objectStorageId int64) error
	MakeCurrentPhotoAnswerInactive(patientId, questionId, patientVisitId, potentialAnswerId, layoutVersionId int64) (err error)

	GetQuestionType(questionId int64) (questionType string, err error)
	GetPatientAnswersForQuestionsInGlobalSections(questionIds []int64, patientId int64) (patientAnswers map[int64][]*common.PatientAnswer, err error)
	GetPatientAnswersForQuestionsInPatientVisit(questionIds []int64, patientId int64, patientVisitId int64) (patientAnswers map[int64][]*common.PatientAnswer, err error)
	GetGlobalSectionIds() (globalSectionIds []int64, err error)
	GetSectionIdsForHealthCondition(healthConditionId int64) (sectionIds []int64, err error)
	GetHealthConditionInfo(healthConditionTag string) (int64, error)
	GetSectionInfo(sectionTag string, languageId int64) (id int64, title string, err error)
	GetQuestionInfo(questionTag string, languageId int64) (id int64, questionTitle string, questionType string, questionSummary string, parentQuestionId int64, err error)
	GetAnswerInfo(questionId int64, languageId int64) (answerInfos []PotentialAnswerInfo, err error)
	GetTipSectionInfo(tipSectionTag string, languageId int64) (id int64, tipSectionTitle string, tipSectionSubtext string, err error)
	GetTipInfo(tipTag string, languageId int64) (id int64, tip string, err error)
	GetSupportedLanguages() (languagesSupported []string, languagesSupportedIds []int64, err error)
}

type PatientIntakeLayoutAPI interface {
	GetActiveLayoutInfoForHealthCondition(healthConditionTag, role string) (bucket, key, region string, err error)
	GetStorageInfoOfCurrentActivePatientLayout(languageId, healthConditionId int64) (bucket, key, region string, layoutVersionId int64, err error)
	GetStorageInfoOfCurrentActiveDoctorLayout(healthConditionId int64) (bucket, storage, region string, layoutVersionId int64, err error)
	GetLayoutVersionIdForPatientVisit(patientVisitId int64) (layoutVersionId int64, err error)
	GetStorageInfoForClientLayout(layoutVersionId, languageId int64) (bucket, key, region string, err error)
	MarkNewLayoutVersionAsCreating(objectId int64, syntaxVersion int64, healthConditionId int64, role, comment string) (int64, error)
	MarkNewPatientLayoutVersionAsCreating(objectId int64, languageId int64, layoutVersionId int64, healthConditionId int64) (int64, error)
	UpdatePatientActiveLayouts(layoutId int64, clientLayoutIds []int64, healthConditionId int64) error
	MarkNewDoctorLayoutAsCreating(objectId int64, layoutVersionId int64, healthConditionId int64) (int64, error)
	UpdateDoctorActiveLayouts(layoutId, doctorLayoutId, healthConditionId int64) error
}

type ObjectStorageAPI interface {
	CreateNewUploadCloudObjectRecord(bucket, key, region string) (int64, error)
	UpdateCloudObjectRecordToSayCompleted(id int64) error
}

type DataAPI interface {
	PatientAPI
	PatientIntakeAPI
	PatientVisitAPI
	PatientIntakeLayoutAPI
	ObjectStorageAPI
}

type CloudStorageAPI interface {
	GetObjectAtLocation(bucket, key, region string) (rawData []byte, err error)
	GetSignedUrlForObjectAtLocation(bucket, key, region string, duration time.Time) (url string, err error)
	PutObjectToLocation(bucket, key, region, contentType string, rawData []byte, duration time.Time, dataApi DataAPI) (int64, string, error)
}

type Layout interface {
	VerifyAndUploadIncomingLayout(rawLayout []byte, healthConditionTag string) error
}

type ACLAPI interface {
	IsAuthorizedForCase(accountId, caseId int64) (bool, error)
}
