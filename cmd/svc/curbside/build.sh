#!/bin/bash -e

APP=curbside
if [ "$REV" = "" ]; then
	REV="$GIT_COMMIT"
fi
if [ "$REV" = "" ]; then
	REV=$(git rev-parse HEAD)
fi
if [ "$BRANCH" = "" ]; then
	BRANCH="$GIT_BRANCH"
fi
if [ "$BRANCH" = "" ]; then
	BRANCH=$(git rev-parse --abbrev-ref HEAD)
fi
if [ "$TIME" = "" ]; then
	TIME=$(date)
fi
GO15VENDOREXPERIMENT=1 GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
	go install -tags netgo -ldflags " \
		-X 'github.com/sprucehealth/backend/boot.GitRevision=$REV' \
		-X 'github.com/sprucehealth/backend/boot.GitBranch=$BRANCH' \
		-X 'github.com/sprucehealth/backend/boot.BuildTime=$TIME' \
		-X 'github.com/sprucehealth/backend/boot.BuildNumber=$BUILD_NUMBER'"

# Embed resources in the binary

RESOURCE_ZIP=`pwd`/resources.zip
rm -f $RESOURCE_ZIP
(cd $GOPATH/src/github.com/sprucehealth/backend/cmd/svc/curbside/src ; zip -r $RESOURCE_ZIP templates)
BINPATH=$GOPATH/bin/$APP
if [[ "$(go env GOHOSTOS)" != "linux" ]]; then
	BINPATH=$GOPATH/bin/linux_amd64/$APP
fi
cat $RESOURCE_ZIP >> $BINPATH
zip -A $BINPATH
chmod +x $BINPATH
