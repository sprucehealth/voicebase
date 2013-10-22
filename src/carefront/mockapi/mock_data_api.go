package mockapi

import (
	"errors"
)

type MockDataService struct {
	ToGenerateError bool
}

func (m *MockDataService) CreatePhotoForCase(caseId int64, photoType string) (int64, error) {
	if m.ToGenerateError {
		return int64(0), errors.New("Fake error")
	}

	return int64(0), nil
}

func (m *MockDataService) MarkPhotoUploadComplete(caseId, photoId int64) error {
	if m.ToGenerateError {
		return  errors.New("Fake error")
	}

	return nil
}

func (m *MockDataService) GetPhotosForCase(caseId int64) ([]string, error) {
	return make([]string, 5), nil
}
