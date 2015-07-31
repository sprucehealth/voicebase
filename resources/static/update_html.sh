#!/bin/sh

cp ../templates/static/terms.html terms
gzip -9 terms
mv terms.gz terms

cp ../templates/static/informed_consent.html consent
gzip -9 consent
mv consent.gz consent

source ~/.aws.prod
s3cmd --add-header Content-Encoding:gzip -m text/html -P put terms s3://carefront-static/terms
s3cmd --add-header Content-Encoding:gzip -m text/html -P put consent s3://carefront-static/consent

source ~/.aws.dev
s3cmd --add-header Content-Encoding:gzip -m text/html -P put terms s3://dev-carefront-static/terms
s3cmd --add-header Content-Encoding:gzip -m text/html -P put consent s3://dev-carefront-static/consent

s3cmd --add-header Content-Encoding:gzip -m text/html -P put terms s3://staging-carefront-static/terms
s3cmd --add-header Content-Encoding:gzip -m text/html -P put consent s3://staging-carefront-static/consent

s3cmd --add-header Content-Encoding:gzip -m text/html -P put terms s3://demo-carefront-static/terms
s3cmd --add-header Content-Encoding:gzip -m text/html -P put consent s3://demo-carefront-static/consent

rm consent terms
