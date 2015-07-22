#!/bin/bash -e

# Start MySQL
mv /var/lib/mysql /mem/mysql
ln -s /mem/mysql /var/lib/mysql
/etc/init.d/mysql start

# Start Consul
mkdir -p /tmp/consul
tmux new -d -s consul 'GOMAXPROCS=2 /usr/local/bin/consul agent -data-dir /tmp/consul -bootstrap -server'

export PATH=/usr/local/go/bin:$PATH
export CF_LOCAL_DB_INSTANCE=127.0.0.1
export CF_LOCAL_DB_PORT=3306
export CF_LOCAL_DB_USERNAME=root
export CF_LOCAL_DB_PASSWORD=
export DOSESPOT_USER_ID=407
export USER=`whoami`
export TEST_CONSUL_ADDRESS=127.0.0.1:8500

export GOPATH=/workspace/go
export PATH=$GOPATH/bin:$PATH
cd $GOPATH/src/github.com/sprucehealth/backend

# TODO(samuel): remove this after Go 1.5 which will do it automatically based on CPUs.
export GOPMAXPROCS=4

go get github.com/golang/lint/golint

# Find all directories that contain Go files (all packages). This lets us
# exclude everything under the vendoring directory.
PKGS=$(find . -name '*.go' | grep -v Godeps | xargs -n 1 dirname | sort | uniq)
echo $PKGS

echo "BUILDING"
echo $PKGS | xargs go build -i

echo "FMT"
FMT=$(echo $PKGS | xargs go fmt)
if [[ ! -z "$FMT" ]]; then
    echo $FMT
    exit 1
fi

echo "VET"
echo $PKGS | xargs go vet

echo "LINT"
echo $PKGS | xargs -n 1 golint | grep -v "_test.go"

echo "BUILDING TESTS"
echo $PKGS | xargs go test -i

PKGSLIST=""
for P in $PKGS; do
    if [[ ! "$P" == *"/apps/"* ]] && [[ ! "$P" == *"/test/"* ]]; then
        P="github.com/sprucehealth/backend$(echo $P | cut -c2-)"
        PKGSLIST+=",$P"
    fi
done
PKGSLIST=$(echo $PKGSLIST | cut -c2-)

echo "TESTING"
if [[ ! -z "$FULLCOVERAGE" ]]; then
    for PKG in $PKGS; do
        # For integration tests tell it to check coverage in all packages,
        # but for other packages just check coverage against themselves.
        if [[ "$PKG" == *"/test/"* ]]; then
            go test -cover -covermode=set -coverprofile="$PKG/cover.out" -coverpkg=$PKGSLIST -test.parallel 4 "$PKG" 2>&1 | grep -v "warning: no packages being tested depend on"
        else
            go test -cover -covermode=set -coverprofile="$PKG/cover.out" -test.parallel 4 "$PKG"
        fi
    done
else
    for PKG in $PKGS; do
        if [[ "$PKG" == *"/test/"* ]]; then
            go test -test.parallel 4 "$PKG"
        else
            go test -cover -covermode=set -coverprofile="$PKG/cover.out" -test.parallel 4 "$PKG"
        fi
    done
fi

go run docker-ci/covermerge.go ./coverage-$BUILD_NUMBER.out ./
go tool cover -html=coverage-$BUILD_NUMBER.out -o coverage.html
cp coverage.html coverage-$BUILD_NUMBER.html
go tool cover -func=coverage-$BUILD_NUMBER.out | grep "total:"

# Test static resources
echo "TESTING STATIC RESOURCES"
flow --version
resources/build.sh
(cd resources/apps ; time flow check)

# Build for deploy
echo "BUILDING"
cd apps/restapi
./build.sh

if [[ "$DEPLOY_TO_S3" != "" ]]; then
    echo "DEPLOYING"

    CMD_NAME="restapi-$GIT_BRANCH-$BUILD_NUMBER"
    rm -rf build # Jenkins preserves the worksapce so remove any old build files
    mkdir build
    cp restapi build/$CMD_NAME
    bzip2 -9 build/$CMD_NAME
    echo $GIT_COMMIT > build/$CMD_NAME.revision
    cp ../../coverage-$BUILD_NUMBER.out build/$CMD_NAME.coverage
    cp ../../coverage-$BUILD_NUMBER.html build/$CMD_NAME.coverage.html
    s3cmd --add-header "x-amz-acl:bucket-owner-full-control" -M --server-side-encryption put build/* s3://spruce-deploy/restapi/

    cd ../../resources/static
    STATIC_PREFIX="s3://spruce-static/web/$BUILD_NUMBER"
    s3cmd --recursive -P --no-preserve -m "text/css" put css/* $STATIC_PREFIX/css/
    s3cmd --recursive -P --no-preserve -m "application/javascript" put js/* $STATIC_PREFIX/js/
    # s3cmd --recursive -P --no-preserve -m "application/x-font-opentype" --add-header "Access-Control-Allow-Origin:*" put fonts/* $STATIC_PREFIX/fonts/
    s3cmd --recursive -P --no-preserve -m "application/octet-stream" --add-header "Access-Control-Allow-Origin:*" put fonts/*.ttf $STATIC_PREFIX/fonts/
    s3cmd --recursive -P --no-preserve -m "application/vnd.ms-fontobject" --add-header "Access-Control-Allow-Origin:*" put fonts/*.eot $STATIC_PREFIX/fonts/
    s3cmd --recursive -P --no-preserve -m "application/font-woff" --add-header "Access-Control-Allow-Origin:*" put fonts/*.woff $STATIC_PREFIX/fonts/
    s3cmd --recursive -P --no-preserve -m "application/font-woff2" --add-header "Access-Control-Allow-Origin:*" put fonts/*.woff2 $STATIC_PREFIX/fonts/
    s3cmd --recursive -P --no-preserve -m "image/svg+xml" --add-header "Access-Control-Allow-Origin:*" put fonts/*.svg $STATIC_PREFIX/fonts/
    s3cmd --recursive -P --no-preserve -M put img/* $STATIC_PREFIX/img/
fi
