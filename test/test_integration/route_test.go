package test_integration

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/test"
)

// TestAuthSetup ensures that every endpoint on the restapi server
// explicitly defines how it handles authorization and authentication
func TestAuthSetup(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// get a list of all the paths registered with the mux
	res, err := http.Get(testData.APIServer.URL + "/listpaths")
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
	var jsonData map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&jsonData)
	test.OK(t, err)

	// ensure that there is atleast one path registered otherwise there's
	// something wrong in the setup
	registeredPaths := jsonData["paths"].([]interface{})
	test.Equals(t, true, len(registeredPaths) > 0)

	// iterate over each registered path and ensure that we get the expected
	// response when verifying that the endpoint explicitly defines how
	// authorization and authentication should work
	var pathsNotSetupForAuthorization []string
	var pathsNotSetupForAuthentication []string
	queryForAuthentication := "?test=authentication"
	queryForAuthorization := "?test=authorization"
	for _, registeredPath := range registeredPaths {
		// verify that the registered path explicitly defines
		// how to handle authentication
		path := registeredPath.(string)
		if !runTestQuery(path, queryForAuthentication, testData, t) {
			pathsNotSetupForAuthentication = append(pathsNotSetupForAuthentication, path)
		}

		// verify that the registered path explicitly defines
		// how to handle authorization
		if !runTestQuery(path, queryForAuthorization, testData, t) {
			pathsNotSetupForAuthorization = append(pathsNotSetupForAuthorization, path)
		}
	}

	if len(pathsNotSetupForAuthentication) > 0 ||
		len(pathsNotSetupForAuthorization) > 0 {
		t.Fatalf("Following paths are not setup for:\nAuthentication:%v\nAuthorization:%v",
			pathsNotSetupForAuthentication, pathsNotSetupForAuthorization)
	}
}

type result struct {
	Code int `json:"result"`
}

func runTestQuery(registeredPath, testQuery string, testData *TestData, t *testing.T) bool {
	// first identify what are the set of allowable methods against the endpoint
	// NOTE: intentionally send the test query parameters so that we can bypass
	// the auth checks when the test query is present in the test environment
	req, err := http.NewRequest("OPTIONS", testData.APIServer.URL+registeredPath+testQuery, nil)
	test.OK(t, err)
	res, err := http.DefaultClient.Do(req)
	test.OK(t, err)
	res.Body.Close()
	allowableMethods := strings.Split(res.Header.Get("Allow"), ", ")
	test.Equals(t, true, len(allowableMethods) > 0)

	req, err = http.NewRequest(allowableMethods[0], testData.APIServer.URL+registeredPath+testQuery, nil)
	test.OK(t, err)
	res, err = http.DefaultClient.Do(req)
	defer res.Body.Close()

	jsonResponse := result{}
	err = json.NewDecoder(res.Body).Decode(&jsonResponse)
	test.OK(t, err)

	return jsonResponse.Code == apiservice.VerifyAuthCode
}
