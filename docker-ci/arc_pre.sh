#!/bin/bash

cat <<EOF | arc call-conduit differential.createcomment
{
	"revision_id": "$REVISION_ID",
	"message": "Build started: ${BUILD_URL} (console: ${BUILD_URL}console)"
}
EOF

cat <<EOF | arc call-conduit harbormaster.createartifact
{
	"buildTargetPHID": "${PHID}",
	"artifactKey": "Jenkins",
	"artifactType": "uri",
	"artifactData": {
		"uri": "${BUILD_URL}",
		"name": "View External Build Results",
		"ui.external": true
	}
}
EOF
