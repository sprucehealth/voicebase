# Settings Service
The settings service is responsible for managing settings for addressable nodes in the system. New settings configurations are registered by different services
and the service validates all settings that are stored and served.

## Local Development
* Get Local Dynamo Runnning

```
brew install awscli
docker run -p 7777:7777 --deatch tray/dynamodb-local -inMemory -port 7777 -delayTransientStatuses
aws --endpoint-url="http://localhost:7777" --color=on dynamodb list-tables
{
    "TableNames": []
}
```

* Apply terraform changes to the local DynmoDB instance

```
cd schema/local
terraform get
terraform plan
terraform apply
```

* Ensure that the tables were created on the local dynamodb instance

```
aws --endpoint-url="http://localhost:7777" --color=on dynamodb list-tables
```
_At this point you should see the tables created that are present in the terraform files_

* Run Service

```
go build -i -v
./settings -env=local
```