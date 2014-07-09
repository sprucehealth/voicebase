package common

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/subtle"
	"hash"
)

// Signer provides an easy way to deal with signatures and key rotation.
// Generating a signature always uses the first key (the latest), but
// verifying a signatures tests all keys and if one matches then the
// signature is conisdered valid. If HashNew is nil then SHA1 is used
// as the hash for HMAC.
type Signer struct {
	Keys    [][]byte
	HashNew func() hash.Hash
}

// Sign the key witih the first key.
func (s *Signer) Sign(msg []byte) ([]byte, error) {
	return s.sign(msg, s.Keys[0])
}

// Verify the message and signature against all keys.
func (s *Signer) Verify(msg, sig []byte) bool {
	// Try all keys every time to avoid leaking key rotation information.
	// If the loop exited early then it would be possible to know which
	// key was used. Probably not a big deal, but the hmac should be fast
	// enough to not matter testing all keys.
	var eq int
	for _, key := range s.Keys {
		exp, err := s.sign(msg, key)
		if err == nil && len(sig) == len(exp) {
			eq += subtle.ConstantTimeCompare(sig, exp)
		}
	}
	return eq != 0
}

func (s *Signer) hashNew() func() hash.Hash {
	if s.HashNew != nil {
		return s.HashNew
	}
	return sha1.New
}

func (s *Signer) sign(msg, key []byte) ([]byte, error) {
	h := hmac.New(s.hashNew(), key)
	if _, err := h.Write(msg); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}
