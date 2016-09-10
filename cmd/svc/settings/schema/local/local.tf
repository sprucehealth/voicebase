variable "env" { default = "local" }

module "dynamodb" {
	source = "../"
	env = "${var.env}"
}

provider "aws" {
    region = "us-east-1"
    dynamodb_endpoint = "http://localhost:7777"
}
