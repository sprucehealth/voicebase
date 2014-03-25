#!/bin/bash -e
TEST_DB=database_${RANDOM}_$(date +%s)
MYSQL_FOLDER=${CAREFRONT_PROJECT_DIR}/src/carefront/mysql
pushd $MYSQL_FOLDER > /dev/null
latestSnapshotNumber=`ls -r snapshot-*.sql | cut -d- -f 2  | cut -d. -f1 | sort -nr | head -1`
latestDataSnapshotNumber=`ls -r data-snapshot-*.sql | cut -d- -f 3  | cut -d. -f1 | sort -nr | head -1`
echo "create database $TEST_DB; use $TEST_DB;"  | cat - snapshot-$latestSnapshotNumber.sql > current.sql
echo "use $TEST_DB;" | cat - data-snapshot-$latestDataSnapshotNumber.sql > current_data.sql
mysql -h 127.0.0.1 -u travis -p12345 < current.sql
mysql -h 127.0.0.1 -u travis -p12345 < current_data.sql
rm current.sql
rm current_data.sql
popd > /dev/null
echo $TEST_DB