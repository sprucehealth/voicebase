#!/bin/bash 

# This script assumes that there is a live RDS instance
# that is publically accessible. The RDS instance is needed
# to validate the migration DDL and create an updated snapshot
RDS_INSTANCE="dev-db-3.ccvrwjdx3gvp.us-east-1.rds.amazonaws.com"
RDS_USERNAME="carefront"
DATABASE_NAME="database_$RANDOM"

# The RDS password for this instance is expected to be set as an environment variable
# named DEV_RDS_PASSWORD

# trapping the TERM signal enables us to instruct the
# process executing the bash script to exit if the TERM 
# signal is sent to it
trap "exit 1" TERM
export TOP_PID=$$

function cleanup {
	echo -e "--- Cleaning up temp files created and dropping database $DATABASE_NAME from rds instance\n"
	rm temp-migration.sql
	rm temp.sql
	echo "drop database $DATABASE_NAME;" > temp-drop-database.sql
	mysql -h $RDS_INSTANCE -u $RDS_USERNAME -p$DEV_RDS_PASSWORD < temp-drop-database.sql
	if [ $? -ne 0 ]; then
		echo "--- ERROR: Unable to drop database $DATABASE_NAME from rds instance"
	fi
	rm temp-drop-database.sql
}

# Identify the latest snapshot of the database that exists
# The latest snapshot is essentially the snapshot with the largest number in the snapshot-N.sql format
latestSnapshotNumber=`ls -r snapshot-*.sql | cut -d- -f 2  | cut -d. -f1 | sort -nr | head -1`
latestMigrationNumber=`ls -r migration-*.sql | cut -d- -f 2  | cut -d. -f1 | sort -nr | head -1`

## add the create database and use database statements before the rest of the sql statements
echo "create database $DATABASE_NAME; use $DATABASE_NAME;"  | cat - snapshot-$latestSnapshotNumber.sql > temp.sql
 
# Use this snapshot as the base to create a random database on a test mysql instance
echo -e "--- Creating database $DATABASE_NAME and restoring schema from snapshot-$latestSnapshotNumber.sql\n"
mysql -h $RDS_INSTANCE -u $RDS_USERNAME -p$DEV_RDS_PASSWORD < temp.sql

# Apply the latest migration file to the database
echo -e "--- Applying DDL in migrate-$latestMigrationNumber.sql to database\n"
echo "use $DATABASE_NAME;" | cat - migration-$latestMigrationNumber.sql > temp-migration.sql
mysql -h $RDS_INSTANCE -u $RDS_USERNAME -p$DEV_RDS_PASSWORD < temp-migration.sql 

if [ $? -ne 0 ]; then
	cleanup
	kill -s TERM $TOP_PID
fi

# If migration successful, snapshotting database again to generate new schema
newSnapshotNumber=$((latestSnapshotNumber + 1))
echo -e "--- Creating new snapshot from database into snapshot-$newSnapshotNumber.sql\n"
`mysqldump -h $RDS_INSTANCE -u $RDS_USERNAME --no-data $DATABASE_NAME -p$DEV_RDS_PASSWORD > snapshot-$newSnapshotNumber.sql`
cleanup
