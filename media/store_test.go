package media

import (
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
)

func TestStore(t *testing.T) {
	signer, err := sig.NewSigner([][]byte{[]byte("xxx")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	baseStore := storage.NewTestStore(nil)
	store := NewStore("http://example.com", signer, baseStore)
	signedURL, err := store.SignedURL(123, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	surl, err := url.Parse(signedURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(signedURL)
	params := surl.Query()
	if params.Get("media_id") != "123" {
		t.Fatalf("Expected media_id of '123', got '%s'", params.Get("media_id"))
	}
	expires, err := strconv.ParseInt(params.Get("expires"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	if !store.ValidateSignature(123, expires, params.Get("sig")) {
		t.Fatal("Signature should have been valid")
	}
}
