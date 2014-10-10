package main

import (
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/sprucehealth/backend/libs/aws/s3"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-librato/librato"
)

type column struct {
	Name      string
	Type      string
	Transform string `json:",omitempty"`
}

type table struct {
	Name    string
	Columns []column
}

func transform(srcDB, destDB *sql.DB, tables []*table, s3c *s3.S3, bucket, prefix string, lib *librato.Client, libSource string) error {
	if len(prefix) > 0 && prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}
	// Set the max connections to 1 since there's no way to hold onto a connection
	// without starting a transaction. When dumping the tables we need to set the
	// isolation level and make sure to get the same connection/session for the next query.
	srcDB.SetMaxOpenConns(1)

	metrics := &librato.Metrics{}
	if lib != nil {
		defer func() {
			if len(metrics.Gauges) != 0 {
				if err := lib.PostMetrics(metrics); err != nil {
					golog.Errorf("Failed to post librato metrics: %s", err.Error())
				}
			}
		}()
	}

	// Dump tables

	for _, tab := range tables {
		golog.Infof("Dumping table %s", tab.Name)

		var columns []string
		for _, c := range tab.Columns {
			if c.Transform != "" {
				columns = append(columns, `"`+c.Transform+`"`)
			} else if strings.Contains(strings.ToUpper(c.Type), "TIMESTAMP") {
				columns = append(columns, `DATE_FORMAT("`+c.Name+`", '%Y-%m-%d %H:%i:%S')`)
			} else {
				columns = append(columns, `"`+c.Name+`"`)
			}
		}

		r, w := io.Pipe()
		headers := map[string][]string{
			"x-amz-server-side-encryption": []string{"AES256"},
			"Content-Encoding":             []string{"gzip"},
		}
		uploadCh := make(chan error, 1)
		go func() {
			uploadCh <- s3c.PutMultiFrom(bucket, prefix+tab.Name+".json.gz", r, "application/json", s3.Private, headers)
		}()

		err := func() error {
			gz := gzip.NewWriter(w)
			defer gz.Close()

			// Avoid locking during the SELECT (hopefully the SELECT happens in the same session/connection)
			if _, err := srcDB.Exec(`SET SESSION TRANSACTION ISOLATION LEVEL READ UNCOMMITTED`); err != nil {
				return err
			}

			rows, err := srcDB.Query(`SELECT ` + strings.Join(columns, ", ") + ` FROM "` + tab.Name + `"`)
			if err != nil {
				return err
			}
			defer rows.Close()

			enc := json.NewEncoder(gz)

			valPtrs := make([]interface{}, len(columns))
			vals := make([]interface{}, len(columns))
			valMap := make(map[string]interface{}, len(columns))
			for i := 0; i < len(vals); i++ {
				valPtrs[i] = &vals[i]
				valMap[tab.Columns[i].Name] = valPtrs[i]
			}
			nRows := 0
			for rows.Next() {
				nRows++
				if err := rows.Scan(valPtrs...); err != nil {
					return err
				}

				for i, v := range vals {
					switch x := v.(type) {
					case []byte:
						vals[i] = string(x)
					}
				}

				if err := enc.Encode(valMap); err != nil {
					return err
				}
			}
			metrics.Gauges = append(metrics.Gauges, &librato.Metric{
				Name:   "redshift.etl.table." + tab.Name + ".rows",
				Source: libSource,
				Value:  float64(nRows),
			})
			return rows.Err()
		}()
		if err != nil {
			w.CloseWithError(err)
			return err
		} else {
			if err := w.Close(); err != nil {
				return err
			}
		}
		if err := <-uploadCh; err != nil {
			return err
		}
	}

	// Import the tables into RedShift

	existingTables := make(map[string]bool)
	rows, err := destDB.Query("SELECT DISTINCT tablename FROM pg_tables")
	if err != nil {
		return err
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		existingTables[name] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	tx, err := destDB.Begin()
	if err != nil {
		return err
	}

	err = func() error {
		// Delete in reverse order to avoid breaking foreign key constraints
		for i := len(tables) - 1; i >= 0; i-- {
			tab := tables[i]
			if existingTables[tab.Name] {
				golog.Infof("Dropping table %s", tab.Name)
				_, err := tx.Exec(`DROP TABLE "` + tab.Name + `"`)
				if err != nil {
					return err
				}
			} else {
				golog.Infof("Table %s does not already exist", tab.Name)
			}
		}

		s3Keys := s3c.Client.Auth.Keys()
		s3Creds := fmt.Sprintf("aws_access_key_id=%s;aws_secret_access_key=%s", s3Keys.AccessKey, s3Keys.SecretKey)
		if s3Keys.Token != "" {
			s3Creds += ";token=" + s3Keys.Token
		}

		var columns []string
		for _, tab := range tables {
			golog.Infof("Creating and loading table %s", tab.Name)
			columns = columns[:0]
			for _, c := range tab.Columns {
				columns = append(columns, fmt.Sprintf(`"%s" %s`, c.Name, c.Type))
			}
			if _, err := tx.Exec(fmt.Sprintf(`CREATE TABLE "%s" (%s)`, tab.Name, strings.Join(columns, ", "))); err != nil {
				return err
			}
			if _, err := tx.Exec(fmt.Sprintf(`GRANT SELECT ON "%s" TO GROUP readonly`, tab.Name)); err != nil {
				return err
			}
			if _, err := tx.Exec(fmt.Sprintf(
				`COPY "%s" FROM 's3://%s/%s%s.json.gz'
				 CREDENTIALS '%s'
				 JSON AS 'auto' GZIP TRUNCATECOLUMNS`,
				tab.Name, bucket, prefix, tab.Name, s3Creds),
			); err != nil {
				return err
			}
		}

		return nil
	}()
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
