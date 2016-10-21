package config

import (
	"crypto/tls"
	"fmt"

	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/deploy"
)

type Config struct {
	App        *boot.App
	DeployAddr string
	TLS        bool
	CACertPath string
}

func (c *Config) DeployClient() (deploy.DeployClient, error) {
	tlsConfig, err := c.TLSConfig()
	if err != nil {
		return nil, errors.Trace(err)
	}
	conn, err := boot.DialGRPC("deploy", c.DeployAddr, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to deploy service: %s", err)
	}
	return deploy.NewDeployClient(conn), nil
}

func (c *Config) STSClient() (stsiface.STSAPI, error) {
	awsSession, err := c.App.AWSSession()
	if err != nil {
		return nil, fmt.Errorf("Unable to create AWSSession: %s", err)
	}
	return sts.New(awsSession), nil
}

func (c *Config) TLSConfig() (*tls.Config, error) {
	if !c.TLS {
		return nil, nil
	}
	tlsConfig := &tls.Config{}
	if c.CACertPath != "" {
		ca, err := boot.CAFromFile(c.CACertPath)
		if err != nil {
			return nil, errors.Errorf("Failed to load CA from %s: %s", c.CACertPath, err)
		}
		tlsConfig.RootCAs = ca
	}
	return tlsConfig, nil
}
