package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sprucehealth/backend/consul"
	"github.com/sprucehealth/backend/libs/aws/s3"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/third_party/github.com/lib/pq"
)

const (
	createdStatus   = "CREATED"
	completedStatus = "COMPLETED"
	erroredStatus   = "ERRORED"
	timeFormat      = "2006-01-02"
)

type migrationItem struct {
	id              *int64
	fileName        *string
	numRowsUpdated  *int
	numRowsInserted *int
	status          *string
	errorMsg        *string
}

type pharmacyUpdateWorker struct {
	db            *sql.DB
	s3Client      *s3.S3
	bucketName    string
	consulService *consul.Service
}

func (w *pharmacyUpdateWorker) start() {
	lock := w.consulService.NewLock("service/pharmacydb/update", nil)
	go func() {
		defer lock.Release()
		for {
			if !lock.Wait() {
				return
			}

			if err := w.updatePharmacyDB(); err != nil {
				golog.Errorf(err.Error())
			}
			time.Sleep(24 * time.Hour)
		}
	}()
}

func (w *pharmacyUpdateWorker) updatePharmacyDB() error {
	// only stop if there are no more files to migrate
	for {
		bucketItems, err := w.nextFilesToMigrate()
		if err != nil {
			return err
		} else if len(bucketItems) == 0 {
			break
		}

		for _, item := range bucketItems {
			if err := w.processFile(item.Key); err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *pharmacyUpdateWorker) processFile(key string) error {

	mItem := &migrationItem{
		fileName: &key,
		status:   strPtr(createdStatus),
	}

	// if the migration is already complete for this file then
	// there is nothing else to do
	_, err := w.getMigrationItemForFile(key)
	if err != sql.ErrNoRows && err != nil {
		return err
	} else if err == nil {
		return nil
	}

	if err := w.addOrUpdateMigrationItem(mItem); err != nil {
		return err
	}

	err = w.updateDBFromFile(mItem)
	if err != nil {
		mItem.status = strPtr(erroredStatus)
		mItem.errorMsg = strPtr(err.Error())
	}

	if err := w.addOrUpdateMigrationItem(mItem); err != nil {
		return err
	}

	return err
}

func (w *pharmacyUpdateWorker) updateDBFromFile(item *migrationItem) error {

	if err := w.sanityCheckCSVFile(*item.fileName); err != nil {
		return err
	}

	reader, err := w.s3Client.GetReader(w.bucketName, *item.fileName)
	if err != nil {
		return err
	}
	defer reader.Close()

	tx, err := w.db.Begin()
	if err != nil {
		return err
	}

	// prepare the bulk copy statement for copying over new pharmacy items
	copyStmt, err := tx.Prepare(pq.CopyIn("pharmacy",
		"id",
		"ncpdpid",
		"store_number",
		"store_name",
		"address_line_1",
		"address_line_2",
		"city",
		"state",
		"zip",
		"phone_primary",
		"fax",
		"active_start_time",
		"active_end_time",
		"service_level",
		"specialty",
		"last_modified_date",
		"twenty_four_hour_flag",
		"version",
		"cross_street",
		"is_from_surescripts"))
	if err != nil {
		tx.Rollback()
		return err
	}

	csvReader := csv.NewReader(reader)
	var rowsInserted int
	var rowsToUpdate [][]string
	for {

		row, err := csvReader.Read()
		if err == io.EOF {
			break
		}

		// prepare the values to be copied in or updated
		vals := strSliceToInterfaceSlice(row)

		// only copy over new rows (identified by the pharmacy id)
		var id int64
		err = w.db.QueryRow(`SELECT id FROM pharmacy WHERE id = $1`, row[0]).Scan(&id)
		if err == sql.ErrNoRows {
			// copy the row if it doesnt already exist
			_, err = copyStmt.Exec(vals...)
			if err != nil {
				tx.Rollback()
				return err
			}
			rowsInserted++
		} else if err == nil {
			rowsToUpdate = append(rowsToUpdate, row)
		} else {
			tx.Rollback()
			return err
		}
	}

	// flush out buffered data and commit transaction
	if _, err := copyStmt.Exec(); err != nil {
		tx.Rollback()
		return err
	} else if err := copyStmt.Close(); err != nil {
		tx.Rollback()
		return err
	} else if err := tx.Commit(); err != nil {
		return err
	}

	// update any existing pharmacy rows
	tx, err = w.db.Begin()
	if err != nil {
		return err
	}

	updateStmt, err := tx.Prepare(`
				UPDATE pharmacy SET 
					ncpdpid = $2,
					store_number = $3,
					store_name = $4,
					address_line_1 = $5,
					address_line_2 = $6,
					city = $7,
					state = $8,
					zip = $9,
					phone_primary = $10,
					fax = $11,
					active_start_time = $12,
					active_end_time = $13,
					service_level = $14,
					specialty = $15,
					last_modified_date = $16,
					twenty_four_hour_flag = $17,
					version = $18,
					cross_street = $19,
					is_from_surescripts = $20
				WHERE id = $1`)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, row := range rowsToUpdate {
		vals := strSliceToInterfaceSlice(row)

		// update the row if it exists in the database
		_, err = updateStmt.Exec(vals...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := updateStmt.Close(); err != nil {
		tx.Rollback()
		return err
	} else if err := tx.Commit(); err != nil {
		tx.Rollback()
		return err
	}

	rowsUpdated := len(rowsToUpdate)
	item.numRowsUpdated = &rowsUpdated
	item.numRowsInserted = &rowsInserted
	item.status = strPtr(completedStatus)
	return nil
}

func (w *pharmacyUpdateWorker) getMigrationItemForFile(fileName string) (*migrationItem, error) {
	var mItem migrationItem
	if err := w.db.QueryRow(`
		SELECT id, file_name, rows_inserted, rows_updated, status, error 
		FROM pharmacy_migration WHERE file_name = $1`, fileName).Scan(
		&mItem.id,
		&mItem.fileName,
		&mItem.numRowsInserted,
		&mItem.numRowsUpdated,
		&mItem.status,
		&mItem.errorMsg); err != nil {
		return nil, err
	}

	return &mItem, nil
}

func (w *pharmacyUpdateWorker) nextFilesToMigrate() ([]*s3.BucketItem, error) {

	// don't proceed with identifying files if there are migrations
	// in incomplete states. Reason for this is that we continuing forth with the
	// migration will actually cause more problems because they have to be played back in order
	var count int64
	if err := w.db.QueryRow(`
		SELECT count(*) 
		FROM pharmacy_migration 
		WHERE status != $1`, completedStatus).Scan(&count); err != nil {
		return nil, err
	} else if count > 0 {
		return nil, fmt.Errorf("Cannot proceed forward with migration because there is an incomplete migration")
	}

	var mItem migrationItem
	if err := w.db.QueryRow(`
		SELECT id, file_name, rows_inserted, rows_updated, status, error 
		FROM pharmacy_migration ORDER BY id desc LIMIT 1`).Scan(
		&mItem.id,
		&mItem.fileName,
		&mItem.numRowsInserted,
		&mItem.numRowsUpdated,
		&mItem.status,
		&mItem.errorMsg); err != nil {
		return nil, err
	}

	// now look for the next file to migrate and only stop looking until today's date as hit
	date, err := time.Parse(timeFormat, (*mItem.fileName)[:len(timeFormat)])
	if err != nil {
		return nil, err
	}

	for {
		date = date.Add(24 * time.Hour)
		if time.Now().Before(date) {
			break
		}

		// lets look for the migration file from the next day
		filePrefix := fmt.Sprintf("%d-%02d-%02d", date.Year(), date.Month(), date.Day())

		listResults, err := w.s3Client.ListBucket(w.bucketName, &s3.ListBucketParams{Prefix: filePrefix})
		if err != nil {
			return nil, err
		} else if len(listResults.Contents) > 0 {
			return listResults.Contents, nil
		}
	}

	return nil, nil
}

func (w *pharmacyUpdateWorker) addOrUpdateMigrationItem(mItem *migrationItem) error {

	if mItem.id == nil {
		var updateId int64
		err := w.db.QueryRow(`INSERT INTO pharmacy_migration (file_name, status) VALUES ($1, $2) RETURNING id`, *mItem.fileName, *mItem.status).Scan(&updateId)
		if err != nil {
			return err
		}

		mItem.id = &updateId
		return nil
	}

	cols := []string{}
	vals := []interface{}{}
	i := 1

	if mItem.numRowsInserted != nil {
		cols = append(cols, fmt.Sprintf("rows_inserted = $%d", i))
		vals = append(vals, *mItem.numRowsInserted)
		i++
	}
	if mItem.numRowsUpdated != nil {
		cols = append(cols, fmt.Sprintf("rows_updated = $%d", i))
		vals = append(vals, *mItem.numRowsUpdated)
		i++
	}
	if mItem.status != nil {
		cols = append(cols, fmt.Sprintf("status = $%d", i))
		vals = append(vals, *mItem.status)
		i++
	}
	if mItem.errorMsg != nil {
		cols = append(cols, fmt.Sprintf("error = $%d", i))
		vals = append(vals, *mItem.errorMsg)
		i++
	}

	if len(cols) == 0 {
		return nil
	}

	vals = append(vals, *mItem.id)
	_, err := w.db.Exec(fmt.Sprintf(`UPDATE pharmacy_migration SET %s WHERE id = $%d`, strings.Join(cols, ","), i), vals...)
	if err != nil {
		return err
	}

	return nil
}

func (w *pharmacyUpdateWorker) sanityCheckCSVFile(key string) error {

	reader, err := w.s3Client.GetReader(w.bucketName, key)
	if err != nil {
		return err
	}
	defer reader.Close()

	csvReader := csv.NewReader(reader)
	for {
		row, err := csvReader.Read()
		if err == io.EOF {
			break
		}

		if len(row) != 0 && len(row) != 20 {
			return fmt.Errorf("Expected 20 items in the row instead got %d, for row %s", len(row), row)
		}
	}
	return nil
}
