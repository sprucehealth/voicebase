package layout

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"sync"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/visitreview"
	"github.com/sprucehealth/mapstructure"
)

// Storage is the interface that any object, that implements a backing
// store for SAML and its generated documents, confirms to.
type Storage interface {
	PutIntake(name string, intake *Intake) (string, error)
	PutReview(name string, review *visitreview.SectionListView) (string, error)
	PutSAML(name, saml string) (string, error)
	GetIntake(location string) (*Intake, error)
	GetReview(location string) (*visitreview.SectionListView, error)
	GetSAML(location string) (string, error)
}

type layoutStore struct {
	store storage.Store
}

var (
	gzipReaderPool sync.Pool
	gzipWriterPool sync.Pool
)

func NewStore(store storage.Store) Storage {
	return &layoutStore{
		store: store,
	}
}

func (s *layoutStore) PutIntake(name string, intake *Intake) (string, error) {
	return s.putCompressedData(name, "application/json", func(w io.Writer) error {
		return json.NewEncoder(w).Encode(intake)
	})
}

func (s *layoutStore) PutReview(name string, review *visitreview.SectionListView) (string, error) {
	return s.putCompressedData(name, "application/json", func(w io.Writer) error {
		return json.NewEncoder(w).Encode(review)
	})
}

func (s *layoutStore) PutSAML(name, saml string) (string, error) {
	return s.putCompressedData(name, "application/octet-stream", func(w io.Writer) error {
		if _, err := w.Write([]byte(saml)); err != nil {
			return err
		}
		return nil
	})
}

func (s *layoutStore) GetIntake(location string) (*Intake, error) {
	reader, err := s.getCompressedData(location)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer reader.Close()

	var intake Intake
	if err := json.NewDecoder(reader).Decode(&intake); err != nil {
		return nil, errors.Trace(err)
	}
	return &intake, nil
}

func (s *layoutStore) GetReview(location string) (*visitreview.SectionListView, error) {
	reader, err := s.getCompressedData(location)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer reader.Close()

	var jsonMap map[string]interface{}
	if err := json.NewDecoder(reader).Decode(&jsonMap); err != nil {
		return nil, errors.Trace(err)
	}

	var sectionList visitreview.SectionListView
	decoderConfig := &mapstructure.DecoderConfig{
		Result:   &sectionList,
		TagName:  "json",
		Registry: *visitreview.TypeRegistry,
	}

	d, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if err := d.Decode(jsonMap); err != nil {
		return nil, errors.Trace(err)
	}

	return &sectionList, nil
}

func (s *layoutStore) GetSAML(location string) (string, error) {
	reader, err := s.getCompressedData(location)
	if err != nil {
		return "", errors.Trace(err)
	}
	defer reader.Close()

	samlData, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", errors.Trace(err)
	}
	return string(samlData), nil
}

func (s *layoutStore) putCompressedData(name, contentType string, writeFunc func(w io.Writer) error) (string, error) {

	var writer *gzipWriter
	if r := gzipWriterPool.Get(); r != nil {
		writer = r.(*gzipWriter)
		writer.Reset()
	} else {
		writer = newGzipWriter()
	}

	if err := writeFunc(writer.zw); err != nil {
		return "", errors.Trace(err)
	}

	if err := writer.zw.Close(); err != nil {
		return "", errors.Trace(err)
	}

	reader := bytes.NewReader(writer.buffer.Bytes())

	// release back into pool
	gzipWriterPool.Put(writer)
	writer = nil

	size, err := media.SeekerSize(reader)
	if err != nil {
		return "", errors.Trace(err)
	}

	location, err := s.store.PutReader(name, reader, size, contentType, nil)
	if err != nil {
		return "", errors.Trace(err)
	}

	return location, nil
}

func (s *layoutStore) getCompressedData(location string) (io.ReadCloser, error) {

	reader, _, err := s.store.GetReader(location)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &gzipReadCloser{rc: reader}, nil
}

type gzipReadCloser struct {
	rc io.ReadCloser
	zr *gzip.Reader
}

type gzipWriter struct {
	buffer bytes.Buffer
	zw     *gzip.Writer
}

func newGzipWriter() *gzipWriter {
	w := &gzipWriter{}
	w.zw = gzip.NewWriter(&w.buffer)
	return w
}

func (w *gzipWriter) Reset() {

	w.zw.Reset(&w.buffer)
	w.buffer.Reset()
}

func (gz *gzipReadCloser) Read(b []byte) (int, error) {
	if gz.zr == nil {
		var zr *gzip.Reader
		if r := gzipReaderPool.Get(); r != nil {
			zr = r.(*gzip.Reader)
			if err := zr.Reset(gz.rc); err != nil {
				return 0, err
			}
		} else {
			var err error
			zr, err = gzip.NewReader(gz.rc)
			if err != nil {
				return 0, err
			}
		}
		gz.zr = zr
	}
	return gz.zr.Read(b)
}

func (gz *gzipReadCloser) Close() error {
	if gz.zr != nil {
		gz.rc.Close()
		err := gz.zr.Close()
		gzipReaderPool.Put(gz.zr)
		gz.zr = nil
		return err
	}
	return nil
}
