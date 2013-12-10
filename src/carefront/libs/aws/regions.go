package aws

// http://docs.aws.amazon.com/general/latest/gr/rande.html#rds_region
type Region struct {
	Name                 string
	EC2Endpoint          string
	S3Endpoint           string
	S3LocationConstraint bool
	S3LowercaseBucket    bool
	SDBEndpoint          string
	SNSEndpoint          string
	SQSEndpoint          string
	IAMEndpoint          string
	RDSEndpoint          string
	KinesisEndpoint      string
}

var USEast = Region{
	"us-east-1",
	"https://ec2.us-east-1.amazonaws.com",
	"https://s3.amazonaws.com",
	false,
	false,
	"https://sdb.amazonaws.com",
	"https://sns.us-east-1.amazonaws.com",
	"https://sqs.us-east-1.amazonaws.com",
	"https://iam.amazonaws.com",
	"https://rds.us-east-1.amazonaws.com",
	"https://kinesis.us-east-1.amazonaws.com",
}

var Regions = map[string]Region{
	// APNortheast.Name:  APNortheast,
	// APSoutheast.Name:  APSoutheast,
	// APSoutheast2.Name: APSoutheast2,
	// EUWest.Name:       EUWest,
	USEast.Name: USEast,
	// USWest.Name:       USWest,
	// USWest2.Name:      USWest2,
	// SAEast.Name:       SAEast,
}
