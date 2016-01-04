#Directory Service
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
$ go run $GOPATH/src/github.com/sprucehealth/backend/cmd/svc/directory/main.go -debug=true -rpc.listen.port=50052
```