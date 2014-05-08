#!/bin/bash -e

# This script makes it easy to apply changes to the development and production database once 
# the schema has been validated. 

RDS_INSTANCE="127.0.0.1"
RDS_USERNAME="carefront"
DATABASE_NAME="carefront_db"

argsArray=($@) 
len=${#argsArray[@]}

if [ $len -lt 2 ];
then
	echo "ERROR: Usage ./apply_schema.sh [local|dev|prod|staging] migration1 migration2 .... migrationN"
	exit 1;
fi

env=${argsArray[0]}
for migrationNumber in ${argsArray[@]:1:$len}
do 
	echo "Applying migration-$migrationNumber.sql"
	
	# ensure that the file exists
	if [ ! -f snapshot-$migrationNumber.sql ] || [ ! -f data-snapshot-$migrationNumber.sql ] || [ ! -f migration-$migrationNumber.sql ]; then
		echo "ERROR: Looks like migration $migrationNumber has not yet been validated using validate_schema.sql and so they will not be applied to database"
		exit 1
	fi

	
	case "$env" in
		
		"local" )
			echo "use $DATABASE_NAME; insert into migrations (migration_id, migration_user) values ($migrationNumber, '$USER');" > temp-migration.sql
			echo "use $DATABASE_NAME;" | cat - migration-$migrationNumber.sql > temp.sql
			mysql -h $RDS_INSTANCE -u $RDS_USERNAME -p$DEV_RDS_PASSWORD < temp.sql
			mysql -h $RDS_INSTANCE -u $RDS_USERNAME -p$DEV_RDS_PASSWORD < temp-migration.sql
		;;	

		"staging" )
			echo "use $STAGING_DB_NAME; insert into migrations (migration_id, migration_user) values ($migrationNumber, '$USER');" > temp-migration.sql
			LOGMSG="{\"env\":\"$env\",\"user\":\"$USER\",\"migration_id\":\"$migrationNumber\"}"
			echo "use $STAGING_DB_NAME;" | cat - migration-$migrationNumber.sql > temp.sql
			scp temp.sql kunal@$STAGING_BASTIAN:~
			scp temp-migration.sql kunal@$STAGING_BASTIAN:~
			ssh -t $USER@$STAGING_DB_INSTANCE "sudo ec2-consistent-snapshot -mysql.config /mysql-data/mysql/backup.cnf -tag migrationId=migration-$migrationNumber"
			ssh $USER@$STAGING_BASTIAN "mysql -h $STAGING_DB_INSTANCE -u $STAGING_DB_USER_NAME -p$STAGING_DB_PASSWORD < temp.sql ; mysql -h $STAGING_DB_INSTANCE -u $STAGING_DB_USER_NAME -p$STAGING_DB_PASSWORD < temp-migration.sql; logger -p user.info -t schema '$LOGMSG'"
		;;

		"dev" )
			echo "use $DATABASE_NAME; insert into migrations (migration_id, migration_user) values ($migrationNumber, '$USER');" > temp-migration.sql
			echo "use $DATABASE_NAME;" | cat - migration-$migrationNumber.sql > temp.sql
			mysql -h $DEV_RDS_INSTANCE -u $RDS_USERNAME -p$DEV_RDS_PASSWORD < temp.sql
			mysql -h $DEV_RDS_INSTANCE -u $RDS_USERNAME -p$DEV_RDS_PASSWORD < temp-migration.sql
		;;
		
		
		"prod" ) 
			echo "use $PROD_DB_NAME; insert into migrations (migration_id, migration_user) values ($migrationNumber, '$USER');" > temp-migration.sql
			LOGMSG="{\"env\":\"$env\",\"user\":\"$USER\",\"migration_id\":\"$migrationNumber\"}"
			echo "use $PROD_DB_NAME;" | cat - migration-$migrationNumber.sql > temp.sql
			scp temp.sql kunal@54.209.10.66:~
			scp temp-migration.sql kunal@54.209.10.66:~
			ssh -t $USER@$PROD_DB_INSTANCE "sudo ec2-consistent-snapshot -mysql.config /mysql-data/mysql/backup.cnf -tag migrationId=migration-$migrationNumber"
			ssh $USER@54.209.10.66 "mysql -h $PROD_DB_INSTANCE -u $PROD_DB_USER_NAME -p$PROD_DB_PASSWORD < temp.sql ; mysql -h $PROD_DB_INSTANCE -u $PROD_DB_USER_NAME -p$PROD_DB_PASSWORD < temp-migration.sql; logger -p user.info -t schema '$LOGMSG'"
		;;
	esac
	
	rm temp.sql temp-migration.sql

done
