package api

import (
	"errors"
	"time"
)

const (
	EN_LANGUAGE_ID = 1
)

var ErrLoginFailed = errors.New("api: login failed")
var ErrSignupFailedUserExists = errors.New("api: signup failed because user exists")

type Auth interface {
	Signup(login, password string) (token string, accountId int64, err error)
	Login(login, password string) (token string, accountId int64, err error)
	Logout(token string) error
	ValidateToken(token string) (valid bool, accountId int64, err error)
}

type Photo interface {
	Upload(data []byte, contentType string, key string, bucket string, duration time.Time) (string, error)
	GenerateSignedUrlsForKeysInBucket(bucket, prefix string, duration time.Time) ([]string, error)
}

const (
	PHOTO_TYPE_FACE_MIDDLE = "face_middle"
	PHOTO_TYPE_FACE_LEFT   = "face_left"
	PHOTO_TYPE_FACE_RIGHT  = "face_right"
	PHOTO_TYPE_BACK        = "back"
	PHOTO_TYPE_CHEST       = "chest"

	PHOTO_STATUS_PENDING_UPLOAD   = "PENDING_UPLOAD"
	PHOTO_STATUS_PENDING_APPROVAL = "PENDING_APPROVAL"
	PHOTO_STATUS_REJECTED         = "REJECTED"
	PHOTO_STATUS_APPROVED         = "APPROVED"
)

type PatientAnswerToQuestion struct {
	PatientInfoIntakeId int64
	QuestionId          int64
	PotentialAnswerId   int64
	LayoutVersionId     int64
	AnswerText          string
}

type PotentialAnswerInfo struct {
	PotentialAnswerId int64
	AnswerType        string
	Answer            string
	AnswerTag         string
	Ordering          int64
}

type PatientAPI interface {
	RegisterPatient(accountId int64, firstName, lastName, gender, zipCode string, dob time.Time) (int64, error)
	CreateNewPatientVisit(patientId, healthConditionId, layoutVersionId int64) (int64, error)
	GetPatientIdFromAccountId(accountId int64) (int64, error)
}

type PatientVisitAPI interface {
	GetActivePatientVisitForHealthCondition(patientId, healthConditionId int64) (int64, error)
}

type PatientIntakeAPI interface {
	StoreFreeTextAnswersForQuestion(patientId, questionId, patientVisitId, layoutVersionId int64, answerIds []int64, answerTexts []string) (patientInfoIntakeIds []int64, err error)
	StoreChoiceAnswersForQuestion(patientId, questionId, patientVisitId, layoutVersionId int64, answerIds []int64) (patientInfoIntakeIds []int64, err error)
	CreatePhotoAnswerForQuestionRecord(patientId, questionId, patientVisitId, potentialAnswerId, layoutVersionId int64) (patientInfoIntakeId int64, err error)
	UpdatePhotoAnswerRecordWithObjectStorageId(patientInfoIntakeId, objectStorageId int64) error
	MakeCurrentPhotoAnswerInactive(patientId, questionId, patientVisitId, potentialAnswerId, layoutVersionId int64) (err error)

	GetQuestionType(questionId int64) (questionType string, err error)
	GetPatientAnswersForQuestionsInGlobalSections(questionIds []int64, patientId int64) (patientAnswers map[int64][]PatientAnswerToQuestion, err error)
	GetPatientAnswersForQuestionsInPatientVisit(questionIds []int64, patientId int64, patientVisitId int64) (patientAnswers map[int64][]PatientAnswerToQuestion, err error)
	GetGlobalSectionIds() (globalSectionIds []int64, err error)
	GetSectionIdsForHealthCondition(healthConditionId int64) (sectionIds []int64, err error)
	GetHealthConditionInfo(healthConditionTag string) (int64, error)
	GetSectionInfo(sectionTag string, languageId int64) (id int64, title string, err error)
	GetQuestionInfo(questionTag string, languageId int64) (id int64, questionTitle string, questionType string, err error)
	GetAnswerInfo(questionId int64, languageId int64) (answerInfos []PotentialAnswerInfo, err error)
	GetTipSectionInfo(tipSectionTag string, languageId int64) (id int64, tipSectionTitle string, tipSectionSubtext string, err error)
	GetTipInfo(tipTag string, languageId int64) (id int64, tip string, err error)
	GetSupportedLanguages() (languagesSupported []string, languagesSupportedIds []int64, err error)
	GetActiveLayoutInfoForHealthCondition(healthConditionTag string) (bucket, key, region string, err error)
}

type PatientIntakeLayoutAPI interface {
	GetStorageInfoOfCurrentActiveClientLayout(languageId, healthConditionId int64) (bucket, key, region string, layoutVersionId int64, err error)
	GetLayoutVersionIdForPatientVisit(patientVisitId int64) (layoutVersionId int64, err error)
	GetStorageInfoForClientLayout(layoutVersionId, languageId int64) (bucket, key, region string, err error)
	MarkNewLayoutVersionAsCreating(objectId int64, syntaxVersion int64, healthConditionId int64, comment string) (int64, error)
	MarkNewPatientLayoutVersionAsCreating(objectId int64, languageId int64, layoutVersionId int64, healthConditionId int64) (int64, error)
	UpdateActiveLayouts(layoutId int64, clientLayoutIds []int64, healthConditionId int64) error
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
	PutObjectToLocation(bucket, key, region, contentType string, rawData []byte, duration time.Time, dataApi DataAPI) (int64, string, error)
}

type Layout interface {
	VerifyAndUploadIncomingLayout(rawLayout []byte, healthConditionTag string) error
}

type ACLAPI interface {
	IsAuthorizedForCase(accountId, caseId int64) (bool, error)
}
