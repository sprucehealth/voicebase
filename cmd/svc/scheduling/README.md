#Scheduling Service
The scheduling service is responsible for tracking events that are intended to occur on a period time schedule

## Local Development
### Database Setup

In `mysql` execute the following from the `root account`.

```
CREATE SCHEMA scheduling;
CREATE USER 'baymax-scheduling'@'localhost' IDENTIFIED BY 'baymax-scheduling';
GRANT ALL PRIVILEGES ON scheduling.* TO 'baymax-scheduling'@'localhost';
```

Initialize the schema a by applying all .sql files in order in `internal/dal/mysql`

### Running the Service
To start the service run

```
$ $GOPATH/src/github.com/sprucehealth/backend/cmd/svc/scheduling/run-local.sh
```