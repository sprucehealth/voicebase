package main

import (
	"database/sql"
	"flag"
	"fmt"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/storage"
)

type migrateBrokenImagesCmd struct {
	cnf          *config
	imageService *media.ImageService
	mediaDB      *sql.DB
}

func newMigrateBrokenImagesCmd(cnf *config) (command, error) {
	mediaDB, err := cnf.db("media")
	if err != nil {
		return nil, err
	}
	session, err := cnf.awsSession()
	if err != nil {
		return nil, err
	}
	store := storage.NewS3(session, fmt.Sprintf("%s-baymax-storage", cnf.Env), "media")
	imageService := media.NewImageService(store, store, 0, 0)
	return &migrateBrokenImagesCmd{
		cnf:          cnf,
		imageService: imageService,
		mediaDB:      mediaDB,
	}, nil
}

func (c *migrateBrokenImagesCmd) run(args []string) error {
	fs := flag.NewFlagSet("migratebrokenimages", flag.ExitOnError)
	args = fs.Args()

	bImages, err := brokenImages(c.mediaDB)
	if err != nil {
		return errors.Trace(err)
	}
	golog.Infof("Found %d broken images", len(bImages))
	for _, id := range bImages {
		if err := c.fixBrokenImage(id); err != nil {
			return errors.Trace(err)
		}
	}
	golog.Infof("Fixed %d broken images", len(bImages))
	return nil
}

func brokenImages(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT id FROM media WHERE mime_type = ?`, "image/*")
	if err != nil {
		return nil, fmt.Errorf("Failed to get list of broken media objects: %s", err)
	}
	defer rows.Close()
	var imageIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("Failed to scan media: %s", err)
		}
		imageIDs = append(imageIDs, id)
	}
	return imageIDs, errors.Trace(rows.Err())
}

func (c *migrateBrokenImagesCmd) fixBrokenImage(id string) error {
	meta, err := c.imageService.GetMeta(id)
	if err != nil {
		return errors.Trace(err)
	}
	golog.Infof("Updating %s to %s", id, meta.MimeType)
	_, err = c.mediaDB.Exec("UPDATE media SET mime_type = ? WHERE id = ?", meta.MimeType, id)
	return errors.Trace(err)
}
