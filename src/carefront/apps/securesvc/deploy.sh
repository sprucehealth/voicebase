#!/bin/bash -e

DATE=$(date +%Y%m%d%H%M)

APP=securesvc
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $APP
rm $APP.bz2 || true
bzip2 $APP

for HOST in 10.0.38.180
do
	scp $APP.bz2 $HOST:/usr/local/apps/$APP/$APP.$DATE.bz2
	ssh $HOST "cd /usr/local/apps/$APP && bzip2 -d $APP.$DATE.bz2 && chmod +x $APP.$DATE && rm -f $APP && ln -s $APP.$DATE $APP && supervisorctl restart $APP"
done
