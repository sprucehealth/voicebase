package main

import (
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
	"fmt"
	"time"
)

func main() {
	auth, err := aws.EnvAuth()
	if err != nil {
		panic(err.Error())
	}

	s3Access := s3.New(auth, aws.USWest) 
	s3Bucket := s3Access.Bucket("carefront-cases")

	additionalHeaders := map[string][]string {
		"x-amz-server-side-encryption" : {"AES256"},	
	}

	err = s3Bucket.Put("testing/testingAnotherFile.txt", make([]byte, 1000), "binary/octet-stream", s3.BucketOwnerFull, additionalHeaders)
	if err != nil {
		panic(err.Error())
	}
	
	signedUrl := s3Bucket.SignedURL("testing/testing1234", time.Now().Add(10*time.Second))
	fmt.Println(signedUrl)
}
