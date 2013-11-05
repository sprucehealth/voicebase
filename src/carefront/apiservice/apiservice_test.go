package apiservice

import (
	"bytes"
	"carefront/mockapi"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

const (
	SignupPath       = "/v1/signup"
	LoginPath        = "/v1/authenticate"
	LogoutPath       = "/v1/logout"
	PhotoUploadPath  = "/v1/upload"
	ContentTypeValue = "application/x-www-form-urlencoded; param=value"
)

// TESTS

func TestSuccesfulSignup(t *testing.T) {
	mux := setupAuthHandlerInMux(SignupPath)
	req, _ := http.NewRequest("POST", SignupPath, strings.NewReader("login=kkjj&password=12345"))
	req.Header.Set("Content-Type", ContentTypeValue)

	responseWriter := createFakeResponseWriter()
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusOK, responseWriter, t)
	validateTokenResponse(responseWriter.body, t)
}

func TestExistingUserInSignup(t *testing.T) {
	mux := setupAuthHandlerInMux(SignupPath)
	req, _ := http.NewRequest("POST", SignupPath, strings.NewReader("login=k1234&password=12345"))
	req.Header.Set("Content-Type", ContentTypeValue)

	responseWriter := createFakeResponseWriter()
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusOK, responseWriter, t)
	validateTokenResponse(responseWriter.body, t)

	req, _ = http.NewRequest("POST", SignupPath, strings.NewReader("login=k1234&password=12345"))
	req.Header.Set("Content-Type", ContentTypeValue)

	responseWriter = createFakeResponseWriter()
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusBadRequest, responseWriter, t)
}

func TestMissingParametersSignup(t *testing.T) {
	mux := setupAuthHandlerInMux(SignupPath)
	req, _ := http.NewRequest("POST", SignupPath, nil)
	req.Header.Set("Content-Type", ContentTypeValue)

	responseWriter := createFakeResponseWriter()
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusForbidden, responseWriter, t)
}

func TestSignupFollowedByLogin(t *testing.T) {
	mux := setupAuthHandlerInMux(SignupPath)
	req, _ := http.NewRequest("POST", SignupPath, strings.NewReader("login=kjkj1&password=12345"))
	req.Header.Set("Content-Type", ContentTypeValue)

	responseWriter := createFakeResponseWriter()
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusOK, responseWriter, t)
	validateTokenResponse(responseWriter.body, t)

	anotherAuthHandler := &AuthenticationHandler{mux.AuthApi}
	mux.Handle(LoginPath, anotherAuthHandler)
	req, _ = http.NewRequest("POST", LoginPath, strings.NewReader("login=kjkj1&password=12345"))
	req.Header.Set("Content-Type", ContentTypeValue)

	responseWriter = createFakeResponseWriter()
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusOK, responseWriter, t)
	validateTokenResponse(responseWriter.body, t)
}

func TestMalformedHeaderSignup(t *testing.T) {
	mux := setupAuthHandlerInMux(SignupPath)
	req, _ := http.NewRequest("POST", SignupPath, strings.NewReader("login=koko&password=12345"))
	req.Header.Set("Content-Type", "WrongContentType")

	responseWriter := createFakeResponseWriter()
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusForbidden, responseWriter, t)
}

func TestSuccessfulLogin(t *testing.T) {
	mux := setupAuthHandlerInMux(LoginPath)

	req, _ := http.NewRequest("POST", LoginPath, strings.NewReader("login=kajham&password=12345"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	responseWriter := createFakeResponseWriter()
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusOK, responseWriter, t)
	validateTokenResponse(responseWriter.body, t)
}

func TestSuccessfulLogout(t *testing.T) {
	mux := setupAuthHandlerInMux(LogoutPath)
	req, _ := http.NewRequest("GET", LogoutPath, nil)
	req.Header.Set("Authorization", "token tokenForKajham")
	responseWriter := createFakeResponseWriter()
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusOK, responseWriter, t)
}

