package cache

import (
	"sort"

	"context"

	"github.com/golang/protobuf/proto"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"google.golang.org/grpc"
)

// EntityGroupCache represents a thread safe map of key to entity objects we have encountered that map to that key
// This cache is intended to be used in conjunction with request context
type EntityGroupCache struct {
	cMap *conc.Map
}

// NewEntityGroupCache returns an initialized instance of NewEntityGroupCache
func NewEntityGroupCache(ini map[string][]*directory.Entity) *EntityGroupCache {
	cMap := conc.NewMap()
	for k, es := range ini {
		cMap.Set(k, es)
	}
	return &EntityGroupCache{
		cMap: cMap,
	}
}

// Get returns the entity mapped to the provided key and nil if it does not exist
func (c *EntityGroupCache) Get(key string) []*directory.Entity {
	esi := c.cMap.Get(key)
	if esi == nil {
		return nil
	}

	ents, ok := esi.([]*directory.Entity)
	if !ok {
		golog.Errorf("EntityGroupCache: Found %+v mapped to %s but failed conversion from interface to []*Entity. Removing from cache.", esi, key)
		go c.cMap.Delete(key)
		return nil
	}
	return ents
}

// GetOnly returns the only entity mapped to the provided key, nil if no entities exist or more than 1 is mapped to the key
// An error is logged in the case that multiple entities are mapped to a key that is provided
func (c *EntityGroupCache) GetOnly(key string) *directory.Entity {
	ents := c.Get(key)
	if len(ents) == 0 {
		return nil
	}
	if len(ents) > 1 {
		golog.Errorf("Expected only 1 entity to be present in EntityGroupCache for key %s, but found %v", key, ents)
		return nil
	}
	return ents[0]
}

// Set maps the provided entities to the provided key
func (c *EntityGroupCache) Set(key string, ents []*directory.Entity) {
	c.cMap.Set(key, ents)
}

// Delete removes the provided key from the cache
func (c *EntityGroupCache) Delete(key string) {
	c.cMap.Delete(key)
}

// Clear removes all entries from the cache
func (c *EntityGroupCache) Clear() {
	c.cMap.Clear()
}

// CachedClient implements a request based context cache of entities and entity groups
// NOTE: We intentionally are not using a composit type here.
// Any person updating the directory service client should manually implement
// the new call here and consider the cache busting implications
type CachedClient struct {
	dc            directory.DirectoryClient
	statCacheHit  *metrics.Counter
	statCacheMiss *metrics.Counter
	statCacheBust *metrics.Counter
}

// NewCachedClient returns an implementation of CachedClient
func NewCachedClient(dc directory.DirectoryClient, metricsRegistry metrics.Registry) directory.DirectoryClient {
	cc := &CachedClient{
		dc:            dc,
		statCacheHit:  metrics.NewCounter(),
		statCacheMiss: metrics.NewCounter(),
		statCacheBust: metrics.NewCounter(),
	}
	metricsRegistry.Add("hit", cc.statCacheHit)
	metricsRegistry.Add("miss", cc.statCacheMiss)
	metricsRegistry.Add("bust", cc.statCacheBust)
	return cc
}

// For now just use the marshalled message as the cache key, this could perhaps be optimized.
func calculateMessageSig(m proto.Message) string {
	sig, err := proto.Marshal(m)
	if err != nil {
		return ""
	}
	return string(sig)
}

func (c *CachedClient) checkCache(ctx context.Context, key string) []*directory.Entity {
	if key == "" {
		return nil
	}
	cache := Entities(ctx)
	if cache == nil {
		golog.Warningf("CachedClient: No entity cache present in context")
		return nil
	}
	es := cache.Get(key)
	if es != nil && c.statCacheHit != nil {
		c.statCacheHit.Inc(1)
	} else if c.statCacheMiss != nil {
		c.statCacheMiss.Inc(1)
	}
	return cache.Get(key)
}

// TODO: It's possible we could extend this client wrapping construct to cache sig to proto.Message for generic request cacheing
func (c *CachedClient) cache(ctx context.Context, key string, ents []*directory.Entity) {
	cache := Entities(ctx)
	if cache == nil {
		golog.Warningf("CachedClient: No entity cache present in context")
		return
	}
	cache.Set(key, ents)
}

