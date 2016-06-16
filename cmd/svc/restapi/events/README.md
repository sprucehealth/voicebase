#Event DB Setup

##Local Development

### Install flyway

```
$ brew update
$ brew install flyway
```

### Install Postgres

```
brew install postgres
```

### Initialize the DB

```
initdb /usr/local/var/postgres
```

###Initialize the user

For the password, use `changeme` verbatim (ironically). Otherwise, you'll have to update the password manually in `apps/restapi/dev.conf`.

```
$ createuser -P -s -e events
```

###Create the DB

```
$ psql -h localhost -d postgres -U events
```

That opens the postgres prompt. Then:

```
postgres=# CREATE DATABASE events;
postgres=# \q
```

### Initialize the schema

```
$ flyway -url=jdbc:postgresql://localhost:5432/events -user=events -password=<user password> -locations=filesystem:$GOPATH/src/github.com/sprucehealth/backend/events/postgres -validateOnMigrate=true migrate
```
