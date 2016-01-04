package dbutil

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
)

// DBConfig represents the data needed to connect to a database
type DBConfig struct {
	Host               string
	Port               int64
	Name               string
	User               string
	Password           string
	EnableTLS          bool
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

	enableTLS := dbconfig.CACert != "" && dbconfig.TLSCert != "" && dbconfig.TLSKey != ""
	tlsOpt := "?parseTime=true"
	if enableTLS {
		rootCertPool := x509.NewCertPool()
		if ok := rootCertPool.AppendCertsFromPEM([]byte(dbconfig.CACert)); !ok {
			return nil, fmt.Errorf("Failed to append PEM.")
		}
		clientCert := make([]tls.Certificate, 0, 1)
		certs, err := tls.X509KeyPair([]byte(dbconfig.TLSCert), []byte(dbconfig.TLSKey))
		if err != nil {
			return nil, err
		}
		clientCert = append(clientCert, certs)
		if err := mysql.RegisterTLSConfig("custom", &tls.Config{
			RootCAs:            rootCertPool,
			Certificates:       clientCert,
			InsecureSkipVerify: true,
		}); err != nil {
			return nil, err
		}
		tlsOpt += "&tls=custom"
	}

	if dbconfig.EnableTLS {
		tlsOpt += "&tls=true"
	}
	if dbconfig.Port == 0 {
		dbconfig.Port = 3306
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s%s&charset=utf8mb4&collation=utf8mb4_unicode_ci&loc=Local&interpolateParams=true",
		dbconfig.User, dbconfig.Password, dbconfig.Host, dbconfig.Port, dbconfig.Name, tlsOpt)
	db, err := sql.Open("mysql", dsn)
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
