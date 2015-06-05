package storage

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/s3"
)

var sseAlgorithm = "AES256"

type S3 struct {
	s3            *s3.S3
	bucket        string
	prefix        string
	latchedExpire bool
}

func NewS3(awsConfig *aws.Config, bucket, prefix string) *S3 {
	// Make sure the path prefix starts and ends with /
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	return &S3{
		s3:     s3.New(awsConfig),
		bucket: bucket,
		prefix: prefix,
	}
}

func (s *S3) LatchedExpire(enabled bool) {
	s.latchedExpire = enabled
}

func (s *S3) IDFromName(name string) string {
	return fmt.Sprintf("s3://%s/%s%s%s", s.s3.Config.Region, s.bucket, s.prefix, name)
}

func (s *S3) Put(name string, data []byte, contentType string, meta map[string]string) (string, error) {
	return s.PutReader(name, bytes.NewReader(data), int64(len(data)), contentType, meta)
}

func (s *S3) PutReader(name string, r io.ReadSeeker, size int64, contentType string, meta map[string]string) (string, error) {
	var m map[string]*string
	if len(meta) != 0 {
		m = make(map[string]*string, len(meta))
		for k, v := range meta {
			m[k] = &v
		}
	}
	if contentType == "" {
		// TODO: could use the mime package to try to detect type based on extension
		contentType = "application/binary"
	}
	path := s.prefix + name
	_, err := s.s3.PutObject(&s3.PutObjectInput{
		Bucket:               &s.bucket,
		Key:                  &path,
		Body:                 r,
		ContentLength:        &size,
		ContentType:          &contentType,
		ServerSideEncryption: &sseAlgorithm,
		Metadata:             m,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("s3://%s/%s%s", s.s3.Config.Region, s.bucket, path), nil
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
	region, bkt, path, err := s.parseURI(id)
	if err != nil {
		return nil, nil, err
	}
	// TODO(samuel): Support different regions
	_ = region
	obj, err := s.s3.GetObject(&s3.GetObjectInput{
		Bucket: &bkt,
		Key:    &path,
	})
	if err != nil {
		return nil, nil, err
	}
	header := http.Header{}
	if obj.ContentType != nil {
		header.Set("Content-Type", *obj.ContentType)
	}
	for k, v := range obj.Metadata {
		if v != nil {
			header.Set(k, *v)
		}
	}
	return obj.Body, header, nil
}

func (s *S3) SignedURL(id string, expires time.Duration) (string, error) {
	region, bkt, path, err := s.parseURI(id)
	if err != nil {
		return "", err
	}
	// TODO(samuel): Support different regions
	_ = region
	req, _ := s.s3.GetObjectRequest(&s3.GetObjectInput{
		Bucket: &bkt,
		Key:    &path,
	})
	now := time.Now().UTC()
	if s.latchedExpire {
		// Set expire time to end of the following period so the actual
		// expire duration is somewhere between `expires` and `2*expires`
		ex := int64(expires / time.Second)
		expires = time.Second * time.Duration(2*ex-(now.Unix()%ex))
	}
	return req.Presign(expires)
}

func (s *S3) Delete(id string) error {
	region, bkt, path, err := s.parseURI(id)
	if err != nil {
		return err
	}
	// TODO(samuel): Support different regions
	_ = region
	_, err = s.s3.DeleteObject(&s3.DeleteObjectInput{
		Bucket: &bkt,
		Key:    &path,
	})
	return err
}

func (s *S3) parseURI(uri string) (region string, bucket string, key string, err error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", "", "", err
	}
	p := strings.SplitN(u.Path, "/", 3)
	if len(p) < 3 {
		return "", "", "", fmt.Errorf("storage: bad S3 path %s", u.Path)
	}
	return u.Host, p[1], "/" + p[2], nil
}
