package server

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/media"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// go vet doesn't like that the first argument to grpcErrorf is not a string so alias the function with a different name :(
func grpcErrorf(c codes.Code, format string, a ...interface{}) error {
	if c == codes.Internal {
		golog.LogDepthf(1, golog.ERR, "Media - Internal GRPC Error: %s", fmt.Sprintf(format, a...))
	}
	return grpcErrf(c, format, a...)
}

var grpcErrf = grpc.Errorf

type server struct {
	dl             dal.DAL
	mediaAPIDomain string
}

// New returns an initialized instance of server
func New(dl dal.DAL, mediaAPIDomain string) media.MediaServer {
	srv := &server{
		dl:             dl,
		mediaAPIDomain: mediaAPIDomain,
	}
	return srv
}

func (s *server) MediaInfos(ctx context.Context, rd *media.MediaInfosRequest) (*media.MediaInfosResponse, error) {
	var err error
	ids := make([]dal.MediaID, len(rd.MediaIDs))
	for i, id := range rd.MediaIDs {
		ids[i], err = dal.ParseMediaID(id)
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "Unable to parse media ID %s", id)
		}
	}
	m, err := s.dl.Medias(ids)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	rms, err := s.transformMediasToResponse(m)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	mMap := make(map[string]*media.MediaInfo, len(rms))
	for _, rm := range rms {
		mMap[rm.ID] = rm
	}
	return &media.MediaInfosResponse{
		MediaInfos: mMap,
	}, nil
}
