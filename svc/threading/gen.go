package threading

//go:generate protoc --gogoslick_out=plugins=grpc:. --proto_path=$GOPATH/src:. svc.proto
// TODO go:generate sed -i "" s/grpc\.Invoke/grpcInvoke/g ./svc.pb.go
//go:generate gofmt -w ./svc.pb.go

// func init() {
// 	// Wrap server methods with tracing
// 	for i, m := range _Threads_serviceDesc.Methods {
// 		methodName := m.MethodName
// 		oldHandler := m.Handler
// 		m.Handler = func(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
// 			ctx, tr := trace.RPCServerContext(ctx, _Threads_serviceDesc.ServiceName, methodName)
// 			// TODO record start time in trace
// 			defer func() {
// 				// TODO record end time in trace
// 				tr.Finish()
// 			}()
// 			return oldHandler(srv, ctx, dec)
// 		}
// 		_Threads_serviceDesc.Methods[i] = m
// 	}
// }

// func grpcInvoke(ctx context.Context, method string, args, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
// 	ctx = trace.RPCClientContext(ctx)
// 	// TODO: record start time in trace
// 	// TODO: defer record end time in trace
// 	return grpc.Invoke(ctx, method, args, reply, cc, opts...)
// }
