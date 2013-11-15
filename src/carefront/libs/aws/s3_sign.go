package aws

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/http"
	"sort"
	"strings"
	"time"
)

var base64Std = base64.StdEncoding

var s3ParamsToSign = map[string]bool{
	// "acl":                          true,
	// "location":                     true,
	// "logging":                      true,
	// "notification":                 true,
	// "partNumber":                   true,
	// "policy":                       true,
	// "requestPayment":               true,
	// "torrent":                      true,
	// "uploadId":                     true,
	// "uploads":                      true,
	// "versionId":                    true,
	// "versioning":                   true,
	// "versions":                     true,
	"response-content-type":        true,
	"response-content-language":    true,
	"response-expires":             true,
	"response-cache-control":       true,
	"response-content-disposition": true,
	"response-content-encoding":    true,
}

func (s3 *S3) sign(req *http.Request) {
	if req.Header.Get("Date") == "" {
		req.Header.Set("Date", time.Now().Format(time.RFC1123Z))
	}

	var md5, ctype, date, xamz string
	var xamzDate bool
	var sarray []string
	for k, v := range req.Header {
		k = strings.ToLower(k)
		switch k {
		case "content-md5":
			md5 = v[0]
		case "content-type":
			ctype = v[0]
		case "date":
			if !xamzDate {
				date = v[0]
			}
		default:
			if strings.HasPrefix(k, "x-amz-") {
				vall := strings.Join(v, ",")
				sarray = append(sarray, k+":"+vall)
				if k == "x-amz-date" {
					xamzDate = true
					date = ""
				}
			}
		}
	}
	if len(sarray) > 0 {
		sort.StringSlice(sarray).Sort()
		xamz = strings.Join(sarray, "\n") + "\n"
	}

	params := req.URL.Query()
	expires := false
	if v := params.Get("Expires"); v != "" {
		// Query string request authentication alternative.
		expires = true
		date = v
		params.Set("AWSAccessKeyId", s3.Client.Keys.AccessKey)
	}

	sarray = sarray[:0]
	for k, v := range params {
		if s3ParamsToSign[k] {
			for _, vi := range v {
				if vi == "" {
					sarray = append(sarray, k)
				} else {
					// "When signing you do not encode these values."
					sarray = append(sarray, k+"="+vi)
				}
			}
		}
	}

	path := req.URL.Path
	if len(sarray) > 0 {
		sort.StringSlice(sarray).Sort()
		path = path + "?" + strings.Join(sarray, "&")
	}

	payload := req.Method + "\n" + md5 + "\n" + ctype + "\n" + date + "\n" + xamz + path
	hash := hmac.New(sha1.New, []byte(s3.Client.Keys.SecretKey))
	hash.Write([]byte(payload))
	signature := make([]byte, base64Std.EncodedLen(hash.Size()))
	base64Std.Encode(signature, hash.Sum(nil))

	if expires {
		params.Set("Signature", string(signature))
		req.URL.RawQuery = params.Encode()
	} else {
		req.Header.Set("Authorization", "AWS "+s3.Client.Keys.AccessKey+":"+string(signature))
	}
}
