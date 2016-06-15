#!/bin/bash

set -e -o pipefail

# Lowercase version of the job name
NAME=${BUILD_TAG,,}

echo 'ssh -i ~/.ssh/id_api $*' > ssh ; chmod +x ssh
export GIT_SSH="./ssh"

# Remove ignored files
git clean -X -f
# Remove code coverage files (not always deleted by git clean if a directory is removed)
find . -name 'cover.out' -exec rm '{}' \;
find . -name 'coverage.xml' -exec rm '{}' \;

docker build --rm --force-rm -t $NAME docker-ci

# get the uid of the user running the job to be able to properly manage permissions
PARENT_UID=$(id -u)

# use docker gid to give job access to docker socket
PARENT_GID=$(getent group docker | cut -d: -f3)

# origin/master -> master
BRANCH=$(echo $GIT_BRANCH | cut -d'/' -f2)
# if building a Phabricator diff then use the revision ID instead
if [[ "$REVISION_ID" != "" ]]; then
	BRANCH="D${REVISION_ID}"
fi

GIT_REV="$GIT_COMMIT"
if [ "$GIT_REV" = "" ]; then
    GIT_REV=$(git rev-parse HEAD)
fi
MEMPATH="/mnt/mem/jenkins/$BUILD_TAG"
mkdir -p $MEMPATH
docker pull 137987605457.dkr.ecr.us-east-1.amazonaws.com/scratch:latest
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
	-e "NO_INTEGRATION_TESTS=$NO_INTEGRATION_TESTS" \
	-v $MEMPATH:/mem \
	-v `pwd`:/workspace/go/src/github.com/sprucehealth/backend \
	-v /var/run/docker.sock:/var/run/docker.sock \
    $NAME

declare -A devDeployableMap
devDeployableMap["auth"]="deployable_0E1V9I2MO0O00"
devDeployableMap["settings"]="deployable_0E2KVI78O0O00"
devDeployableMap["routing"]="deployable_0E2L0O5R80O00"
devDeployableMap["threading"]="deployable_0E2L0O6CO0O00"
devDeployableMap["baymaxgraphql"]="deployable_0E2L0O6UG0O00"
devDeployableMap["invite"]="deployable_0E2L0O7H00O00"
devDeployableMap["excomms"]="deployable_0E2L0O98G0O00,deployable_0E2L0O8HG0O00"
devDeployableMap["notification"]="deployable_0E2L0O9PG0O00"
devDeployableMap["directory"]="deployable_0E2L0Q5000O00"
devDeployableMap["operational"]="deployable_0E38JIGJG0O00"
devDeployableMap["layout"]="deployable_0E6H974S00O00"
devDeployableMap["care"]="deployable_0E91VDOHG0O00"
devDeployableMap["media"]="deployable_0EM1LSDHG0O00,deployable_0EP10F1PG0O00"

if [[ "$DEPLOY_TO_S3" != "" ]]; then
	echo "Pushing images for revision: $GIT_REV"
	# Tag any generated images with the remote repo and push
	IMAGES=$(docker images -q -f label=version=$BRANCH-$BUILD_ID)
	echo $IMAGES
	for IMAGEID in $IMAGES; do
		TAG=$(docker inspect -f '{{index .RepoTags 0}}' $IMAGEID)
		echo "Pushing $TAG"
		REMOTETAG="137987605457.dkr.ecr.us-east-1.amazonaws.com/$TAG"
		docker tag $TAG $REMOTETAG
		NEXT_WAIT_TIME=0
		until docker push $REMOTETAG || [ $NEXT_WAIT_TIME -eq 4 ]; do
			sleep $(( NEXT_WAIT_TIME++ ))
		done
		IFS=':' read -a STAG <<< "$TAG"

		if [[ "$MANUAL_DEPLOY" == "" ]]; then
			if [[ ${devDeployableMap[${STAG[0]}]} != "" ]]; then
				IFS=',' read -a DEPLOYABLES <<< "${devDeployableMap[${STAG[0]}]}"
				for DEPLOYABLE in ${DEPLOYABLES[@]}; do
					echo "Notifying Deploy Service of successful build $DEPLOYABLE - $TAG"
					aws sqs send-message --queue-url=https://sqs.us-east-1.amazonaws.com/137987605457/corp-deploy-events --message-body="{\"deployable_id\":\"$DEPLOYABLE\",\"build_number\":\"$BUILD_ID\",\"image\":\"$REMOTETAG\",\"git_hash\":\"$GIT_REV\"}" --region=us-east-1
				done
			fi
		fi

		docker rmi $REMOTETAG
		docker rmi $TAG
	done
fi
