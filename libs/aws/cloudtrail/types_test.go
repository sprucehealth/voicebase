package cloudtrail

import (
	"encoding/json"
	"testing"
)

func TestTime(t *testing.T) {
	js := `{
    "Records": [
        {
            "awsRegion": "us-east-1",
            "eventName": "DescribeVolumes",
            "eventSource": "ec2.amazonaws.com",
            "eventTime": "2014-02-03T14:09:35Z",
            "eventVersion": "1.0",
            "requestParameters": {
                "filterSet": {
                    "items": [
                        {
                            "name": "status",
                            "valueSet": {
                                "items": [
                                    {
                                        "value": "available"
                                    },
                                    {
                                        "value": "in-use"
                                    }
                                ]
                            }
                        }
                    ]
                },
                "volumeSet": {}
            },
            "responseElements": null,
            "sourceIPAddress": "1.2.3.4",
            "userAgent": "aws-sdk-ruby/1.9.5 ruby/1.9.3 x86_64-linux",
            "userIdentity": {
                "accessKeyId": "AKxxxxx",
                "accountId": "123123123123132",
                "arn": "arn:aws:iam::123123123123132:user/librato",
                "principalId": "AIxxxxxxx",
                "type": "IAMUser",
                "userName": "librato"
            }
        },
        {
            "awsRegion": "us-east-1",
            "eventName": "DescribeInstances",
            "eventSource": "ec2.amazonaws.com",
            "eventTime": "2014-02-03T14:09:34Z",
            "eventVersion": "1.0",
            "requestParameters": {
                "filterSet": {
                    "items": [
                        {
                            "name": "instance-state-name",
                            "valueSet": {
                                "items": [
                                    {
                                        "value": "running"
                                    }
                                ]
                            }
                        }
                    ]
                },
                "instancesSet": {}
            },
            "responseElements": null,
            "sourceIPAddress": "1.2.3.4",
            "userAgent": "aws-sdk-ruby/1.9.5 ruby/1.9.3 x86_64-linux",
            "userIdentity": {
                "accessKeyId": "AKxxxxxx",
                "accountId": "123123123123",
                "arn": "arn:aws:iam::123123123:user/librato",
                "principalId": "AIxxxxxx",
                "type": "IAMUser",
                "userName": "librato"
            }
        },
        {
            "awsRegion": "us-east-1",
            "eventName": "DescribeDBInstances",
            "eventSource": "rds.amazonaws.com",
            "eventTime": "2014-02-03T14:09:35Z",
            "eventVersion": "1.0",
            "requestParameters": null,
            "responseElements": null,
            "sourceIPAddress": "2.3.4.5",
            "userAgent": "aws-sdk-ruby/1.9.5 ruby/1.9.3 x86_64-linux",
            "userIdentity": {
                "accessKeyId": "AKxxxxxxx",
                "accountId": "123123123123",
                "arn": "arn:aws:iam::123123123123:user/librato",
                "principalId": "AIxxxxxxxx",
                "type": "IAMUser",
                "userName": "librato"
            }
        }
    ]
    }`
	var log Log
	if err := json.Unmarshal([]byte(js), &log); err != nil {
		t.Fatal(err)
	}
	for _, r := range log.Records {
		t.Logf("%+v\n", r)
	}
}
