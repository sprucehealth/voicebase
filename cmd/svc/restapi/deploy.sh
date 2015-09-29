#!/bin/bash -e

DATE=$(date +%Y%m%d%H%M)
DEV_HOSTS="dev-restapi-1.node.dev-us-east-1.spruce"
PROD_HOSTS="prod-restapi-1.node.prod-us-east-1.spruce prod-restapi-2.node.prod-us-east-1.spruce"
STAGING_HOSTS="staging-restapi-1.node.staging-us-east-1.spruce staging-restapi-2.node.staging-us-east-1.spruce"
DEMO_HOSTS="54.210.97.69"
DEPLOY_ENV="$1"
DEPLOY_BUILD="$2"
DEPLOY_BRANCH="$3"
APP="restapi"
if [ "$DEPLOY_BRANCH" = "" ]; then
	DEPLOY_BRANCH=$(git rev-parse --abbrev-ref HEAD)
fi
GOVERSION=$(go version)

case "$DEPLOY_ENV" in
	"prod" )
		HOSTS="$PROD_HOSTS"
		DEPLOY_BRANCH="master"

		if [ "$DEPLOY_BUILD" = "" ]; then
			echo "Missing build number. Cannot deploy to production from local code."
			exit 2
		fi
		if [ "$DEPLOY_BRANCH" != "master" ]; then
			echo "Can only deploy the master branch to production."
			exit 2
		fi

		read -p "To be sure you want to deploy to production, type PROD if you wish to deploy to production: " confirmation
		case $confirmation in
			PROD ) HOSTS=$PROD_HOSTS;;
			* ) exit;;
		esac
	;;

	"dev" )
		HOSTS=$DEV_HOSTS

		if [ "$DEPLOY_BUILD" = "" ]; then
			. ./build.sh
		fi
	;;

	"staging" )
		HOSTS=$STAGING_HOSTS

		if [ "$DEPLOY_BUILD" = "" ]; then
			echo "Missing build number. Cannot deploy to staging from local code."
			exit 2
		fi
		if [ "$DEPLOY_BRANCH" != "master" ]; then
			echo "Can only deploy the master branch to production."
			exit 2
		fi
	;;

	"demo" )
		HOSTS=$DEMO_HOSTS

		if [ "$DEPLOY_BUILD" = "" ]; then
			echo "Missing build number. Cannot deploy to staging from local code."
			exit 2
		fi
		if [ "$DEPLOY_BRANCH" != "master" ]; then
			echo "Can only deploy the master branch to production."
			exit 2
		fi
	;;


	* )
		echo "ERROR: Usage : ./deploy.sh [staging|dev|prod] [build number] [branch]" >&2
		exit 1;
	;;
esac

set -e

if [ "$DEPLOY_BUILD" = "" ]; then
	for HOST in $HOSTS
	do
		LOGMSG="{\"env\":\"$DEPLOY_ENV\",\"user\":\"$USER\",\"app\":\"$APP\",\"date\":\"$DATE\",\"host\":\"$HOST\",\"goversion\":\"$GOVERSION\",\"rev\":\"$REV\",\"branch\":\"$BRANCH\"}"
		scp -C $APP $HOST:/usr/local/apps/$APP/$APP.$DATE
		ssh -t $HOST "cd /usr/local/apps/$APP && chmod +x $APP.$DATE && rm -f $APP && ln -s $APP.$DATE $APP && sudo supervisorctl restart $APP ; logger -p user.info -t deploy '$LOGMSG'"
	done
else
	BRANCH="$DEPLOY_BRANCH"
	for HOST in $HOSTS
	do
		LOGMSG="{\"env\":\"$DEPLOY_ENV\",\"user\":\"$USER\",\"app\":\"$APP\",\"date\":\"$DATE\",\"host\":\"$HOST\",\"goversion\":\"$GOVERSION\",\"rev\":\"$REV\",\"branch\":\"$BRANCH\",\"build\":$DEPLOY_BUILD}"
		NAME="$APP-$DEPLOY_BRANCH-$DEPLOY_BUILD"
		ssh -t $HOST "cd /usr/local/apps/$APP && s3cmd -c s3cfg --force get s3://spruce-deploy/$APP/$NAME.bz2 && bzip2 -fd $NAME.bz2 && chmod +x $NAME && rm -f $APP && ln -s $NAME $APP && sudo supervisorctl restart $APP ; logger -p user.info -t deploy '$LOGMSG'"
	done
fi


### Post a message to the slack notifications channel upon deployment if the 
### incoming webhook for slack is set
if [ "$SLACK_NOTIFY_WEBHOOK" != "" ]; then
	curl -X POST --data-urlencode "payload={
		\"icon_emoji\": \":ghost:\",
		\"attachments\":[
		      {
		         \"fallback\":\"$USER deployed $APP\",
		         \"pretext\":\"$USER deployed $APP\",
		         \"color\":\"good\",
		         \"fields\":[
		           	{
		               \"title\":\"Environment\",
		               \"value\":\"$DEPLOY_ENV\",
		               \"short\":true
		            },
					{
		               \"title\":\"Travis Build ID\",
		               \"value\":\"$DEPLOY_BUILD ($DEPLOY_BRANCH)\",
		               \"short\":true
		            }
		         ]
		      }
		   ]}" "$SLACK_NOTIFY_WEBHOOK"
fi
