package aws

import "os"

type Auth interface {
	Keys() Keys
}

// Keys holds a set of Amazon Security Credentials.
type Keys struct {
	AccessKey string
	SecretKey string
	Token     string
}

func (k Keys) Keys() Keys {
	return k
}

// Initializes and returns a Keys using the AWS_ACCESS_KEY and AWS_SECRET_KEY
// environment variables.
func KeysFromEnvironment() Keys {
	return Keys{
		AccessKey: os.Getenv("AWS_ACCESS_KEY"),
		SecretKey: os.Getenv("AWS_SECRET_KEY"),
	}
}
