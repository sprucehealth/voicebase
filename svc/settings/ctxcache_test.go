package settings

import (
	"testing"

	"github.com/sprucehealth/backend/libs/test"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type testClient struct {
	gets int
	sets int
}

func (testClient) RegisterConfigs(ctx context.Context, in *RegisterConfigsRequest, opts ...grpc.CallOption) (*RegisterConfigsResponse, error) {
	return &RegisterConfigsResponse{}, nil
}

func (testClient) GetConfigs(ctx context.Context, in *GetConfigsRequest, opts ...grpc.CallOption) (*GetConfigsResponse, error) {
	return &GetConfigsResponse{}, nil
}

func (tc *testClient) SetValue(ctx context.Context, in *SetValueRequest, opts ...grpc.CallOption) (*SetValueResponse, error) {
	tc.sets++
	return &SetValueResponse{}, nil
}

func (tc *testClient) GetValues(ctx context.Context, in *GetValuesRequest, opts ...grpc.CallOption) (*GetValuesResponse, error) {
	tc.gets += len(in.Keys)
	res := &GetValuesResponse{}
	values := map[ConfigKey]*Value{
		ConfigKey{Key: "foo"}: {
			Key:  &ConfigKey{Key: "foo"},
			Type: ConfigType_STRING_LIST,
		},
		ConfigKey{Key: "bar"}: {
			Key:  &ConfigKey{Key: "bar"},
			Type: ConfigType_BOOLEAN,
		},
	}
	if in.NodeID == "node1" {
		for _, k := range in.Keys {
			if v := values[*k]; v != nil {
				res.Values = append(res.Values, v)
			}
		}
	}
	return res, nil
}

func TestContextCache(t *testing.T) {
	tc := &testClient{}
	c := NewContextCacheClient(tc)
	ctx := context.Background()
	ctx = InitContextCache(ctx)

	res, err := c.GetValues(ctx, &GetValuesRequest{
		NodeID: "node1",
		Keys: []*ConfigKey{
			{Key: "foo"},
		},
	})
	test.OK(t, err)
	test.Equals(t, 1, len(res.Values))
	test.Equals(t, 1, tc.gets)
	test.Equals(t, &ConfigKey{Key: "foo"}, res.Values[0].Key)
	test.Equals(t, ConfigType_STRING_LIST, res.Values[0].Type)

	// Should be cached now so tc.gets doesn't change

	res, err = c.GetValues(ctx, &GetValuesRequest{
		NodeID: "node1",
		Keys: []*ConfigKey{
			{Key: "foo"},
		},
	})
	test.OK(t, err)
	test.Equals(t, 1, len(res.Values))
	test.Equals(t, 1, tc.gets)
	test.Equals(t, &ConfigKey{Key: "foo"}, res.Values[0].Key)
	test.Equals(t, ConfigType_STRING_LIST, res.Values[0].Type)

	// One value from cache, one value from service

	res, err = c.GetValues(ctx, &GetValuesRequest{
		NodeID: "node1",
		Keys: []*ConfigKey{
			{Key: "foo"},
			{Key: "bar"},
		},
	})
	test.OK(t, err)
	test.Equals(t, 2, len(res.Values))
	test.Equals(t, 2, tc.gets)
	test.Equals(t, &ConfigKey{Key: "bar"}, res.Values[0].Key)
	test.Equals(t, ConfigType_BOOLEAN, res.Values[0].Type)
	test.Equals(t, &ConfigKey{Key: "foo"}, res.Values[1].Key)
	test.Equals(t, ConfigType_STRING_LIST, res.Values[1].Type)

	// New value should be in cache

	res, err = c.GetValues(ctx, &GetValuesRequest{
		NodeID: "node1",
		Keys: []*ConfigKey{
			{Key: "bar"},
		},
	})
	test.OK(t, err)
	test.Equals(t, 1, len(res.Values))
	test.Equals(t, 2, tc.gets)
	test.Equals(t, &ConfigKey{Key: "bar"}, res.Values[0].Key)
	test.Equals(t, ConfigType_BOOLEAN, res.Values[0].Type)

	// Setting a value should add it to the cache

	_, err = c.SetValue(ctx, &SetValueRequest{
		NodeID: "node1",
		Value: &Value{
			Key:  &ConfigKey{Key: "foo"},
			Type: ConfigType_MULTI_SELECT,
		},
	})
	test.OK(t, err)
	test.Equals(t, 1, tc.sets)

	// Updated value should be in cache

	res, err = c.GetValues(ctx, &GetValuesRequest{
		NodeID: "node1",
		Keys: []*ConfigKey{
			{Key: "foo"},
		},
	})
	test.OK(t, err)
	test.Equals(t, 1, len(res.Values))
	test.Equals(t, 2, tc.gets)
	test.Equals(t, &ConfigKey{Key: "foo"}, res.Values[0].Key)
	test.Equals(t, ConfigType_MULTI_SELECT, res.Values[0].Type)
}
