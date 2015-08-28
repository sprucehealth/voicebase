#!/bin/bash -e

APP=restapi
REV="$GIT_COMMIT"
if [ "$REV" = "" ]; then
	REV="$TRAVIS_COMMIT"
fi
if [ "$REV" = "" ]; then
	REV=$(git rev-parse HEAD)
fi
BRANCH="$GIT_BRANCH"
if [ "$BRANCH" = "" ]; then
	BRANCH="$TRAVIS_BRANCH"
fi
if [ "$BRANCH" = "" ]; then
	BRANCH=$(git rev-parse --abbrev-ref HEAD)
fi
# Jenkins sets BUILD_NUMBER
if [ "$BUILD_NUMBER" = "" ]; then
	BUILD_NUMBER="$TRAVIS_BUILD_NUMBER"
fi
TIME=$(date)
LATEST_MIGRATION=$(ls -r $GOPATH/src/github.com/sprucehealth/backend/mysql/snapshot-*.sql | cut -d- -f 2  | cut -d. -f1 | sort -nr | head -1)
GO15VENDOREXPERIMENT=1 GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
	go build -i -a -tags netgo -ldflags " \
		-X 'github.com/sprucehealth/backend/common/config.GitRevision=$REV' \
		-X 'github.com/sprucehealth/backend/common/config.GitBranch=$BRANCH' \
		-X 'github.com/sprucehealth/backend/common/config.BuildTime=$TIME' \
		-X 'github.com/sprucehealth/backend/common/config.BuildNumber=$BUILD_NUMBER' \
		-X 'github.com/sprucehealth/backend/common/config.MigrationNumber=$LATEST_MIGRATION'" -o $APP

# Embed resources in the binary

RESOURCE_ZIP=`pwd`/resources.zip
rm -f $RESOURCE_ZIP
(cd $GOPATH/src/github.com/sprucehealth/backend/resources ; zip -r $RESOURCE_ZIP templates static/img/logo-small.png)
cat $RESOURCE_ZIP >> $APP
zip -A $APP
