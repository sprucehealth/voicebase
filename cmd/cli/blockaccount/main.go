// This script is responsible for publishing an encrypted request to the block account topic
// to block access to a particular account. This is a worker listening on an SQS queue to do the
// actual work of blocking account access.
package main

import (
	"encoding/base64"
	"encoding/csv"
	"flag"
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/svc/operational"
)

var (
	flagKMSKeyARN               = flag.String("kms_key_arn", "", "the arn of the master key that should be used for encrypting data")
	flagAccountIDsCSV           = flag.String("account_ids_csv", "", "file name containing the list of account IDs that map to accounts to block")
	flagBlockAccountSNSTopicARN = flag.String("block_account_sns_topic_arn", "", "arn of the sns topic to which to publish the block account request")
	flagAWSAccessKey            = flag.String("aws_access_key", "", "Access `key` for AWS")
	flagAWSSecretKey            = flag.String("aws_secret_key", "", "Secret `key` for AWS")
	flagAWSToken                = flag.String("aws_token", "", "Temporary access `token` for AWS")
	flagAWSRegion               = flag.String("aws_region", "us-east-1", "AWS `region`")
)

func main() {
	flag.Parse()
	validate()

	awsConfig, err := awsutil.Config(*flagAWSRegion, *flagAWSAccessKey, *flagAWSSecretKey, *flagAWSToken)
	if err != nil {
		golog.Fatalf(err.Error())
	}
	awsSession := session.New(awsConfig)

	eSNS, err := awsutil.NewEncryptedSNS(*flagKMSKeyARN, kms.New(awsSession), sns.New(awsSession))
	if err != nil {
		golog.Fatalf("Unable to initialize enrypted sns: %s", err.Error())
	}

	csvFile, err := os.Open(*flagAccountIDsCSV)
	if err != nil {
		golog.Fatalf(err.Error())
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	reader.Comma = '\n'
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			golog.Fatalf(err.Error())
		}

		if err := publish(eSNS, *flagBlockAccountSNSTopicARN, &operational.BlockAccountRequest{
			AccountID: row[0],
		}); err != nil {
			golog.Fatalf(err.Error())
		} else {
			golog.Infof("Published block account request for %s", row[0])
		}
	}
}

func validate() {
	if *flagKMSKeyARN == "" {
		golog.Fatalf("ARN for KMS Key not specified")
	} else if *flagAccountIDsCSV == "" {
		golog.Fatalf("Filename of file containing emails not specified")
	} else if *flagBlockAccountSNSTopicARN == "" {
		golog.Fatalf("SNS Topic ARN for blocking account not specified")
	} else if *flagAWSAccessKey == "" {
		golog.Fatalf("AWS Access Key not specified")
	} else if *flagAWSSecretKey == "" {
		golog.Fatalf("AWS Secret Key not specified")
	}

}

func publish(snsCLI snsiface.SNSAPI, topic string, bar *operational.BlockAccountRequest) error {
	data, err := bar.Marshal()
	if err != nil {
		return errors.Trace(err)
	}

	_, err = snsCLI.Publish(&sns.PublishInput{
		Message:  ptr.String(base64.StdEncoding.EncodeToString(data)),
		TopicArn: ptr.String(topic),
	})
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}
