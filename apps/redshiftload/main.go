package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/sprucehealth/backend/libs/aws"
	_ "github.com/sprucehealth/backend/third_party/github.com/lib/pq"
)

type appConfig struct {
	DropExisting bool
	// AWS
	AWSAccessKey string
	AWSSecretKey string
	AWSToken     string
	// Database
	DBHost    string
	DBPort    int
	DBName    string
	DBUser    string
	DBPass    string
	DBSSLMode string

	awsAuth aws.Auth
}

var config = &appConfig{}

func init() {
	flag.StringVar(&config.DBHost, "db.host", "", "Database hostname")
	flag.IntVar(&config.DBPort, "db.port", 5439, "Database port")
	flag.StringVar(&config.DBName, "db.name", "", "Database name")
	flag.StringVar(&config.DBUser, "db.user", "", "Database username")
	flag.StringVar(&config.DBPass, "db.pass", "", "Database password")
	flag.StringVar(&config.DBSSLMode, "db.sslmode", "require", "disable, require, or verify-full")
	flag.StringVar(&config.AWSAccessKey, "aws.access_key", "", "AWS access key")
	flag.StringVar(&config.AWSSecretKey, "aws.secret_key", "", "AWS secret key")
	flag.StringVar(&config.AWSToken, "aws.token", "", "AWS auth token")
	flag.BoolVar(&config.DropExisting, "dropexisting", false, "Drop and recreate the database table if it already exists")
}

func (c *appConfig) verify() {
	if c.DBHost == "" {
		log.Fatalf("db.host is required")
	}
	if c.DBName == "" {
		log.Fatalf("db.name is required")
	}
	if c.DBSSLMode == "" {
		log.Fatalf("db.sslmode is required")
	}

	if c.AWSAccessKey != "" && c.AWSSecretKey != "" {
		c.awsAuth = aws.Keys{
			AccessKey: c.AWSAccessKey,
			SecretKey: c.AWSSecretKey,
			Token:     c.AWSToken,
		}
	} else if keys := aws.KeysFromEnvironment(); keys.AccessKey != "" {
		c.awsAuth = keys
	} else {
		var err error
		c.awsAuth, err = aws.CredentialsForRole("")
		if err != nil {
			log.Fatal(err)
		}
	}
}

var tableNameRE = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func main() {
	// Allow pulling config values from the environment. Flag names get mangled,
	// for instance db.name becomes REDSHIFT_DB_NAME
	flag.VisitAll(func(f *flag.Flag) {
		name := "REDSHIFT_" + strings.Replace(strings.ToUpper(f.Name), ".", "_", -1)
		if v := os.Getenv(name); v != "" {
			f.Value.Set(v)
		}
	})

	flag.Parse()
	config.verify()

	if len(flag.Args()) < 3 {
		fmt.Fprintf(os.Stderr, "syntax: redshiftload [options] schemafile tablename s3path\n")
		os.Exit(1)
	}

	schemaFile := flag.Arg(0)
	tableName := flag.Arg(1)
	s3Path := flag.Arg(2)

	if !tableNameRE.MatchString(flag.Arg(1)) {
		log.Fatalf("Table name is invalid")
	}

	schemaB, err := ioutil.ReadFile(schemaFile)
	if err != nil {
		log.Fatal(err)
	}
	schema := strings.Replace(string(schemaB), "_TABLE_NAME_", tableName, -1)

	dbArgs := fmt.Sprintf("host=%s port=%d dbname=%s sslmode=%s", config.DBHost, config.DBPort, config.DBName, config.DBSSLMode)
	if config.DBUser != "" {
		dbArgs += " user=" + config.DBUser
	}
	if config.DBPass != "" {
		dbArgs += " password=" + config.DBPass
	}
	db, err := sql.Open("postgres", dbArgs)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Make sure the database connection is working
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %s", err.Error())
	}

	if _, err := db.Exec(schema); err != nil {
		if config.DropExisting && strings.Contains(err.Error(), "already exists") {
			// Must drop the view since it's likely to be referencing the events tables
			dropView(db)
			if _, err := db.Exec("DROP TABLE " + tableName); err != nil {
				log.Fatal(err)
			}
			if _, err := db.Exec(schema); err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatalf("Can't create table: %s", err.Error())
		}
	}

	buildView(db)
	updateGrants(db)

	keys := config.awsAuth.Keys()
	credentials := fmt.Sprintf("aws_access_key_id=%s;aws_secret_access_key=%s", keys.AccessKey, keys.SecretKey)
	if keys.Token != "" {
		credentials += ";token=" + keys.Token
	}

	// TRUNCATECOLUMNS - truncate values that don't fit in a given varchar
	// TRIMBLANKS - trim spaces from end of varchar fields
	if _, err := db.Exec(fmt.Sprintf(`
		COPY %s FROM '%s'
		CREDENTIALS '%s' JSON AS 'auto'
		TRUNCATECOLUMNS
		TRIMBLANKS`,
		tableName, s3Path, credentials),
	); err != nil {
		log.Fatalf("Can't load data: %s", err.Error())
	}
}

func eventTables(db *sql.DB) []string {
	// TODO: shouldn't hard code the name 'client_event'
	rows, err := db.Query(`SELECT DISTINCT "tablename" FROM "pg_table_def" WHERE "tablename" LIKE 'client_event_%'`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Fatal(err)
		}
		tables = append(tables, name)
	}
	return tables
}

func dropView(db *sql.DB) {
	// TODO: shouldn't hard code the name 'client_event'
	_, err := db.Exec(`DROP VIEW "client_event"`)
	// Unfortunately RedShift doesn't support "IF EXISTS"
	if err != nil && !strings.Contains(err.Error(), "does not exist") {
		log.Fatalf("Can't drop old view: %s", err.Error())
	}
}

func buildView(db *sql.DB) {
	tables := eventTables(db)
	for i, t := range tables {
		tables[i] = `SELECT * FROM "` + t + `"`
	}
	if _, err := db.Exec(`CREATE OR REPLACE VIEW "client_event" AS ` + strings.Join(tables, " UNION ")); err != nil {
		log.Fatalf("Can't create view: %s", err.Error())
	}
}

func updateGrants(db *sql.DB) {
	tables := eventTables(db)
	if _, err := db.Exec(`GRANT SELECT ON ` + strings.Join(tables, ", ") + `, client_event TO GROUP readonly`); err != nil {
		log.Fatalf("Failed to update grants: %s", err.Error())
	}
}
