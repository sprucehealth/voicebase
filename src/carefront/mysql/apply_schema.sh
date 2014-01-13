#!/bin/bash -e

# This script makes it easy to apply changes to the development and production database once 
# the schema has been validated. 

RDS_INSTANCE="dev-db-3.ccvrwjdx3gvp.us-east-1.rds.amazonaws.com"
RDS_USERNAME="carefront"
DATABASE_NAME="carefront_db"

argsArray=($@) 
len=${#argsArray[@]}

if [ $len -lt 2 ];
then
	echo "ERROR: Usage ./apply_schema.sh [local|dev|prod] migration1 migration2 .... migrationN"
	exit 1;
fi

env=${argsArray[0]}
for migrationNumber in ${argsArray[@]:1:$len}
do 
	# ensure that the file exists
	if [ ! -f snapshot-$migrationNumber.sql ] || [ ! -f data-snapshot-$migrationNumber.sql ] || [ ! -f migration-$migrationNumber.sql ]; then
		echo "ERROR: Looks like migration $migrationNumber has not yet been validated using validate_schema.sql and so they will not be applied to database"
		exit 1
	fi

	case "$env" in
		
		"local" )
			echo "use $DATABASE_NAME;" | cat - migration-$migrationNumber.sql > temp.sql
			mysql -h $RDS_INSTANCE -u $RDS_USERNAME -p$DEV_RDS_PASSWORD < temp.sql
		;;

		"dev" )
			echo "use $DATABASE_NAME;" | cat - migration-$migrationNumber.sql > temp.sql
			mysql -h $DEV_RDS_INSTANCE -u $RDS_USERNAME -p$DEV_RDS_PASSWORD < temp.sql
		;;
		
		
		"prod" ) 
			LOGMSG="{\"env\":\"$env\",\"user\":\"$USER\",\"migration_id\":\"$migrationNumber\"}"
			echo "use $PROD_DB_NAME;" | cat - migration-$migrationNumber.sql > temp.sql
			scp temp.sql kunal@54.209.10.66:~
			ssh -t $USER@$PROD_DB_INSTANCE "sudo ec2-consistent-snapshot -mysql.config /mysql-data/mysql/backup.cnf -tag migrationId=migration-$migrationNumber"
			ssh $USER@54.209.10.66 "mysql -h $PROD_DB_INSTANCE -u $PROD_DB_USER_NAME -p$PROD_DB_PASSWORD < temp.sql ; logger -p user.info -t schema '$LOGMSG'"
		;;
	esac
	
	rm temp.sql

done
