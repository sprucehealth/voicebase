package dbutil

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/go-sql-driver/mysql"
)

// DBConfig represents the data needed to connect to a database
type DBConfig struct {
	Host               string
	Port               int
	Name               string
	User               string
	Password           string
	EnableTLS          bool
	SkipVerifyTLS      bool
	CACert             string
	TLSCert            string
	TLSKey             string
	MaxOpenConnections int
	MaxIdleConnections int
}

// ConnectMySQL uses the provided information to initialize a mysql connection
func ConnectMySQL(dbconfig *DBConfig) (*sql.DB, error) {
	if dbconfig.User == "" || dbconfig.Host == "" || dbconfig.Name == "" {
		return nil, errors.New("missing one or more of user, host, or name for db config")
	}
	if dbconfig.Port == 0 {
		dbconfig.Port = 3306
	}

	cfg := &mysql.Config{
		User:              dbconfig.User,
		Passwd:            dbconfig.Password,
		Net:               "tcp",
		Addr:              fmt.Sprintf("%s:%d", dbconfig.Host, dbconfig.Port),
		DBName:            dbconfig.Name,
		Collation:         "utf8mb4_unicode_ci",
		Loc:               time.Local,
		Strict:            true,
		ParseTime:         true,
		InterpolateParams: true,
		Params: map[string]string{
			"sql_notes": "0",
			"charset":   "utf8mb4",
		},
	}

	if dbconfig.CACert != "" || (dbconfig.TLSCert != "" && dbconfig.TLSKey != "") {
		var rootCertPool *x509.CertPool

		if dbconfig.CACert != "" {
			rootCertPool = x509.NewCertPool()
			pem, err := ioutil.ReadFile(dbconfig.CACert)
			if err != nil {
				return nil, fmt.Errorf("failed to read DB CA cert file: %s", err)
			}
			if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
				return nil, fmt.Errorf("failed to append PEM")
			}
		} else {
			rootCertPool = x509.NewCertPool()
		}

		tlsConfig := &tls.Config{
			RootCAs:            rootCertPool,
			InsecureSkipVerify: false,
		}

		if dbconfig.TLSCert != "" && dbconfig.TLSKey != "" {
			cert, err := ioutil.ReadFile(dbconfig.TLSCert)
			if err != nil {
				return nil, fmt.Errorf("failed to read TLS cert file: %s", err)
			}
			key, err := ioutil.ReadFile(dbconfig.TLSKey)
			if err != nil {
				return nil, fmt.Errorf("failed to read TLS key file: %s", err)
			}
			pair, err := tls.X509KeyPair(cert, key)
			if err != nil {
				return nil, err
			}
			tlsConfig.Certificates = append(tlsConfig.Certificates, pair)
		}

		cfg.TLSConfig = "custom"
		if err := mysql.RegisterTLSConfig("custom", tlsConfig); err != nil {
			return nil, fmt.Errorf("failed to register custom DB TLS config: %s", err)
		}
	} else if dbconfig.EnableTLS {
		if dbconfig.SkipVerifyTLS {
			cfg.TLSConfig = "skip-verify"
		} else {
			// The MySQL pkg doesn't handle 'true' correctly in that it doesn't set the ServerName on the TLS config
			// so we need to use a custom config.
			tlsConfig := &tls.Config{
				ServerName: dbconfig.Host,
			}
			cfg.TLSConfig = "custom"
			if err := mysql.RegisterTLSConfig("custom", tlsConfig); err != nil {
				return nil, fmt.Errorf("failed to register custom DB TLS config: %s", err)
			}
		}
	}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}
	// test the connection to the database by running a ping against it
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	if dbconfig.MaxOpenConnections != 0 {
		db.SetMaxOpenConns(dbconfig.MaxOpenConnections)
	}
	if dbconfig.MaxIdleConnections != 0 {
		db.SetMaxOpenConns(dbconfig.MaxIdleConnections)
	}
	return db, nil
}
