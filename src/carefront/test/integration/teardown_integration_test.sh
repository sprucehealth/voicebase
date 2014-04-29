#!/bin/bash

PASSWORD_ARG="-p$RDS_PASSWORD"
if [ "$RDS_PASSWORD" = "" ]; then
	PASSWORD_ARG=""
fi
echo "drop database $1;" | mysql -h $RDS_INSTANCE -u $RDS_USERNAME $PASSWORD_ARG
