package api

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateToken() (string, error) {
	tokBytes := make([]byte, 16)
	if _, err := rand.Read(tokBytes); err != nil {
		return "", err
	}

	tok := hex.EncodeToString(tokBytes)
	return tok, nil
}
