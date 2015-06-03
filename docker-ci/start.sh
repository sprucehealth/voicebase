#!/bin/bash

/bin/bash -e /run.sh
EXIT=$?

# Cleanup
killall -9 mysqld mysqld_safe
rm -rf /mem/*

exit $EXIT
