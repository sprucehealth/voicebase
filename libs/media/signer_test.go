package media

import (
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/test"
)

func TestSigner(t *testing.T) {
	si, err := sig.NewSigner([][]byte{[]byte("hello")}, nil)
	test.OK(t, err)
	signer := NewSigner("http://test.com", si)
	accountID := "12312414515"

	signedURL, err := signer.SignedURL("hello:12345", "image/jpeg", "12312414515", 100, 100, true)
	test.OK(t, err)

	u, err := url.Parse(signedURL)
	test.OK(t, err)
	params := u.Query()

	parsedMediaID := params.Get("id")
	parsedMimetype := params.Get("mimetype")
	parsedWidth, err := strconv.Atoi(params.Get("width"))
	test.OK(t, err)
	parsedHeight, err := strconv.Atoi(params.Get("height"))
	test.OK(t, err)
	parsedCrop, err := strconv.ParseBool(params.Get("crop"))
	test.OK(t, err)
	parsedSignature := params.Get("sig")

	test.Equals(t, true, signer.ValidateSignature(parsedMediaID, parsedMimetype, accountID, parsedWidth, parsedHeight, parsedCrop, time.Time{}, parsedSignature))
	test.Equals(t, false, signer.ValidateSignature(parsedMediaID, parsedMimetype, accountID, parsedWidth, 110, parsedCrop, time.Time{}, parsedSignature))
}

func TestExpiringSigner(t *testing.T) {
	si, err := sig.NewSigner([][]byte{[]byte("hello")}, nil)
	test.OK(t, err)
	signer := NewSigner("http://test.com", si)
	clk := clock.NewManaged(time.Now())
	signer.clk = clk
	accountID := "12312414515"

	expires := clk.Now().Add(time.Minute)
	expiringSignedURL, err := signer.ExpiringSignedURL("hello:12345", "image/jpeg", "12312414515", 100, 100, true, expires)
	test.OK(t, err)

	u, err := url.Parse(expiringSignedURL)
	test.OK(t, err)
	params := u.Query()

	parsedMediaID := params.Get("id")
	parsedMimetype := params.Get("mimetype")
	parsedWidth, err := strconv.Atoi(params.Get("width"))
	test.OK(t, err)
	parsedHeight, err := strconv.Atoi(params.Get("height"))
	test.OK(t, err)
	parsedCrop, err := strconv.ParseBool(params.Get("crop"))
	test.OK(t, err)
	parsedExpiration, err := strconv.ParseUint(params.Get("expires"), 10, 64)
	test.OK(t, err)
	expiration := time.Unix(int64(parsedExpiration), 0)
	parsedSignature := params.Get("sig")

	// Expiring Signed
	test.Equals(t, true, signer.ValidateSignature(parsedMediaID, parsedMimetype, accountID, parsedWidth, parsedHeight, parsedCrop, expiration, parsedSignature))
	test.Equals(t, false, signer.ValidateSignature(parsedMediaID, parsedMimetype, accountID, parsedWidth, 110, parsedCrop, expiration, parsedSignature))

	// Expire the URL
	clk.WarpForward(time.Minute * 2)
	test.Equals(t, false, signer.ValidateSignature(parsedMediaID, parsedMimetype, accountID, parsedWidth, parsedHeight, parsedCrop, expiration, parsedSignature))
}
