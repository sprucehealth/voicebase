package ec2

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"carefront/libs/aws"
)

const apiVersion = "2013-10-15"

type EC2 struct {
	aws.Region
	Client *aws.Client
	host   string
}

func (ec2 *EC2) Get(action string, params url.Values, response interface{}) error {
	if ec2.Client.HttpClient == nil {
		ec2.Client.HttpClient = http.DefaultClient
	}
	if ec2.host == "" {
		if u, err := url.Parse(ec2.Region.EC2Endpoint); err != nil {
			return err
		} else {
			ec2.host = u.Host
		}
	}
	params.Set("Action", action)
	params.Set("Version", apiVersion)
	params.Set("Timestamp", time.Now().In(time.UTC).Format(time.RFC3339))
	ec2.sign("GET", "/", params, ec2.host)
	req, err := http.NewRequest("GET", ec2.Region.EC2Endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	res, err := ec2.Client.HttpClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		defer res.Body.Close()
		return ParseErrorResponse(res)
	}
	defer res.Body.Close()
	dec := xml.NewDecoder(res.Body)
	return dec.Decode(response)
}

func (ec2 *EC2) CreateTags(resourceIds []string, tags map[string]string) error {
	params := url.Values{}
	for i, id := range resourceIds {
		params.Set(fmt.Sprintf("ResourceId.%d", i+1), id)
	}
	i := 1
	for key, value := range tags {
		params.Set(fmt.Sprintf("Tag.%d.Key", i), key)
		params.Set(fmt.Sprintf("Tag.%d.Value", i), value)
		i++
	}
	res := &CreateTagsResponse{}
	err := ec2.Get("CreateTags", params, res)
	if err != nil {
		return err
	}
	if !res.Return {
		return fmt.Errorf("ec2:CreateTags failed with return==false")
	}
	return nil
}

func (ec2 *EC2) DescribeInstances(ids []string, maxResults int, nextToken string, filters map[string][]string) (*DescribeInstancesResponse, error) {
	params := url.Values{}
	for i, id := range ids {
		params.Set(fmt.Sprintf("InstanceId.%d", i+1), id)
	}
	if maxResults > 0 {
		params.Set("MaxResults", strconv.Itoa(maxResults))
	}
	if nextToken != "" {
		params.Set("NextToken", nextToken)
	}
	i := 1
	for name, values := range filters {
		params.Set(fmt.Sprintf("Filter.%d.Name", i), name)
		for j, val := range values {
			params.Set(fmt.Sprintf("Filter.%d.Value.%d", i, j+1), val)
		}
		i++
	}
	res := &DescribeInstancesResponse{}
	err := ec2.Get("DescribeInstances", params, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
