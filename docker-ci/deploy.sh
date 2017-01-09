#!/bin/bash

set -e -o pipefail

declare -A devDeployableMap
devDeployableMap["auth"]="deployable_0E1V9I2MO0O00"
devDeployableMap["settings"]="deployable_0E2KVI78O0O00"
devDeployableMap["routing"]="deployable_0E2L0O5R80O00"
devDeployableMap["threading"]="deployable_0E2L0O6CO0O00"
devDeployableMap["baymaxgraphql"]="deployable_0E2L0O6UG0O00"
devDeployableMap["invite"]="deployable_0E2L0O7H00O00,deployable_0FAE0Q5O80O00"
devDeployableMap["excomms"]="deployable_0E2L0O98G0O00,deployable_0E2L0O8HG0O00"
devDeployableMap["notification"]="deployable_0E2L0O9PG0O00"
devDeployableMap["directory"]="deployable_0E2L0Q5000O00"
devDeployableMap["operational"]="deployable_0E38JIGJG0O00"
devDeployableMap["layout"]="deployable_0E6H974S00O00"
devDeployableMap["care"]="deployable_0E91VDOHG0O00"
devDeployableMap["media"]="deployable_0EM1LSDHG0O00,deployable_0EP10F1PG0O00"
devDeployableMap["admin"]="deployable_0FRP18V1G0O00"
devDeployableMap["payments"]="deployable_0G6QO9CUO0O00"
devDeployableMap["patientsync"]="deployable_0GGFHCRPO0O00,deployable_0GGFI5HIO0O00"
devDeployableMap["scheduling"]="deployable_0J5FPR4C00O00"

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

	if [[ "$SKIP_DEPLOY" == "" ]]; then
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
