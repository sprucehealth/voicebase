package awsutil

import (
	"strings"

	"github.com/sprucehealth/backend/libs/errors"
)

func ResourceNameFromARN(arn string) (string, error) {
	idx := strings.LastIndex(arn, ":")
	if idx == -1 {
		return "", errors.Errorf("resource name not found in topic arn %s", arn)
	}

	return arn[idx+1:], nil
}

func ResourceNameFromSQSURL(url string) (string, error) {
	idx := strings.LastIndex(url, "/")
	if idx == -1 {
		return "", errors.Errorf("resource name not found in queue url %s", url)
	}

	return url[idx+1:], nil
}
