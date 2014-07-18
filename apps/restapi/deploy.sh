#!/bin/bash

DATE=$(date +%Y%m%d%H%M)
DEV_HOSTS="54.209.3.156"
PROD_HOSTS="10.0.43.89 10.0.89.31"
STAGING_HOSTS="10.1.5.105 10.1.22.127"
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
