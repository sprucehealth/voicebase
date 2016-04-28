package awsutil

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/sprucehealth/backend/libs/ptr"
)

func AssumedECSCli(stsCli stsiface.STSAPI, roleARN, sessionName string) (ecsiface.ECSAPI, error) {
	creds, err := assumedCreds(stsCli, roleARN, sessionName)
	if err != nil {
		return nil, err
	}
	sess, err := sessionFromCreds(creds)
	if err != nil {
		return nil, err
	}
	return ecs.New(sess), nil
}

func assumedCreds(stsCli stsiface.STSAPI, roleARN, sessionName string) (*sts.Credentials, error) {
	res, err := stsCli.AssumeRole(&sts.AssumeRoleInput{
		RoleArn:         ptr.String(roleARN),
		RoleSessionName: ptr.String(sessionName),
	})
	if err != nil {
		return nil, err
	}
	return res.Credentials, nil
}

func sessionFromCreds(sCreds *sts.Credentials) (*session.Session, error) {
	fmt.Println("AccessID:", *sCreds.AccessKeyId)
	fmt.Println("SecretKey:", *sCreds.SecretAccessKey)
	fmt.Println("SessionToken:", *sCreds.SessionToken)
	// TODO: Hack region for now
	awsConfig, err := Config("us-east-1", *sCreds.AccessKeyId, *sCreds.SecretAccessKey, *sCreds.SessionToken)
	if err != nil {
		return nil, err
	}
	return session.New(awsConfig), nil
}
