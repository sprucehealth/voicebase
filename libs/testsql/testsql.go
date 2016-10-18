package testsql

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/test"
)

type DB struct {
	DB   *sql.DB
	name string
}

// Setup creates a new temporary database and applies all schemas matching the provided glob ordered by name.
func Setup(t *testing.T, migrationGlob string) *DB {
	user := os.Getenv("TEST_DB_USER")
	if user == "" {
		t.Skip("Missing TEST_DB_USER")
	}
	password := os.Getenv("TEST_DB_PASSWORD")
	host := os.Getenv("TEST_DB_HOST")
	if host == "" {
		host = "localhost"
	}

	migrations, err := filepath.Glob(migrationGlob)
	test.OK(t, err)
	sort.Strings(migrations)

	dbID, err := randomID()
	test.OK(t, err)

	dbName := fmt.Sprintf("test_db_" + dbID)
	db, err := dbutil.ConnectMySQL(&dbutil.DBConfig{
		Host: host,
		// Have to connect to something before we can create the database.
		// TODO: The mysql database should always exist though we may not actually have permissions on it
		Name:     "mysql",
		User:     user,
		Password: password,
	})
	test.OK(t, err)
	_, err = db.Exec(`CREATE DATABASE ` + dbName)
	test.OK(t, err)
	_, err = db.Exec(`USE ` + dbName)
	test.OK(t, err)
	for _, m := range migrations {
		b, err := ioutil.ReadFile(m)
		if err != nil {
			db.Exec(`DELETE DATABASE ` + dbName)
			t.Fatal(err)
		}
		s := string(b)
		lines := strings.Split(s, "\n")
		delimiter := ";"
		nonEmpty := make([]string, 0, len(lines))
		for _, l := range lines {
			if i := strings.Index(l, "--"); i >= 0 {
				l = l[:i]
			}
			l := strings.TrimSpace(l)
			if l != "" {
				// Only support delimiter statement on first non-blank line (i.e. only one delimiter per file)
				if len(nonEmpty) == 0 && strings.HasPrefix(strings.ToUpper(l), "DELIMITER ") {
					delimiter = strings.Split(l, " ")[1]
				} else {
					nonEmpty = append(nonEmpty, l)
				}
			}
		}
		stmts := strings.Split(strings.Join(nonEmpty, "\n"), delimiter)
		for _, st := range stmts {
			st = strings.TrimSpace(st)
			if st != "" {
				if _, err := db.Exec(st); err != nil {
					db.Exec(`DELETE DATABASE ` + dbName)
					t.Fatalf("Failed to apply migration %s: %s\nstatement: %s", m, err, st)
				}
			}
		}
	}
	// Reconnect so that we use the actual database name in case of reconnect
	test.OK(t, db.Close())
	db, err = dbutil.ConnectMySQL(&dbutil.DBConfig{
		Host:     host,
		Name:     dbName,
		User:     user,
		Password: password,
	})
	test.OK(t, err)
	return &DB{
		DB:   db,
		name: dbName,
	}
}

// Cleanup drops the test database
func (d *DB) Cleanup(t *testing.T) {
	_, err := d.DB.Exec(`DROP DATABASE ` + d.name)
	if err != nil {
		t.Log(err)
	}
}

func randomID() (string, error) {
	var b [8]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
