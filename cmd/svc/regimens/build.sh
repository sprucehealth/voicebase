#!/bin/bash -e

APP=regimens
REV="$GIT_COMMIT"
if [ "$REV" = "" ]; then
	REV=$(git rev-parse HEAD)
fi
BRANCH="$GIT_BRANCH"
if [ "$BRANCH" = "" ]; then
	BRANCH=$(git rev-parse --abbrev-ref HEAD)
fi
TIME=$(date)
GO15VENDOREXPERIMENT=1 GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
	go install -a -tags netgo -ldflags " \
		-X 'github.com/sprucehealth/backend/boot.GitRevision=$REV' \
		-X 'github.com/sprucehealth/backend/boot.GitBranch=$BRANCH' \
		-X 'github.com/sprucehealth/backend/boot.BuildTime=$TIME' \
		-X 'github.com/sprucehealth/backend/boot.BuildNumber=$BUILD_NUMBER'"
