#!/bin/bash -e

set -e -o pipefail

BGPID=""

# checkedwait waits for background jobs to complete with non-zero code if any fail
checkedwait() {
    FAIL=0
    echo "Waiting on $BGPID"
    for job in $BGPID; do
        wait $job || let "FAIL+=1"
    done
    if [ "$FAIL" != "0" ]; then
        echo "FAIL"
        exit 1
    fi
    BGPID=""
}

# savepid saves the last started bg job's PID for later use by checkedwait
savepid() {
    BGPID="$BGPID $!"
}

export HOME=/workspace

PHABRICATOR_COMMENT=".phabricator-comment"

# Start Consul
mkdir -p /tmp/consul
tmux new -d -s consul 'GOMAXPROCS=2 /usr/local/bin/consul agent -data-dir /tmp/consul -bootstrap -server'

# Start Memcached (used by rate limiter tests)
memcached -m 4 -d -u nobody

export PATH=/usr/local/go/bin:$PATH
export CF_LOCAL_DB_INSTANCE=127.0.0.1
export CF_LOCAL_DB_PORT=3306
export CF_LOCAL_DB_USERNAME=root
export CF_LOCAL_DB_PASSWORD=
export DOSESPOT_USER_ID=407
export USER=`whoami`
export TEST_CONSUL_ADDRESS=127.0.0.1:8500
export TEST_MEMCACHED=127.0.0.1:11211

export GO15VENDOREXPERIMENT=1
export GOPATH=/workspace/go
export PATH=$GOPATH/bin:$PATH
export MONOREPO_PATH=$GOPATH/src/github.com/sprucehealth/backend
cd $MONOREPO_PATH

go version
go get github.com/golang/lint/golint

# Find all directories that contain Go files (all packages). This lets us
# exclude everything under the vendoring directory.
PKGS=$(find . -name '*.go' | grep -v vendor/ | xargs -n 1 dirname | sort | uniq)
# TODO: PKGS=$(go list ./... | grep -v /vendor/) -- this requires some updates below however but should be more reliable way to get package list
echo $PKGS

echo "BUILDING"
echo $PKGS | xargs go build -i

echo "FMT"
FMT=$(echo $PKGS | xargs go fmt)
if [[ ! -z "$FMT" ]]; then
    echo $FMT | tee -a $PHABRICATOR_COMMENT
    exit 1
fi

echo "VET"
echo $PKGS | xargs go vet | tee -a $PHABRICATOR_COMMENT

echo "LINT"
echo $PKGS | xargs -n 1 golint | grep -v "_test.go" | grep -v ".pb.go"

echo "BUILDING TESTS"
echo $PKGS | xargs go test -i

PKGSLIST=""
for P in $PKGS; do
    if [[ ! "$P" == *"/cmd/"* ]] && [[ ! "$P" == *"/test/"* ]]; then
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
go tool cover -func=coverage-$BUILD_NUMBER.out | grep "total:" | tee -a $PHABRICATOR_COMMENT

flow version | tee -a $PHABRICATOR_COMMENT
npm version | tee -a $PHABRICATOR_COMMENT

# Disable some steps for dev builds (that aren't related to testing)
export BUILDENV=dev
if [[ "$DEPLOY_TO_S3" != "" ]]; then
    export BUILDENV=prod
fi

# Test static resources (restapi)
echo "TESTING STATIC RESOURCES (restapi)"
time (
    cd $MONOREPO_PATH/resources
    ./build.sh
    cd apps
    flow check
) &
savepid

# Test static resources (curbside)
echo "TESTING STATIC RESOURCES (curbside)"
time (
    cd $MONOREPO_PATH/cmd/svc/curbside
    ./build_resources.sh
    flow check
) &
savepid

# Test static resources (carefinder)
echo "TESTING STATIC RESOURCES (carefinder)"
time (
    cd $MONOREPO_PATH/cmd/svc/carefinder
    rm -rf resources/static/js
    mkdir resources/static/js
    rm -rf resources/static/css
    mkdir resources/static/css
    ./build_resources.sh
    flow check
) &
savepid

checkedwait

# Clean binaries before building to make sure we get a clean build for deployment
rm -rf $GOPATH/pkg $GOPATH/bin

