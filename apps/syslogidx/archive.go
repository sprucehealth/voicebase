package main

import (
	"github.com/sprucehealth/backend/libs/aws/s3"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	flagArchiveSrcPath  = flag.String("archive_src_path", "/var/log/syslog.1", "Source path for archiving")
	flagArchiveDestPath = flag.String("archive_dest_path", "s3://carefront-audit-logs/syslog/", "Where to copy archived logs")
)

type Stat struct {
	Ctime time.Time
	Size  int64
	Inode uint64
}

func archiveLog(path, bucket, key string, st *Stat) error {
	isCompressed := false
	extension := ".gz"
	if strings.HasSuffix(path, ".gz") || strings.HasSuffix(path, ".bz2") {
		isCompressed = true
		idx := strings.LastIndex(path, ".")
		extension = path[idx:]
	}
	key += extension

	if _, err := s3Client.Head(bucket, key); err == nil {
		log.Printf("Log already archived: %s", key)
		return nil
	}

	fi, err := os.Open(path)
	if err != nil {
		log.Printf("Failed to open log: %+v", err)
		return err
	}
	defer fi.Close()

	if isCompressed {
		if err := s3Client.PutFrom(bucket, key, fi, st.Size, "application/x-gzip", s3.Private, nil); err != nil {
			log.Printf("Failed to put log: %+v", err)
			return err
		}
		return nil
	}

	tempFile, err := ioutil.TempFile("", "syslogidx")
	if err != nil {
		log.Printf("Failed to create temp file: %+v", err)
		return err
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()

	gzWr := gzip.NewWriter(tempFile)
	defer gzWr.Close()
	if _, err := io.Copy(gzWr, fi); err != nil {
		log.Printf("Failed to gzip file: %+v", err)
		return err
	}
	if err := gzWr.Flush(); err != nil {
		log.Printf("Failed to flush gzip: %+v", err)
		return err
	}
	if err := tempFile.Sync(); err != nil {
		log.Printf("Failed to sync temp file: %+v", err)
		return err
	}

	size, _ := tempFile.Seek(0, 1)
	if _, err := tempFile.Seek(0, 0); err != nil {
		log.Printf("Failed to seek to beginning of temp file: %+v", err)
		return err
	}
	if err := s3Client.PutFrom(bucket, key, tempFile, size, "application/x-gzip", s3.Private, nil); err != nil {
		log.Printf("Failed to put log: %+v", err)
		return err
	}
	log.Printf("Archived %s to %s", path, key)

	return nil
}

func startLogArchiving() error {
	destUrl, err := url.Parse(*flagArchiveDestPath)
	if err != nil {
		return err
	}
	if destUrl.Scheme != "s3" {
		return errors.New("Unsupported scheme for log archiving: " + destUrl.Scheme)
	}
	bucket := destUrl.Host
	path := destUrl.Path
	if path[0] == '/' {
		path = path[1:]
	}

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	go func() {
		var lastInode uint64
		first := true
		for {
			if !first {
				time.Sleep(time.Hour)
			}
			first = false

			fileInfo, err := os.Stat(*flagArchiveSrcPath)
			if err != nil {
				log.Printf("Unable to stat %s: %+v", *flagArchiveSrcPath, err)
				continue
			}

			stat, err := GetStat(fileInfo)
			if err != nil {
				log.Println(err)
				continue
			}

			if stat.Inode == lastInode {
				// log.Println("Already seen log file")
				continue
			}

			key := fmt.Sprintf("%s%s-%s-%d", path, stat.Ctime.Format("2006/01/02/15"), hostname, stat.Inode)

			if err := archiveLog(*flagArchiveSrcPath, bucket, key, stat); err != nil {
				continue
			}

			lastInode = stat.Inode
		}
	}()

	return nil
}
