package ec2

import (
	"testing"

	"carefront/libs/aws"
)

func TestDescribeSnapshots(t *testing.T) {
	// keys := aws.KeysFromEnvironment()
	// if keys.AccessKey == "" || keys.SecretKey == "" {
	// 	t.Skip("Skipping aws.ec2 tests. AWS keys not found in environment.")
	// }
	// cli := &aws.Client{
	// 	Auth: keys,
	// }
	// ec2 := &EC2{
	// 	Region: aws.USEast,
	// 	Client: cli,
	// }
	// snaps, err := ec2.DescribeSnapshots(nil, []string{"self"}, nil, nil)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// for _, s := range snaps {
	// 	t.Logf("%+v", s)
	// }
}

func TestDescribeVolumes(t *testing.T) {
	// keys := aws.KeysFromEnvironment()
	// if keys.AccessKey == "" || keys.SecretKey == "" {
	// 	t.Skip("Skipping aws.ec2 tests. AWS keys not found in environment.")
	// }
	// cli := &aws.Client{
	// 	Auth: keys,
	// }
	// ec2 := &EC2{
	// 	Region: aws.USEast,
	// 	Client: cli,
	// }
	// vols, err := ec2.DescribeVolumes(nil, nil)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// for _, v := range vols {
	// 	t.Logf("%+v %+v", v, v.Attachment)
	// }
}
