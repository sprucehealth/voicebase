#!/bin/bash 

latestSnapshotNumber=`ls -r snapshot-*.sql | cut -d- -f 2  | cut -d. -f1 | sort -nr | head -1`
echo "create database carefront_test; use carefront_test;"  | cat - snapshot-$latestSnapshotNumber.sql > current.sql