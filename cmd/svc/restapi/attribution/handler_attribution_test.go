package attribution

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/attribution/model"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
	"github.com/sprucehealth/backend/libs/test"
)

type mockAttributionDAL struct {
	insertAttributionDataErr   error
	insertAttributionDataParam *model.AttributionData
	insertAttributionData      int64
}

func (h *mockAttributionDAL) InsertAttributionData(attributionData *model.AttributionData) (int64, error) {
	h.insertAttributionDataParam = attributionData
	return h.insertAttributionData, h.insertAttributionDataErr
}

func TestAttributionHandlerPOSTDataRequired(t *testing.T) {
	r, err := http.NewRequest(httputil.Post, "mock.api.request", bytes.NewReader([]byte(`{}`)))
	test.OK(t, err)
	dal := &mockAttributionDAL{}
	handler := NewAttributionHandler(dal)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestAttributionHandlerPOSTDeviceIDHeaderRequired(t *testing.T) {
	r, err := http.NewRequest(httputil.Post, "mock.api.request", bytes.NewReader([]byte(`{"data":{}}`)))
	test.OK(t, err)
	dal := &mockAttributionDAL{}
	handler := NewAttributionHandler(dal)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestAttributionHandlerPOSTInsertAttributionRecordError(t *testing.T) {
	r, err := http.NewRequest(httputil.Post, "mock.api.request", bytes.NewReader([]byte(`{"data":{}}`)))
	r.Header.Add("S-Device-ID", "DeviceID")
	test.OK(t, err)
	dal := &mockAttributionDAL{insertAttributionDataErr: errors.New("Foo")}
	handler := NewAttributionHandler(dal)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestAttributionHandlerPOSTDeviceIDHappyCase(t *testing.T) {
	r, err := http.NewRequest(httputil.Post, "mock.api.request", bytes.NewReader([]byte(`{"data":{}}`)))
	r.Header.Add("S-Device-ID", "DeviceID")
	test.OK(t, err)
	dal := &mockAttributionDAL{}
	handler := NewAttributionHandler(dal)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusOK, responseWriter.Code)
	test.Assert(t, dal.insertAttributionDataParam.DeviceID != nil, "Expected non nil")
	test.Assert(t, dal.insertAttributionDataParam.AccountID == nil, "Expected nil")
	test.Equals(t, *dal.insertAttributionDataParam.DeviceID, "DeviceID")
}

func TestAttributionHandlerPOSTAccountIDHappyCase(t *testing.T) {
	r, err := http.NewRequest(httputil.Post, "mock.api.request", bytes.NewReader([]byte(`{"data":{}}`)))
	r.Header.Add("Authorization", "token 12345")
	test.OK(t, err)
	dal := &mockAttributionDAL{}
	handler := NewAttributionHandler(dal)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r.WithContext(apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 100})))
	test.Equals(t, http.StatusOK, responseWriter.Code)
	test.Assert(t, dal.insertAttributionDataParam.AccountID != nil, "Expected non nil")
	test.Assert(t, dal.insertAttributionDataParam.DeviceID == nil, "Expected nil")
	test.Equals(t, *dal.insertAttributionDataParam.AccountID, int64(100))
}
