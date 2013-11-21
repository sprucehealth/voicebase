#!/bin/bash -e

DATE=$(date +%Y%m%d%H%M)

GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o server
rm server.bz2 || true
bzip2 server

for HOST in 10.0.43.114 10.0.95.240
do
	scp server.bz2 $HOST:/usr/local/apps/restapi/server.$DATE.bz2
	ssh $HOST "cd /usr/local/apps/restapi && bzip2 -d server.$DATE.bz2 && chmod +x server.$DATE && rm -f server && ln -s server.$DATE server && supervisorctl restart restapi"
done
