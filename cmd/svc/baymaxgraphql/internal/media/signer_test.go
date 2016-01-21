package media

import (
	"net/url"
	"strconv"

	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/test"

	"testing"
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

	test.Equals(t, true, signer.ValidateSignature(parsedMediaID, parsedMimetype, accountID, parsedWidth, parsedHeight, parsedCrop, parsedSignature))

	test.Equals(t, false, signer.ValidateSignature(parsedMediaID, parsedMimetype, accountID, parsedWidth, 110, parsedCrop, parsedSignature))
}
