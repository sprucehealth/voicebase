package twilio

// JSON Web Token implementation
// Minimum implementation based on this spec:
// http://self-issued.info/docs/draft-jones-json-web-token-01.html

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"hash"
)

const jwtType = "JWT"

type errInvalidJWT string

func (e errInvalidJWT) Error() string {
	return "twilio: JWT failed validation: " + string(e)
}

type jwtSigner struct {
	name    string
	size    int
	hashNew func() hash.Hash
}

type jwtHeader struct {
	Type      string `json:"typ"`
	Algorithm string `json:"alg"`
}

var (
	hs256 = jwtSigner{name: "HS256", size: sha256.Size, hashNew: sha256.New}
	hs384 = jwtSigner{name: "HS384", size: sha512.Size384, hashNew: sha512.New384}
	hs512 = jwtSigner{name: "HS512", size: sha512.Size, hashNew: sha512.New}
)

func (s jwtSigner) sign(dest, key, data []byte) ([]byte, error) {
	h := hmac.New(s.hashNew, key)
	if _, err := h.Write(data); err != nil {
		return nil, err
	}
	return h.Sum(dest), nil
}

func jwtEncode(payload interface{}, key []byte, signer jwtSigner, header map[string]interface{}) ([]byte, error) {
	if header == nil {
		header = make(map[string]interface{})
	}
	header["typ"] = jwtType
	header["alg"] = signer.name

	// Generate the following while minimizing allocations:
	// data = base64(json(header)) + "." + base64(json(payload))
	// signature = sign(data)
	// result = data + "." + base64(signature)

	s1, err := json.Marshal(header)
	if err != nil {
		return nil, err
	}
	s2, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	n1 := base64.RawURLEncoding.EncodedLen(len(s1))
	n2 := base64.RawURLEncoding.EncodedLen(len(s2))
	segments := make([]byte, n1+n2+base64.RawURLEncoding.EncodedLen(signer.size)+2)
	base64.RawURLEncoding.Encode(segments, s1)
	segments[n1] = '.'
	base64.RawURLEncoding.Encode(segments[n1+1:], s2)

	// Sign and append signature
	s2, err = signer.sign(s2[:0], key, segments[:n1+n2+1])
	segments[n1+n2+1] = '.'
	base64.RawURLEncoding.Encode(segments[n1+n2+2:], s2)
	return segments, nil
}

func jwtDecode(token, key []byte, v interface{}) error {
	ix := bytes.LastIndexByte(token, '.')
	if ix <= 0 {
		return errInvalidJWT("missing segment")
	}
	signed := token[:ix]
	sigB64 := token[ix+1:]
	ix = bytes.IndexByte(signed, '.')
	if ix <= 0 {
		return errInvalidJWT("missing segment")
	}
	headerB64 := signed[:ix]
	payloadB64 := signed[ix+1:]

	// Figure out the maximum size of a decoded segment to minimize allocation
	headLen := base64.RawURLEncoding.DecodedLen(len(headerB64))
	payloadLen := base64.RawURLEncoding.DecodedLen(len(payloadB64))
	sigLen := base64.RawURLEncoding.DecodedLen(len(sigB64))
	maxN := headLen
	if payloadLen > maxN {
		maxN = payloadLen
	}
	if sigLen > maxN {
		maxN = sigLen
	}

	b := make([]byte, maxN+sigLen) // bit of padding to right out the decoded sig as well as the generated sig
	n, err := base64.RawURLEncoding.Decode(b, headerB64)
	if err != nil {
		return errInvalidJWT("invalid header base64")
	}
	var header jwtHeader
	if err := json.Unmarshal(b[:n], &header); err != nil {
		return errInvalidJWT("invalid header json")
	}
	if header.Type != jwtType {
		return errInvalidJWT("invalid type")
	}
	var signer jwtSigner
	switch header.Algorithm {
	case hs256.name:
		signer = hs256
	case hs384.name:
		signer = hs384
	case hs512.name:
		signer = hs512
	default:
		return errInvalidJWT("unknown algorithm")
	}
	expSig, err := signer.sign(b[:0], key, signed)
	if err != nil {
		return err
	}
	sig := b[len(b)-sigLen:]
	if _, err := base64.RawURLEncoding.Decode(sig, sigB64); err != nil {
		return errInvalidJWT("invalid signature base64")
	}
	if subtle.ConstantTimeCompare(sig, expSig) != 1 {
		return errInvalidJWT("invalid signature")
	}

	n, err = base64.RawURLEncoding.Decode(b, payloadB64)
	if err != nil {
		return errInvalidJWT("invalid payload base64")
	}
	if err := json.Unmarshal(b[:n], v); err != nil {
		return errInvalidJWT("invalid payload json")
	}
	return nil
}
