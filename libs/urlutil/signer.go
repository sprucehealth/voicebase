package urlutil

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/sig"
)

// Signer provides mechanisms for signing urls
type Signer struct {
	baseURL string
	signer  *sig.Signer
	clk     clock.Clock
}

// NewSigner created a generic URL signer with no param type or ordering optimizations
func NewSigner(baseURL string, signer *sig.Signer, clk clock.Clock) *Signer {
	return &Signer{
		baseURL: baseURL,
		signer:  signer,
		clk:     clk,
	}
}

const SigParamName = "sig"
const expiresParamName = "expires"

// SignedURL generates the signed url for the provided params and expires at the provided time. If nil or a zero value time is provided, the url will never expire.
// Note: This function reserves the right to use the "expires" param name
func (s *Signer) SignedURL(uPath string, params url.Values, expires *time.Time) (string, error) {
	if expires != nil && !expires.IsZero() {
		params.Set(expiresParamName, strconv.FormatInt(expires.Unix(), 10))
	}
	sig, err := s.signer.Sign(makeSignedData(uPath, params))
	if err != nil {
		return "", err
	}
	params.Set(SigParamName, base64.URLEncoding.EncodeToString(sig))
	return path.Join(s.baseURL, uPath) + "?" + params.Encode(), nil
}

// ErrExpiredURL is returned when the provided signature has expired
var ErrExpiredURL = errors.New("the url has expired")

// ErrSignatureMismatch is returned when the provided signature does not match
var ErrSignatureMismatch = errors.New("the signature does not match")

// ValidateSignature validates that the provided signature matches the params and path
func (s *Signer) ValidateSignature(uPath string, params url.Values) error {
	sig := params.Get(SigParamName)
	if sig == "" {
		return fmt.Errorf("no signature in param set %v for path %s", params, uPath)
	}
	params.Del(SigParamName)
	expires := params.Get(expiresParamName)
	if expires != "" {
		expiresEpoch, err := strconv.ParseInt(expires, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse expiration epoch param %s", expires)
		}
		if s.clk.Now().After(time.Unix(expiresEpoch, 0)) {
			return ErrExpiredURL
		}
	}
	decSig, err := base64.URLEncoding.DecodeString(sig)
	if err != nil {
		return fmt.Errorf("unable to decode signature %s", sig)
	}
	if !s.signer.Verify(makeSignedData(uPath, params), decSig) {
		return ErrSignatureMismatch
	}
	return nil
}

// Note: This is a generic signature mechanism and does not have a lot of optimizations that a typed/paramed implementation could have
func makeSignedData(uPath string, params url.Values) []byte {
	return []byte(uPath + "?" + params.Encode())
}
