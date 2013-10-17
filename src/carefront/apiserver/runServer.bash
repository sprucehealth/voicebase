#!/usr/bin/env bash
CERT_KEY=cert.pem \
PRIVATE_KEY=key.pem \
CASE_BUCKET=carefront-cases \
GOPATH=/Users/kunaljham/Dropbox/personal/workspace/backend/medellin \
AWS_SECRET_ACCESS_KEY="eGc+Y28/t7q8LcgnYkSZyi5H8D4tzpSeFSt/158S" \
AWS_ACCESS_KEY_ID=AKIAJVJQW6IJT7ZOQMWA \
DB_USER=ejabberd \
DB_PASSWORD=ejabberd \
DB_HOST=ejabberd-db-dev.c83wlsbcftxz.us-west-1.rds.amazonaws.com \
DB_NAME=carefront_db \
./apiserver
