#!/bin/bash

APPS="admin dashboard dronboard home"
RESOURCEPATH="$(cd "$(dirname "$0")/.."; pwd)/resources"

if [ "$1" != "" ]; then
	APPS="$1"
fi

if [ ! "$APPS" == "css" ]; then
	for APP in $APPS; do
		browserify -w -e $RESOURCEPATH/static/jsx/$APP/app.js -t reactify -o $RESOURCEPATH/static/js/$APP.dev.js -d
		browserify -w -e $RESOURCEPATH/static/jsx/$APP/app.js -t reactify -t uglifyify -o $RESOURCEPATH/static/js/$APP.min.js
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
