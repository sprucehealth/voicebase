package cmd

import (
	"flag"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/cmd/cli/sqsadmin/internal/config"
)

type listQueuesCmd struct {
	cnf *config.Config
	sqs sqsiface.SQSAPI
}

func NewListQueuesCmd(cnf *config.Config) (Command, error) {
	sqs, err := cnf.SQSClient()
	if err != nil {
		return nil, err
	}
	return &listQueuesCmd{
		cnf: cnf,
		sqs: sqs,
	}, nil
}

func (c *listQueuesCmd) Run(args []string) error {
	fs := flag.NewFlagSet("list_queues", flag.ExitOnError)
	prefix := fs.String("prefix", "", "Prefix for filtering queue names")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	if *prefix == "" {
		prefix = nil
	}

	resp, err := c.sqs.ListQueues(&sqs.ListQueuesInput{
		QueueNamePrefix: prefix,
	})
	if err != nil {
		return err
	}

	pprint(resp.String())
	return nil
}
