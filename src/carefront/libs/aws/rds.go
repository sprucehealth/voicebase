package aws

import (
	"encoding/xml"
	"net/url"
	"strconv"
	"time"

	"github.com/bmizerany/aws4"
)

type DescribeDBLogFilesResponse struct {
	RequestId          string                      `xml:"ResponseMetadata>RequestId"`
	Marker             string                      `xml:"DescribeDBLogFilesResult>Marker"`
	DescribeDBLogFiles []DescribeDBLogFilesDetails `xml:"DescribeDBLogFilesResult>DescribeDBLogFiles>DescribeDBLogFilesDetails"`
}

type DescribeDBLogFilesDetails struct {
	LastWritten int64 // timestamp in ms
	LogFileName string
	Size        int64
}

type DownloadDBLogFilePortionResponse struct {
	RequestId             string `xml:"ResponseMetadata>RequestId"`
	Marker                string `xml:"DownloadDBLogFilePortionResult>Marker"`
	LogFileData           string `xml:"DownloadDBLogFilePortionResult>LogFileData"`
	AdditionalDataPending bool   `xml:"DownloadDBLogFilePortionResult>AdditionalDataPending"`
}

type RDS struct {
	Region
	Client *aws4.Client
}

func (rds *RDS) Request(action string, args url.Values, response interface{}) error {
	if args == nil {
		args = url.Values{}
	}
	if args.Get("Version") == "" {
		args.Set("Version", "2013-02-12")
	}
	if args.Get("Timestamp") == "" && args.Get("Expires") == "" {
		args.Set("Timestamp", time.Now().In(time.UTC).Format(time.RFC3339))
	}
	args.Set("Action", action)
	res, err := rds.Client.PostForm(rds.RDSEndpoint, args)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return ErrBadStatusCode(res.StatusCode)
	}
	dec := xml.NewDecoder(res.Body)
	return dec.Decode(response)
}

/*
Return a list of DB log files for the DB instance, dbInstanceId.

FileLastWritten: Filters the available log files for files written since the specified date, in POSIX timestamp format.

FileSize: Filters the available log files for files larger than the specified size.

FilenameContains: Filters the available log files for log file names that contain the specified string.

Marker:	The pagination token provided in the previous request. If this parameter is specified the response includes only records beyond the marker, up to MaxRecords.

MaxRecords:	The maximum number of records to include in the response. If more records exist than the specified MaxRecords value, a pagination token called a marker is included in the response so that the remaining results can be retrieved.
*/
func (rds *RDS) DescribeDBLogFiles(dbInstanceId string, fileLastWritten, fileSize int64, filenameContains, marker string, maxRecords int) (*DescribeDBLogFilesResponse, error) {
	args := url.Values{}
	args.Set("DBInstanceIdentifier", dbInstanceId)
	if fileLastWritten > 0 {
		args.Set("FileLastWritten", strconv.FormatInt(fileLastWritten, 64))
	}
	if fileSize > 0 {
		args.Set("FileSize", strconv.FormatInt(fileSize, 64))
	}
	if filenameContains != "" {
		args.Set("FilenameContains", filenameContains)
	}
	if marker != "" {
		args.Set("Marker", marker)
	}
	if maxRecords > 0 {
		args.Set("MaxRecords", strconv.Itoa(maxRecords))
	}
	res := &DescribeDBLogFilesResponse{}
	return res, rds.Request("DescribeDBLogFiles", args, res)
}

// Downloads the last line of the specified log file (logFileName) from the
// database specified by dbInstanceId. Marker is the pagination token
// provided in a previous request, and numberOfLines is the number of lines
// remaining to be downloaded.
func (rds *RDS) DownloadDBLogFilePortion(dbInstanceId string, logFileName, marker string, numberOfLines int) (*DownloadDBLogFilePortionResponse, error) {
	args := url.Values{}
	args.Set("DBInstanceIdentifier", dbInstanceId)
	args.Set("LogFileName", logFileName)
	if marker != "" {
		args.Set("Marker", marker)
	}
	if numberOfLines > 0 {
		args.Set("NumberOfLines", strconv.Itoa(numberOfLines))
	}
	res := &DownloadDBLogFilePortionResponse{}
	return res, rds.Request("DownloadDBLogFilePortion", args, res)

}
