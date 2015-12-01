#### Overview
----
The Routing Service is responsible for routing messages to/from external channels after resolving external channel information via the directory service. 

#### Generating service definition and models
----
You should have the following pieces of software setup on your computer to to run the command below:
- [Protocol Buffers compiler](https://github.com/google/protobuf)
- [Protocol Buffers for Go with Gadgets](https://github.com/gogo/protobuf)

Run the following command to generate models for the routing service. 
```
	protoc --gogo_out=plugins=grpc:. *.proto
```
