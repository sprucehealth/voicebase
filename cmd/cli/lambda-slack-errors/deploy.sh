#!/bin/sh

# This script builds the command and bundles everything into a zip. The file
# config.env should be a JSON encoded config with environment provided to
# the command. A variable SLACK_WEBHOOKURL is expected to exist (see main.go
# for other possible options aka flags).

set -e

if [ ! -e "config.env" ]; then
	echo "config.env does not exist but is required"
	exit 1
fi

GO15VENDOREXPERIMENT=1 GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -v -i
zip lambda-slack-errors.zip main.py lambda-slack-errors config.env
s3cmd put lambda-slack-errors.zip s3://spruce-lambda-functions/lambda-slack-errors.zip
