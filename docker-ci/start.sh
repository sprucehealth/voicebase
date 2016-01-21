#!/bin/bash

groupadd -g $PARENT_GID ci
useradd -d `pwd` -u $PARENT_UID -g ci ci
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
