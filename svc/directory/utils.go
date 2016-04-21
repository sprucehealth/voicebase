package directory

import (
	"fmt"

	"github.com/sprucehealth/backend/libs/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// EntityIDs is a convenience method for retrieving ID's from a list
// Note: This could be made more gneeric using reflection but don't want the performance cost
func EntityIDs(es []*Entity) []string {
	ids := make([]string, len(es))
	for i, e := range es {
		ids[i] = e.ID
	}
	return ids
}

var (
	ErrEntityNotFound = errors.New("entity not found")
)

// SingleEntity returns a single entity for the given lookup request. If just 1 entity is not found an error is returned.
func SingleEntity(ctx context.Context, client DirectoryClient, req *LookupEntitiesRequest) (*Entity, error) {
	res, err := client.LookupEntities(ctx, req)
	if err != nil && grpc.Code(err) == codes.NotFound {
		return nil, ErrEntityNotFound
	} else if err != nil {
		return nil, errors.Trace(err)
	} else if len(res.Entities) == 0 {
		return nil, ErrEntityNotFound
	} else if len(res.Entities) != 1 {
		return nil, fmt.Errorf("expected single entity but got %d", len(res.Entities))
	}
	return res.Entities[0], nil
}

// SingleEntityByContact returns a single entity for a given contact value. If just 1 entity not found error is returned.
func SingleEntityByContact(ctx context.Context, client DirectoryClient, req *LookupEntitiesByContactRequest) (*Entity, error) {
	res, err := client.LookupEntitiesByContact(ctx, req)
	if err != nil && grpc.Code(err) == codes.NotFound {
		return nil, ErrEntityNotFound
	} else if err != nil {
		return nil, errors.Trace(err)
	} else if len(res.Entities) == 0 {
		return nil, ErrEntityNotFound
	} else if len(res.Entities) != 1 {
		return nil, fmt.Errorf("expected single entity but got %d", len(res.Entities))
	}

	return res.Entities[0], nil
}
