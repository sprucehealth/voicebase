#!/bin/bash -e

if [ "$NPM" == "" ]; then
	NPM="npm"
fi

time $NPM install
time $NPM run build