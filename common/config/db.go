package config

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sprucehealth/backend/third_party/github.com/go-sql-driver/mysql"
)

type DB struct {
	User     string `long:"db_user" description:"Username for accessing database"`
	Password string `long:"db_password" description:"Password for accessing database"`
	Host     string `long:"db_host" description:"Database host"`
	Port     int    `long:"db_port" description:"Database port"`
	Name     string `long:"db_name" description:"Database name"`
	CACert   string `long:"db_cacert" description:"Database TLS CA certificate path"`
	TLSCert  string `long:"db_cert" description:"Database TLS client certificate path"`
	TLSKey   string `long:"db_key" description:"Database TLS client key path"`
}

func (c *DB) Connect(bconf *BaseConfig) (*sql.DB, error) {
	if c.User == "" || c.Host == "" || c.Name == "" {
		return nil, errors.New("missing one or more of user, host, or name for db config")
	}

	enableTLS := c.CACert != "" && c.TLSCert != "" && c.TLSKey != ""
	if enableTLS {
		rootCertPool := x509.NewCertPool()
		pem, err := bconf.ReadURI(c.CACert)
		if err != nil {
			return nil, err
		}
		if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
			return nil, fmt.Errorf("Failed to append PEM.")
		}
		clientCert := make([]tls.Certificate, 0, 1)
		cert, err := bconf.ReadURI(c.TLSCert)
		if err != nil {
			return nil, err
		}
		key, err := bconf.ReadURI(c.TLSKey)
		if err != nil {
			return nil, err
		}
		certs, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return nil, err
		}
		clientCert = append(clientCert, certs)
		mysql.RegisterTLSConfig("custom", &tls.Config{
			RootCAs:            rootCertPool,
			Certificates:       clientCert,
			InsecureSkipVerify: true,
		})
	}

	tlsOpt := "?parseTime=true"
	if enableTLS {
		tlsOpt += "&tls=custom"
	}
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s%s", c.User, c.Password, c.Host, c.Port, c.Name, tlsOpt))
	if err != nil {
		return nil, err
	}
	// test the connection to the database by running a ping against it
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
