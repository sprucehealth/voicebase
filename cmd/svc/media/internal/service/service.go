package service

import (
	"context"
	"io"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/mediactx"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/mime"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
)

// Service implements a multi media storage service.
type Service interface {
	CanAccess(ctx context.Context, mediaID dal.MediaID, accountID string) error
	CopyMedia(ctx context.Context, ownerType dal.MediaOwnerType, ownerID string, sourceID dal.MediaID) (*MediaMeta, error)
	ExpiringURL(ctx context.Context, mediaID dal.MediaID, exp time.Duration) (string, error)
	GetReader(ctx context.Context, mediaID dal.MediaID) (io.ReadCloser, *MediaMeta, error)
	GetThumbnailReader(ctx context.Context, mediaID dal.MediaID, size *media.ImageSize) (io.ReadCloser, *media.ImageMeta, error)
	IsPublic(ctx context.Context, mediaID dal.MediaID) (bool, error)
	PutMedia(ctx context.Context, mFile io.ReadSeeker, fileName string, mediaType *mime.Type, mThumb io.ReadSeeker) (*MediaMeta, error)
}

// New returns a new initialized multi media service.
func New(
	dal dal.DAL,
	directory directory.DirectoryClient,
	threads threading.ThreadsClient,
	care care.CareClient,
	imageService *media.ImageService,
	audioService *media.AudioService,
	videoService *media.VideoService,
	binaryService *media.BinaryService,
) Service {
	return &service{
		dal:           dal,
		directory:     directory,
		threads:       threads,
		care:          care,
		imageService:  imageService,
		audioService:  audioService,
		videoService:  videoService,
		binaryService: binaryService,
	}
}

// Service implements a multi media storage service.
type service struct {
	dal           dal.DAL
	directory     directory.DirectoryClient
	threads       threading.ThreadsClient
	care          care.CareClient
	imageService  *media.ImageService
	audioService  *media.AudioService
	videoService  *media.VideoService
	binaryService *media.BinaryService
}

// ErrUnsupportedContentType represents an attempt to put unsupported media into the service
var ErrUnsupportedContentType = errors.New("Unsupported content-type")

// ErrAccessDenied represents a request for access to a resource the caller cannot access
var ErrAccessDenied = errors.New("Access denied")

// MediaMeta represents the metadata being tracked by the system common to all media types
type MediaMeta struct {
	MediaID  dal.MediaID
	MIMEType string
}

const thumbnailSuffix = "-thumbnail"

