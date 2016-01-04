#Auth Service
## Local Development
### Database Setup
If you don't already have it setup install `flyway`

```
$ brew update
$ brew install flyway
```

In `mysql` execute the following from the `root account`.

```
CREATE SCHEMA auth;
CREATE USER 'baymax-auth'@'localhost' IDENTIFIED BY 'baymax-auth';
GRANT ALL PRIVILEGES ON auth.* TO 'baymax-auth'@'localhost';
```

Initialize the schema

```
$ flyway -url=jdbc:mysql://localhost:3306/auth -user=baymax-auth -password=baymax-auth -locations=filesystem:$GOPATH/src/github.com/sprucehealth/backend/cmd/svc/auth/internal/dal/mysql -validateOnMigrate=true migrate
```

To update the schema you would run the previous command again as new files are added to `dal/mysql`

### Running the Service
To start the service run

```
$ go run $GOPATH/src/github.com/sprucehealth/backend/cmd/svc/auth/main.go -debug=true -rpc.listen.port=50051
```