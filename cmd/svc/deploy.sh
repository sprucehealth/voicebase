#!/bin/bash -e

DATE=$(date +%Y%m%d%H%M)
DEV_HOSTS="dev-restapi-1.node.dev-us-east-1.spruce"
PROD_HOSTS="prod-restapi-1.node.prod-us-east-1.spruce prod-restapi-2.node.prod-us-east-1.spruce"
STAGING_HOSTS="staging-restapi-1.node.staging-us-east-1.spruce staging-restapi-2.node.staging-us-east-1.spruce"
DEMO_HOSTS="54.210.97.69"
APP="$1"
DEPLOY_ENV="$2"
DEPLOY_BUILD="$3"
DEPLOY_BRANCH="$4"

if [ "$APP" == "" ]; then
	echo "Missing app name."
	exit 2
fi

if [ "$DEPLOY_BUILD" = "" ]; then
	echo "Missing build number."
	exit 2
fi

if [ "$DEPLOY_BRANCH" != "master" ]; then
	echo "Can only deploy the master branch to production."
	exit 2
fi

GOVERSION=$(go version)

case "$DEPLOY_ENV" in
	"prod" )
		HOSTS="$PROD_HOSTS"

		read -p "To be sure you want to deploy to production, type PROD if you wish to deploy to production: " confirmation
		case $confirmation in
			PROD ) HOSTS=$PROD_HOSTS;;
			* ) exit;;
		esac
	;;

	"staging" )
		HOSTS=$STAGING_HOSTS
	;;

	"dev" )
		HOSTS=$DEV_HOSTS
	;;

	"demo" )
		HOSTS=$DEMO_HOSTS
	;;

	* )
		echo "ERROR: Usage : ./deploy.sh [appname] [staging|dev|prod] [build number] [branch]" >&2
		exit 1;
	;;
esac

set -e

BRANCH="$DEPLOY_BRANCH"
for HOST in $HOSTS
do
	LOGMSG="{\"env\":\"$DEPLOY_ENV\",\"user\":\"$USER\",\"app\":\"$APP\",\"date\":\"$DATE\",\"host\":\"$HOST\",\"goversion\":\"$GOVERSION\",\"rev\":\"$REV\",\"branch\":\"$BRANCH\",\"build\":$DEPLOY_BUILD}"
	NAME="$APP-$DEPLOY_BRANCH-$DEPLOY_BUILD"
	ssh -t $HOST "cd /usr/local/apps/$APP && s3cmd -c s3cfg --force get s3://spruce-deploy/$APP/$NAME.bz2 && bzip2 -fd $NAME.bz2 && chmod +x $NAME && rm -f $APP && ln -s $NAME $APP && sudo supervisorctl restart $APP ; logger -p user.info -t deploy '$LOGMSG'"
done


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
