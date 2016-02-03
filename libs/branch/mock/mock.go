package mock

import (
	"testing"

	"github.com/sprucehealth/backend/libs/branch"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

// Build time check for matching against the interface
var d branch.Client = &Mock{}

// Mock is a mock version of the Branch client
type Mock struct {
	*mock.Expector
}

// New return a new mock version of the Branch client
func New(t *testing.T) *Mock {
	return &Mock{&mock.Expector{T: t}}
}

// URL implements branch.Client
func (m *Mock) URL(linkData map[string]interface{}) (string, error) {
	rets := m.Expector.Record(linkData)
	return rets[0].(string), mock.SafeError(rets[1])
}
