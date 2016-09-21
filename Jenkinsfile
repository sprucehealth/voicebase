#!groovy

// env.NO_INTEGRATION_TESTS = "true"
// sh "docker-ci/jenkins.sh"

node {
	stage 'Setup'

	checkout scm

	// Remove ignored files
	sh 'git clean -X -f'

	// Lowercase version of the job name
	def name = env.BUILD_TAG.toLowerCase()

	//echo 'ssh -i ~/.ssh/id_api $*' > ssh ; chmod +x ssh
	//export GIT_SSH="./ssh"

	// Remove code coverage files (not always deleted by git clean if a directory is removed)
	sh "find . -name cover.out -exec rm '{}' \\;"
	sh "find . -name coverage.xml -exec rm '{}' \\;"

	sh "docker build --rm --force-rm -t ${name} docker-ci"

	// get the uid of the user running the job to be able to properly manage permissions
	def parentUID = sh(script: 'id -u', returnStdout: true).trim()

	// use docker gid to give job access to docker socket
	def parentGID = sh(script: 'getent group docker | cut -d: -f3', returnStdout: true).trim()

	// origin/master -> master
	//def gitBranch = sh(script: "git rev-parse --abbrev-ref HEAD", returnStdout: true).trim()
	def gitBranch = env.BRANCH_NAME
	def gitCommit = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
	echo "${gitBranch}"
	echo "${gitCommit}"

	def memPath = "/mnt/mem/jenkins/" + name
	sh "mkdir -p ${memPath}"
	sh "docker pull 137987605457.dkr.ecr.us-east-1.amazonaws.com/scratch:latest"

	def deploy = false
	if (gitBranch == "master") {
		echo "DEPLOYING"
		deploy = true
	} else {
		echo "NOT DEPLOYING"
	}

	def deployToS3 = ""
	if (deploy) {
		deployToS3 = "true"
	}
	env.FULLCOVERAGE = ""
	env.TEST_S3_BUCKET = ""
	env.NO_INTEGRATION_TESTS = "true"
	env.NO_COVERAGE = "true"

	stage 'Build and Test'

	def workspace = pwd()
	sh """docker run --rm=true --name=${name} \
        -e "BUILD_NUMBER=${env.BUILD_NUMBER}" \
        -e "BUILD_ID=${env.BUILD_ID}" \
        -e "BUILD_URL=${env.BUILD_URL}" \
        -e "BUILD_TAG=${env.BUILD_TAG}" \
        -e "GIT_COMMIT=${gitCommit}" \
        -e "GIT_BRANCH=${gitBranch}" \
        -e "JOB_NAME=${env.JOB_NAME}" \
        -e "DEPLOY_TO_S3=${deployToS3}" \
        -e "FULLCOVERAGE=${env.FULLCOVERAGE}" \
        -e "TEST_S3_BUCKET=${env.TEST_S3_BUCKET}" \
        -e "PARENT_UID=${parentUID}" \
        -e "PARENT_GID=${parentGID}" \
        -e "NO_INTEGRATION_TESTS=${env.NO_INTEGRATION_TESTS}" \
        -e "NO_COVERAGE=${env.NO_COVERAGE}" \
        -v ${memPath}:/mem \
        -v ${workspace}:/workspace/go/src/github.com/sprucehealth/backend \
        -v /var/run/docker.sock:/var/run/docker.sock \
        ${name}"""

	stage 'Deploy'

	if (deploy) {
		sh "./docker-ci/deploy.sh"
	}
}
