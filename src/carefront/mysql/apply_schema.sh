#!/bin/bash -e

# This script makes it easy to apply changes to the development and production database once 
# the schema has been validated. 

RDS_INSTANCE="dev-db-3.ccvrwjdx3gvp.us-east-1.rds.amazonaws.com"
RDS_USERNAME="carefront"
DATABASE_NAME="carefront_db"

latestSnapshotNumber=`ls -r snapshot-*.sql | cut -d- -f 2  | cut -d. -f1 | sort -nr | head -1`
latestDataSnapshotNumber=`ls -r data-snapshot-*.sql | cut -d- -f 3  | cut -d. -f1 | sort -nr | head -1`
latestMigrationNumber=`ls -r migration-*.sql | cut -d- -f 2  | cut -d. -f1 | sort -nr | head -1`

if [  "$latestMigrationNumber" != "$latestSnapshotNumber"  -o "$latestSnapshotNumber" != "$latestDataSnapshotNumber" ]; then
	echo "ERROR: Looks like migration statements have not yet been validated using validate_schema.sql and so they will not be applied to database"
	exit 1
fi

echo "use $DATABASE_NAME;" | cat - migration-$latestMigrationNumber.sql > temp.sql
mysql -h $RDS_INSTANCE -u $RDS_USERNAME -p$DEV_RDS_PASSWORD < temp.sql
scp temp.sql kunal@54.209.10.66:~
ssh kunal@54.209.10.66 "mysql -h $PROD_RDS_INSTANCE -u $RDS_USERNAME -p$PROD_RDS_PASSWORD < temp.sql"
rm temp.sql

