package media

import (
	"encoding/base64"
	"encoding/binary"
	"net/url"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/sig"
)

// TODO: Move this to a signer lib
type Signer struct {
	url    string
	signer *sig.Signer
	clk    clock.Clock
}

func NewSigner(serverURL string, signer *sig.Signer) *Signer {
	return &Signer{
		url:    serverURL,
		signer: signer,
		clk:    clock.New(),
	}
}

func (s *Signer) SignedURL(mediaID, mimetype, accountID string, width, height int, crop bool) (string, error) {
	params := url.Values{
		"id":       []string{mediaID},
		"mimetype": []string{mimetype},
	}

	if width > 0 {
		params.Set("width", strconv.Itoa(width))
	}
	if height > 0 {
		params.Set("height", strconv.Itoa(height))
	}
	if crop {
		params.Set("crop", strconv.FormatBool(crop))
	}

	sig, err := s.signer.Sign(makeSignedMsg(mediaID, mimetype, accountID, uint32(width), uint32(height), crop, time.Time{}))
	if err != nil {
		return "", err
	}
	params.Set("sig", base64.URLEncoding.EncodeToString(sig))
	return s.url + "?" + params.Encode(), nil
}

func (s *Signer) ExpiringSignedURL(mediaID, mimetype, accountID string, width, height int, crop bool, expires time.Time) (string, error) {
	params := url.Values{
		"id":       []string{mediaID},
		"mimetype": []string{mimetype},
	}

	if width > 0 {
		params.Set("width", strconv.Itoa(width))
	}
	if height > 0 {
		params.Set("height", strconv.Itoa(height))
	}
	if crop {
		params.Set("crop", strconv.FormatBool(crop))
	}
	params.Set("expires", strconv.FormatInt(expires.Unix(), 10))

	sig, err := s.signer.Sign(makeSignedMsg(mediaID, mimetype, accountID, uint32(width), uint32(height), crop, expires))
	if err != nil {
		return "", err
	}
	params.Set("sig", base64.URLEncoding.EncodeToString(sig))
	return s.url + "?" + params.Encode(), nil
}

func (s *Signer) ValidateSignature(mediaID, mimetype, accountID string, width, height int, crop bool, expires time.Time, sig string) bool {
	decSig, err := base64.URLEncoding.DecodeString(sig)
	if err != nil {
		return false
	}
	if s.signer.Verify(makeSignedMsg(mediaID, mimetype, accountID, uint32(width), uint32(height), crop, expires), decSig) {
		if !expires.IsZero() && s.clk.Now().After(expires) {
			return false
		}
		return true
	}
	return false
}

func makeSignedMsg(mediaID, mimetype, accountID string, width, height uint32, crop bool, expires time.Time) []byte {
	signedMsg := make([]byte, 10, 18+len(accountID)+len(mediaID)+len(mimetype))
	binary.BigEndian.PutUint32(signedMsg[0:4], width)
	binary.BigEndian.PutUint32(signedMsg[4:8], height)
	if crop {
		binary.BigEndian.PutUint16(signedMsg[8:], uint16(1))
	} else {
		binary.BigEndian.PutUint16(signedMsg[8:], uint16(0))
	}
	signedMsg = append(signedMsg, []byte(accountID)...)
	signedMsg = append(signedMsg, []byte(mediaID)...)
	signedMsg = append(signedMsg, []byte(mimetype)...)
	if !expires.IsZero() {
		signedMsg = signedMsg[:len(signedMsg)+8]
		binary.BigEndian.PutUint64(signedMsg[len(signedMsg)-8:], uint64(expires.Unix()))
	}
	return signedMsg
}
