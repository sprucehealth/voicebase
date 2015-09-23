#!/bin/bash

useradd -d `pwd` -u $PARENT_UID ci
chown -R ci /workspace /usr/local
export HOME=/workspace

# Start MySQL
mv /var/lib/mysql /mem/mysql
ln -s /mem/mysql /var/lib/mysql
/etc/init.d/mysql start

su ci -c "/bin/bash -e /run.sh"
EXIT=$?

# Cleanup
killall -9 mysqld mysqld_safe
rm -rf /mem/*

exit $EXIT
