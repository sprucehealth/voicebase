#!/bin/bash -e

APP=curbside
REV="$GIT_COMMIT"
if [ "$REV" = "" ]; then
	REV=$(git rev-parse HEAD)
fi
BRANCH="$GIT_BRANCH"
if [ "$BRANCH" = "" ]; then
	BRANCH=$(git rev-parse --abbrev-ref HEAD)
fi
TIME=$(date)
LATEST_MIGRATION=$(ls -r $GOPATH/src/github.com/sprucehealth/backend/mysql/snapshot-*.sql | cut -d- -f 2  | cut -d. -f1 | sort -nr | head -1)
GO15VENDOREXPERIMENT=1 GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
	go build -i -a -tags netgo -ldflags " \
		-X 'github.com/sprucehealth/backend/boot.GitRevision=$REV' \
		-X 'github.com/sprucehealth/backend/boot.GitBranch=$BRANCH' \
		-X 'github.com/sprucehealth/backend/boot.BuildTime=$TIME' \
		-X 'github.com/sprucehealth/backend/boot.BuildNumber=$BUILD_NUMBER' \
		-X 'github.com/sprucehealth/backend/boot.MigrationNumber=$LATEST_MIGRATION'" -o $APP

# Embed resources in the binary

RESOURCE_ZIP=`pwd`/resources.zip
rm -f $RESOURCE_ZIP
(cd $GOPATH/src/github.com/sprucehealth/backend/cmd/svc/curbside/src ; zip -r $RESOURCE_ZIP templates)
cat $RESOURCE_ZIP >> $APP
zip -A $APP
