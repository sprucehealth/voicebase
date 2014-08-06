package test_integration

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/passreset"
	"github.com/sprucehealth/backend/test"
)

func TestAuth(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	email, pass, pass2 := "someone@somewhere.com", "somepass", "newPass"

	platform := api.Platform("test")

	sAccountID, err := testData.AuthApi.CreateAccount(email, pass, "DOCTOR")
	test.OK(t, err)
	if sAccountID <= 0 {
		t.Fatalf("CreateAccount returned invalid AccountId: %d", sAccountID)
	}

	sToken, err := testData.AuthApi.CreateToken(sAccountID, platform)
	if err != nil {
		t.Fatal(err)
	}
	if sToken == "" {
		t.Fatalf("CreateToken returned a blank Token")
	}

	// Make sure token is valid
	if account, err := testData.AuthApi.ValidateToken(sToken, platform); err != nil {
		t.Fatal(err)
	} else if account.ID != sAccountID {
		t.Fatalf("ValidateToken returned differnet AccountId")
	} else if account.Role != "DOCTOR" {
		t.Fatalf("ValidateToken returned role '%s', expected 'DOCTOR'", account.Role)
	}
	lAccount, err := testData.AuthApi.Authenticate(email, pass)
	test.OK(t, err)

	if sAccountID != lAccount.ID {
		t.Fatalf("AccountId doesn't match between login and singup")
	}
	_, err = testData.AuthApi.CreateToken(lAccount.ID, platform)
	if err != nil {
		t.Fatal(err)
	}
	// Make sure token from Signup is no longer valid
	if _, err := testData.AuthApi.ValidateToken(sToken, platform); err == api.TokenDoesNotExist {
		// Expected
	} else if err != nil {
		t.Fatal(err)
	} else {
		t.Fatalf("Token returned by Signup still valid after Login")
	}
	if err := testData.AuthApi.SetPassword(lAccount.ID, pass2); err != nil {
		t.Fatal(err)
	}
	// Make sure token from Signup is no longer valid
	if _, err := testData.AuthApi.ValidateToken(sToken, platform); err == api.TokenDoesNotExist {
		// Expected
	} else if err != nil {
		t.Fatal(err)
	} else {
		t.Fatalf("Token returned by Login still valid after SetPassword")
	}
	// Try to login with new password
	lAccount, err = testData.AuthApi.Authenticate(email, pass2)
	test.OK(t, err)

	if sAccountID != lAccount.ID {
		t.Fatalf("AccountId doesn't match between login and singup")
	}

	token, err := testData.AuthApi.CreateToken(lAccount.ID, platform)
	if err != nil {
		t.Fatal(err)
	}

	if a, err := testData.AuthApi.ValidateToken(token, platform); err != nil {
		t.Fatal(err)
	} else if a.ID != lAccount.ID {
		t.Fatalf("ValidateToken returned differnet AccountId")
	}

	if err := testData.AuthApi.DeleteToken(token); err != nil {
		t.Fatal(err)
	}
	// Make sure token is no longer valid
	if _, err := testData.AuthApi.ValidateToken(token, platform); err == api.TokenDoesNotExist {
		// Expected
	} else if err != nil {
		t.Fatal(err)
	} else {
		t.Fatalf("Token returned by Login still valid after Logout")
	}
}

func TestLostPassword(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	em := &email.TestService{}

	h := passreset.NewForgotPasswordHandler(testData.DataApi, testData.AuthApi, em, "support@sprucehealth.com", "www")

	req := JSONPOSTRequest(t, "/", &passreset.ForgotPasswordRequest{Email: "does-not-exist@nowhere.com"})
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if !strings.Contains(res.Body.String(), "No account with") {
		t.Fatalf("Expected 'No account' error. Got '%s'", res.Body.String())
	}

	validEmail := "exists@somewhere.com"
	_, err := testData.AuthApi.CreateAccount(validEmail, "xxx", "DOCTOR")
	test.OK(t, err)

	req = JSONPOSTRequest(t, "/", &passreset.ForgotPasswordRequest{Email: validEmail})
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if body := strings.TrimSpace(res.Body.String()); body != `{"result":"success"}` {
		t.Fatalf(`Expected '{"result":"success"}' got '%s'`, body)
	}

	if len(em.Email) != 1 {
		t.Fatalf("Expected 1 sent email. Got %d", len(em.Email))
	}

	t.Log(em.Email[0].BodyText)
}
