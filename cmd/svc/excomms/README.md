# ExComms Service

### Overview
The goal of this service is to process inbound messages from a variety of different channels (email, sms, fax, etc) and publish the message for any interested parties to consume. 

It is also responsible for all unified communications related functionality, such as provisioning phone numbers, initiating phone calls between provider and external entity, processing unsubscribes, etc.


### Running locally
Check out `./excomms --help` to see a list of arguments to support. Note that `excomms_api_url` refers to the public facing API url for the excomms API service.