// EntityStatusSlice aliases a slice on EntityStatus records
type EntityStatusSlice []directory.EntityStatus

func (p EntityStatusSlice) Len() int           { return len(p) }
func (p EntityStatusSlice) Less(i, j int) bool { return p[i] < p[j] }
func (p EntityStatusSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// EntityTypeSlice aliases a slice on EntityType records
type EntityTypeSlice []directory.EntityType

func (p EntityTypeSlice) Len() int           { return len(p) }
func (p EntityTypeSlice) Less(i, j int) bool { return p[i] < p[j] }
func (p EntityTypeSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// EntityInformationSlice aliases a slice on EntityInformation records
type EntityInformationSlice []directory.EntityInformation

func (p EntityInformationSlice) Len() int           { return len(p) }
func (p EntityInformationSlice) Less(i, j int) bool { return p[i] < p[j] }
func (p EntityInformationSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// TODO: Add sort.Interface support for gogoprotobuf generation
func normalizeLookupEntitiesRequest(r *directory.LookupEntitiesRequest) {
	sort.Sort(EntityStatusSlice(r.Statuses))
	sort.Sort(EntityTypeSlice(r.RootTypes))
	sort.Sort(EntityTypeSlice(r.ChildTypes))
	if r.RequestedInformation != nil {
		sort.Sort(EntityInformationSlice(r.RequestedInformation.EntityInformation))
	}
	if r.LookupKeyType == directory.LookupEntitiesRequest_BATCH_ENTITY_ID {
		sort.Strings(r.GetBatchEntityID().IDs)
	}
}

// Cache Busters
func (c *CachedClient) bustCache(ctx context.Context) {
	cache := Entities(ctx)
	if cache == nil {
		golog.Warningf("CachedClient: No entity cache present in context")
		return
	}
	if c.statCacheBust != nil {
		c.statCacheBust.Inc(1)
	}
	cache.Clear()
}

func (c *CachedClient) CreateContact(ctx context.Context, in *directory.CreateContactRequest, opts ...grpc.CallOption) (*directory.CreateContactResponse, error) {
	c.bustCache(ctx)
	return c.dc.CreateContact(ctx, in, opts...)
}

func (c *CachedClient) CreateContacts(ctx context.Context, in *directory.CreateContactsRequest, opts ...grpc.CallOption) (*directory.CreateContactsResponse, error) {
	c.bustCache(ctx)
	return c.dc.CreateContacts(ctx, in, opts...)
}

func (c *CachedClient) CreateEntityDomain(ctx context.Context, in *directory.CreateEntityDomainRequest, opts ...grpc.CallOption) (*directory.CreateEntityDomainResponse, error) {
	c.bustCache(ctx)
	return c.dc.CreateEntityDomain(ctx, in, opts...)
}

func (c *CachedClient) CreateEntity(ctx context.Context, in *directory.CreateEntityRequest, opts ...grpc.CallOption) (*directory.CreateEntityResponse, error) {
	c.bustCache(ctx)
	return c.dc.CreateEntity(ctx, in, opts...)
}

func (c *CachedClient) CreateExternalIDs(ctx context.Context, in *directory.CreateExternalIDsRequest, opts ...grpc.CallOption) (*directory.CreateExternalIDsResponse, error) {
	c.bustCache(ctx)
	return c.dc.CreateExternalIDs(ctx, in, opts...)
}

func (c *CachedClient) CreateMembership(ctx context.Context, in *directory.CreateMembershipRequest, opts ...grpc.CallOption) (*directory.CreateMembershipResponse, error) {
	c.bustCache(ctx)
	return c.dc.CreateMembership(ctx, in, opts...)
}

func (c *CachedClient) DeleteContacts(ctx context.Context, in *directory.DeleteContactsRequest, opts ...grpc.CallOption) (*directory.DeleteContactsResponse, error) {
	c.bustCache(ctx)
	return c.dc.DeleteContacts(ctx, in, opts...)
}

func (c *CachedClient) DeleteEntity(ctx context.Context, in *directory.DeleteEntityRequest, opts ...grpc.CallOption) (*directory.DeleteEntityResponse, error) {
	c.bustCache(ctx)
	return c.dc.DeleteEntity(ctx, in, opts...)
}

func (c *CachedClient) ExternalIDs(ctx context.Context, in *directory.ExternalIDsRequest, opts ...grpc.CallOption) (*directory.ExternalIDsResponse, error) {
	return c.dc.ExternalIDs(ctx, in, opts...)
}

// LookupEntities implements a cached version of LookupEntities
func (c *CachedClient) LookupEntities(ctx context.Context, in *directory.LookupEntitiesRequest, opts ...grpc.CallOption) (*directory.LookupEntitiesResponse, error) {
	normalizeLookupEntitiesRequest(in)
	sig := calculateMessageSig(in)
	ents := c.checkCache(ctx, sig)
	if ents != nil {
		return &directory.LookupEntitiesResponse{
			Entities: ents,
		}, nil
	}
	resp, err := c.dc.LookupEntities(ctx, in, opts...)
	if err != nil {
		return resp, err
	}
	c.cache(ctx, sig, resp.Entities)
	return resp, nil
}

func (c *CachedClient) LookupEntitiesByContact(ctx context.Context, in *directory.LookupEntitiesByContactRequest, opts ...grpc.CallOption) (*directory.LookupEntitiesByContactResponse, error) {
	return c.dc.LookupEntitiesByContact(ctx, in, opts...)
}

func (c *CachedClient) LookupEntityDomain(ctx context.Context, in *directory.LookupEntityDomainRequest, opts ...grpc.CallOption) (*directory.LookupEntityDomainResponse, error) {
	return c.dc.LookupEntityDomain(ctx, in, opts...)
}

func (c *CachedClient) Profile(ctx context.Context, in *directory.ProfileRequest, opts ...grpc.CallOption) (*directory.ProfileResponse, error) {
	return c.dc.Profile(ctx, in, opts...)
}

func (c *CachedClient) SerializedEntityContact(ctx context.Context, in *directory.SerializedEntityContactRequest, opts ...grpc.CallOption) (*directory.SerializedEntityContactResponse, error) {
	return c.dc.SerializedEntityContact(ctx, in, opts...)
}

func (c *CachedClient) UpdateContacts(ctx context.Context, in *directory.UpdateContactsRequest, opts ...grpc.CallOption) (*directory.UpdateContactsResponse, error) {
	c.bustCache(ctx)
	return c.dc.UpdateContacts(ctx, in, opts...)
}

func (c *CachedClient) UpdateEntity(ctx context.Context, in *directory.UpdateEntityRequest, opts ...grpc.CallOption) (*directory.UpdateEntityResponse, error) {
	c.bustCache(ctx)
	return c.dc.UpdateEntity(ctx, in, opts...)
}

func (c *CachedClient) UpdateEntityDomain(ctx context.Context, in *directory.UpdateEntityDomainRequest, opts ...grpc.CallOption) (*directory.UpdateEntityDomainResponse, error) {
	return c.dc.UpdateEntityDomain(ctx, in, opts...)
}

func (c *CachedClient) UpdateProfile(ctx context.Context, in *directory.UpdateProfileRequest, opts ...grpc.CallOption) (*directory.UpdateProfileResponse, error) {
	c.bustCache(ctx)
	return c.dc.UpdateProfile(ctx, in, opts...)
}

func (c *CachedClient) CreateExternalLink(ctx context.Context, in *directory.CreateExternalLinkRequest, opts ...grpc.CallOption) (*directory.CreateExternalLinkResponse, error) {
	c.bustCache(ctx)
	return c.dc.CreateExternalLink(ctx, in, opts...)
}

func (c *CachedClient) DeleteExternalLink(ctx context.Context, in *directory.DeleteExternalLinkRequest, opts ...grpc.CallOption) (*directory.DeleteExternalLinkResponse, error) {
	c.bustCache(ctx)
	return c.dc.DeleteExternalLink(ctx, in, opts...)
}

func (c *CachedClient) LookupExternalLinksForEntity(ctx context.Context, in *directory.LookupExternalLinksForEntityRequest, opts ...grpc.CallOption) (*directory.LookupExternalLinksforEntityResponse, error) {
	return c.dc.LookupExternalLinksForEntity(ctx, in, opts...)
}
