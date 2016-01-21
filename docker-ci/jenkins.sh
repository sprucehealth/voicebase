#!/bin/bash -x

# Lowercase version of the job name
NAME=${BUILD_TAG,,}

echo 'ssh -i ~/.ssh/id_api $*' > ssh ; chmod +x ssh
export GIT_SSH="./ssh"

# Remove ignored files
git clean -X -f
# Remove cover.out files (not always deleted by git clean if a directory is removed)
find . -name 'cover.out' -exec rm '{}' \;

docker build --rm --force-rm -t $NAME docker-ci

# get the uid of the user running the job to be able to properly manage permissions
PARENT_UID=$(id -u)

# use docker gid to give job access to docker socket
PARENT_GID=$(getent group docker | cut -d: -f3)

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
	-e "TEST_S3_BUCKET=$TEST_S3_BUCKET" \
	-e "PARENT_UID=$PARENT_UID" \
	-e "PARENT_GID=$PARENT_GID" \
	-v $MEMPATH:/mem \
	-v `pwd`:/workspace/go/src/github.com/sprucehealth/backend \
	-v /var/run/docker.sock:/var/run/docker.sock \
    $NAME

if [[ "$DEPLOY_TO_S3" != "" ]]; then
	# Tag any generated images with the remote repo and push
	IMAGES=$(docker images -q -f label=version=$BRANCH-$BUILD_ID)
	echo $IMAGES
	for IMAGEID in $IMAGES; do
		TAG=$(docker inspect -f '{{index .RepoTags 0}}' $IMAGEID)
		echo "Pushing $TAG"
		REMOTETAG="137987605457.dkr.ecr.us-east-1.amazonaws.com/$TAG"
		docker tag $TAG $REMOTETAG
		docker push $REMOTETAG
		docker rmi $REMOTETAG
	done
fi
