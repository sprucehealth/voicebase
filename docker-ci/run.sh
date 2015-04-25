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

if [[ "$JOB_NAME" == "Backend-master" ]]; then
    CMD_NAME="restapi-$GIT_BRANCH-$BUILD_NUMBER"
    rm -rf build # Jenkins preserves the worksapce so remove any old build files
    mkdir build
    cp restapi build/$CMD_NAME
    bzip2 -9 build/$CMD_NAME
    echo $GIT_COMMIT > build/$CMD_NAME.revision
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
