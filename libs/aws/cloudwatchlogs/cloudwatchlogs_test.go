package cloudwatchlogs

import (
	"testing"
	"time"

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
		t.Logf("%+v", g)
	}

	streams, err := cli.DescribeLogStreams(groups.LogGroups[0].LogGroupName, "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range streams.LogStreams {
		t.Logf("%+v", s)
	}
	events, err := cli.GetLogEvents(
		groups.LogGroups[0].LogGroupName,
		streams.LogStreams[0].LogStreamName,
		false, time.Time{}, time.Time{}, "", 5)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range events.Events {
		t.Logf("%+v", e)
	}
}
