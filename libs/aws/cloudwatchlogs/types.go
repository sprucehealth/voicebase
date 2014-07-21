package cloudwatchlogs

import (
	"strconv"
	"time"
)

type Time struct {
	time.Time
}

func (t Time) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(t.UnixMilli(), 10)), nil
}

func (t *Time) UnmarshalJSON(b []byte) error {
	s := string(b)
	switch s {
	case "", "null":
		*t = Time{time.Time{}}
		return nil
	}
	ms, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*t = Time{time.Unix(ms/1000, (ms%1000)*1e6)}
	return nil
}

func (t *Time) UnixMilli() int64 {
	return t.UnixNano() / 1e6
}

type LogGroup struct {
	ARN               string `json:"Arn"`
	CreationTime      Time
	LogGroupName      string
	MetricFilterCount int
	RetentionInDays   int
	StoredBytes       int64
}

type LogStream struct {
	ARN                 string `json:"Arn"`
	CreationTime        Time
	FirstEventTimestamp Time
	LastEventTimestamp  Time
	LastIngestionTime   Time
	LogStreamName       string
	StoredBytes         int64
	UploadSequenceToken string
}

type LogGroups struct {
	LogGroups []*LogGroup
	NextToken string
}

type LogStreams struct {
	LogStreams []*LogStream
	NextToken  string
}

type MetricTransformation struct {
	MetricName      string
	MetricNamespace string
	MetricValue     string
}

type MetricFilter struct {
	CreationTime          Time
	FilterName            string
	FilterPattern         string
	MetricTransformations []*MetricTransformation
}

type MetricFilters struct {
	MetricFilters []*MetricFilter
	NextToken     string
}

type Event struct {
	IngestionTime Time
	Timestamp     Time
	Message       string
}

type InputEvent struct {
	Timestamp Time   `json:"timestamp"`
	Message   string `json:"message"`
}

type InputMetricTransformation struct {
	MetricName      string `json:"metricName"`
	MetricNamespace string `json:"metricNamespace"`
	MetricValue     string `json:"metricValue"`
}

type Events struct {
	Events            []*Event
	NextBackwardToken string
	NextForwardToken  string
}

type FiltertMatch struct {
	EventMessage    string
	EventNumber     int
	ExtractedValues map[string]string
}

type FilterMatches struct {
	Matches []*FiltertMatch
}

type logGroupRequest struct {
	LogGroupName string `json:"logGroupName"`
}

type logStreamRequest struct {
	LogGroupName  string `json:"logGroupName"`
	LogStreamName string `json:"logStreamName"`
}

type metricFilterRequest struct {
	LogGroupName string `json:"logGroupName"`
	FilterName   string `json:"filterName"`
}

type describeLogGroupsRequest struct {
	LogGroupNamePrefix string `json:"logGroupNamePrefix,omitempty"`
	NextToken          string `json:"nextToken,omitempty"`
	Limit              *int   `json:"limit,omitempty"`
}

type describeLogStreamsRequest struct {
	LogGroupName        string `json:"logGroupName"`
	LogStreamNamePrefix string `json:"logStreamNamePrefix,omitempty"`
	NextToken           string `json:"nextToken,omitempty"`
	Limit               *int   `json:"limit,omitempty"`
}

type describeMetricFiltersRequest struct {
	LogGroupName     string `json:"logGroupName"`
	FilterNamePrefix string `json:"filterNamePrefix,omitempty"`
	NextToken        string `json:"nextToken,omitempty"`
	Limit            *int   `json:"limit,omitempty"`
}

type getLogEventsRequest struct {
	StartTime     Time   `json:"startTime"`
	EndTime       Time   `json:"endTime"`
	StartFromHead bool   `json:"startFromHead"`
	LogGroupName  string `json:"logGroupName"`
	LogStreamName string `json:"logStreamName"`
	NextToken     string `json:"nextToken,omitempty"`
	Limit         *int   `json:"limit,omitempty"`
}

type putLogEventsRequest struct {
	LogEvents     []*InputEvent `json:"logEvents"`
	LogGroupName  string        `json:"logGroupName"`
	LogStreamName string        `json:"logStreamName"`
	SequenceToken string        `json:"sequenceToken,omitempty"`
}

type putLogEventsResponse struct {
	NextSequenceToken string
}

type putMetricFiltersRequest struct {
	FilterName            string                       `json:"filterName"`
	FilterPattern         string                       `json:"filterPattern"`
	LogGroupName          string                       `json:"logGroupName"`
	MetricTransformations []*InputMetricTransformation `json:"metricTransformations"`
}

type putRetentionPolicyRequest struct {
	LogGroupName    string `json:"logGroupName"`
	RetentionInDays int    `json:"retentionInDays"`
}

type testMetricFilterRequest struct {
	FilterPattern    string   `json:"filterPattern"`
	LogEventMessages []string `json:"logEventMessages"`
}
