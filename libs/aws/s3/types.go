package s3

import "time"

type Owner struct {
	ID          string
	DisplayName string
}

type BucketItem struct {
	Key          string
	LastModified time.Time
	ETag         string
	Size         int64
	StorageClass string
	Owner        Owner
}

type ListBucketsResult struct {
	Name        string
	Prefix      string
	Marker      string
	MaxKeys     int
	IsTruncated bool
	Contents    []*BucketItem
}
