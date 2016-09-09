package main

import (
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/libs/golog"
)

func main() {
	app := boot.NewApp()
	flagImagesToKeep := flag.Int("keep", 700, "Maximum number of images to keep")
	flagPrefix := flag.String("prefix", "master-", "Tag prefix to match")
	flag.Parse()

	awsSess, err := app.AWSSession()
	if err != nil {
		golog.Fatalf(err.Error())
	}
	cr := ecr.New(awsSess)
	out, err := cr.DescribeRepositories(&ecr.DescribeRepositoriesInput{})
	if err != nil {
		golog.Fatalf(err.Error())
	}
	for _, r := range out.Repositories {
		fmt.Printf("Repository %s\n", *r.RepositoryName)
		var imageTags []string
		var nextToken *string
		for {
			out, err := cr.ListImages(&ecr.ListImagesInput{
				RepositoryName: r.RepositoryName,
				NextToken:      nextToken,
			})
			if err != nil {
				golog.Fatalf(err.Error())
			}
			for _, i := range out.ImageIds {
				if i.ImageTag != nil && strings.HasPrefix(*i.ImageTag, *flagPrefix) {
					imageTags = append(imageTags, *i.ImageTag)
				}
			}
			if out.NextToken == nil {
				break
			}
			nextToken = out.NextToken
		}
		if len(imageTags) > *flagImagesToKeep {
			sort.Strings(imageTags)

			n := len(imageTags) - *flagImagesToKeep
			toDelete := make([]*ecr.ImageIdentifier, n)
			for i, tag := range imageTags[:n] {
				fmt.Printf("\tdeleting %s\n", tag)
				toDelete[i] = &ecr.ImageIdentifier{ImageTag: aws.String(tag)}
			}
			// Delete in batches of 100 which is the max size for BatchDelete
			for len(toDelete) > 0 {
				ids := toDelete
				if len(ids) > 100 {
					ids = ids[:100]
				}
				toDelete = toDelete[len(ids):]
				_, err = cr.BatchDeleteImage(&ecr.BatchDeleteImageInput{
					RepositoryName: r.RepositoryName,
					ImageIds:       ids,
				})
				if err != nil {
					golog.Fatalf(err.Error())
				}
			}
		}
	}
}
