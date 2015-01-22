package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/test"
)

type testForm struct {
	Email    string
	State    string
	Platform string
	DeviceID string
}

func (t *testForm) TableColumnValues() (string, []string, []interface{}) {
	return "test_form", []string{"email", "state", "platform", "device_id"}, []interface{}{t.Email, t.State, t.Platform, t.DeviceID}
}

func (n *notifyMeHandlerDataAPI) RecordForm(form api.Form, source string, requestID int64) error {
	n.recordedForm = form
	return nil
}

type notifyMeHandlerDataAPI struct {
	api.DataAPI
	recordedForm api.Form
}

func (n *notifyMeHandlerDataAPI) State(stateCode string) (string, string, error) {
	return "California", "CA", nil
}

func TestNotifyMeHandler(t *testing.T) {
	testNotifyMeHandler("POST", t)
	testNotifyMeHandler("PUT", t)
}

func testNotifyMeHandler(httpVerb string, t *testing.T) {
	dataAPI := &notifyMeHandlerDataAPI{}
	h := NewNotifyMeHandler(dataAPI)
	rd := &notifyMeRequest{
		Email: "test@test.com",
		State: "CA",
	}
	jsonData, err := json.Marshal(rd)
	test.OK(t, err)

	server := httptest.NewServer(h)
	defer server.Close()

	r, err := http.NewRequest(httpVerb, server.URL, bytes.NewReader(jsonData))
	test.OK(t, err)
	r.Header.Add("S-Version", "Patient;test;1.0.0")
	r.Header.Add("S-OS", "iOS;7.1")
	r.Header.Add("S-Device", "iPhone6,1")
	deviceID := "123456dgggddg6787573"
	r.Header.Add("S-Device-ID", deviceID)
	r.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(r)
	test.OK(t, err)
	test.Equals(t, http.StatusOK, res.StatusCode)

	_, _, values := dataAPI.recordedForm.TableColumnValues()
	test.Equals(t, rd.Email, values[0])
	test.Equals(t, rd.State, values[1])
	test.Equals(t, deviceID, values[3])
}
