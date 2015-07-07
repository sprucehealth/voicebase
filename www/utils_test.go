package www

import (
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
)

type stubAuthAPI struct {
	api.AuthAPI
	account *common.Account
}

func (s *stubAuthAPI) ValidateToken(token string, platform api.Platform) (*common.Account, error) {
	return s.account, nil
}

func TestValidateRedirectURL(t *testing.T) {
	cases := []struct {
		url   string
		valid bool
		next  string
	}{
		{url: "http://localhost", valid: false},
		{url: "http://localhost/", valid: true, next: "/"},
		{url: "http://localhost/admin", valid: true, next: "/admin"},
		{url: "http://localhost/admin?blah=123", valid: true, next: "/admin"},
	}
	for _, tc := range cases {
		next, ok := validateRedirectURL(tc.url)
		test.Equals(t, tc.valid, ok)
		test.Equals(t, tc.next, next)
	}
}

func TestCookieAuth(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	test.OK(t, err)
	r.Host = "api.spruth.com:1234"
	cookie := NewAuthCookie("monster", r)
	test.Equals(t, "api.spruth.com", cookie.Domain)

	authAPI := &stubAuthAPI{
		account: &common.Account{ID: 6},
	}

	_, err = ValidateAuth(authAPI, r)
	test.Equals(t, http.ErrNoCookie, err)

	r.AddCookie(cookie)
	account, err := ValidateAuth(authAPI, r)
	test.OK(t, err)
	test.Assert(t, account != nil, "Account should not be nil")
	test.Equals(t, int64(6), account.ID)
}
