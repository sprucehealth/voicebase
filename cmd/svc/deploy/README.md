#Deploy Service
## Local Development
### Database Setup
If you don't already have it setup install `flyway`

```
$ brew update
$ brew install flyway
```

In `mysql` execute the following from the `root account`.

```
CREATE SCHEMA deploy;
CREATE USER 'deploy'@'localhost' IDENTIFIED BY 'deploy';
GRANT ALL PRIVILEGES ON deploy.* TO 'deploy'@'localhost';
```

Initialize the schema

```
$ flyway -url=jdbc:mysql://localhost:3306/deploy -user=deploy -password=deploy -locations=filesystem:$GOPATH/src/github.com/sprucehealth/backend/cmd/svc/deploy/internal/dal/mysql -validateOnMigrate=true migrate
```

To update the schema you would run the previous command again as new files are added to `dal/mysql`

### Running the Service
To start the service run

```
$ go run $GOPATH/src/github.com/sprucehealth/backend/cmd/svc/deploy/main.go -debug=true -rpc_listen_port=51050
```