# Build services for deploy
cd $MONOREPO_PATH
REV="$GIT_COMMIT"
if [ "$REV" = "" ]; then
    REV=$(git rev-parse HEAD)
fi
BRANCH="$GIT_BRANCH"
if [ "$BRANCH" = "" ]; then
    BRANCH=$(git rev-parse --abbrev-ref HEAD)
fi
TIME=$(date)
export TAG="$BRANCH-$BUILD_NUMBER"

SVCS="auth baymaxgraphql directory excomms notification regimensapi routing threading settings"
for SVC in $SVCS; do
    echo "BUILDING ($SVC)"
    cd $MONOREPO_PATH/cmd/svc/$SVC
    GO15VENDOREXPERIMENT=1 GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
        go install -tags netgo -ldflags " \
            -X 'github.com/sprucehealth/backend/boot.GitRevision=$REV' \
            -X 'github.com/sprucehealth/backend/boot.GitBranch=$BRANCH' \
            -X 'github.com/sprucehealth/backend/boot.BuildTime=$TIME' \
            -X 'github.com/sprucehealth/backend/boot.BuildNumber=$BUILD_NUMBER'"

    BINPATH=$GOPATH/bin/$SVC
    if [[ "$(go env GOHOSTOS)" != "linux" ]]; then
        BINPATH=$GOPATH/bin/linux_amd64/$SVC
    fi
    rm -rf build
    mkdir build
    cp $BINPATH build/
    if [[ -e resources ]]; then
        cp -r resources build/
    fi
    cp /etc/ssl/certs/ca-certificates.crt build/
    cat > build/Dockerfile <<EOF
FROM scratch

LABEL version=$TAG
LABEL svc=$SVC

WORKDIR /workspace
ADD . /workspace
COPY ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
USER 65534
CMD ["/workspace/$SVC"]
EOF
    docker build --rm=true -t $SVC:$TAG -f build/Dockerfile build
done

# Build for deploy (restapi)
echo "BUILDING (restapi)"
time (
    cd $MONOREPO_PATH/cmd/svc/restapi
    ./build.sh
) &
savepid

checkedwait

