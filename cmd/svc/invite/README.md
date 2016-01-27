Invite Service
==============

Setup for local development
---------------------------

Start a local dynamodb

	$ docker run -p 7777:7777 tray/dynamodb-local -inMemory -port 7777 -delayTransientStatuses

Setup tables using terraform

	$ mkdir schema/env-local
	$ cd schema/env-local
	# replace the IP address below with an appropriate one"
	$ cat > aws.tf <<EOF
	provider "aws" {
	    region = "us-east-1"
	    dynamodb_endpoint = "http://192.168.99.100:7777"
	}
	EOF
	ln -s ../dynamodb.tf
	terraform apply
