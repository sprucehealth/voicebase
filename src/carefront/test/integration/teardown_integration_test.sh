#!/bin/bash

echo "drop database $1;" > drop_database.sql
PASSWORD_ARG="-p$RDS_PASSWORD"
if [ "$RDS_PASSWORD" = "" ]; then
	PASSWORD_ARG=""
fi
mysql -h $RDS_INSTANCE -u $RDS_USERNAME $PASSWORD_ARG < drop_database.sql
rm drop_database.sql
