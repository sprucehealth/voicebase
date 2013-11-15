package aws

type ACL string

const (
	Private           ACL = "private"
	PublicRead        ACL = "public-read"
	PublicReadWrite   ACL = "public-read-write"
	AuthenticatedRead ACL = "authenticated-read"
	BucketOwnerRead   ACL = "bucket-owner-read"
	BucketOwnerFull   ACL = "bucket-owner-full-control"
)

type StorageClass string

const (
	StandardStorage          StorageClass = "STANDARD"
	ReducedRedundancyStorage StorageClass = "REDUCED_REDUNDANCY"
)

const (
	// Request Headers

	HeaderMetaPrefix   = "x-amz-meta-"
	HeaderMetaMFA      = "x-amz-mfa"
	HeaderStorageClass = "x-amz-storage-class"
	HeaderACL          = "x-amz-acl"
	// grants: type=value pair with type one of emailAddress, id (user ID), or uri (group)
	HeaderGrantRead        = "x-amz-grant-read"
	HeaderGrantWrite       = "x-amz-grant-write"
	HeaderGrantReadACP     = "x-amz-grant-read-acp"
	HeaderGrantWriteACP    = "x-amz-grant-write-acp"
	HeaderGrantFullControl = "x-amz-grant-full-control"

	// Response Headers

	HeaderDeleteMarker = "x-amz-delete-marker"
	HeaderExpiration   = "x-amz-expiration"
	HeaderRestore      = "x-amz-restore"
	HeaderVersionId    = "x-amz-version-id"

	// Request & Response Headers

	HeaderServerSideEncryption    = "x-amz-server-side-encryption"
	HeaderWebsiteRedirectLocation = "x-amz-website-redirect-location"
)
