package api

import (
	"errors"
	"io"
)

var ErrLoginFailed = errors.New("api: login failed")

type Auth interface {
	Login(login, password string) (token string, err error)
	Logout(token string) error
	ValidateToken(token string) (valid bool, accountId int64, err error)
}

type PhotoService interface {
	Upload(imageData io.Reader, key string, bucket string) (string, error)
	GenerateSignedUrls(key,bucket string) (string, error)
}

type DataService interface {
	CreatePhotoForCase(caseId int64) (string, error)
	MarkPhotoUploadComplete(caseId, photoId int64) (string, error)
	GetPhotosForCase(caseId int64) ([]string, error)	
}

type ACLService interface {
	IsAuthorizedForCase(accountId, caseId int64) (bool, error)
}
