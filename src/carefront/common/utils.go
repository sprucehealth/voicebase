package common

import (
	"crypto/rand"
	"encoding/base64"

	"carefront/libs/aws"
	goamz "launchpad.net/goamz/aws"
)

func GenerateToken() (string, error) {
	tokBytes := make([]byte, 16)
	if _, err := rand.Read(tokBytes); err != nil {
		return "", err
	}

	tok := base64.URLEncoding.EncodeToString(tokBytes)
	return tok, nil
}

func AWSAuthAdapter(auth aws.Auth) goamz.Auth {
	keys := auth.Keys()
	return goamz.Auth{
		AccessKey: keys.AccessKey,
		SecretKey: keys.SecretKey,
		Token:     keys.Token,
	}
}
