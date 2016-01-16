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
	storage.DeterministicStore
	url    string
	signer *sig.Signer
}

func NewStore(serverURL string, signer *sig.Signer, store storage.DeterministicStore) *Store {
	return &Store{
		DeterministicStore: store,
		url:                serverURL,
		signer:             signer,
	}
}

func (s *Store) SignedURL(mediaID, mimetype string, accountID uint64, expiration time.Duration, width, height int, crop bool) (string, error) {
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

	var expireTime int64
	if expiration != 0 {
		expireTime = time.Now().Add(expiration).Unix()
		params.Set("expires", strconv.FormatInt(expireTime, 10))
	}

	sig, err := s.signer.Sign(makeSignedMsg(mediaID, mimetype, accountID, uint64(expireTime), uint32(width), uint32(height), crop))
	if err != nil {
		return "", err
	}
	params.Set("sig", base64.URLEncoding.EncodeToString(sig))
	return s.url + "?" + params.Encode(), nil
}

func (s *Store) ValidateSignature(mediaID, mimetype string, accountID, expireTime uint64, width, height int, crop bool, sig string) bool {
	decSig, err := base64.URLEncoding.DecodeString(sig)
	if err != nil {
		return false
	}
	return s.signer.Verify(makeSignedMsg(mediaID, mimetype, accountID, expireTime, uint32(width), uint32(height), crop), decSig)
}

func makeSignedMsg(mediaID, mimetype string, expireTime, accountID uint64, width, height uint32, crop bool) []byte {
	signedMsg := make([]byte, 0, (8*3+2)+len(mediaID)+len(mimetype))
	binary.BigEndian.PutUint64(signedMsg[:8], expireTime)
	binary.BigEndian.PutUint64(signedMsg[8:16], accountID)
	binary.BigEndian.PutUint32(signedMsg[16:20], width)
	binary.BigEndian.PutUint32(signedMsg[20:24], height)
	if crop {
		binary.BigEndian.PutUint16(signedMsg[24:26], uint16(1))
	} else {
		binary.BigEndian.PutUint16(signedMsg[24:26], uint16(0))
	}
	signedMsg = append(signedMsg, []byte(mediaID)...)
	signedMsg = append(signedMsg, []byte(mimetype)...)

	return signedMsg
}
