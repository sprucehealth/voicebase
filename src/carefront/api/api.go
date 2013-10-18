package api

import (
	"errors"
	"time"
)

var ErrLoginFailed = errors.New("api: login failed")
var ErrSignupFailedUserExists = errors.New("api: signup failed because user exists")

type Auth interface {
	Signup(login, password string) (token string, err error)
	Login(login, password string) (token string, err error)
	Logout(token string) error
	ValidateToken(token string) (valid bool, accountId int64, err error)
}

type Photo interface {
	Upload(data []byte, key string, bucket string, duration time.Time) (string, error)
	GenerateSignedUrlsForKeysInBucket(bucket, prefix string, duration time.Time) ([]string, error)
}

const (
	PHOTO_TYPE_FACE_MIDDLE = "face_middle"
	PHOTO_TYPE_FACE_LEFT = "face_left"
	PHOTO_TYPE_FACE_RIGHT = "face_right"
	PHOTO_TYPE_BACK = "back"
	PHOTO_TYPE_CHEST = "chest"

	PHOTO_STATUS_PENDING_UPLOAD = "PENDING_UPLOAD"
	PHOTO_STATUS_PENDING_APPROVAL = "PENDING_APPROVAL"
	PHOTO_STATUS_REJECTED = "REJECTED"
	PHOTO_STATUS_APPROVED = "APPROVED"
)

type DataAPI interface {
	CreatePhotoForCase(caseId int64, photoType string) (int64, error)
	MarkPhotoUploadComplete(caseId, photoId int64) error
	GetPhotosForCase(caseId int64) ([]string, error)
}

type ACLAPI interface {
	IsAuthorizedForCase(accountId, caseId int64) (bool, error)
}
