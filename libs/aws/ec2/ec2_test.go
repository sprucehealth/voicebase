package ec2

import (
	"os"
	"testing"

	"github.com/sprucehealth/backend/libs/aws"
)

func testClient(t *testing.T) *EC2 {
	if os.Getenv("TEST_EC2") == "" {
		t.Skip("TEST_EC2 not set")
	}
	keys := aws.KeysFromEnvironment()
	if keys.AccessKey == "" || keys.SecretKey == "" {
		t.Skip("AWS keys not found in environment.")
	}
	cli := &aws.Client{
		Auth: keys,
	}
	ec2 := &EC2{
		Region: aws.USEast,
		Client: cli,
	}
	return ec2
}

func TestDescribeSnapshots(t *testing.T) {
	ec2 := testClient(t)
	snaps, err := ec2.DescribeSnapshots(nil, []string{"self"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range snaps {
		t.Logf("%+v", s)
	}
}

func TestDescribeVolumes(t *testing.T) {
	ec2 := testClient(t)
	vols, err := ec2.DescribeVolumes(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vols {
		t.Logf("%+v %+v", v, v.Attachment)
	}
}

func TestDescribeNetworkACLs(t *testing.T) {
	ec2 := testClient(t)
	acls, err := ec2.DescribeNetworkACLs(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range acls {
		t.Logf("%+v %+v", v, v.Associations)
	}
}

func TestDescribeSecurityGroups(t *testing.T) {
	ec2 := testClient(t)
	groups, err := ec2.DescribeSecurityGroups(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range groups {
		t.Logf("%+v", v)
	}
}

func TestDescribeSubnets(t *testing.T) {
	ec2 := testClient(t)
	subnets, err := ec2.DescribeSubnets(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range subnets {
		t.Logf("%+v", v)
	}
}

func TestDescribeVPCs(t *testing.T) {
	ec2 := testClient(t)
	vpcs, err := ec2.DescribeVPCs(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vpcs {
		t.Logf("%+v", v)
	}
}

func TestDescribeVPCPeeringConnections(t *testing.T) {
	ec2 := testClient(t)
	conns, err := ec2.DescribeVPCPeeringConnections(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range conns {
		t.Logf("%+v", v)
	}
}
