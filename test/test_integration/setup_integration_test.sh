#!/bin/bash -e

TEST_DB=database_${RANDOM}_$(date +%s)
MYSQL_FOLDER=${GOPATH}/src/github.com/sprucehealth/backend/mysql
pushd $MYSQL_FOLDER > /dev/null
latestSnapshotNumber=`ls -r snapshot-*.sql | cut -d- -f 2  | cut -d. -f1 | sort -nr | head -1`
latestDataSnapshotNumber=`ls -r data-snapshot-*.sql | cut -d- -f 3  | cut -d. -f1 | sort -nr | head -1`
PASSWORD_ARG="-p$RDS_PASSWORD"
if [ "$RDS_PASSWORD" = "" ]; then
	PASSWORD_ARG=""
fi
echo "create database $TEST_DB; use $TEST_DB;"  | cat - snapshot-$latestSnapshotNumber.sql | mysql -h $RDS_INSTANCE -u $RDS_USERNAME $PASSWORD_ARG
echo "use $TEST_DB;" | cat - data-snapshot-$latestDataSnapshotNumber.sql | mysql -h $RDS_INSTANCE -u $RDS_USERNAME $PASSWORD_ARG
popd > /dev/null
echo $TEST_DB
