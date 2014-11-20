package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	_ "github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/go-sql-driver/mysql"
	_ "github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/lib/pq"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-librato/librato"
	"github.com/sprucehealth/backend/libs/aws"
	"github.com/sprucehealth/backend/libs/aws/s3"
	"github.com/sprucehealth/backend/libs/golog"
)

type appConfig struct {
	DropExisting   bool
	Month          bool
	TransformS3URL string
	Verbose        bool
	// AWS
	AWSAccessKey string
	AWSSecretKey string
	AWSToken     string
	// Analytics Database
	DBHost    string
	DBPort    int
	DBName    string
	DBUser    string
	DBPass    string
	DBSSLMode string
	// MySQL
	MySQLHost     string
	MySQLPort     int
	MySQLDatabase string
	MySQLUser     string
	MySQLPass     string
	// Librato
	LibratoUsername string
	LibratoToken    string
	LibratoSource   string

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
	flag.StringVar(&config.MySQLHost, "mysql.host", "", "Database hostname")
	flag.IntVar(&config.MySQLPort, "mysql.port", 5439, "MySQL port")
	flag.StringVar(&config.MySQLDatabase, "mysql.name", "", "MySQL name")
	flag.StringVar(&config.MySQLUser, "mysql.user", "", "MySQL username")
	flag.StringVar(&config.MySQLPass, "mysql.pass", "", "MySQL password")
	flag.StringVar(&config.TransformS3URL, "transform.s3url", "", "S3 url for dumped MySQL tables (e.g. s3://bucket/prefix)")
	flag.StringVar(&config.AWSAccessKey, "aws.access_key", "", "AWS access key")
	flag.StringVar(&config.AWSSecretKey, "aws.secret_key", "", "AWS secret key")
	flag.StringVar(&config.AWSToken, "aws.token", "", "AWS auth token")
	flag.StringVar(&config.LibratoUsername, "librato.username", "", "Librato username")
	flag.StringVar(&config.LibratoToken, "librato.token", "", "Librato token")
	flag.StringVar(&config.LibratoSource, "librato.source", "", "Librato source")
	flag.BoolVar(&config.DropExisting, "dropexisting", false, "Drop and recreate the database table if it already exists")
	flag.BoolVar(&config.Month, "month", false, "Import a month instead of a day")
	flag.BoolVar(&config.Verbose, "v", false, "Verbose output")
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

var categoryRE = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func main() {
	log.SetFlags(log.Lshortfile)

	// Allow pulling config values from the environment. Flag names get mangled,
	// for instance db.name becomes DB_NAME
	flag.VisitAll(func(f *flag.Flag) {
		name := strings.Replace(strings.ToUpper(f.Name), ".", "_", -1)
		if v := os.Getenv(name); v != "" {
			f.Value.Set(v)
		}
	})

	flag.Parse()
	config.verify()

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

	if len(flag.Args()) < 1 || (flag.Arg(0) != "events" && flag.Arg(0) != "transform") {
		fmt.Fprintf(os.Stderr, "syntax: redshiftload [options] events|transform ...\n")
		os.Exit(1)
	}

	if config.Verbose {
		golog.Default().SetLevel(golog.DEBUG)
	} else {
		golog.Default().SetLevel(golog.WARN)
	}

	var lib *librato.Client
	if config.LibratoToken != "" && config.LibratoUsername != "" {
		lib = &librato.Client{
			Username: config.LibratoUsername,
			Token:    config.LibratoToken,
		}
		if config.LibratoSource == "" {
			config.LibratoSource, err = os.Hostname()
			if err != nil {
				golog.Errorf("Failed to get hostname for librato source: %s", err.Error())
			}
		}
	}

	if flag.Arg(0) == "transform" {
		if len(flag.Args()) < 2 {
			fmt.Fprintf(os.Stderr, "syntax: redshiftload [options] transform <tables.json>\n")
			os.Exit(1)
		}

		var tables []*table
		f, err := os.Open(flag.Arg(1))
		if err != nil {
			log.Fatalf("Failed to open tables.json file: %s", err.Error())
		}
		if json.NewDecoder(f).Decode(&tables); err != nil {
			log.Fatalf("Failed to decode tables: %s", err.Error())
		}
		f.Close()

		mdb, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci",
			config.MySQLUser, config.MySQLPass, config.MySQLHost, config.MySQLPort, config.MySQLDatabase))
		if err != nil {
			log.Fatalf("Failed to connect to MySQL: %s", err.Error())
		}
		defer mdb.Close()
		// test the connection to the database by running a ping against it
		if err := mdb.Ping(); err != nil {
			log.Fatalf("Failed to ping MySQL: %s", err.Error())
		}

		s3c := &s3.S3{
			Region: aws.USEast,
			Client: &aws.Client{
				Auth: config.awsAuth,
			},
		}

		u, err := url.Parse(config.TransformS3URL)
		if err != nil {
			log.Fatalf("Failed to parse transform.s3url")
		}
		bucket := u.Host
		prefix := u.Path
		if bucket == "" {
			log.Fatalf("transform.s3url missing host (bucket)")
		}
		if len(prefix) > 0 && prefix[0] == '/' {
			prefix = prefix[1:]
		}

		err = libratoTimer(lib, config.LibratoSource, "redshift.etl", func() error {
			return transform(mdb, db, tables, s3c, bucket, prefix, lib, config.LibratoSource)
		})
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if len(flag.Args()) < 5 {
		fmt.Fprintf(os.Stderr, "syntax: redshiftload [options] events <schemafile> <category> <s3path> <date>\n")
		os.Exit(1)
	}

	schemaFile := flag.Arg(1)
	category := flag.Arg(2)
	s3Path := flag.Arg(3)
	date, err := time.Parse("2006-01-02", flag.Arg(4))
	if err != nil {
		log.Fatalf("Failed to parse date: %s", err.Error())
	}

	if !categoryRE.MatchString(category) {
		log.Fatalf("Table name is invalid")
	}

	err = libratoTimer(lib, config.LibratoSource, "redshift.loadevents", func() error {
		return loadEvents(db, schemaFile, category, s3Path, date)
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}

func loadEvents(db *sql.DB, schemaFile, category, s3Path string, date time.Time) error {
	baseName := category + "_event"

	var tableName string
	if config.Month {
		tableName = baseName + "_" + date.Format("200601")
		if s3Path[len(s3Path)-1] == '/' {
			s3Path = s3Path[:len(s3Path)-1]
		}
		s3Path = fmt.Sprintf("%s/%s/%s", s3Path, category, date.Format("2006/01"))
	} else {
		tableName = baseName + "_" + date.Format("20060102")
		if s3Path[len(s3Path)-1] == '/' {
			s3Path = s3Path[:len(s3Path)-1]
		}
		s3Path = fmt.Sprintf("%s/%s/%s", s3Path, category, date.Format("2006/01/02"))
	}

	golog.Debugf("baseName=%s tableName=%s s3Path=%s\n", baseName, tableName, s3Path)

	schemaB, err := ioutil.ReadFile(schemaFile)
	if err != nil {
		return err
	}
	schema := strings.Replace(string(schemaB), "_TABLE_NAME_", tableName, -1)

	golog.Debugf("Creating schema...")
	if _, err := db.Exec(schema); err != nil {
		if config.DropExisting && strings.Contains(err.Error(), "already exists") {
			// Must drop the view since it's likely to be referencing the events tables
			if err := dropView(db, baseName); err != nil {
				return err
			}
			if _, err := db.Exec("DROP TABLE " + tableName); err != nil {
				return err
			}
			if _, err := db.Exec(schema); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("Can't create table: %s", err.Error())
		}
	}

	golog.Debugf("Building view...")
	if err := buildView(db, baseName); err != nil {
		return err
	}
	golog.Debugf("Updating grants...")
	if err := updateGrants(db, baseName); err != nil {
		return err
	}

	keys := config.awsAuth.Keys()
	credentials := fmt.Sprintf("aws_access_key_id=%s;aws_secret_access_key=%s", keys.AccessKey, keys.SecretKey)
	if keys.Token != "" {
		credentials += ";token=" + keys.Token
	}

	golog.Debugf("Importing data...")
	// TRUNCATECOLUMNS - truncate values that don't fit in a given varchar
	// TRIMBLANKS - trim spaces from end of varchar fields
	if _, err := db.Exec(fmt.Sprintf(`
		COPY %s FROM '%s'
		CREDENTIALS '%s' JSON AS 'auto'
		TRUNCATECOLUMNS
		TRIMBLANKS`,
		tableName, s3Path, credentials),
	); err != nil {
		return fmt.Errorf("Can't load data: %s", err.Error())
	}

	return nil
}

func eventTables(db *sql.DB, baseName string) ([]string, error) {
	rows, err := db.Query(`SELECT DISTINCT "tablename" FROM "pg_table_def" WHERE "tablename" LIKE $1`, baseName+"_%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, nil
}

func dropView(db *sql.DB, baseName string) error {
	_, err := db.Exec(`DROP VIEW "` + baseName + `"`)
	// Unfortunately RedShift doesn't support "IF EXISTS"
	if err != nil && !strings.Contains(err.Error(), "does not exist") {
		return fmt.Errorf("Can't drop old view: %s", err.Error())
	}
	return nil
}

func buildView(db *sql.DB, baseName string) error {
	tables, err := eventTables(db, baseName)
	if err != nil {
		return err
	}
	for i, t := range tables {
		tables[i] = `SELECT * FROM "` + t + `"`
	}
	if _, err := db.Exec(`CREATE OR REPLACE VIEW "` + baseName + `" AS ` + strings.Join(tables, " UNION ")); err != nil {
		return fmt.Errorf("Can't create view: %s", err.Error())
	}
	return nil
}

func updateGrants(db *sql.DB, baseName string) error {
	tables, err := eventTables(db, baseName)
	if err != nil {
		return err
	}
	if _, err := db.Exec(`GRANT SELECT ON ` + strings.Join(tables, ", ") + `, ` + baseName + ` TO GROUP readonly`); err != nil {
		return fmt.Errorf("Failed to update grants: %s", err.Error())
	}
	return nil
}

func libratoTimer(lib *librato.Client, source, prefix string, f func() error) error {
	if lib == nil {
		return f()
	}

	startTime := time.Now()
	err := f()
	dt := time.Since(startTime).Seconds()

	metrics := &librato.Metrics{}
	if err != nil {
		metrics.Gauges = append(metrics.Gauges,
			&librato.Metric{
				Name:   prefix + ".failure",
				Source: source,
				Value:  1,
			},
			&librato.Metric{
				Name:   prefix + ".success",
				Source: source,
				Value:  0,
			},
		)
	} else {
		metrics.Gauges = append(metrics.Gauges,
			&librato.Metric{
				Name:   prefix + ".time",
				Source: source,
				Value:  dt,
			},
			&librato.Metric{
				Name:   prefix + ".success",
				Source: source,
				Value:  1,
			},
			&librato.Metric{
				Name:   prefix + ".failure",
				Source: source,
				Value:  0,
			},
		)
	}
	if err := lib.PostMetrics(metrics); err != nil {
		golog.Errorf("Failed to post librato metrics: %s", err.Error())
	}
	return err
}
