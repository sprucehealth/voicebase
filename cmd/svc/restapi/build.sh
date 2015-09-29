#!/bin/bash -e

APP=restapi
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
	go install -a -tags netgo -ldflags " \
		-X 'github.com/sprucehealth/backend/boot.GitRevision=$REV' \
		-X 'github.com/sprucehealth/backend/boot.GitBranch=$BRANCH' \
		-X 'github.com/sprucehealth/backend/boot.BuildTime=$TIME' \
		-X 'github.com/sprucehealth/backend/boot.BuildNumber=$BUILD_NUMBER' \
		-X 'github.com/sprucehealth/backend/boot.MigrationNumber=$LATEST_MIGRATION'"

# Embed resources in the binary

RESOURCE_ZIP=`pwd`/resources.zip
rm -f $RESOURCE_ZIP
(cd $GOPATH/src/github.com/sprucehealth/backend/resources ; zip -r $RESOURCE_ZIP templates static/img/logo-small.png)
BINPATH=$GOPATH/bin/$APP
if [[ "$(go env GOHOSTOS)" != "linux" ]]; then
	BINPATH=$GOPATH/bin/linux_amd64/$APP
fi
cat $RESOURCE_ZIP >> $BINPATH
zip -A $BINPATH
chmod +x $BINPATH
