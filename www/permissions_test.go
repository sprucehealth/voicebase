package www

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type mockAPI struct {
	perms map[int64][]string
}

func (m *mockAPI) PermissionsForAccount(accountID int64) ([]string, error) {
	return m.perms[accountID], nil
}

func TestPermissions(t *testing.T) {
	perms := Permissions(map[string]bool{"abc": true, "xxx": true})
	if !perms.Has("abc") {
		t.Error("Permissions.Has failed true case")
	}
	if perms.Has("123") {
		t.Error("Permissions.Has failed false case")
	}
	if !perms.HasAll([]string{"abc", "xxx"}) {
		t.Error("Permissions.HasAll failed true case")
	}
	if perms.HasAll([]string{"abc", "111"}) {
		t.Error("Permissions.HasAll failed false case")
	}
	if !perms.HasAny([]string{"abc", "xxx"}) {
		t.Error("Permissions.HasAny failed true case")
	}
	if perms.HasAny([]string{"222", "111"}) {
		t.Error("Permissions.HasAny failed false case")
	}
}

func TestPermissionsHandler(t *testing.T) {
	api := &mockAPI{
		perms: map[int64][]string{
			1: []string{"aaa", "bbb"},
			2: []string{"123"},
		},
	}

	okHandler := httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	failedHandler := httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})

	h := PermissionsRequiredHandler(api, map[string][]string{"GET": []string{"aaa"}, "POST": []string{"aaa", "123"}}, okHandler, failedHandler)

	// Allowed matching 1 of 1 required premissions

	r, _ := http.NewRequest("GET", "/", nil)
	account := &common.Account{ID: 1}
	w := httptest.NewRecorder()
	h.ServeHTTP(CtxWithAccount(context.Background(), account), w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("GET request failed")
	}

	// Disallowed matching 0 of 1 required premissions

	r, _ = http.NewRequest("GET", "/", nil)
	account = &common.Account{ID: 2}
	w = httptest.NewRecorder()
	h.ServeHTTP(CtxWithAccount(context.Background(), account), w, r)
	if w.Code == http.StatusOK {
		t.Fatalf("GET request should have failed")
	}

	// Allowed matching 1 of 2 required premissions

	r, _ = http.NewRequest("POST", "/", nil)
	account = &common.Account{ID: 1}
	w = httptest.NewRecorder()
	h.ServeHTTP(CtxWithAccount(context.Background(), account), w, r)
	if w.Code == http.StatusForbidden {
		t.Fatalf("POST request should have been allowed")
	}
}
