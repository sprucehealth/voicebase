Backend Monorepo
================
[![Build Status](https://magnum.travis-ci.com/SpruceHealth/backend.svg?token=NtmZSFxujHkPCqsPtfXC&branch=master)](https://magnum.travis-ci.com/SpruceHealth/backend)

Running integration tests locally
---------------------------------

Prerequisites:

	$ brew install mysql
	$ mysql.server start

Setup the environment:

	$ export CAREFRONT_PROJECT_DIR=$GOPATH
	$ export RDS_INSTANCE=localhost
	$ export RDS_USERNAME=$USER
	$ export AWS_ACCESS_KEY=<for dev account>
	$ export AWS_SECRET_KEY=<for dev account>

Run tests:

	$ go test -v carefront/test/integration
