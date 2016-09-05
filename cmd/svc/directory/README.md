#Directory Service
The directory service is responsible for managing addressable "entities" or nodes in the system. Entities include providers, patients, organizations, individuals addressable by email/sms, and system objects (like Spruce Support). It is a flat structure where each node can have members and memberships. 

## Local Development
### Database Setup
If you don't already have it setup install `flyway`

```
$ brew update
$ brew install flyway
```

In `mysql` execute the following from the `root account`.

```
CREATE SCHEMA directory;
CREATE USER 'baymax-directory'@'localhost' IDENTIFIED BY 'baymax-directory';
GRANT ALL PRIVILEGES ON directory.* TO 'baymax-directory'@'localhost';
```

Initialize the schema

```
$ flyway -url=jdbc:mysql://localhost:3306/directory -user=baymax-directory -password=baymax-directory -locations=filesystem:$GOPATH/src/github.com/sprucehealth/backend/cmd/svc/directory/internal/dal/mysql -validateOnMigrate=true migrate
```

To update the schema you would run the previous command again as new files are added to `dal/mysql`

### Running the Service
To start the service run

```
$ go run $GOPATH/src/github.com/sprucehealth/backend/cmd/svc/directory/main.go -debug=true -rpc_listen_port=50052
```

./directory_svc -debug=true \
-rpc_listen_port=50052 \
-db_host=dev-spruceapi.ckwporuc939i.us-east-1.rds.amazonaws.com \
-db_user=baymax \
-db_password=i3nun2beL9edD0 \
-db_name=directory \
-db_max_open_connections=10 \
-db_max_idle_connections=5