package apiservice

import (
	"net/http"
	"testing"
)

func TestContext(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	ctx := GetContext(req)
	if ctx == nil {
		t.Fatal("Expected a valid empty Context from first call to GetContext")
	}
	if ctx.AccountID != 0 {
		t.Fatal("Expected AccountId of 0 on new Context")
	}
	ctx.AccountID = 123
	if ctx2 := GetContext(req); ctx2.AccountID != ctx.AccountID {
		t.Fatal("Write to context failed")
	}
	DeleteContext(req)
	if ctx := GetContext(req); ctx.AccountID != 0 {
		t.Fatal("DeleteContext failed")
	}
}
