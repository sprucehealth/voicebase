#!/usr/bin/env bash
GOPATH=/Users/kunaljham/Dropbox/personal/workspace/backend/medellin \
CAREFRONT_BUCKET=carefront-cases \
DB_USER=ejabberd \
DB_PASSWORD=ejabberd \
DB_HOST=ejabberd-db-dev.c83wlsbcftxz.us-west-1.rds.amazonaws.com \
DB_NAME=carefront_db \
go build
