package util

import (
	"carefront/libs/aws"
	goamz "launchpad.net/goamz/aws"
)

type AWSAdapter struct {
	*aws.Client
}

func (ad *AWSAdapter) Auth() goamz.Auth {
	return goamz.Auth{
		AccessKey: ad.Keys.AccessKey,
		SecretKey: ad.Keys.SecretKey,
	}
}
