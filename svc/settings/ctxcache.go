package settings

import (
	"github.com/sprucehealth/backend/libs/conc"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type cacheCtxKey struct{}

type cacheKey struct {
	nodeID    string
	configKey ConfigKey
}

// InitContextCache initializes the context with the needed machanisms for caching
func InitContextCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, cacheCtxKey{}, conc.NewMap())
}

func contextCache(ctx context.Context) *conc.Map {
	cm, _ := ctx.Value(cacheCtxKey{}).(*conc.Map)
	return cm
}

type contextCacheClient struct {
	c SettingsClient
}

// NewContextCacheClient returns a wrapped version of a settings client that uses a cache in the context.
// On each request InitContextCache should be called to add the cache map to the context.
func NewContextCacheClient(c SettingsClient) SettingsClient {
	return &contextCacheClient{c}
}

func (c *contextCacheClient) RegisterConfigs(ctx context.Context, in *RegisterConfigsRequest, opts ...grpc.CallOption) (*RegisterConfigsResponse, error) {
	return c.c.RegisterConfigs(ctx, in, opts...)
}

func (c *contextCacheClient) GetConfigs(ctx context.Context, in *GetConfigsRequest, opts ...grpc.CallOption) (*GetConfigsResponse, error) {
	return c.c.GetConfigs(ctx, in, opts...)
}

func (c *contextCacheClient) SetValue(ctx context.Context, in *SetValueRequest, opts ...grpc.CallOption) (*SetValueResponse, error) {
	res, err := c.c.SetValue(ctx, in, opts...)
	if err == nil {
		contextCache(ctx).Set(cacheKey{nodeID: in.NodeID, configKey: *in.Value.Key}, in.Value)
	}
	return res, err
}

func (c *contextCacheClient) GetValues(ctx context.Context, in *GetValuesRequest, opts ...grpc.CallOption) (*GetValuesResponse, error) {
	cm := contextCache(ctx)
	toLookup := make([]*ConfigKey, 0, len(in.Keys))
	fromCache := make([]*Value, 0, len(in.Keys))
	for _, k := range in.Keys {
		vi := cm.Get(cacheKey{nodeID: in.NodeID, configKey: *k})
		if vi != nil {
			fromCache = append(fromCache, vi.(*Value))
		} else {
			toLookup = append(toLookup, k)
		}
	}

	var res *GetValuesResponse
	if len(toLookup) != 0 {
		in.Keys = toLookup
		var err error
		res, err = c.c.GetValues(ctx, in, opts...)
		if err != nil {
			return nil, err
		}
		for _, v := range res.Values {
			cm.Set(cacheKey{nodeID: in.NodeID, configKey: *v.Key}, v)
		}
	} else {
		res = &GetValuesResponse{
			Values: make([]*Value, 0, len(fromCache)),
		}
	}

	for _, v := range fromCache {
		res.Values = append(res.Values, v)
	}
	return res, nil
}
