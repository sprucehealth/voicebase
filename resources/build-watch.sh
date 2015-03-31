#!/bin/bash

RESOURCEPATH="$(cd "$(dirname "$0")/.."; pwd)/resources"
SASS="sass -I=$RESOURCEPATH/static/css-sass --sourcemap=none"
$SASS --watch $RESOURCEPATH/static/css-sass:$RESOURCEPATH/static/css
