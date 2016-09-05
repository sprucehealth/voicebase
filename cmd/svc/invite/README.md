Invite Service
==============
The invite service is responsible for managing the patient and provider invites. This includes:
- Managing patient invites when a provider wants to invite a patient to a secure conversation.
- Managing practice links on an org specific basis
- Managing provider invites for providers to join existing team memnbers on spruce.

Setup for local development
---------------------------

Start a local dynamodb

	$ docker run -p 7777:7777 tray/dynamodb-local -inMemory -port 7777 -delayTransientStatuses

Setup tables using terraform

	$ mkdir -p schema/env-local
	$ cd schema/env-local
	# replace the IP address below with an appropriate one"
	$ cat > aws.tf <<EOF
	provider "aws" {
	    region = "us-east-1"
	    dynamodb_endpoint = "http://localhost:7777"
	}
	EOF
	ln -s ../dynamodb.tf
	terraform apply

Setup up Terraform
------------------

	$ cd schema
	$ mkdir env-dev env-staging env-prod
	$ cd env-dev
	$ terraform remote config -backend=s3 -backend-config="bucket=spruce-terraform" -backend-config="encrypt=true" -backend-config="key=invite-dev.tfstate" -backend-config="region=us-east-1"
	$ cd ../env-staging
	$ terraform remote config -backend=s3 -backend-config="bucket=spruce-terraform" -backend-config="encrypt=true" -backend-config="key=invite-staging.tfstate" -backend-config="region=us-east-1"
	$ cd ../env-prod
	$ terraform remote config -backend=s3 -backend-config="bucket=spruce-infra" -backend-config="encrypt=true" -backend-config="key=invite-prod.tfstate" -backend-config="region=us-east-1"