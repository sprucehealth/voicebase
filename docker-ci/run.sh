#!/bin/bash -e -x -v

mv /var/lib/mysql /mem/mysql
ln -s /mem/mysql /var/lib/mysql
/etc/init.d/mysql start

export PATH=/usr/local/go/bin:$PATH
export CF_LOCAL_DB_INSTANCE=127.0.0.1
export CF_LOCAL_DB_PORT=3306
export CF_LOCAL_DB_USERNAME=root
export CF_LOCAL_DB_PASSWORD=
export DOSESPOT_USER_ID=407
export USER=`whoami`

export GOPATH=/workspace/go
cd $GOPATH/src/github.com/sprucehealth/backend

# Find all directories that contain Go files (all packages). This lets us
# exclude everything under the vendoring directory.
PKGS=$(find . -name '*.go' | grep -v Godeps | xargs -n 1 dirname | sort | uniq)
echo $PKGS

echo "BUILDING"
echo $PKGS | xargs go build

echo "FMT"
FMT=$(echo $PKGS | xargs go fmt)
if [[ ! -z "$FMT" ]]; then
	echo $FMT
	exit 1
fi

echo "VET"
echo $PKGS | xargs go vet

echo "BUILDING TESTS"
echo $PKGS | xargs go test -i

echo "TESTING"
echo $PKGS | xargs -n 1 go test -v -test.parallel 8

# Test static resources
resources/build.sh
(cd resources/apps ; flow check)

# Build for deploy
cd apps/restapi
./build.sh
