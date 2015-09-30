#!/bin/bash

set -e

BGPID=""

# checkedwait waits for background jobs to complete with non-zero code if any fail
checkedwait() {
	FAIL=0
	echo "Waiting on $BGPID"
	for job in $BGPID; do
		wait $job || let "FAIL+=1"
	done
	if [ "$FAIL" != "0" ]; then
		echo "FAIL"
	    exit 1
	fi
	BGPID=""
}

# savepid saves the last started bg job's PID for later use by checkedwait
savepid() {
	BGPID="$BGPID $!"
}

APPS="admin dronboard home parental-consent practice-extension"
RESOURCEPATH="$(cd "$(dirname "$0")/.."; pwd)/resources"

if [ "$1" != "" ]; then
	APPS="$1"
fi

if [ "$NPM" == "" ]; then
	NPM="npm"
fi

if [ ! "$APPS" == "css" ]; then
	echo "Installing dependencies..."
	(
		cd $RESOURCEPATH/apps/libs
		$NPM install
	) &
	for APP in $APPS; do
		(
			cd $RESOURCEPATH/apps/$APP
			$NPM install
		) &
		savepid
	done
	checkedwait
	echo "Building js..."
	for APP in $APPS; do
		(
			cd $RESOURCEPATH/apps/$APP
			if [[ "$BUILDENV" == "prod" ]]; then
				PATH="$($NPM bin):$PATH" $NPM run build 2>&1 | grep -v "WARN: " | grep -v "util.error: Use console.error instead"
			fi
			PATH="$($NPM bin):$PATH" $NPM run build-dev 2>&1 | grep -v "WARN: " | grep -v "util.error: Use console.error instead"
		) &
		savepid
	done
	checkedwait
fi

SASS="sass -I=$RESOURCEPATH/static/css-sass --sourcemap=none"
if [ -e "/usr/local/bin/sassc" ]; then
	SASS="/usr/local/bin/sassc -I $RESOURCEPATH/static/css-sass"
fi

echo "Building css..."
for scss in $(ls $RESOURCEPATH/static/css-sass); do
	# Ignore files starting with an underscore
	if [ "$(echo $scss | cut -c1)" != "_" ]; then
		# Remove scss extension
		name=$(echo $scss | cut -d. -f1)
		devName="$name.css"
		minName="$name.min.css"
		$SASS -tnested "$RESOURCEPATH/static/css-sass/$scss" "$RESOURCEPATH/static/css/$devName" &
		savepid
		$SASS -tcompressed "$RESOURCEPATH/static/css-sass/$scss" "$RESOURCEPATH/static/css/$minName" &
		savepid
	fi
done
checkedwait
