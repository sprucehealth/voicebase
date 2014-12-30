package test_integration

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/test"
)

func TestAuth(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()

	email, pass, pass2 := "someone@somewhere.com", "somepass", "newPass"

	platform := api.Platform("test")

	sAccountID, err := testData.AuthAPI.CreateAccount(email, pass, "DOCTOR")
	test.OK(t, err)
	if sAccountID <= 0 {
		t.Fatalf("CreateAccount returned invalid AccountId: %d", sAccountID)
	}

	sToken, err := testData.AuthAPI.CreateToken(sAccountID, platform, false)
	if err != nil {
		t.Fatal(err)
	}
	if sToken == "" {
		t.Fatalf("CreateToken returned a blank Token")
	}

	// Make sure token is valid
	if account, err := testData.AuthAPI.ValidateToken(sToken, platform); err != nil {
		t.Fatal(err)
	} else if account.ID != sAccountID {
		t.Fatalf("ValidateToken returned differnet AccountId")
	} else if account.Role != "DOCTOR" {
		t.Fatalf("ValidateToken returned role '%s', expected 'DOCTOR'", account.Role)
	}
	lAccount, err := testData.AuthAPI.Authenticate(email, pass)
	test.OK(t, err)

	if sAccountID != lAccount.ID {
		t.Fatalf("AccountId doesn't match between login and singup")
	}
	_, err = testData.AuthAPI.CreateToken(lAccount.ID, platform, false)
	if err != nil {
		t.Fatal(err)
	}
	// Make sure token from Signup is no longer valid
	if _, err := testData.AuthAPI.ValidateToken(sToken, platform); err == api.TokenDoesNotExist {
		// Expected
	} else if err != nil {
		t.Fatal(err)
	} else {
		t.Fatalf("Token returned by Signup still valid after Login")
	}
	if err := testData.AuthAPI.SetPassword(lAccount.ID, pass2); err != nil {
		t.Fatal(err)
	}
	// Make sure token from Signup is no longer valid
	if _, err := testData.AuthAPI.ValidateToken(sToken, platform); err == api.TokenDoesNotExist {
		// Expected
	} else if err != nil {
		t.Fatal(err)
	} else {
		t.Fatalf("Token returned by Login still valid after SetPassword")
	}
	// Try to login with new password
	lAccount, err = testData.AuthAPI.Authenticate(email, pass2)
	test.OK(t, err)

	if sAccountID != lAccount.ID {
		t.Fatalf("AccountId doesn't match between login and singup")
	}

	token, err := testData.AuthAPI.CreateToken(lAccount.ID, platform, false)
	if err != nil {
		t.Fatal(err)
	}

	if a, err := testData.AuthAPI.ValidateToken(token, platform); err != nil {
		t.Fatal(err)
	} else if a.ID != lAccount.ID {
		t.Fatalf("ValidateToken returned differnet AccountId")
	}

	if err := testData.AuthAPI.DeleteToken(token); err != nil {
		t.Fatal(err)
	}
	// Make sure token is no longer valid
	if _, err := testData.AuthAPI.ValidateToken(token, platform); err == api.TokenDoesNotExist {
		// Expected
	} else if err != nil {
		t.Fatal(err)
	} else {
		t.Fatalf("Token returned by Login still valid after Logout")
	}
}

func TestAuth_ExtendedAuth(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()

	authAPI, err := api.NewAuthAPI(testData.DB, time.Second, time.Second/2, time.Second*10, time.Second*5, nullHasher{})
	test.OK(t, err)

	email, pass := "someone@somewhere.com", "somepass"
	platform := api.Platform("test")
	platform2 := api.Platform("test2")

	sAccountID, err := testData.AuthAPI.CreateAccount(email, pass, api.PATIENT_ROLE)
	test.OK(t, err)
	if sAccountID <= 0 {
		t.Fatalf("CreateAccount returned invalid AccountId: %d", sAccountID)
	}

	// login with regular auth to ensure that auth fails on regular auth expiration
	_, err = authAPI.Authenticate(email, pass)
	test.OK(t, err)
	sToken, err := authAPI.CreateToken(sAccountID, platform, false)
	test.OK(t, err)
	// non-extended auth token should expire
	time.Sleep(time.Second * 2)
	_, err = authAPI.ValidateToken(sToken, platform)
	test.Equals(t, api.TokenExpired, err)

	// now act as though we are logging in with extended auth
	_, err = authAPI.Authenticate(email, pass)
	test.OK(t, err)
	sToken, err = authAPI.CreateToken(sAccountID, platform, true)
	test.OK(t, err)
	// auth token should still be valid after 2 seconds given that
	// we are dealing with extended auth
	time.Sleep(time.Second)
	_, err = authAPI.ValidateToken(sToken, platform)
	test.OK(t, err)

	// now act as though we are logging in on a different platform with regular auth
	// in this case make sure to ensure that extended auth setting does not spill onto this new platform
	_, err = authAPI.Authenticate(email, pass)
	test.OK(t, err)
	sToken, err = authAPI.CreateToken(sAccountID, platform2, false)
	test.OK(t, err)
	time.Sleep(time.Second * 2)
	_, err = authAPI.ValidateToken(sToken, platform2)
	test.Equals(t, api.TokenExpired, err)

	// now login again as regular auth with the same account to ensure that you can switch of extended auth feature
	_, err = authAPI.Authenticate(email, pass)
	test.OK(t, err)
	sToken, err = authAPI.CreateToken(sAccountID, platform, false)
	test.OK(t, err)
	// auth token should no longer be valid for this platform given that we switched off the extended
	// auth feature for the platform
	time.Sleep(time.Second * 2)
	_, err = authAPI.ValidateToken(sToken, platform)
	test.Equals(t, api.TokenExpired, err)
}

func TestLostPassword(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	cli := PatientClient(testData, t, 0)

	// Should silently ignore non-existant email (return success)
	test.OK(t, cli.ResetPassword("does-not-exist@example.com"))

	validEmail := "exists@somewhere.com"
	_, err := testData.AuthAPI.CreateAccount(validEmail, "xxx", "DOCTOR")
	test.OK(t, err)

	test.OK(t, cli.ResetPassword(validEmail))

	_, templated := testData.EmailService.Reset()
	if len(templated) != 1 {
		t.Fatalf("Expected 1 sent email. Got %d", len(templated))
	}
}

func TestTrackAppDeviceInfo(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// signup doctor
	_, email, password := SignupRandomTestDoctor(t, testData)

	// login doctor
	params := url.Values{}
	params.Set("email", email)
	params.Set("password", password)

	req, err := http.NewRequest("POST", testData.APIServer.URL+apipaths.DoctorAuthenticateURLPath, strings.NewReader(params.Encode()))
	test.OK(t, err)
	req.Header.Set("S-Version", "Patient;Feature;0.9.0;000105")
	req.Header.Set("S-OS", "iOS;7.1.1")
	req.Header.Set("S-Device", "Phone;iPhone6,1;640;1136;2.0")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	account, err := testData.AuthAPI.GetAccountForEmail(email)
	test.OK(t, err)

	appInfo, err := testData.AuthAPI.LatestAppInfo(account.ID)
	test.OK(t, err)
	test.Equals(t, true, appInfo != nil)

}