if [[ "$DEPLOY_TO_S3" != "" ]]; then
    echo "DEPLOYING (restapi)"

    CMD_NAME="restapi-$GIT_BRANCH-$BUILD_NUMBER"
    rm -rf build # Jenkins preserves the workspace so remove any old build files
    mkdir build
    cp $GOPATH/bin/restapi build/$CMD_NAME
    bzip2 -9 build/$CMD_NAME
    echo $GIT_COMMIT > build/$CMD_NAME.revision
    cp $MONOREPO_PATH/coverage-$BUILD_NUMBER.out build/$CMD_NAME.coverage
    cp $MONOREPO_PATH/coverage-$BUILD_NUMBER.html build/$CMD_NAME.coverage.html
    s3cmd --add-header "x-amz-acl:bucket-owner-full-control" -M --server-side-encryption put build/* s3://spruce-deploy/restapi/

    cd $MONOREPO_PATH/resources/static
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

# Build for deploy (curbside)
echo "BUILDING (curbside)"
cd $MONOREPO_PATH/cmd/svc/curbside
./build.sh

if [[ "$DEPLOY_TO_S3" != "" ]]; then
    echo "DEPLOYING (curbside)"

    CMD_NAME="curbside-$GIT_BRANCH-$BUILD_NUMBER"
    rm -rf bin # Jenkins preserves the workspace so remove any old build files
    mkdir bin
    cp $GOPATH/bin/curbside bin/$CMD_NAME
    bzip2 -9 bin/$CMD_NAME
    echo $GIT_COMMIT > bin/$CMD_NAME.revision
    cp $MONOREPO_PATH/coverage-$BUILD_NUMBER.out bin/$CMD_NAME.coverage
    cp $MONOREPO_PATH/coverage-$BUILD_NUMBER.html bin/$CMD_NAME.coverage.html
    s3cmd --add-header "x-amz-acl:bucket-owner-full-control" -M --server-side-encryption put bin/* s3://spruce-deploy/curbside/

    cd $MONOREPO_PATH/cmd/svc/curbside/build
    STATIC_PREFIX="s3://spruce-static/curbside/$BUILD_NUMBER"
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

# Build for deploy (regimensapi)
echo "BUILDING (regimensapi)"
cd $MONOREPO_PATH/cmd/svc/regimensapi
./build.sh

if [[ "$DEPLOY_TO_S3" != "" ]]; then
    echo "DEPLOYING (regimensapi)"

    CMD_NAME="regimensapi-$GIT_BRANCH-$BUILD_NUMBER"
    rm -rf build # Jenkins preserves the workspace so remove any old build files
    mkdir build
    cp $GOPATH/bin/regimensapi build/$CMD_NAME
    bzip2 -9 build/$CMD_NAME
    echo $GIT_COMMIT > build/$CMD_NAME.revision
    cp $MONOREPO_PATH/coverage-$BUILD_NUMBER.out build/$CMD_NAME.coverage
    cp $MONOREPO_PATH/coverage-$BUILD_NUMBER.html build/$CMD_NAME.coverage.html
    s3cmd --add-header "x-amz-acl:bucket-owner-full-control" -M --server-side-encryption put build/* s3://spruce-deploy/regimensapi/
fi

# Build for deploy (carefinder)
echo "BUILDING (carefinder)"
cd $MONOREPO_PATH/cmd/svc/carefinder
./build.sh

if [[ "$DEPLOY_TO_S3" != "" ]]; then
    echo "DEPLOYING (carefinder)"

    CMD_NAME="carefinder-$GIT_BRANCH-$BUILD_NUMBER"
    rm -rf build # Jenkins preserves the workspace so remove any old build files
    mkdir build
    cp $GOPATH/bin/carefinder build/$CMD_NAME
    bzip2 -9 build/$CMD_NAME
    echo $GIT_COMMIT > build/$CMD_NAME.revision
    cp $MONOREPO_PATH/coverage-$BUILD_NUMBER.out build/$CMD_NAME.coverage
    cp $MONOREPO_PATH/coverage-$BUILD_NUMBER.html build/$CMD_NAME.coverage.html
    s3cmd --add-header "x-amz-acl:bucket-owner-full-control" -M --server-side-encryption put build/* s3://spruce-deploy/carefinder/

    # Copy over the fonts from the shared location
    LOCAL_CAREFINDER_STATIC_PATH="$MONOREPO_PATH/cmd/svc/carefinder/resources/static"
    rm -rf $LOCAL_CAREFINDER_STATIC_PATH/fonts
    mkdir $LOCAL_CAREFINDER_STATIC_PATH/fonts
    cp $MONOREPO_PATH/resources/static/fonts/* $LOCAL_CAREFINDER_STATIC_PATH/fonts

    cd $LOCAL_CAREFINDER_STATIC_PATH
    STATIC_PREFIX="s3://spruce-static/carefinder/$BUILD_NUMBER"
    s3cmd --recursive -P --no-preserve -m "application/javascript" put js/* $STATIC_PREFIX/js/
    s3cmd --recursive -P --no-preserve -m "text/css" put css/* $STATIC_PREFIX/css/
    s3cmd --recursive -P --no-preserve -m "application/octet-stream" --add-header "Access-Control-Allow-Origin:*" put fonts/*.ttf $STATIC_PREFIX/fonts/
    s3cmd --recursive -P --no-preserve -m "application/vnd.ms-fontobject" --add-header "Access-Control-Allow-Origin:*" put fonts/*.eot $STATIC_PREFIX/fonts/
    s3cmd --recursive -P --no-preserve -m "application/font-woff" --add-header "Access-Control-Allow-Origin:*" put fonts/*.woff $STATIC_PREFIX/fonts/
    s3cmd --recursive -P --no-preserve -m "application/font-woff2" --add-header "Access-Control-Allow-Origin:*" put fonts/*.woff2 $STATIC_PREFIX/fonts/
    s3cmd --recursive -P --no-preserve -m "image/svg+xml" --add-header "Access-Control-Allow-Origin:*" put fonts/*.svg $STATIC_PREFIX/fonts/
    s3cmd --recursive -P --no-preserve -M put img/* $STATIC_PREFIX/img/
fi
