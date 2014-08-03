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

	sAccountID, sToken, err := testData.AuthApi.SignUp(email, pass, "DOCTOR")
	test.OK(t, err)
	if sAccountID <= 0 {
		t.Fatalf("Signup returned invalid AccountId: %d", sAccountID)
	}
	if sToken == "" {
		t.Fatalf("Signup returned a blank Token")
	}

	// Make sure token is valid
	if account, err := testData.AuthApi.ValidateToken(sToken); err != nil {
		t.Fatal(err)
	} else if account.ID != sAccountID {
		t.Fatalf("ValidateToken returned differnet AccountId")
	} else if account.Role != "DOCTOR" {
		t.Fatalf("ValidateToken returned role '%s', expected 'DOCTOR'", account.Role)
	}
	lAccount, token, err := testData.AuthApi.LogIn(email, pass)
	test.OK(t, err)

	if sAccountID != lAccount.ID {
		t.Fatalf("AccountId doesn't match between login and singup")
	}
	// Make sure token from Signup is no longer valid
	if _, err := testData.AuthApi.ValidateToken(sToken); err == api.TokenDoesNotExist {
		// Expected
	} else if err != nil {
		t.Fatal(err)
	} else {
		t.Fatalf("Token returned by Signup still valid after Login")
	}
	// Make sure login token is valid
	if a, err := testData.AuthApi.ValidateToken(token); err != nil {
		t.Fatal(err)
	} else if a.ID != lAccount.ID {
		t.Fatalf("ValidateToken returned differnet AccountId")
	}
	if err := testData.AuthApi.SetPassword(lAccount.ID, pass2); err != nil {
		t.Fatal(err)
	}
	// Make sure token from Signup is no longer valid
	if _, err := testData.AuthApi.ValidateToken(sToken); err == api.TokenDoesNotExist {
		// Expected
	} else if err != nil {
		t.Fatal(err)
	} else {
		t.Fatalf("Token returned by Login still valid after SetPassword")
	}
	// Try to login with new password
	lAccount, token, err = testData.AuthApi.LogIn(email, pass2)
	test.OK(t, err)

	if sAccountID != lAccount.ID {
		t.Fatalf("AccountId doesn't match between login and singup")
	}

	// Make sure login token is valid
	if a, err := testData.AuthApi.ValidateToken(token); err != nil {
		t.Fatal(err)
	} else if a.ID != lAccount.ID {
		t.Fatalf("ValidateToken returned differnet AccountId")
	}

	if err := testData.AuthApi.LogOut(token); err != nil {
		t.Fatal(err)
	}
	// Make sure token is no longer valid
	if _, err := testData.AuthApi.ValidateToken(token); err == api.TokenDoesNotExist {
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
	_, _, err := testData.AuthApi.SignUp(validEmail, "xxx", "DOCTOR")
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
