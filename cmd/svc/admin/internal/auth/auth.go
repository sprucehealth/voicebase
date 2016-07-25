package auth

import (
	"crypto/rand"
	"encoding/base64"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"context"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/sig"
)

var (
	up = big.NewInt(math.MaxInt64)
)

const (
	tokenDuration = time.Second * 60 * 60 * 24 * 30
)

func hash(id, nonce, expirationts string, signer *sig.Signer) (string, error) {
	h, err := signer.Sign([]byte(id + expirationts + nonce))
	return base64.StdEncoding.EncodeToString(h), errors.Trace(err)
}

// NewToken generates a signed token and expiration timestamp
func NewToken(ctx context.Context, uid string, signer *sig.Signer) (string, time.Time, error) {
	nonce := make([]byte, 32)
	_, err := rand.Read(nonce)
	if err != nil {
		golog.ContextLogger(ctx).Debugf("Error while generating nonce: %s", err)
		return "", time.Time{}, errors.Trace(err)
	}
	snonce := base64.StdEncoding.EncodeToString(nonce)
	expires := time.Now().Add(tokenDuration)
	expirationts := strconv.FormatInt(expires.Unix(), 10)
	h, err := hash(uid, snonce, expirationts, signer)
	return strings.Join([]string{h, uid, snonce, expirationts}, ":"), expires, errors.Trace(err)
}

// IsTokenValid returns a boolean value representing if the provided token is valid and the uid of the user if they are
func IsTokenValid(ctx context.Context, token string, signer *sig.Signer) (bool, string) {
	// We expect the token to be of the format token:uid:nonce:expiration_ts
	tSegs := strings.Split(token, ":")
	if len(tSegs) != 4 {
		golog.ContextLogger(ctx).Debugf("Expected 4 segments but got %v", tSegs)
		return false, ""
	}
	t := tSegs[0]
	uid := tSegs[1]
	nonce := tSegs[2]
	expirationts := tSegs[3]
	ht, err := hash(uid, nonce, expirationts, signer)
	if err != nil {
		golog.ContextLogger(ctx).Debugf("Error while matching hash for %s, %s, %s: %s", uid, nonce, expirationts, err)
		return false, ""
	}

	dt, err := base64.StdEncoding.DecodeString(t)
	if err != nil {
		golog.ContextLogger(ctx).Debugf("Error while decoding token %s: %s", t, err)
		return false, ""
	}

	if !signer.Verify([]byte(uid+expirationts+nonce), dt) {
		golog.ContextLogger(ctx).Debugf("Verification failed for %s - %s", string(t), string(ht))
		return false, ""
	}

	// If the hash matches assert that it's not expired
	ets, err := strconv.ParseInt(expirationts, 10, 64)
	if err != nil {
		golog.ContextLogger(ctx).Debugf("Error while parsing expiration timestamp", tSegs)
		return false, ""
	}

	if !time.Now().Before(time.Unix(ets, 0)) {
		golog.ContextLogger(ctx).Debugf("The token %s is expired at timestamp %d", token, ets)
		return false, ""
	}
	return true, uid
}
