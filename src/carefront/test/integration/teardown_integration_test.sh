#!/bin/bash 
echo "drop database $1;" > drop_database.sql
mysql -h 127.0.0.1 -u root < drop_database.sql
rm drop_database.sql