#Notification Service
## Local Development
### Database Setup
If you don't already have it setup install `flyway`

```
$ brew update
$ brew install flyway
```

In `mysql` execute the following from the `root account`.

```
CREATE SCHEMA notification;
CREATE USER 'baymax-notif'@'localhost' IDENTIFIED BY 'baymax-notif';
GRANT ALL PRIVILEGES ON notification.* TO 'baymax-notif'@'localhost';
```

Initialize the schema

```
$ flyway -url=jdbc:mysql://localhost:3306/notification -user=baymax-notif -password=baymax-notif -locations=filesystem:$GOPATH/src/github.com/sprucehealth/backend/cmd/svc/notification/internal/dal/mysql -validateOnMigrate=true migrate
```

To update the schema you would run the previous command again as new files are added to `dal/mysql`

### Running the Service
To start the service run

```
$ go run main.go  -debug=true \
-sqs_device_registration_url=https://sqs.us-east-1.amazonaws.com/758505115169/dev-baymax_notification_device_registration \
-sqs_notification_url=https://sqs.us-east-1.amazonaws.com/758505115169/dev-baymax_notification \
-sns_apple_device_registration_arn=arn:aws:sns:us-east-1:758505115169:app/APNS_SANDBOX/dev-baymax_apple_push_notification \
-sns_android_device_registration_arn=arn:aws:sns:us-east-1:758505115169:app/GCM/dev-baymax_apple_push_notification \
-directory_addr=:50052
-aws_access_key=$AWS_ACCESS_KEY \
-aws_secret_key=$AWS_SECRET_KEY
```