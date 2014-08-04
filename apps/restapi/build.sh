#!/bin/bash -e

APP=restapi
REV="$TRAVIS_COMMIT"
if [ "$REV" = "" ]; then
	REV=$(git rev-parse HEAD)
fi
BRANCH="$TRAVIS_BRANCH"
if [ "$BRANCH" = "" ]; then
	BRANCH=$(git rev-parse --abbrev-ref HEAD)
fi
TIME=$(date)
LATEST_MIGRATION=$(ls -r ../../mysql/snapshot-*.sql | cut -d- -f 2  | cut -d. -f1 | sort -nr | head -1)
GOOS=linux GOARCH=amd64 \
	go build -ldflags " \
		-X github.com/sprucehealth/backend/common/config.GitRevision '$REV' \
		-X github.com/sprucehealth/backend/common/config.GitBranch '$BRANCH' \
		-X github.com/sprucehealth/backend/common/config.BuildTime '$TIME' \
		-X github.com/sprucehealth/backend/common/config.BuildNumber '$TRAVIS_BUILD_NUMBER' \
		-X github.com/sprucehealth/backend/common/config.MigrationNumber '$LATEST_MIGRATION'" -o $APP

# Embed resources in the binary

RESOURCE_ZIP=`pwd`/resources.zip
rm -f $RESOURCE_ZIP
(cd ../../resources ; zip -r $RESOURCE_ZIP templates)
cat $RESOURCE_ZIP >> $APP
zip -A $APP
