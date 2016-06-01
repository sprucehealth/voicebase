package service

import (
	"io"
	"time"

	"golang.org/x/net/context"

	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/mime"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/media"
)

// Service implements a multi media storage service.
type Service interface {
	PutMedia(ctx context.Context, mFile io.ReadSeeker, mediaType *mime.Type, mThumb io.ReadSeeker) (*MediaMeta, error)
	GetReader(ctx context.Context, mediaID dal.MediaID) (io.ReadCloser, *MediaMeta, error)
	GetThumbnailReader(ctx context.Context, mediaID dal.MediaID, size *media.ImageSize) (io.ReadCloser, *media.ImageMeta, error)
	ExpiringURL(ctx context.Context, mediaID dal.MediaID, exp time.Duration) (string, error)
}

// New returns a new initialized multi media service.
func New(
	dal dal.DAL,
	imageService *media.ImageService,
	audioService *media.AudioService,
	videoService *media.VideoService,
	binaryService *media.BinaryService,
) Service {
	return &service{
		dal:           dal,
		imageService:  imageService,
		audioService:  audioService,
		videoService:  videoService,
		binaryService: binaryService,
	}
}

// Service implements a multi media storage service.
type service struct {
	dal           dal.DAL
	imageService  *media.ImageService
	audioService  *media.AudioService
	videoService  *media.VideoService
	binaryService *media.BinaryService
}

// ErrUnsupportedContentType represents an attempt to put unsupported media into the service
var ErrUnsupportedContentType = errors.New("Unsupported content-type")

// MediaMeta represents the metadata being tracked by the system common to all media types
type MediaMeta struct {
	MediaID  dal.MediaID
	MIMEType string
}

const thumbnailSuffix = "-thumbnail"

// PutReader sends the provided reader to the storage layers. Optionally, if a thumbnail is provided, that is mapped to the media
func (s *service) PutMedia(ctx context.Context, mFile io.ReadSeeker, mediaType *mime.Type, mThumb io.ReadSeeker) (*MediaMeta, error) {
	mediaID, err := dal.NewMediaID()
	if err != nil {
		return nil, err
	}
	parallel := conc.NewParallel()
	var size uint64
	var duration time.Duration
	var url string
	parallel.Go(func() error {
		switch mediaType.Type {
		case "image":
			im, err := s.imageService.PutReader(mediaID.String(), mFile)
			if err != nil {
				return err
			}
			size = im.Size
			url = im.URL
		case "audio":
			am, err := s.audioService.PutReader(mediaID.String(), mFile, mediaType.String())
			if err != nil {
				return err
			}
			size = am.Size
			duration = am.Duration
			url = am.URL
		case "video":
			vm, err := s.videoService.PutReader(mediaID.String(), mFile, mediaType.String())
			if err != nil {
				return err
			}
			size = vm.Size
			duration = vm.Duration
			url = vm.URL
		default:
			bm, err := s.binaryService.PutReader(mediaID.String(), mFile, mediaType.String())
			if err != nil {
				return err
			}
			size = bm.Size
			url = bm.URL
		}
		return nil
	})
	// If thumbnail information for the media was provided, upload and map it
	if mThumb != nil {
		parallel.Go(func() error {
			if _, err := s.imageService.PutReader(mediaID.String()+thumbnailSuffix, mThumb); err != nil {
				return err
			}
			return nil
		})
	}
	if err := parallel.Wait(); err != nil {
		return nil, err
	}
	_, err = s.dal.InsertMedia(&dal.Media{
		ID:         mediaID,
		URL:        url,
		MimeType:   mediaType.String(),
		OwnerType:  dal.MediaOwnerTypeEntity,
		OwnerID:    "TODO",
		SizeBytes:  size,
		DurationNS: uint64(duration.Nanoseconds()),
	})
	if err != nil {
		return nil, err
	}
	return &MediaMeta{
		MediaID:  mediaID,
		MIMEType: mediaType.String(),
	}, nil
}

func (s *service) GetReader(ctx context.Context, mediaID dal.MediaID) (io.ReadCloser, *MediaMeta, error) {
	media, err := s.dal.Media(mediaID)
	if err != nil {
		return nil, nil, err
	}
	mt, err := mime.ParseType(media.MimeType)
	if err != nil {
		return nil, nil, err
	}

	var rc io.ReadCloser
	switch mt.Type {
	case "image":
		rc, _, err = s.imageService.GetReader(mediaID.String(), nil)
		if err != nil {
			return nil, nil, err
		}
	case "audio":
		rc, err = s.audioService.GetReader(mediaID.String())
		if err != nil {
			return nil, nil, err
		}
	case "video":
		rc, err = s.videoService.GetReader(mediaID.String())
		if err != nil {
			return nil, nil, err
		}
	default:
		rc, err = s.binaryService.GetReader(mediaID.String())
		if err != nil {
			return nil, nil, err
		}
	}
	return rc, &MediaMeta{
		MediaID:  mediaID,
		MIMEType: media.MimeType,
	}, nil
}

func (s *service) GetThumbnailReader(ctx context.Context, mediaID dal.MediaID, size *media.ImageSize) (io.ReadCloser, *media.ImageMeta, error) {
	var thumbID string
	m, err := s.dal.Media(mediaID)
	// If we didn't find it in our data store, assume it's a legacy media segment and that the id is consistent
	if errors.Cause(err) != dal.ErrNotFound {
		thumbID = mediaID.String()
	} else if err != nil {
		return nil, nil, err
	} else {
		thumbID = thumbnailID(m)
	}
	rc, meta, err := s.imageService.GetReader(thumbID, size)
	if errors.Cause(err) == media.ErrNotFound {
		// Attempt a placholder fallback
		rc, meta, err = s.imageService.GetReader(placeholderID(m), size)
		if err != nil {
			return nil, nil, err
		}
	} else if err != nil {
		return nil, nil, err
	}
	return rc, meta, err
}

func (s *service) ExpiringURL(ctx context.Context, mediaID dal.MediaID, exp time.Duration) (string, error) {
	return s.binaryService.ExpiringURL(mediaID.String(), exp)
}

func thumbnailID(m *dal.Media) string {
	t, err := mime.ParseType(m.MimeType)
	if err != nil {
		golog.Errorf("Unable to parse content type for media %s: %s - %s", m.ID, t, err)
		return m.ID.String() + thumbnailSuffix
	}
	if t.Type == "image" {
		return m.ID.String()
	}
	return m.ID.String() + thumbnailSuffix
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