func TestUnauthorizedLogout(t *testing.T) {
	mux := setupAuthHandlerInMux(LogoutPath)
	req, _ := http.NewRequest("GET", LogoutPath, nil)
	responseWriter := createFakeResponseWriter()
	mux.ServeHTTP(responseWriter, req)
	checkStatusCode(http.StatusForbidden, responseWriter, t)
}
func TestUnsuccessfulLoginDueToPassword(t *testing.T) {
	mux := setupAuthHandlerInMux(LoginPath)

	req, _ := http.NewRequest("POST", LoginPath, strings.NewReader("login=kajham&password=ShouldFail"))
	req.Header.Set("Content-Type", ContentTypeValue)

	responseWriter := createFakeResponseWriter()
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusForbidden, responseWriter, t)
}

func TestUnsuccessfulLoginDueToUsername(t *testing.T) {
	mux := setupAuthHandlerInMux(LoginPath)

	req, _ := http.NewRequest("POST", LoginPath, strings.NewReader("login=kajaja&password=12345"))
	req.Header.Set("Content-Type", ContentTypeValue)

	responseWriter := createFakeResponseWriter()
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusForbidden, responseWriter, t)
}

func TestUnsuccessfulLoginDueToMissingParams(t *testing.T) {
	mux := setupAuthHandlerInMux(LoginPath)

	req, _ := http.NewRequest("POST", LoginPath, nil)

	responseWriter := createFakeResponseWriter()
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusForbidden, responseWriter, t)
}

func TestMissingAuthHeaderPhotoUpload(t *testing.T) {
	mux := setupPhotoUploadHandlerInMux()
	req, _ := http.NewRequest("POST", PhotoUploadPath, strings.NewReader("case_id=1234&photo_type=face_middle&photo=xxx"))
	responseWriter := createFakeResponseWriter()
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusForbidden, responseWriter, t)
}

func TestMissingCaseIdInPhotoUpload(t *testing.T) {
	testHelperForMissingParameter([]string{"photo", "photo_type"}, t)
}

func TestMissingPhotoTypeInPhotoUpload(t *testing.T) {
	testHelperForMissingParameter([]string{"case_id", "photo"}, t)
}

func TestSuccessfulPhotoUpload(t *testing.T) {
	mux := setupPhotoUploadHandlerInMux()

	buf, w := createMultiPartFormDataWithParameters([]string{"photo", "photo_type", "case_id"}, t)

	req, _ := http.NewRequest("POST", PhotoUploadPath, buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "token tokenForKajham")

	responseWriter := createFakeResponseWriter()
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusOK, responseWriter, t)
}

func testHelperDataOrPhotoServiceError(photoServiceError, dataServiceError bool, t *testing.T) {
	fakeAuthApi := createAndReturnFakeAuthApi()
	fakePhotoApi := &mockapi.MockPhotoService{photoServiceError}
	fakeDataApi := &mockapi.MockDataService{dataServiceError}

	photoUploadHandler := &PhotoUploadHandler{fakePhotoApi, "testing", fakeDataApi}
	mux := &AuthServeMux{*http.NewServeMux(), fakeAuthApi}
	mux.Handle(PhotoUploadPath, photoUploadHandler)

	buf, w := createMultiPartFormDataWithParameters([]string{"photo", "photo_type", "case_id"}, t)
	req, _ := http.NewRequest("POST", PhotoUploadPath, buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "token tokenForKajham")

	responseWriter := createFakeResponseWriter()
	mux.ServeHTTP(responseWriter, req)
	checkStatusCode(http.StatusInternalServerError, responseWriter, t)
}

func TestUploadErrorPhotoUpload(t *testing.T) {
	testHelperDataOrPhotoServiceError(true, false, t)
}

func TestDataServiceErrorPhotoUpload(t *testing.T) {
	testHelperDataOrPhotoServiceError(false, true, t)
}

// Private Methods

