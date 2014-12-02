package test_integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/test"
)

func TestForcedUpgrade(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()

	minimumAppVersionConfigs := config.MinimumAppVersionConfigs(map[string]*config.MinimumAppVersionConfig{
		"Patient-Dev": &config.MinimumAppVersionConfig{
			AppVersion: &common.Version{Major: 1, Minor: 2, Patch: 0},
		},
	})
	testData.Config.MinimumAppVersionConfigs = &minimumAppVersionConfigs
	testData.StartAPIServer(t)

	r, err := http.NewRequest("GET", testData.APIServer.URL+apipaths.SettingsURLPath, nil)
	test.OK(t, err)
	r.Header.Add("S-Version", "Patient;Dev;0.9.5")

	// should require forced upgrade with an older version of the app
	resp, err := http.DefaultClient.Do(r)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	var responseData map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseData)
	test.OK(t, err)
	settingsResponse, ok := responseData["settings"]
	test.Equals(t, true, ok)
	_, ok = settingsResponse.(map[string]interface{})["upgrade_info"]
	test.Equals(t, true, ok)

	// should not require forced upgrade with a newer version of the app
	r.Header.Set("S-Version", "Patient;Dev;1.3.4")
	responseData = map[string]interface{}{}
	resp, err = http.DefaultClient.Do(r)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
	err = json.NewDecoder(resp.Body).Decode(&responseData)
	test.OK(t, err)
	_, ok = responseData["settings"]
	test.Equals(t, true, !ok)

	// should not require forced upgrade because there does not exist a config
	r.Header.Set("S-Version", "Patient;Live;1.3.4")
	responseData = map[string]interface{}{}
	resp, err = http.DefaultClient.Do(r)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
	err = json.NewDecoder(resp.Body).Decode(&responseData)
	test.OK(t, err)
	_, ok = responseData["settings"]
	test.Equals(t, true, !ok)
}
