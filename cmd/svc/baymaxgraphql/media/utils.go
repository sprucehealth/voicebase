package media

import (
	"fmt"
	"net/url"
	"strings"
)

func ParseMediaID(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	parts := strings.Split(u.Path, "/")
	if len(parts) != 4 {
		return "", fmt.Errorf("Expected uri of form s3://region/bucket/prefix/name but got %s", uri)
	}

	return parts[3], nil
}
