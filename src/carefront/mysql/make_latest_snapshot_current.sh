#!/bin/bash 

latestSnapshotNumber=`ls -r snapshot-*.sql | cut -d- -f 2  | cut -d. -f1 | sort -nr | head -1`
echo "create database database_$TRAVIS_BUILD_ID; use database_$TRAVIS_BUILD_ID;"  | cat - snapshot-$latestSnapshotNumber.sql > current.sql
echo "Going to load latest schame into database_$TRAVIS_BUILD_ID"