package patient_file

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/compat"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/test"
)

type mockDataAPI_patientCapabilitiesHandler struct {
	api.DataAPI
}

func (a *mockDataAPI_patientCapabilitiesHandler) Patient(id common.PatientID, basic bool) (*common.Patient, error) {
	return &common.Patient{ID: id, AccountID: encoding.NewObjectID(id.Uint64())}, nil
}

type mockAuthAPI_patientCapabilitiesHandler struct {
	api.AuthAPI
	appInfo *api.AppInfo
}

func (a *mockAuthAPI_patientCapabilitiesHandler) LatestAppInfo(accountID int64) (*api.AppInfo, error) {
	if a.appInfo == nil {
		return nil, api.ErrNotFound("app_info")
	}
	return a.appInfo, nil
}

func TestPatientCapabilitiesHandler(t *testing.T) {
	dataAPI := &mockDataAPI_patientCapabilitiesHandler{}
	authAPI := &mockAuthAPI_patientCapabilitiesHandler{}
	var features compat.Features
	features.Register([]*compat.Feature{
		{
			Name: "feature1",
			AppVersions: map[string]encoding.VersionRange{
				"ios-patient":     {MinVersion: &encoding.Version{Major: 1, Minor: 0, Patch: 0}},
				"android-patient": {MinVersion: &encoding.Version{Major: 1, Minor: 0, Patch: 0}},
			},
		},
		{
			Name: "feature2",
			AppVersions: map[string]encoding.VersionRange{
				"ios-patient": {MinVersion: &encoding.Version{Major: 1, Minor: 5, Patch: 0}},
			},
		},
	})
	h := NewPatientCapabilitiesHandler(dataAPI, authAPI, features)
	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{Role: api.RoleDoctor, ID: 1})
	r, err := http.NewRequest("GET", "/?patient_id=2", nil)
	test.OK(t, err)

	// Missing app info
	w := httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusNotFound, w)

	// Missing app info
	authAPI.appInfo = &api.AppInfo{
		Platform: device.IOS,
		Version:  &encoding.Version{Major: 1, Minor: 0, Patch: 0},
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)
	test.Equals(t, "{\"features\":[\"feature1\"]}\n", w.Body.String())
}
