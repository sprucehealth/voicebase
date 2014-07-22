package cloudwatchlogs

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/libs/aws"
)

const apiVersion = "Logs_20140328."

type Client struct {
	aws.Region
	Client *aws.Client
	host   string
}

func (c *Client) do(action string, request, response interface{}) error {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	if err := enc.Encode(request); err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.Region.CloudWatchLogsEndpoint, buf)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("Content-Length", strconv.Itoa(buf.Len()))
	req.Header.Set("X-Amz-Target", apiVersion+string(action))
	res, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return parseErrorResponse(res)
	}
	// Some actions only use the StatusCode with no body
	if response == nil {
		return nil
	}

	dec := json.NewDecoder(res.Body)
	return dec.Decode(response)
}

// CreateLogGroup creates a new log group with the specified name. The name of
// the log group must be unique within a region for an AWS account.You can
// create up to 500 log groups per account.
//
// You must use the following guidelines when naming a log group:
// 	• Log group names can be between 1 and 512 characters long.
// 	• Allowed characters are a-z, A-Z, 0-9, '_' (underscore), '-' (hyphen),
// 	  '/' (forward slash), and '.' (period).
func (c *Client) CreateLogGroup(name string) error {
	return c.do("CreateLogGroup", &logGroupRequest{LogGroupName: name}, nil)
}

// CreateLogStream creates a new log stream in the specified log group. The
// name of the log stream must be unique within the log group. There is no
// limit on the number of log streams that can exist in a log group.
//
// You must use the following guidelines when naming a log stream:
// 	• Log stream names can be between 1 and 512 characters long.
// 	• The ':' colon character is not allowed.
func (c *Client) CreateLogStream(groupName, streamName string) error {
	return c.do("CreateLogStream", &logStreamRequest{LogGroupName: groupName, LogStreamName: streamName}, nil)
}

// DeleteLogGroup deletes the log group with the specified name and
// permanently deletes all the archived log events associated with it.
func (c *Client) DeleteLogGroup(name string) error {
	return c.do("DeleteLogGroup", &logGroupRequest{LogGroupName: name}, nil)
}

// DeleteLogStream deletes a log stream and permanently deletes all the archived
// log events associated with it.
func (c *Client) DeleteLogStream(groupName, streamName string) error {
	return c.do("DeleteLogStream", &logStreamRequest{LogGroupName: groupName, LogStreamName: streamName}, nil)
}

// DeleteMetricFilter deletes a metric filter associated with the specified log group
func (c *Client) DeleteMetricFilter(groupName, filterName string) error {
	return c.do("DeleteMetricFilter", &metricFilterRequest{LogGroupName: groupName, FilterName: filterName}, nil)
}

// DeleteRetentionPolicy deletes the retention policy of the specified log group. Log
// events would not expire if they belong to log groups without a retention policy
func (c *Client) DeleteRetentionPolicy(groupName string) error {
	return c.do("DeleteRetentionPolicy", &logGroupRequest{LogGroupName: groupName}, nil)
}

