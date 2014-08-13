package storage

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/aws"
	"github.com/sprucehealth/backend/libs/golog"
	goamz "github.com/sprucehealth/backend/third_party/launchpad.net/goamz/aws"
	"github.com/sprucehealth/backend/third_party/launchpad.net/goamz/s3"
)

type S3 struct {
	auth   aws.Auth
	region goamz.Region
	bucket string
	prefix string
}

func NewS3(auth aws.Auth, region, bucket, prefix string) *S3 {
	// Make sure the path prefix starts and ends with /
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	reg, ok := goamz.Regions[region]
	if !ok {
		reg = goamz.USEast
	}
	return &S3{
		auth:   auth,
		region: reg,
		bucket: bucket,
		prefix: prefix,
	}
}

func (s *S3) bkt() *s3.Bucket {
	return s3.New(common.AWSAuthAdapter(s.auth), s.region).Bucket(s.bucket)
}

func (s *S3) parseURI(uri string) (*s3.Bucket, string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, "", err
	}
	region, ok := goamz.Regions[u.Host]
	if !ok {
		return nil, "", fmt.Errorf("storage: unknown S3 region %s", u.Host)
	}
	p := strings.SplitN(u.Path, "/", 3)
	if len(p) < 3 {
		return nil, "", fmt.Errorf("storage: bad S3 path %s", u.Path)
	}
	bucket := p[1]
	path := "/" + p[2]
	return s3.New(common.AWSAuthAdapter(s.auth), region).Bucket(bucket), path, nil
}

func (s *S3) Put(name string, data []byte, headers http.Header) (string, error) {
	return s.PutReader(name, bytes.NewReader(data), int64(len(data)), headers)
}

func (s *S3) PutReader(name string, r io.Reader, size int64, headers http.Header) (string, error) {
	if headers == nil {
		golog.Infof("there is no header")
		headers = http.Header{}
	}
	headers.Set("x-amz-server-side-encryption", "AES256")
	if headers.Get("Content-Type") == "" {
		// TODO: could use the mime package to try to detect type based on extension
		headers.Set("Content-Type", "application/binary")
	}
	path := s.prefix + name
	err := s.bkt().PutReader(path, r, size, headers.Get("Content-Type"), s3.BucketOwnerFull, headers)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("s3://%s/%s%s", s.region.Name, s.bucket, path), nil
}

func (s *S3) Get(id string) ([]byte, http.Header, error) {
	r, headers, err := s.GetReader(id)
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, r); err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), headers, nil
}

func (s *S3) GetReader(id string) (io.ReadCloser, http.Header, error) {
	bkt, path, err := s.parseURI(id)
	if err != nil {
		return nil, nil, err
	}
	return bkt.GetReader(path)
}

func (s *S3) GetSignedURL(id string, expires time.Time) (string, error) {
	bkt, path, err := s.parseURI(id)
	if err != nil {
		return "", err
	}
	// The URL is signed so that it is valid for 24 hours, so it can be used to cache the file on the client. However this logic
	// needs to be double-checked because when the visit is reloaded, we will send them a new URL that is signed from the time of the
	// new request, so it will have a different signature/URL.
	return bkt.SignedURL(path, expires, nil), nil
}

func (s *S3) Delete(id string) error {
	bkt, path, err := s.parseURI(id)
	if err != nil {
		return err
	}
	return bkt.Del(path)
}
