#!/bin/sh

APPS="admin dronboard home"
RESOURCEPATH="$(cd "$(dirname "$0")/.."; pwd)/resources"

for APP in $APPS; do
	browserify -w -e $RESOURCEPATH/static/jsx/$APP/app.js -t reactify -o $RESOURCEPATH/static/js/$APP.dev.js -d
	browserify -w -e $RESOURCEPATH/static/jsx/$APP/app.js -t reactify -t uglifyify -o $RESOURCEPATH/static/js/$APP.min.js
done
