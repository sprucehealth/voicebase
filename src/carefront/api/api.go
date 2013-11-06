package api

import (
	"errors"
	"time"
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

type DataAPI interface {
	CreatePhotoForCase(caseId int64, photoType string) (int64, error)
	MarkPhotoUploadComplete(caseId, photoId int64) error
	GetPhotosForCase(caseId int64) ([]string, error)

	/*
	* Patient Information Intake APIs
	 */
	GetTreatmentInfo(treatmentTag string, languageId int64) (int64, error)
	GetSectionInfo(sectionTag string, languageId int64) (id int64, title string, err error)
	GetQuestionInfo(questionTag string, languageId int64) (id int64, questionTitle string, questionType string, err error)
	GetOutcomeInfo(outcomeTag string, languageId int64) (id int64, outcome string, outcomeType string, err error)
	GetTipSectionInfo(tipSectionTag string, languageId int64) (id int64, tipSectionTitle string, tipSectionSubtext string, err error)
	GetTipInfo(tipTag string, languageId int64) (id int64, tip string, err error)

	GetSupportedLanguages() (languagesSupported []string, languagesSupportedIds []int64, err error)
	GetCurrentActiveLayoutInfoForTreatment(treatmentTag string) (bucket, key, region string, err error)
	UploadAndMarkActiveNewLayoutForTreatment(rawLayout []byte, bucket, key, region string, err error)
	UploadAndMarkActiveNewClientLayoutForTreatment(rawLayout []byte, bucket, key, region string, languageId int64, err error)
}

type Layout interface {
	VerifyAndUploadIncomingLayout(rawLayout []byte, treatmentTag string) error
}

type ACLAPI interface {
	IsAuthorizedForCase(accountId, caseId int64) (bool, error)
}
