#!/bin/bash

cat <<EOF | arc call-conduit differential.createcomment
{
	"revision_id": "$REVISION_ID",
	"action": "reject",
	"message": "Build has FAILED\n\nBuild: ${BUILD_URL}\nConsole: ${BUILD_URL}console"
}
EOF

cat <<EOF | arc call-conduit harbormaster.sendmessage
{
	"buildTargetPHID": "${PHID}",
	"type": "fail"
}
EOF

exit 1
