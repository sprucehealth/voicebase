#!/bin/bash -e

DATE=$(date +%Y%m%d%H%M)
DEV_HOSTS="54.209.125.122"
PROD_HOSTS="10.0.43.95 10.0.95.22"
APP=restapi
HOSTS=$DEV_HOSTS
deploy_env=$1

if [ "$deploy_env" == "prod" ]; 
then
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

GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $APP

for HOST in $HOSTS
do
	scp -C $APP $HOST:/usr/local/apps/$APP/$APP.$DATE
	ssh $HOST "cd /usr/local/apps/$APP && chmod +x $APP.$DATE && rm -f $APP && ln -s $APP.$DATE $APP && supervisorctl restart $APP"
done
