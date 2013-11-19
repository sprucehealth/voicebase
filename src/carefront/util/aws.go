package util

import (
	"carefront/libs/aws"
	goamz "launchpad.net/goamz/aws"
)

func AWSAuthAdapter(auth aws.Auth) goamz.Auth {
	keys := auth.Keys()
	return goamz.Auth{
		AccessKey: keys.AccessKey,
		SecretKey: keys.SecretKey,
		Token:     keys.Token,
	}
}
