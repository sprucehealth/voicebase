package mock

import (
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/firehose/firehoseiface"
)

// Firehose mocks out the functionality of the kinesis firehose client for use in tests
type Firehose struct {
	*Expector
	firehoseiface.FirehoseAPI
	// Outputs should be set to stage return calls from the corresponding method
	PutRecordBatchOutputs []*firehose.PutRecordBatchOutput
	PutRecordBatchErrs    []error
}

// PutRecordBatch is a mocked implementation
func (f *Firehose) PutRecordBatch(in *firehose.PutRecordBatchInput) (*firehose.PutRecordBatchOutput, error) {
	defer f.Record(in)
	out := f.PutRecordBatchOutputs[0]
	f.PutRecordBatchOutputs = f.PutRecordBatchOutputs[1:]

	var err error
	f.PutRecordBatchErrs, err = NextError(f.PutRecordBatchErrs)
	return out, err
}
