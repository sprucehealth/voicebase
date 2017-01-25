package boot

import (
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/errors"
)

func getAWSAccountID(awsSession *session.Session) (string, error) {
	if environment.IsLocal() {
		accountID, err := getAccountIDFromSTS(awsSession)
		if err != nil {
			return "", errors.Trace(err)
		}
		return accountID, nil
	}
	accountID, err := getAccountIDFromEC2Metadata(awsSession)
	if err != nil {
		return "", errors.Trace(err)
	}
	return accountID, nil
}

func getAccountIDFromEC2Metadata(awsSession *session.Session) (string, error) {
	metadataClient := ec2metadata.New(awsSession)
	doc, err := metadataClient.GetInstanceIdentityDocument()
	if err != nil {
		return "", errors.Errorf("Unable to get instance identity document from ec2 metadata: %s", err)
	}

	return doc.AccountID, nil
}

func getAccountIDFromSTS(awsSession *session.Session) (string, error) {
	stsAPI := sts.New(awsSession)
	resp, err := stsAPI.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", errors.Errorf("Unable to get caller identity document from STS: %s", err)
	}
	return *resp.Account, nil
}
