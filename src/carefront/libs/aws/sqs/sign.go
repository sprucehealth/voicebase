package sqs

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"sort"
	"strings"

	"carefront/libs/aws"
)

var b64 = base64.StdEncoding

func (sqs *SQS) sign(method, path string, params url.Values, host string) {
	authKeys := sqs.Client.Auth.Keys()

	params.Set("AWSAccessKeyId", authKeys.AccessKey)
	params.Set("SignatureVersion", "2")
	params.Set("SignatureMethod", "HmacSHA256")
	if authKeys.Token != "" {
		params.Set("SecurityToken", authKeys.Token)
	}

	// AWS specifies that the parameters in a signed request must
	// be provided in the natural order of the keys. This is distinct
	// from the natural order of the encoded value of key=value.
	// Percent and equals affect the sorting order.
	var keys, sarray []string
	for k, _ := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		sarray = append(sarray, aws.Encode(k)+"="+aws.Encode(params[k][0]))
	}
	joined := strings.Join(sarray, "&")
	payload := method + "\n" + host + "\n" + path + "\n" + joined
	hash := hmac.New(sha256.New, []byte(authKeys.SecretKey))
	hash.Write([]byte(payload))
	params.Set("Signature", b64.EncodeToString(hash.Sum(nil)))
}
