package mock

import "github.com/aws/aws-sdk-go/service/dynamodb"

// DynamoDB mocks out the functionality of the dynamodb client for use in tests
type DynamoDB struct {
	*Expector
	// Outputs should be set to stage return calls from the corresponding method
	BatchWriteItemOutputs []*dynamodb.BatchWriteItemOutput
	BatchWriteItemErrs    []error
	CreateTableOutputs    []*dynamodb.CreateTableOutput
	CreateTableErrs       []error
	DescribeTableOutputs  []*dynamodb.DescribeTableOutput
	DescribeTableErrs     []error
	GetItemOutputs        []*dynamodb.GetItemOutput
	GetItemErrs           []error
	QueryOutputs          []*dynamodb.QueryOutput
	QueryErrs             []error
}

// BatchWriteItem is a mocked implementation that returns the queued data
func (d *DynamoDB) BatchWriteItem(input *dynamodb.BatchWriteItemInput) (*dynamodb.BatchWriteItemOutput, error) {
	defer d.Record(input)
	out := d.BatchWriteItemOutputs[0]
	d.BatchWriteItemOutputs = d.BatchWriteItemOutputs[1:]

	var err error
	d.BatchWriteItemErrs, err = NextError(d.BatchWriteItemErrs)
	return out, err
}

// CreateTable is a mocked implementation that returns the queued data
func (d *DynamoDB) CreateTable(input *dynamodb.CreateTableInput) (*dynamodb.CreateTableOutput, error) {
	defer d.Record(input)
	out := d.CreateTableOutputs[0]
	d.CreateTableOutputs = d.CreateTableOutputs[1:]

	var err error
	d.CreateTableErrs, err = NextError(d.CreateTableErrs)
	return out, err
}

// DescribeTable is a mocked implementation that returns the queued data
func (d *DynamoDB) DescribeTable(input *dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error) {
	defer d.Record(input)
	out := d.DescribeTableOutputs[0]
	d.DescribeTableOutputs = d.DescribeTableOutputs[1:]

	var err error
	d.DescribeTableErrs, err = NextError(d.DescribeTableErrs)
	return out, err
}

// GetItem is a mocked implementation that returns the queued data
func (d *DynamoDB) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	defer d.Record(input)
	out := d.GetItemOutputs[0]
	d.GetItemOutputs = d.GetItemOutputs[1:]

	var err error
	d.GetItemErrs, err = NextError(d.GetItemErrs)
	return out, err
}

// Query is a mocked implementation that returns the queued data
func (d *DynamoDB) Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	defer d.Record(input)
	out := d.QueryOutputs[0]
	d.QueryOutputs = d.QueryOutputs[1:]

	var err error
	d.QueryErrs, err = NextError(d.QueryErrs)
	return out, err
}
