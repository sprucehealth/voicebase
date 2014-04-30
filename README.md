Backend Monorepo
================
[![Build Status](https://magnum.travis-ci.com/SpruceHealth/backend.svg?token=NtmZSFxujHkPCqsPtfXC&branch=master)](https://magnum.travis-ci.com/SpruceHealth/backend)

Running integration tests locally
---------------------------------

Prerequisites:

	# Need the mysql client
	brew install mysql

Setup the environment:

	export CAREFRONT_PROJECT_DIR=$GOPATH
	export RDS_INSTANCE=dev-db-3.ccvrwjdx3gvp.us-east-1.rds.amazonaws.com
	export RDS_USERNAME=carefront
	export RDS_PASSWORD=<password>
	export AWS_ACCESS_KEY=<for dev account>
	export AWS_SECRET_KEY=<for dev account>

Run tests:

	go test -v carefront/test/integration
