package manager

import (
	"fmt"
	"reflect"

	"github.com/gogo/protobuf/proto"
)

const (
	protobuf = "protobuf"
)

func init() {
	mustRegisterSerializer(protobuf, &protoBufSerializer{})
}

type serializer interface {
	marshal(val interface{}) ([]byte, error)
	typed
}

var serializers = map[string]reflect.Type{}

func mustRegisterSerializer(typeName string, s serializer) {
	if typeName == "" {
		panic("typeName not set")
	}

	_, ok := serializers[typeName]
	if ok {
		panic(fmt.Sprintf("%s already defined in registry", typeName))
	}

	serializers[typeName] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(s)).Interface())
}

func serializerForType(typeName string) (serializer, error) {
	dataType, ok := serializers[typeName]
	if !ok {
		return nil, fmt.Errorf("Unable to get serializer for type: %s", typeName)
	}
	return reflect.New(dataType).Interface().(serializer), nil
}

type protoBufSerializer struct{}

func (protoBufSerializer) TypeName() string {
	return protobuf
}

func (protoBufSerializer) marshal(val interface{}) ([]byte, error) {
	msg, ok := val.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("Data not of type proto.Message as is expected. Got %T", val)
	}
	return proto.Marshal(msg)
}
