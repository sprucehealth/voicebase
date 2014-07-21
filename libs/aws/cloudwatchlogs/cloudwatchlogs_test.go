package cloudwatchlogs

import (
	"testing"

	"github.com/sprucehealth/backend/libs/aws"
)

func TestDescribeLogGroups(t *testing.T) {
	keys := aws.KeysFromEnvironment()
	if keys.AccessKey == "" || keys.SecretKey == "" {
		t.Skip("Skipping aws.kinesis tests. AWS keys not found in environment.")
	}

	cli := &Client{
		Region: aws.USEast,
		Client: &aws.Client{
			Auth: keys,
		},
	}

	// if err := cli.CreateLogGroup("foobar"); err != nil {
	// 	t.Fatal(err)
	// }

	// if err := cli.CreateLogStream("foobar", "abc"); err != nil {
	// 	t.Fatal(err)
	// }

	// if err := cli.DeleteLogStream("foobar", "abc"); err != nil {
	// 	t.Fatal(err)
	// }

	// if err := cli.DeleteLogGroup("foobar"); err != nil {
	// 	t.Fatal(err)
	// }

	groups, err := cli.DescribeLogGroups("", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range groups.LogGroups {
		t.Logf("%+v\n", g)
	}

	streams, err := cli.DescribeLogStreams(groups.LogGroups[0].LogGroupName, "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range streams.LogStreams {
		t.Logf("%+v\n", s)
	}
}
