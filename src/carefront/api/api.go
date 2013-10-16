package api

import (
	"errors"
	"time"
)

var ErrLoginFailed = errors.New("api: login failed")

type Auth interface {
	Login(login, password string) (token string, err error)
	Logout(token string) error
	ValidateToken(token string) (valid bool, accountId int64, err error)
}

type Photo interface {
	Upload(data []byte, key string, bucket string, duration time.Time) (string, error)
	GenerateSignedUrlsForKeysInBucket(bucket, prefix string, duration time.Time) ([]string, error)
}

type DataService interface {
	CreatePhotoForCase(caseId int64) (string, error)
	MarkPhotoUploadComplete(caseId, photoId int64) (string, error)
	GetPhotosForCase(caseId int64) ([]string, error)	
}

type ACLService interface {
	IsAuthorizedForCase(accountId, caseId int64) (bool, error)
}
