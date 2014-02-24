#!/bin/bash -e

APP=restapi
REV=$(git rev-parse HEAD)
BRANCH=$(git rev-parse --abbrev-ref HEAD)
TIME=$(date)
LATEST_MIGRATION=$(ls -r ../../mysql/migration-*.sql | cut -d- -f 2  | cut -d. -f1 | sort -nr | head -1)
GOOS=linux GOARCH=amd64 \
	go build -ldflags " \
	    -X carefront/common/config.GitRevision '$REV' \
		-X carefront/common/config.GitBranch '$BRANCH' \
		-X carefront/common/config.BuildTime '$TIME' \
		-X carefront/common/config.BuildNumber '$TRAVIS_BUILD_NUMBER' \
		-X carefront/common/config.MigrationNumber '$LATEST_MIGRATION'" -o $APP
