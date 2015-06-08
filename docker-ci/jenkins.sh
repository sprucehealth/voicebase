#!/bin/bash -x

# Lowercase version of the job name
NAME=${BUILD_TAG,,}

echo 'ssh -i ~/.ssh/id_api $*' > ssh ; chmod +x ssh
export GIT_SSH="./ssh"
git submodule update --init
# Remove ignored files
git clean -X -f
# Remove cover.out files (not always deleted by git clean if a directory is removed)
find . -name 'cover.out' -exec rm '{}' \;

docker build --rm --force-rm -t $NAME docker-ci

# origin/master -> master
BRANCH=$(echo $GIT_BRANCH | cut -d'/' -f2)
MEMPATH="/mnt/mem/jenkins/$BUILD_TAG"
mkdir -p $MEMPATH
docker run --rm=true --name=$NAME \
	-e "BUILD_NUMBER=$BUILD_NUMBER" \
	-e "BUILD_ID=$BUILD_ID" \
	-e "BUILD_URL=$BUILD_URL" \
	-e "BUILD_TAG=$BUILD_TAG" \
	-e "GIT_COMMIT=$GIT_COMMIT" \
	-e "GIT_URL=$GIT_URL" \
	-e "GIT_BRANCH=$BRANCH" \
	-e "JOB_NAME=$JOB_NAME" \
	-e "DEPLOY_TO_S3=$DEPLOY_TO_S3" \
	-e "FULLCOVERAGE=$FULLCOVERAGE" \
	-v $MEMPATH:/mem \
	-v `pwd`:/workspace/go/src/github.com/sprucehealth/backend \
    $NAME
