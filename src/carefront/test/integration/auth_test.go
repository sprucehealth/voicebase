package integration

import (
	"testing"
	"time"
)

func TestAuth(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	email, pass, pass2 := "someone@somewhere.com", "somepass", "newPass"

	ts := time.Now()
	signup, err := testData.AuthApi.SignUp(email, pass)
	if err != nil {
		t.Fatal(err)
	}
	if signup.AccountId <= 0 {
		t.Fatalf("Signup returned invalid AccountId: %d", signup.AccountId)
	}
	if signup.Token == "" {
		t.Fatalf("Signup returned a blank Token")
	}
	t.Logf("Time to signup %.3f seconds", float64(time.Since(ts))/float64(time.Second))

	ts = time.Now()
	// Make sure token is valid
	if res, err := testData.AuthApi.ValidateToken(signup.Token); err != nil {
		t.Fatal(err)
	} else if !res.IsValid {
		t.Fatalf("ValidateToken failed for token returned from Signup")
	} else if *res.AccountId != signup.AccountId {
		t.Fatalf("ValidateToken returned differnet AccountId")
	}
	t.Logf("Time to validate token %.3f seconds", float64(time.Since(ts))/float64(time.Second))

	ts = time.Now()
	login, err := testData.AuthApi.LogIn(email, pass)
	if err != nil {
		t.Fatal(err)
	}

	if signup.AccountId != login.AccountId {
		t.Fatalf("AccountId doesn't match between login and singup")
	}
	t.Logf("Time to login %.3f seconds", float64(time.Since(ts))/float64(time.Second))

	ts = time.Now()
	// Make sure token from Signup is no longer valid
	if res, err := testData.AuthApi.ValidateToken(signup.Token); err != nil {
		t.Fatal(err)
	} else if res.IsValid {
		t.Fatalf("Token returned by Signup still valid after new Login")
	}
	t.Logf("Time to validate token after signup %.3f seconds", float64(time.Since(ts))/float64(time.Second))

	ts = time.Now()
	// Make sure login token is valid
	if res, err := testData.AuthApi.ValidateToken(login.Token); err != nil {
		t.Fatal(err)
	} else if !res.IsValid {
		t.Fatalf("ValidateToken failed for token returned from Login")
	} else if *res.AccountId != login.AccountId {
		t.Fatalf("ValidateToken returned differnet AccountId")
	}
	t.Logf("Time to validate token after login  %.3f seconds", float64(time.Since(ts))/float64(time.Second))

	ts = time.Now()
	if err := testData.AuthApi.SetPassword(login.AccountId, pass2); err != nil {
		t.Fatal(err)
	}
	t.Logf("Time to set password %.3f seconds", float64(time.Since(ts))/float64(time.Second))

	ts = time.Now()
	// Make sure token from Signup is no longer valid
	if res, err := testData.AuthApi.ValidateToken(signup.Token); err != nil {
		t.Fatal(err)
	} else if res.IsValid {
		t.Fatalf("Token returned by Login still valid after SetPassword")
	}
	t.Logf("Time to validate token after password change %.3f seconds", float64(time.Since(ts))/float64(time.Second))

	ts = time.Now()
	// Try to login with new password
	login, err = testData.AuthApi.LogIn(email, pass2)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Time to login after password change %.3f seconds", float64(time.Since(ts))/float64(time.Second))

	if signup.AccountId != login.AccountId {
		t.Fatalf("AccountId doesn't match between login and singup")
	}

	ts = time.Now()
	// Make sure login token is valid
	if res, err := testData.AuthApi.ValidateToken(login.Token); err != nil {
		t.Fatal(err)
	} else if !res.IsValid {
		t.Fatalf("ValidateToken failed for token returned from Login")
	} else if *res.AccountId != login.AccountId {
		t.Fatalf("ValidateToken returned differnet AccountId")
	}
	t.Logf("Time to validate token after login after password change %.3f seconds", float64(time.Since(ts))/float64(time.Second))

	ts = time.Now()
	if err := testData.AuthApi.LogOut(login.Token); err != nil {
		t.Fatal(err)
	}
	t.Logf("Time to logout %.3f seconds", float64(time.Since(ts))/float64(time.Second))

	ts = time.Now()
	// Make sure token is no longer valid
	if res, err := testData.AuthApi.ValidateToken(login.Token); err != nil {
		t.Fatal(err)
	} else if res.IsValid {
		t.Fatalf("Token returned by Login still valid after Logout")
	}
	t.Logf("Time to validate token after logout %.3f seconds", float64(time.Since(ts))/float64(time.Second))
}
