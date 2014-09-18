package main

import (
	"database/sql"

	"github.com/sprucehealth/backend/libs/aws/s3"
)

const (
	createdStatus   = "CREATED"
	completedStatus = "COMPLETED"
	erroredStatus   = "ERRORED"
	testPharmacyId  = 47731
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

type worker struct {
	db         *sql.DB
	s3Client   *s3.S3
	bucketName string
}
