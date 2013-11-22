package config

import "testing"

type TestConfig struct {
	BaseConfig
	TestArg int `long:"test_arg" description:"testing"`
}

func TestBasic(t *testing.T) {
	args := []string{
		"--test_arg", "1234",
		"--aws_region", "test-region",
	}
	config := &TestConfig{}
	args2, err := ParseFlagsAndConfig(config, args)
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
