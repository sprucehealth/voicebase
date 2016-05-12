package mock

import (
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/libs/httputil"
)

var _ httputil.Client = &mockClient{}

type mockClient struct {
	*Expector
}

func NewHttpClient(t *testing.T) *mockClient {
	return &mockClient{
		Expector: &Expector{
			T: t,
		},
	}
}

func (m *mockClient) Head(url string) (*http.Response, error) {
	rets := m.Record(url)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*http.Response), SafeError(rets[1])
}
