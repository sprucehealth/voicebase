Backend Monorepo
================
[![Build Status](http://dev-ci-1503083087.us-east-1.elb.amazonaws.com/job/backend/badge/icon)](http://dev-ci-1.node.dev-us-east-1.spruce:8080/job/backend/)

Setting up your environment & running the `gotour`
---------------------------------

	$ brew update
	$ brew doctor # ensure there are no issues with Brew or your system
	$ brew install go
	$ brew install mercurial
	$ export GOPATH=$HOME/go
	$ go get golang.org/x/tour/gotour
	$ $HOME/go/bin/gotour # runs the gotour executable and opens it in a browser window

Building the app
---------------------------------

	# checks out to $GOPATH/src/github.com/sprucehealth/backend/
	$ go get github.com/sprucehealth/backend
	
Note: `go get` uses [HTTPS by default](http://golang.org/doc/faq#git_https)
([how to use SSH by default](http://michaelheap.com/golang-how-to-go-get-private-repos/)).

One of the great things about Go is getting external packages is as simple as
the above command. You're encouraged to create your package under the path
that you'd upload it if you wanted to open source your project. So any
personal package you wrote you could create under github.com/GITHUB_USERNAME.
Someone else may have the exact same package name but its under their own
unique path.

Next, `cd` to the backend app's directory and `go build` it:

	$ cd /Users/YOU/go/src/github.com/sprucehealth/backend/apps/restapi
	$ go build
	# binary should be at /Users/YOU/go/src/github.com/sprucehealth/backend/apps/restapi/restapi

_Having issues? See the [troubleshooting](#troubleshooting) section._

Getting environment setup
---------------------------------

Set up the AWS keys as environment variables by adding the following to `~/.bashrc` or `~/.zshrc`:

	export GOPATH=$HOME/go
	export AWS_ACCESS_KEY='ASK_KUNAL_OR_SAM_FOR_ME'
	export AWS_SECRET_KEY='ASK_KUNAL_OR_SAM_FOR_ME'

Then:

	$ source ~/.bashrc # or source ~/.zshrc

Add the following lines to `/etc/hosts`. Reason you need this is because we
currently use the same binary for the restapi as well as the website, and
requests are routed based on the incoming URI

	127.0.0.1       www.spruce.loc
	127.0.0.1       api.spruce.loc

Local database setup (automatic method)
---------------------------------

1. `cd mysql`
2. Find number of the latest migration file (ex: if `migration-395.sql` is the
   highest-numbered file, then `395` is your number)
3. ./bootstrap_mysql.sh carefront_db carefront changethis <latest-migration-id>


Local database setup (manual method)
---------------------------------

Before running the backend server locally, we want to get a local instance of
mysql running, and setup with the database schema and boostrapped data.

Install MySQL and get it running:

	$ brew install mysql
	$ mysql.server start

Setup the expected user for the restapi (user="carefront" password="changethis"):

	$ mysql -u root
	mysql> CREATE USER 'carefront'@'localhost' IDENTIFIED BY 'changethis';
	mysql> CREATE DATABASE carefront_db;
	mysql> GRANT ALL on *.* to 'carefront'@localhost;

Ensure that you have access to your local mysql instance and the `carefront_db`
as "carefront". *First ensure to log out of your session by typing exit*

	$ mysql -u carefront -pchangethis
	$ use carefront_db;

Now that you have mysql up and running, lets populate the database just created
with the schema and boostrapped data. Anytime we have to update the schema we
create a migration filed under the mysql directory in the form migration-X.sql
where X represents the migration number. A validation script loads a database
with the current schema, runs the migration on this database, and then spits
out snapshot-X.sql and data-snapshot-X.sql files that represent the database
schema and boostrapped data respectively.

Open a new terminal tab and `cd $GOPATH/src/github.com/sprucehealth/backend/mysql`:

	$ echo "use carefront_db;" | cat - snapshot-<latest_migration_id>.sql > temp.sql
	$ echo "use carefront_db;" | cat - data-snapshot-<latest_migration_id>.sql > data_temp.sql

Now seed the database:

	mysql -u carefront -pchangethis < temp.sql
	mysql -u carefront -pchangethis < data_temp.sql

Go back to your mysql session tab. Log the latest migration id in the
migrations table to indicate to the application the last migration that
was completed:

	mysql> INSERT INTO migrations (migration_id, migration_user) VALUES (<latest_migration_id>, "carefront");


Running the server locally
---------------------------------

Let's try running the server locally.

`cd` to the restapi folder under apps:

	cd $GOPATH/src/github.com/sprucehealth/backend/apps/restapi


Build the app and execute the run_server.bash script which tells the
application where to get the config file for the local config from:

	go build
	./run_server.bash


_Having issues? See the [troubleshooting](#troubleshooting) section._

Clone the dev db:
-----------------

	$ mysqldump -h dev-db-2b.ckwporuc939i.us-east-1.rds.amazonaws.com -u carefront -p carefront_db > devdb.sql

This will prompt for a password -- get the password from Meldium or ask
someone on the backend team. It'll also take a few minutes to download
the data dump.

	$ mysql -u carefront -pchangethis carefront_db < ./devdb.sql

Setting up an admin user (for `http://www.spruce.loc:8443/admin/`)
------------------------------------------------------------------

_NOTE: obviously, you don't need to do this if you cloned dev your user is
       already already set up as an admin user on dev_

Creating an admin account. The reason we need to create an admin account
is because there are operational tasks we have to carry out to upload the
patient visit intake and doctor review layouts, and only an admin user can
do that. Currently, the easiest way to create an admin account is to create
a _patient account_ and then modify its role type to be that of an admin user.

But first make sure to build and start running the app:

	$ go build
	$ ./run_server.bash

> Open the [PAW file](https://github.com/SpruceHealth/api-response-examples/tree/master/v1) in [PAW (Mac App Store)](https://itunes.apple.com/us/app/paw-http-client/id584653203?mt=12) and create a new patient (ex: `jon@sprucehealth.com`):
<img src="http://f.cl.ly/items/221c0k392Z3n2R3O3Z0z/Screen%20Shot%202014-11-26%20at%201.17.28%20PM.png" />

Log back in to mysql as `carefront` and change the account's role type to:

	$ mysql -u carefront -pchangethis;
	mysql> USE carefront_db;
	mysql> UPDATE account SET role_type_id=(SELECT id FROM role_type WHERE role_type_tag='ADMIN') WHERE email='<admin_email>';


Open the PAW file again:

* In the `Layout upload (Initial Visit)`, upload the latest versions of
  `intake-*`, `review-*`, and `diagnose-*` json files. You'll have to locate
  them on disk in `./info_intake/`.

> <img src="http://f.cl.ly/items/2E3X1k0X1X2l0y1I3O2g/Screen%20Shot%202014-11-26%20at%203.14.09%20PM.png" />

* In the `Layout upload (Initial Visit)`, upload the `followup-intake-*` and
  `followup-review-*` json files. You'll have to locate them on disk in
  `./info_intake/`.

> <img src="http://f.cl.ly/items/021Z1q3h1u3Z3i0O3I1A/Screen%20Shot%202014-11-26%20at%203.14.11%20PM.png" />

Create a cost entry for the initial and the followup visits so that there
exists a cost type to query against for the patient app when attempting to
determine the cost of each of the visits:

	INSERT INTO item_cost (sku_id, status) VALUES ((SELECT id FROM sku WHERE type = 'acne_visit'), 'ACTIVE');
	INSERT INTO line_item (currency, description, amount, item_cost_id) VALUES ('USD', 'Acne visit', 4000, 1);
	INSERT INTO item_cost (sku_id, status) VALUES ((SELECT id FROM sku WHERE type = 'acne_followup'), 'ACTIVE');
	INSERT INTO line_item (currency, description, amount, item_cost_id) VALUES ('USD', 'Acne Followup', 2000, 2);


Make yourself a boss:

	INSERT INTO carefront_db.account_group_member (group_id, account_id)
	VALUES (
		(SELECT id FROM carefront_db.account_group WHERE name = 'superuser'),
		(SELECT id FROM carefront_db.account WHERE email = '<account_email>'));

Building to run the website(s)
------------------------------

Install dependencies:

	brew install npm sassc

1. `cd` to `resources`
2. `$ ./build.sh`
3. `cd` to `apps/restapi`
4. `./run_server.bash`

For development on on JS app it's possible to start a watcher that will rebuild
the bundle when any file changes.

1. `$ cd resources/apps/$APP`
2. `$ npm install`
3. `$ npm start`

If you find that you need more (perhaps more have been added since this
writing), look for the `.travis.yml` file for dependencies that our CI
server needs, and try installing those.

> The public-facing website will be at: https://www.spruce.loc:8443/

> The admin website will be at: https://www.spruce.loc:8443/admin/

Events setup (optional)
---------------------------------

### Database setup

First, follow the [Events Setup README](https://github.com/SpruceHealth/backend/blob/master/events/README.md).

To have launchd start postgresql at login:

`    ln -sfv /usr/local/opt/postgresql/*.plist ~/Library/LaunchAgents`

Then to load postgresql now:

`    launchctl load ~/Library/LaunchAgents/homebrew.mxcl.postgresql.plist`

Or, if you don't want/need launchctl, you can just run:

`    postgres -D /usr/local/var/postgres`

Add the following to `dev.conf`:

```
[events_database]		
User = "events"		
Name = "events"		
Password = "changeme"		
Port = 5432		
Host = "localhost"	
```

### Testing it

Run the app locally and do something that would trigger an event, like creating a visit (you don't have to submit).

Then open the postgres prompt:

```
psql -h localhost -U events
```

Inside postgres, try one of the following queries:

```
select * from web_request_events;
```
or
```
select * from server_event order by timestamp desc;
```

You should see some events, for example:
```
            name             | timestamp  | session_id | account_id | patient_id | doctor_id | visit_id | case_id | treatment_plan_id | role | extra_json 
-----------------------------+------------+------------+------------+------------+-----------+----------+---------+-------------------+------+------------
 visit_started               | 2015-04-22 |            |            |       4160 |           |     3803 |    3956 |                   |      | 
 visit_pre_submission_triage | 2015-04-11 |            |            |            |           |     3797 |    3950 |                   |      | 
 visit_started               | 2015-04-11 |            |            |       4158 |           |     3798 |    3951 |                   |      | 
 visit_started               | 2015-04-10 |            |            |       4158 |           |     3797 |    3950 |                   |      | 

```

Running integration tests locally
---------------------------------

Add the following to `~/.bashrc` or `~/.zshrc`:

	export CF_LOCAL_DB_USERNAME='carefront'
	export CF_LOCAL_DB_PASSWORD='changethis'
	export CF_LOCAL_DB_INSTANCE='127.0.0.1'
	export DOSESPOT_USER_ID=407

Then:

	$ source ~/.bashrc # or source ~/.zshrc

To run the tests serially:

	$ cd ./test/test_integration
	$ go test -v ./...

To run the tests in parallel:

	$ cd ./test/test_integration
	$ go test -v -parallel 4 ./...


Troubleshooting
---------------

### Issues during `go build`:

Error:

	github.com/sprucehealth/backend/app_url
../../app_url/action.go:9: import /Users/jonsibley/go/pkg/darwin_amd64/github.com/sprucehealth/backend/libs/golog.a: object is [darwin amd64 go1.3.3 X:precisestack] expected [darwin amd64 go1.4.1 X:precisestack]

Solution:

	go clean -r -i
	go install -a
	go build

### Issues while attempting to run the app:

Error:

	dial tcp 127.0.0.1:3306: connection refused

Solution:

	mysql.server start # note: there are ways to automatically start mysql when your machine starts, too

Error:

	Error 1045: Access denied for user 'carefront'@'localhost' (using password: YES)

Solution:

You need to set up the `carefront` user with access to the database `carefront_db` (as described above).
