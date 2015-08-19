package awsutil

import (
	"bytes"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sprucehealth/backend/libs/golog"
)

const multiChunkSize = 5 << 20

// PutMultiFrom uploads an object of unknown size using multipart upload with chunking.
func PutMultiFrom(s3c *s3.S3, bucket, path string, rd io.Reader, contentType, contentEncoding, acl string, meta map[string]*string) error {
	var multi *s3.CreateMultipartUploadOutput
	var parts []*s3.CompletedPart

	lr := &io.LimitedReader{R: rd, N: multiChunkSize}
	buf := bytes.NewBuffer(make([]byte, 0, bytes.MinRead))
	for nChunk := 1; ; nChunk++ {
		lr.N = multiChunkSize
		buf.Reset()
		if n, err := buf.ReadFrom(lr); err != nil {
			if multi != nil {
				_, err := s3c.AbortMultipartUpload(&s3.AbortMultipartUploadInput{
					Bucket:   &bucket,
					Key:      &path,
					UploadId: multi.UploadId,
				})
				if err != nil {
					golog.Errorf("Failed to abort multipart S3 upload: %s", err.Error())
				}
			}
			return err
		} else if n == 0 {
			break
		}

		if nChunk == 1 {
			if lr.N != 0 {
				// If there's less than one chunk of data then don't bother with multi-party
				_, err := s3c.PutObject(&s3.PutObjectInput{
					Bucket:               &bucket,
					Key:                  &path,
					Body:                 bytes.NewReader(buf.Bytes()),
					ContentType:          &contentType,
					ContentEncoding:      &contentEncoding,
					ACL:                  &acl,
					Metadata:             meta,
					ServerSideEncryption: aws.String("AES256"),
				})
				return err
			}
			var err error
			multi, err = s3c.CreateMultipartUpload(&s3.CreateMultipartUploadInput{
				Bucket:               &bucket,
				Key:                  &path,
				ContentType:          &contentType,
				ContentEncoding:      &contentEncoding,
				ACL:                  &acl,
				Metadata:             meta,
				ServerSideEncryption: aws.String("AES256"),
			})
			if err != nil {
				return err
			}
		}

		p, err := s3c.UploadPart(&s3.UploadPartInput{
			Bucket:        &bucket,
			Key:           &path,
			Body:          bytes.NewReader(buf.Bytes()),
			ContentLength: aws.Int64(int64(buf.Len())),
			PartNumber:    aws.Int64(int64(nChunk)),
			UploadId:      multi.UploadId,
		})
		if err != nil {
			_, err := s3c.AbortMultipartUpload(&s3.AbortMultipartUploadInput{
				Bucket:   &bucket,
				Key:      &path,
				UploadId: multi.UploadId,
			})
			if err != nil {
				golog.Errorf("Failed to abort multipart S3 upload: %s", err.Error())
			}
			return err
		}
		parts = append(parts, &s3.CompletedPart{ETag: p.ETag, PartNumber: aws.Int64(int64(nChunk))})
	}

	_, err := s3c.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
		Bucket:   &bucket,
		Key:      &path,
		UploadId: multi.UploadId,
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: parts,
		},
	})
	if err != nil {
		_, err := s3c.AbortMultipartUpload(&s3.AbortMultipartUploadInput{
			Bucket:   &bucket,
			Key:      &path,
			UploadId: multi.UploadId,
		})
		if err != nil {
			golog.Errorf("Failed to abort multipart S3 upload: %s", err.Error())
		}
		return err
	}
	return nil
}
