package attribution

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/attribution/model"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

type mockAttributionDAL struct {
	insertAttributionDataErr error
	insertAttributionData    int64
}

func (h *mockAttributionDAL) InsertAttributionData(attributionData *model.AttributionData) (int64, error) {
	return h.insertAttributionData, h.insertAttributionDataErr
}

func TestAttributionHandlerPOSTDataRequired(t *testing.T) {
	r, err := http.NewRequest(httputil.Post, "mock.api.request", bytes.NewReader([]byte(`{}`)))
	test.OK(t, err)
	dal := &mockAttributionDAL{}
	handler := NewAttributionHandler(dal)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestAttributionHandlerPOSTDeviceIDHeaderRequired(t *testing.T) {
	r, err := http.NewRequest(httputil.Post, "mock.api.request", bytes.NewReader([]byte(`{"data":{}}`)))
	test.OK(t, err)
	dal := &mockAttributionDAL{}
	handler := NewAttributionHandler(dal)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestAttributionHandlerPOSTInsertAttributionRecordError(t *testing.T) {
	r, err := http.NewRequest(httputil.Post, "mock.api.request", bytes.NewReader([]byte(`{"data":{}}`)))
	r.Header.Add("S-Device-ID", "DeviceID")
	test.OK(t, err)
	dal := &mockAttributionDAL{insertAttributionDataErr: errors.New("Foo")}
	handler := NewAttributionHandler(dal)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestAttributionHandlerPOSTHappyCase(t *testing.T) {
	r, err := http.NewRequest(httputil.Post, "mock.api.request", bytes.NewReader([]byte(`{"data":{}}`)))
	r.Header.Add("S-Device-ID", "DeviceID")
	test.OK(t, err)
	dal := &mockAttributionDAL{}
	handler := NewAttributionHandler(dal)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, http.StatusOK, responseWriter.Code)
}