func createFakeResponseWriter() *FakeResponseWriter {
	return &FakeResponseWriter{make(map[string][]string), make([]byte, 20)}
}

func createAndReturnFakeAuthApi() *mockapi.MockAuth {
	return &mockapi.MockAuth{
		Accounts: map[string]mockapi.MockAccount{
			"kajham": mockapi.MockAccount{
				Id:       1,
				Login:    "kajham",
				Password: "12345",
			},
		},
		Tokens: map[string]int64{
			"tokenForKajham": 1,
		},
	}
}

func setupAuthHandlerInMux(path string) *AuthServeMux {
	fakeAuthApi := createAndReturnFakeAuthApi()
	authHandler := &AuthenticationHandler{fakeAuthApi}
	mux := &AuthServeMux{*http.NewServeMux(), fakeAuthApi}
	mux.Handle(path, authHandler)

	return mux
}

func setupPhotoUploadHandlerInMux() *AuthServeMux {
	fakeAuthApi := createAndReturnFakeAuthApi()
	fakePhotoApi := &mockapi.MockPhotoService{false}
	fakeDataApi := &mockapi.MockDataService{false}

	photoUploadHandler := &PhotoUploadHandler{fakePhotoApi, "testing", fakeDataApi}
	mux := &AuthServeMux{*http.NewServeMux(), fakeAuthApi}
	mux.Handle(PhotoUploadPath, photoUploadHandler)

	return mux
}

func validateTokenResponse(data []byte, t *testing.T) {
	type TokenJson struct {
		Token string
	}

	// test body
	var tokenJson TokenJson
	err := json.Unmarshal(data, &tokenJson)
	if err != nil {
		t.Errorf("Expected an auth token to be returned as response to the login called. %s", err.Error())
	}
	if tokenJson.Token == "" {
		t.Errorf("token not expected to be empty in return!")
	}
}

func checkForErrorInResponse(data []byte, t *testing.T) {
	type ErrorJson struct {
		Error string
	}

	var errorJson ErrorJson
	err := json.Unmarshal(data, &errorJson)
	if err != nil {
		t.Errorf("Expected an error to be returned in the response %s", err.Error())
	}
}

func checkStatusCode(expected int, responseWriter *FakeResponseWriter, t *testing.T) {
	statusCode := responseWriter.Headers.Get("Status")
	if statusCode != strconv.Itoa(expected) {
		t.Errorf("Expected status code %d, but got %q", expected, statusCode)
	}
}

func createMultiPartFormDataWithParameters(parameters []string, t *testing.T) (*bytes.Buffer, *multipart.Writer) {
	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)
	for _, parameter := range parameters {
		switch parameter {
		case "photo":
			photo, err := w.CreateFormFile("photo", "photo.png")
			if err != nil {
				t.Errorf("Something went wrong in test setup %s", err.Error())
			}
			photo.Write(make([]byte, 100))
		case "photo_type":
			photoType, err := w.CreateFormField("photo_type")
			if err != nil {
				t.Errorf("Something went wrong in adding photo_type to form %s", err.Error())
			}
			photoType.Write([]byte("face_middle"))
		case "case_id":
			caseId, err := w.CreateFormField("case_id")
			if err != nil {
				t.Errorf("Something went wrong in adding case_id to form %s", err.Error())
			}
			caseId.Write([]byte("12345"))
		}
	}
	w.Close()

	return buf, w
}

func testHelperForMissingParameter(parameters []string, t *testing.T) {
	mux := setupPhotoUploadHandlerInMux()
	buf, w := createMultiPartFormDataWithParameters(parameters, t)
	req, _ := http.NewRequest("POST", PhotoUploadPath, buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "token tokenForKajham")
	responseWriter := createFakeResponseWriter()
	mux.ServeHTTP(responseWriter, req)

	checkStatusCode(http.StatusBadRequest, responseWriter, t)
	checkForErrorInResponse(responseWriter.body, t)
}
