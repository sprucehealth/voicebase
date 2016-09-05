# Settings Service
The settings service is responsible for managing settings for addressable nodes in the system. New settings configurations are registered by different services
and the service validates all settings that are stored and served.

## Local Development
* Get Local Dynamo Runnning

```
brew install awscli
docker run -p 7777:7777 tray/dynamodb-local -inMemory -port 7777 -delayTransientStatuses
aws --endpoint-url="http://$(docker-machine ip spruce):7777" --color=on dynamodb list-tables
{
    "TableNames": []
}
```

* Create the Setting and SettingConfig tables

- Setting
```
aws --endpoint-url="http://$(docker-machine ip spruce):7777" --color=on dynamodb  create-table --table-name Setting --attribute-definitions AttributeName=nodeID,AttributeType=S AttributeName=key,AttributeType=S --key-schema AttributeName=nodeID,KeyType=HASH AttributeName=key,KeyType=RANGE --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5
```

- SettingConfig
```
aws --endpoint-url="http://$(docker-machine ip spruce):7777" --color=on dynamodb  create-table --table-name SettingConfig --attribute-definitions AttributeName=key,AttributeType=S --key-schema AttributeName=key,KeyType=HASH --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5
```

* Run Service

```
go build -i && \
SETTINGS_ENV=local \
SETTINGS_LOCAL_DYNAMODB_ENDPOINT=http://$(docker-machine ip spruce):7777 \
SETTINGS_DYNAMODB_TABLE_NAME_SETTINGS="Setting" \
SETTINGS_DYNAMODB_TABLE_NAME_SETTING_CONFIGS="SettingConfig" \
SETTINGS_AWS_REGION="us-east-1" \
SETTINGS_AWS_SECRET_KEY="something-seekirt" \
SETTINGS_AWS_ACCESS_KEY="<AWS_ACCESS_KEY>" \
SETTINGS_DEBUG=true \
./settings
```