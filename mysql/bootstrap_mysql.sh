#!/bin/bash

# Derived from: http://jetpackweb.com/blog/2009/07/20/bash-script-to-create-mysql-database-and-user/
 
EXPECTED_ARGS=4
E_BADARGS=65
MYSQL=`which mysql`
 
Q1="CREATE DATABASE IF NOT EXISTS $1;"
Q2="GRANT ALL ON *.* TO '$2'@'localhost' IDENTIFIED BY '$3';"
Q3="FLUSH PRIVILEGES;"
SQL="${Q1}${Q2}${Q3}"
 
if [ $# -ne $EXPECTED_ARGS ]
then
  echo "Usage: $0 dbname dbuser dbpass latest_migration_number"
  exit $E_BADARGS
fi
 
$MYSQL -uroot -p -e "$SQL"

echo "...creating schema and data snapshot files as of migration $4"

echo "use $1;" | cat - snapshot-$4.sql > temp.sql
echo "use $1;" | cat - data-snapshot-$4.sql > data_temp.sql

echo "...seeding schema and data snapshots"

$MYSQL -u $2 -p$3 < temp.sql
$MYSQL -u $2 -p$3 < data_temp.sql

echo "...inserting successful migration $4 as an event/row into the migrations table"

RECORD_MIGRATION_SQL="use $1; insert into migrations (migration_id, migration_user) values ($4, '$3');"

$MYSQL -u $2 -p$3 -e "$RECORD_MIGRATION_SQL"

echo "...cleaning up schema and data snapshot files"

rm ./temp.sql
rm ./data_temp.sql