// DescribeLogGroups returns all the log groups that are associated with the AWS account
// making the request. The list returned in the response is ASCII-sorted by log group name.
//
// By default, this operation returns up to 50 log groups. If there are more log
// groups to list, the response would contain a nextToken value in the response
// body.You can also limit the number of log groups returned in the response by
// specifying the limit parameter in the request.
func (c *Client) DescribeLogGroups(prefix, nextToken string, limit int) (*LogGroups, error) {
	var res LogGroups
	if err := c.do("DescribeLogGroups", &describeLogGroupsRequest{Limit: intPtrIfNonZero(limit), LogGroupNamePrefix: prefix, NextToken: nextToken}, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// DescribeLogStreams returns all the log streams that are associated with the specified
// log group. The list returned in the response is ASCII-sorted by log stream name.
//
// By default, this operation returns up to 50 log streams. If there are more log streams
// to list, the response would contain a nextToken value in the response body.You can also
// limit the number of log streams returned in the response by specifying the limit
// parameter in the request.
func (c *Client) DescribeLogStreams(groupName, prefix, nextToken string, limit int) (*LogStreams, error) {
	var res LogStreams
	if err := c.do("DescribeLogStreams", &describeLogStreamsRequest{Limit: intPtrIfNonZero(limit), LogGroupName: groupName, LogStreamNamePrefix: prefix, NextToken: nextToken}, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// DescribeMetricFilters returns all the metrics filters associated with the specified
// log group. The list returned in the response is ASCII-sorted by filter name.
//
// By default, this operation returns up to 50 metric filters. If there are more metric
// filters to list, the response would contain a nextToken value in the response body.
// You can also limit the number of metric filters returned in the response by specifying
// the limit parameter in the request
func (c *Client) DescribeMetricFilters(groupName, prefix, nextToken string, limit int) (*MetricFilters, error) {
	var res MetricFilters
	if err := c.do("DescribeMetricFilters", &describeMetricFiltersRequest{Limit: intPtrIfNonZero(limit), LogGroupName: groupName, FilterNamePrefix: prefix, NextToken: nextToken}, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetLogEvents retrieves log events from the specified log stream.You can provide an
// optional time range to filter the results on the event timestamp.
//
// By default, this operation returns as much log events as can fit in a response size
// of 1MB, up to 10,000 log events. The response will always include a nextForwardToken
// and a nextBackwardToken in the response body.You can use any of these tokens in
// subsequent GetLogEvents requests to paginate through events in either forward or
// backward direction. You can also limit the number of log events returned in the
// response by specifying the limit parameter in the request.
func (c *Client) GetLogEvents(groupName, streamName string, startFromHead bool, startTime, endTime time.Time, nextToken string, limit int) (*Events, error) {
	req := &getLogEventsRequest{
		StartFromHead: startFromHead,
		LogGroupName:  groupName,
		LogStreamName: streamName,
		NextToken:     nextToken,
		Limit:         intPtrIfNonZero(limit),
	}
	if !startTime.IsZero() {
		req.StartTime = (Time{startTime}).UnixMilli()
	}
	if !endTime.IsZero() {
		req.EndTime = (Time{endTime}).UnixMilli()
	}
	var res Events
	if err := c.do("GetLogEvents", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// PutLogEvents uploads a batch of log events to the specified log stream.
//
// Every PutLogEvents request must include the sequenceToken obtained from the
// response of the previous request. An upload in a newly created log stream
// does not require a sequenceToken.
//
// The maximum batch size is 32,768 bytes, and this size is calculated as the sum of all event messages
// in UTF-8, plus 26 bytes for each log event.
// 	• None of the log events in the batch can be more than 2 hours in the future.
// 	• None of the log events in the batch can be older than 14 days or the retention period of the log group.
// 	• The log events in the batch must be in chronological ordered by their timestamp.
// 	• The maximum number of log events in a batch is 1,000.
func (c *Client) PutLogEvents(groupName, streamName string, events []*InputEvent, sequenceToken string) (nextSequenceToken string, err error) {
	req := &putLogEventsRequest{
		LogEvents:     events,
		LogGroupName:  groupName,
		LogStreamName: streamName,
		SequenceToken: sequenceToken,
	}
	var res putLogEventsResponse
	if err := c.do("PutLogEvents", req, &res); err != nil {
		return "", err
	}
	return res.NextSequenceToken, nil
}

// PutMetricFilter creates or updates a metric filter and associates it with the
// specified log group. Metric filters allow you to configure rules to extract
// metric data from log events ingested through PutLogEvents requests.
func (c *Client) PutMetricFilter(groupName, filterName, filterPattern string, transformations []*InputMetricTransformation) error {
	req := &putMetricFiltersRequest{
		LogGroupName:          groupName,
		FilterName:            filterName,
		FilterPattern:         filterPattern,
		MetricTransformations: transformations,
	}
	return c.do("PutMetricFilter", req, nil)
}

// PutRetentionPolicy sets the retention of the specified log group. A
// retention policy allows you to configure the number of days you want to
// retain log events in the specified log group.
//
// Possible number of days are 1, 3, 5, 7, 14, 30, 60, 90, 120, 150, 180,
// 365, 400, 545, 731, 1827, 3653.
func (c *Client) PutRetentionPolicy(groupName string, days int) error {
	req := &putRetentionPolicyRequest{
		LogGroupName:    groupName,
		RetentionInDays: days,
	}
	return c.do("PutRetentionPolicy", req, nil)
}

// TestMetricFilter tests the filter pattern of a metric filter against a
// sample of log event messages. You can use this operation to validate the
// correctness of a metric filter pattern.
//
// eventMessages must have a minimum of 1 item and a maximum of 50 items.
func (c *Client) TestMetricFilter(filterPattern string, eventMessages []string) (*FilterMatches, error) {
	req := &testMetricFilterRequest{
		FilterPattern:    filterPattern,
		LogEventMessages: eventMessages,
	}
	var res FilterMatches
	if err := c.do("TestMetricFilter", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
