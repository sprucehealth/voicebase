#Payments Service
## Local Development
### Database Setup

In `mysql` execute the following from the `root account`.

```
CREATE SCHEMA payments;
CREATE USER 'baymax-payments'@'localhost' IDENTIFIED BY 'baymax-payments';
GRANT ALL PRIVILEGES ON payments.* TO 'baymax-payments'@'localhost';
```

Initialize the schema a by applying all .sql files in order in `internal/dal/mysql`

### Running the Service
To start the service run

```
$ go run $GOPATH/src/github.com/sprucehealth/backend/cmd/svc/payments/main.go -debug=true -env=debug
```