#### Overview
----
The Excomms Service is responsible for sending and receiving messages over external channels like voice, sms, email, etc. It is also responsible for communicating unsubscribes to the routing service so that the routing service can know which external channels to direct messages over.

#### Generating service definition and models
----
You should have the following pieces of software setup on your computer to to run the command below:
- [Protocol Buffers compiler](https://github.com/google/protobuf)
- [Protocol Buffers for Go with Gadgets](https://github.com/gogo/protobuf)

Run the following command to generate models for the excomms service. 
```
	protoc --gogo_out=plugins=grpc:. *.proto
```
