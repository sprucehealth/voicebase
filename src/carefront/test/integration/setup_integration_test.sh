#!/bin/bash 
TEST_DB=database_${RANDOM}_$(date +%s)
MYSQL_FOLDER=${CAREFRONT_PROJECT_DIR}/src/carefront/mysql
pushd $MYSQL_FOLDER > /dev/null
latestSnapshotNumber=`ls -r snapshot-*.sql | cut -d- -f 2  | cut -d. -f1 | sort -nr | head -1`
latestDataSnapshotNumber=`ls -r data-snapshot-*.sql | cut -d- -f 3  | cut -d. -f1 | sort -nr | head -1`
echo "create database $TEST_DB; use $TEST_DB;"  | cat - snapshot-$latestSnapshotNumber.sql > current.sql
echo "use $TEST_DB;" | cat - data-snapshot-$latestDataSnapshotNumber.sql > current_data.sql
mysql -h $RDS_INSTANCE -u $RDS_USERNAME -p$RDS_PASSWORD < current.sql
mysql -h $RDS_INSTANCE -u $RDS_USERNAME -p$RDS_PASSWORD < current_data.sql
rm current.sql
rm current_data.sql
popd > /dev/null
echo $TEST_DB