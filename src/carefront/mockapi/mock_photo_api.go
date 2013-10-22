package mockapi

import (
	"time"
	"errors"
)

type MockPhotoService struct {
	ToGenerateError bool
}

func (m *MockPhotoService) Upload(data []byte, key string, bucket string, duration time.Time) (string, error) {
	if m.ToGenerateError {
		return "", errors.New("Fake error")
	}
	return "", nil
}

func (m *MockPhotoService) GenerateSignedUrlsForKeysInBucket(bucket, prefix string, duration time.Time) ([]string, error) {
	if m.ToGenerateError {
		return make([]string, 5),errors.New("Fake error")
	}
	return make([]string, 5), nil
}
