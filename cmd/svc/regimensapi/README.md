# RegimensAPI Service
## Local Development
* Get Local Dynamo Runnning

```
brew install awscli
docker run -p 7777:7777 tray/dynamodb-local -inMemory -port 7777 -delayTransientStatuses
aws --endpoint-url="http://$(boot2docker ip):7777" --color=on dynamodb list-tables
{
    "TableNames": []
}
```

* Run Service

```
go build && \
REGIMENS_ENV=local \
REGIMENS_AWS_DYNAMODB_ENDPOINT=http://$(boot2docker ip):7777 \
REGIMENS_AWS_DYNAMODB_REGION=us-east-1 \
REGIMENS_AWS_DYNAMODB_DISABLE_SSL=true \
REGIMENS_AUTH_SECRET="something-seekrit" \
REGIMENS_WEB_DOMAIN="http://localhost:8445/" \
REGIMENS_HTTP=:8445 \
./regimensapi
```