#!/bin/bash -e

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

rm -rf resources/static/js
mkdir resources/static/js
rm -rf resources/static/css
mkdir resources/static/css

if [ "$NPM" == "" ]; then
	NPM="npm"
fi

time $NPM install
time $NPM run build
time $NPM run build-dev

RESOURCEPATH="$(pwd)/resources"

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
