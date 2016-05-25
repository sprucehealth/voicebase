package mock

import (
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/libs/visitreview"
	"github.com/sprucehealth/backend/svc/layout"
)

type Store struct {
	*mock.Expector
}

func NewStore(t testing.TB) *Store {
	return &Store{
		Expector: &mock.Expector{T: t},
	}
}

func (s *Store) PutIntake(name string, intake *layout.Intake) (string, error) {
	rets := s.Record(name, intake)
	if len(rets) == 0 {
		return "", nil
	}
	return rets[0].(string), mock.SafeError(rets[1])
}
func (s *Store) PutReview(name string, review *visitreview.SectionListView) (string, error) {
	rets := s.Record(name, review)
	if len(rets) == 0 {
		return "", nil
	}
	return rets[0].(string), mock.SafeError(rets[1])
}
func (s *Store) PutSAML(name, saml string) (string, error) {
	rets := s.Record(name, saml)
	if len(rets) == 0 {
		return "", nil
	}
	return rets[0].(string), mock.SafeError(rets[1])
}
func (s *Store) GetIntake(location string) (*layout.Intake, error) {
	rets := s.Record(location)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*layout.Intake), mock.SafeError(rets[1])
}
func (s *Store) GetReview(location string) (*visitreview.SectionListView, error) {
	rets := s.Record(location)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*visitreview.SectionListView), mock.SafeError(rets[1])
}
func (s *Store) GetSAML(location string) (string, error) {
	rets := s.Record(location)
	if len(rets) == 0 {
		return "", nil
	}
	return rets[0].(string), mock.SafeError(rets[1])
}
