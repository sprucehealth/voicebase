package media

import (
	"encoding/base64"
	"encoding/binary"
	"net/url"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
)

type Store struct {
	storage.Store
	url    string
	signer *sig.Signer
}

func NewStore(serveURL string, signer *sig.Signer, store storage.Store) *Store {
	return &Store{
		Store:  store,
		url:    serveURL,
		signer: signer,
	}
}

func (s *Store) SignedURL(mediaID int64, expiration time.Duration) (string, error) {
	params := url.Values{
		"media_id": []string{strconv.FormatInt(mediaID, 10)},
	}
	var expireTime int64
	if expiration != 0 {
		expireTime = time.Now().Add(expiration).Unix()
		params.Set("expires", strconv.FormatInt(expireTime, 10))
	}
	sig, err := s.signer.Sign(makeSignedMsg(mediaID, expireTime))
	if err != nil {
		return "", err
	}
	params.Set("sig", base64.URLEncoding.EncodeToString(sig))
	return s.url + "?" + params.Encode(), nil
}

func (s *Store) ValidateSignature(mediaID, expireTime int64, sig string) bool {
	decSig, err := base64.URLEncoding.DecodeString(sig)
	if err != nil {
		return false
	}
	return s.signer.Verify(makeSignedMsg(mediaID, expireTime), decSig)
}

func makeSignedMsg(mediaID, expireTime int64) []byte {
	signedMsg := make([]byte, 8*2)
	binary.BigEndian.PutUint64(signedMsg[:8], uint64(mediaID))
	binary.BigEndian.PutUint64(signedMsg[8:16], uint64(expireTime))
	return signedMsg
}