func (s *service) CopyMedia(ctx context.Context, ownerType dal.MediaOwnerType, ownerID string, sourceID dal.MediaID) (*MediaMeta, error) {
	med, err := s.media(ctx, sourceID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	mimeType, err := mime.ParseType(med.MimeType)
	if err != nil {
		return nil, errors.Trace(err)
	}

	newID, err := dal.NewMediaID()
	if err != nil {
		return nil, errors.Trace(err)
	}

	par := conc.NewParallel()
	par.Go(func() error {
		switch mimeType.Type {
		case "image":
			return errors.Trace(s.imageService.Copy(newID.String(), sourceID.String()))
		case "audio":
			return errors.Trace(s.audioService.Copy(newID.String(), sourceID.String()))
		case "video":
			return errors.Trace(s.videoService.Copy(newID.String(), sourceID.String()))
		}
		return errors.Trace(s.binaryService.Copy(newID.String(), sourceID.String()))
	})
	par.Go(func() error {
		if err := s.imageService.Copy(newID.String()+thumbnailSuffix, sourceID.String()+thumbnailSuffix); err != nil && errors.Cause(err) != media.ErrNotFound {
			return errors.Trace(err)
		}
		return nil
	})
	if err := par.Wait(); err != nil {
		return nil, errors.Trace(err)
	}

	_, err = s.dal.InsertMedia(&dal.Media{
		ID:         newID,
		Name:       med.Name,
		MimeType:   med.MimeType,
		OwnerType:  ownerType,
		OwnerID:    ownerID,
		SizeBytes:  med.SizeBytes,
		DurationNS: med.DurationNS,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &MediaMeta{
		MediaID:  newID,
		MIMEType: med.MimeType,
	}, nil
}

func (s *service) GetReader(ctx context.Context, mediaID dal.MediaID) (io.ReadCloser, *MediaMeta, error) {
	media, err := s.media(ctx, mediaID)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	mt, err := mime.ParseType(media.MimeType)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	var rc io.ReadCloser
	switch mt.Type {
	case "image":
		rc, _, err = s.imageService.GetReader(mediaID.String(), nil)
		if err != nil {
			return nil, nil, errors.Trace(err)
		}
	case "audio":
		rc, err = s.audioService.GetReader(mediaID.String())
		if err != nil {
			return nil, nil, errors.Trace(err)
		}
	case "video":
		rc, err = s.videoService.GetReader(mediaID.String())
		if err != nil {
			return nil, nil, errors.Trace(err)
		}
	default:
		rc, err = s.binaryService.GetReader(mediaID.String())
		if err != nil {
			return nil, nil, errors.Trace(err)
		}
	}
	return rc, &MediaMeta{
		MediaID:  mediaID,
		MIMEType: media.MimeType,
	}, nil
}

func (s *service) GetThumbnailReader(ctx context.Context, mediaID dal.MediaID, size *media.ImageSize) (io.ReadCloser, *media.ImageMeta, error) {
	var thumbID string
	m, err := s.media(ctx, mediaID)
	// If we didn't find it in our data store, assume it's a legacy media segment and that the id is consistent
	if errors.Cause(err) == dal.ErrNotFound {
		thumbID = mediaID.String()
	} else if err != nil {
		return nil, nil, errors.Trace(err)
	} else {
		thumbID = thumbnailID(m)
	}

	rc, meta, err := s.imageService.GetReader(thumbID, size)
	if errors.Cause(err) == media.ErrNotFound {
		// Attempt a placholder fallback
		rc, meta, err = s.imageService.GetReader(placeholderID(m), size)
		if err != nil {
			return nil, nil, errors.Trace(err)
		}
	} else if err != nil {
		return nil, nil, errors.Trace(err)
	}
	return rc, meta, errors.Trace(err)
}

func thumbnailID(m *dal.Media) string {
	t, err := mime.ParseType(m.MimeType)
	if err != nil {
		golog.Errorf("Unable to parse content type for media %s: %s - %s", m.ID, t, err)
		return m.ID.String() + thumbnailSuffix
	}
	// Thumbnails served to clients are dynamically generated for the requested size. For images
	// we don't store a specific thumbnail image and instead use the original. For other types
	// there should be a thumbnail image.
	if t.Type == "image" {
		return m.ID.String()
	}
	return m.ID.String() + thumbnailSuffix
}

func (s *service) ExpiringURL(ctx context.Context, mediaID dal.MediaID, exp time.Duration) (string, error) {
	return s.binaryService.ExpiringURL(mediaID.String(), exp)
}

// PutReader sends the provided reader to the storage layers. Optionally, if a thumbnail is provided, that is mapped to the media
func (s *service) PutMedia(ctx context.Context, mFile io.ReadSeeker, mFileName string, mediaType *mime.Type, mThumb io.ReadSeeker) (*MediaMeta, error) {
	acc, err := mediactx.Account(ctx)
	if err != nil {
		return nil, errors.Trace(err)
	}
	mediaID, err := dal.NewMediaID()
	if err != nil {
		return nil, errors.Trace(err)
	}
	parallel := conc.NewParallel()
	var size uint64
	var duration time.Duration
	parallel.Go(func() error {
		switch mediaType.Type {
		case "image":
			im, err := s.imageService.PutReader(mediaID.String(), mFile)
			if err != nil {
				return errors.Trace(err)
			}
			size = im.Size
			// Trust what the decoder/uploader sent
			mediaType, err = mime.ParseType(im.MimeType)
			if err != nil {
				return errors.Trace(err)
			}
		case "audio":
			am, err := s.audioService.PutReader(mediaID.String(), mFile, mediaType.String())
			if err != nil {
				return errors.Trace(err)
			}
			size = am.Size
			duration = am.Duration
		case "video":
			vm, err := s.videoService.PutReader(mediaID.String(), mFile, mediaType.String())
			if err != nil {
				return errors.Trace(err)
			}
			size = vm.Size
			duration = vm.Duration
		default:
			bm, err := s.binaryService.PutReader(mediaID.String(), mFile, mediaType.String())
			if err != nil {
				return errors.Trace(err)
			}
			size = bm.Size
		}
		return nil
	})
	// If thumbnail information for the media was provided, upload and map it
	if mThumb != nil {
		parallel.Go(func() error {
			if _, err := s.imageService.PutReader(mediaID.String()+thumbnailSuffix, mThumb); err != nil {
				return errors.Trace(err)
			}
			return nil
		})
	}
	if err := parallel.Wait(); err != nil {
		return nil, errors.Trace(err)
	}
	_, err = s.dal.InsertMedia(&dal.Media{
		ID:         mediaID,
		Name:       mFileName,
		MimeType:   mediaType.String(),
		OwnerType:  dal.MediaOwnerTypeAccount,
		OwnerID:    acc.ID,
		SizeBytes:  size,
		DurationNS: uint64(duration.Nanoseconds()),
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &MediaMeta{
		MediaID:  mediaID,
		MIMEType: mediaType.String(),
	}, nil
}

// media fetches the media metadata from the database. If the metadata doesn't exist
// then it attempts to check the blob store and saves the metdata to the db.
func (s *service) media(ctx context.Context, id dal.MediaID) (*dal.Media, error) {
	med, err := s.dal.Media(id)
	if err == nil {
		return med, nil
	}
	if errors.Cause(err) != dal.ErrNotFound {
		return nil, errors.Trace(err)
	}
	meta, err := s.binaryService.GetMeta(id.String())
	if err != nil {
		return nil, errors.Trace(err)
	}
	med = &dal.Media{
		ID:         id,
		Name:       meta.Name,
		MimeType:   meta.MimeType,
		SizeBytes:  meta.Size,
		OwnerType:  dal.MediaOwnerTypeLegacy,
		DurationNS: uint64(meta.Duration.Nanoseconds()),
	}
	_, err = s.dal.InsertMedia(med)
	return med, errors.Trace(err)
}

const (
	imagePHThumbID  = "ph_image_thumb"
	videoPHThumbID  = "ph_video_thumb"
	audioPHThumbID  = "ph_audio_thumb"
	binaryPHThumbID = "ph_binary_thumb"
)

func placeholderID(m *dal.Media) string {
	t, err := mime.ParseType(m.MimeType)
	if err != nil {
		golog.Errorf("Unable to parse content type for media %s: %s - %s", m.ID, t, err)
		return binaryPHThumbID
	}
	switch t.Type {
	case "image":
		return imagePHThumbID
	case "audio":
		return audioPHThumbID
	case "video":
		return videoPHThumbID
	}
	return binaryPHThumbID
}
