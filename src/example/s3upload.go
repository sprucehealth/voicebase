package main

import (
	"fmt"
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
)

func main() {
	auth, err := aws.EnvAuth()
	if err != nil {
		panic(err.Error())
	}

	s3Access := s3.New(auth, aws.USWest)
	s3Bucket := s3Access.Bucket("carefront-cases")

	listBucketResult, err := s3Bucket.List("1234/", "/", "", 100)
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("%q", listBucketResult.Contents)
}
