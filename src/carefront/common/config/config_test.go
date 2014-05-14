package config

import (
	"strings"
	"testing"
	"time"
)

type TestConfig struct {
	*BaseConfig
	TestArg int `long:"test_arg" description:"testing"`
}

func TestBasic(t *testing.T) {
	args := []string{
		"--test_arg", "1234",
		"--aws_region", "test-region",
		"--app_name", "test-name",
		"--env", "test",
	}
	config := &TestConfig{}
	args2, err := ParseArgs(config, args)
	if err != nil {
		t.Fatal(err)
	}
	_ = args2
	t.Logf("%+v\n", config)
	if config.TestArg != 1234 {
		t.Fatal("Failed to set TestARg")
	}
	if config.BaseConfig.AWSRegion != "test-region" {
		t.Fatal("Failed to set BaseConfig.AWSRegion")
	}
}

func TestSMTPTimeout(t *testing.T) {
	smtpConnectTimeout = time.Millisecond * 200
	config := &BaseConfig{
		SMTPAddr:   "127.0.0.123:25",
		AlertEmail: "noone@nowhere.com'",
	}
	_, err := config.SMTPConnection()
	if err == nil {
		t.Fatal("Expected a timeout error")
	} else if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("Expected timeout. Got '%s'", err.Error())
	}
}
