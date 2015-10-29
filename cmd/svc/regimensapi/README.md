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
go build -i && \
REGIMENS_ENV=local \
REGIMENS_CORS_ALLOW_ALL=true \
REGIMENS_AWS_DYNAMODB_ENDPOINT=http://$(docker-machine ip spruce):7777 \
REGIMENS_AWS_DYNAMODB_REGION=us-east-1 \
REGIMENS_AWS_DYNAMODB_DISABLE_SSL=true \
REGIMENS_AUTH_SECRET="something-seekrit" \
REGIMENS_WEB_DOMAIN="http://web.localhost:8445/" \
REGIMENS_API_DOMAIN="http://localhost:8445/" \
REGIMENS_HTTP=:8445 \
REGIMENS_ANALYTICS_DEBUG=true
./regimensapi
```