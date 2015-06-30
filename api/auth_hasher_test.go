package api

import (
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestAuthHasherBCrypt(t *testing.T) {
	h := NewBcryptHasher(1)
	hash, err := h.GenerateFromPassword([]byte("abc"))
	test.OK(t, err)
	test.OK(t, h.CompareHashAndPassword(hash, []byte("abc")))
	test.Assert(t, nil != h.CompareHashAndPassword(hash, []byte("xyz")), "Wrong password matched")
	test.Assert(t, nil != h.CompareHashAndPassword([]byte(""), []byte("")), "Empty hash should not match against empty password")
}
