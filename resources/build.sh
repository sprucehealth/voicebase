#!/bin/bash

APPS="admin dashboard dronboard home"
RESOURCEPATH="$(cd "$(dirname "$0")/.."; pwd)/resources"

if [ "$1" != "" ]; then
	APPS="$1"
fi

if [ "$NPM" == "" ]; then
	NPM="npm"
fi

if [ ! "$APPS" == "css" ]; then
	(
		cd $RESOURCEPATH/apps/libs
		$NPM install
	)
	for APP in $APPS; do
		(
			cd $RESOURCEPATH/apps/$APP
			$NPM install
			PATH="$($NPM bin):$PATH" $NPM run build 2>&1 | grep -v "WARN: " | grep -v "util.error: Use console.error instead"
		)
	done
fi

SASS="sass -I=$RESOURCEPATH/static/css-sass --sourcemap=none"
if [ -e "/usr/local/bin/sassc" ]; then
	SASS="/usr/local/bin/sassc -I $RESOURCEPATH/static/css-sass"
fi

for scss in $(ls $RESOURCEPATH/static/css-sass); do
	# Ignore files starting with an underscore
	if [ "$(echo $scss | cut -c1)" != "_" ]; then
		# Remove scss extension
		name=$(echo $scss | cut -d. -f1)
		devName="$name.css"
		minName="$name.min.css"
		$SASS -tnested "$RESOURCEPATH/static/css-sass/$scss" "$RESOURCEPATH/static/css/$devName"
		$SASS -tcompressed "$RESOURCEPATH/static/css-sass/$scss" "$RESOURCEPATH/static/css/$minName"
	fi
done
