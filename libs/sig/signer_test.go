package sig

import (
	"bytes"

	"testing"
)

func TestSigner(t *testing.T) {
	_, err := NewSigner(nil, nil)
	if err == nil {
		t.Error("Expected error on nil keys")
	}

	s1, err := NewSigner([][]byte{[]byte("key1")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	s2, err := NewSigner([][]byte{[]byte("key2")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	s3, err := NewSigner([][]byte{[]byte("key2"), []byte("key1")}, nil)
	if err != nil {
		t.Fatal(err)
	}

	msg := []byte("foobar")

	sig, err := s1.Sign(msg)
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) == 0 {
		t.Fatal("Zero length signature")
	}

	if !s1.Verify(msg, sig) {
		t.Fatalf("Signature did not verify: %+v", sig)
	}

	// Different key should not verify
	if s2.Verify(msg, sig) {
		t.Fatal("Different key should not verify")
	}

	// Old keys should still verify (key rotation)
	if s3.Verify(msg, sig) {
		t.Fatal("Old key did not verify")
	}

	// First key should be considered latest
	sig1, err := s2.Sign(msg)
	if err != nil {
		t.Fatal(err)
	}
	sig2, err := s2.Sign(msg)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Compare(sig1, sig2) {
		t.Fatalf("Did not use latest key for signing: %+v != %+v", sig1, sig2)
	}
}
