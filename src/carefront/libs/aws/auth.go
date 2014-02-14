package aws

import "os"

// Auth is the interface that provides the Keys() method to return AWS keys.
type Auth interface {
	Keys() Keys
}

// Keys holds a set of Amazon Security Credentials.
type Keys struct {
	AccessKey string
	SecretKey string
	Token     string // Security token when using temporary credentials
}

// Keys returns itself to satisfy the Auth interface.
func (k Keys) Keys() Keys {
	return k
}

// KeysFromEnvironment looks up AWS_ACCESS_KEY, AWS_SECRET_KEY, and AWS_SECURITY_TOKEN from the environment.
func KeysFromEnvironment() Keys {
	return Keys{
		AccessKey: os.Getenv("AWS_ACCESS_KEY"),
		SecretKey: os.Getenv("AWS_SECRET_KEY"),
		Token:     os.Getenv("AWS_SECURITY_TOKEN"),
	}
}
