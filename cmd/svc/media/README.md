#Media Service
## Local Development
### Database Setup
If you don't already have it setup install `flyway`

```
$ brew update
$ brew install flyway
```

In `mysql` execute the following from the `root account`.

```
CREATE SCHEMA media;
CREATE USER 'baymax-media'@'localhost' IDENTIFIED BY 'baymax-media';
GRANT ALL PRIVILEGES ON media.* TO 'baymax-media'@'localhost';
```

Initialize the schema

```
$ flyway -url=jdbc:mysql://localhost:3306/media -user=baymax-media -password=baymax-media -locations=filesystem:$GOPATH/src/github.com/sprucehealth/backend/cmd/svc/media/internal/dal/mysql -validateOnMigrate=true migrate
```

To update the schema you would run the previous command again as new files are added to `dal/mysql`

### Running the Service
To start the service run

```
$ go run $GOPATH/src/github.com/sprucehealth/backend/cmd/svc/media/main.go -debug=true -rpc_listen_port=50059
```