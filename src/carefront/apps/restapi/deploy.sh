#!/bin/bash

DATE=$(date +%Y%m%d%H%M)
DEV_HOSTS="54.209.125.122"
PROD_HOSTS="10.0.43.95 10.0.95.22"
APP=restapi
HOSTS=$DEV_HOSTS
deploy_env=$1

GOVERSION=$(go version)
REV=$(git rev-parse HEAD)
BRANCH=$(git rev-parse --abbrev-ref HEAD)

if [ "$deploy_env" == "prod" ]; 
then
	# Make sure the current branch is master and is the latest version according to origin/master

	if [ "$BRANCH" != "master" ]; then
		echo "Current branch is $BRANCH. Please only deploy from master."
		exit 2;
	fi
	# Pull in latest origin
	git fetch
	git diff --quiet origin/master
	if [ "$?" != "0" ]; then
		echo "Your repo does not match origin/master. Please make sure there's no uncommited changes and you have the latest changes before deploying."
		exit 3
	fi

	read -p "To be sure you want to deploy to production, type PROD if you wish to deploy to production: " confirmation
	case $confirmation in
		PROD ) HOSTS=$PROD_HOSTS;;
		* ) exit;;
	esac
elif [ "$deploy_env" != "dev" ];
then 
	echo "ERROR: Usage : ./deploy.sh [dev|prod] " >&2
	exit 1;
fi

set -e

GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $APP

for HOST in $HOSTS
do
	LOGMSG="{\"env\":\"$deploy_env\",\"user\":\"$USER\",\"app\":\"$APP\",\"date\":\"$DATE\",\"host\":\"$HOST\",\"goversion\":\"$GOVERSION\",\"rev\":\"$REV\",\"branch\":\"$BRANCH\"}"
	scp -C $APP $HOST:/usr/local/apps/$APP/$APP.$DATE
	ssh $HOST "cd /usr/local/apps/$APP && chmod +x $APP.$DATE && rm -f $APP && ln -s $APP.$DATE $APP && supervisorctl restart $APP ; logger -p user.info -t deploy '$LOGMSG'"
done
