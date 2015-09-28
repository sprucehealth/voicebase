#!/bin/bash -e

if [ "$NPM" == "" ]; then
	NPM="npm"
fi

$NPM install
$NPM run build
