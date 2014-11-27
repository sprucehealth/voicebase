#!/bin/bash

PASSWORD_ARG="-p$CF_LOCAL_DB_PASSWORD"
if [ "$CF_LOCAL_DB_PASSWORD" = "" ]; then
	PASSWORD_ARG=""
fi
echo "drop database $1;" | mysql -h $CF_LOCAL_DB_INSTANCE -u $CF_LOCAL_DB_USERNAME $PASSWORD_ARG
