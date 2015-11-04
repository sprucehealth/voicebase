#!/bin/bash

cat <<EOF | arc call-conduit differential.createcomment
{
	"revision_id": "$REVISION_ID",
	"action": "resign",
	"message": "Build is green\n\nBuild: ${BUILD_URL}\nConsole: ${BUILD_URL}console"
}
EOF

cat <<EOF | arc call-conduit harbormaster.sendmessage
{
	"buildTargetPHID": "${PHID}",
	"type": "pass"
}
EOF

exit 0
