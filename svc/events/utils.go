package events

import (
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/errors"
)

func getAccountIDFromEC2Metadata(awsSession *session.Session) (string, error) {
	metadataClient := ec2metadata.New(awsSession)
	doc, err := metadataClient.GetInstanceIdentityDocument()
	if err != nil {
		return "", errors.Errorf("Unable to get instance identity document from ec2 metadat: %s", err)
	}

	return doc.AccountID, nil
}

func nameOfEvent(m interface{}) string {
	v := reflect.ValueOf(m)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v.Type().Name()
}

func basePackageOfEvent(m interface{}) string {
	return path.Base(reflect.TypeOf(m).PkgPath())
}

func resourceNameFromEvent(m interface{}) string {
	return fmt.Sprintf("%s-%s-%s", environment.GetCurrent(), basePackageOfEvent(m), strings.ToLower(nameOfEvent(m)))
}

func resourceNameFromARN(arn string) (string, error) {
	idx := strings.LastIndex(arn, ":")
	if idx == -1 {
		return "", errors.Errorf("resource name not found in topic arn %s", arn)
	}

	return arn[idx+1:], nil
}

func newInstanceFromType(t reflect.Type) interface{} {
	var typ reflect.Type
	if t.Kind() == reflect.Ptr {
		typ = t.Elem()
	} else {
		typ = t
	}

	return reflect.New(typ).Interface()
}
