package boot

import (
	"flag"
	"sync"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/sprucehealth/backend/libs/awsutil"
)

type App struct {
	flags struct {
		debug        bool
		env          string
		awsAccessKey string
		awsSecretKey string
		awsToken     string
		awsRegion    string
	}
	awsSessionOnce sync.Once
	awsSession     *session.Session
	awsSessionErr  error
}

// NewApp should be called at the start of an application
func NewApp() *App {
	app := &App{}
	flag.BoolVar(&app.flags.debug, "debug", false, "Enable debug logging")
	flag.StringVar(&app.flags.env, "env", "", "Execution environment")
	flag.StringVar(&app.flags.awsAccessKey, "aws_access_key", "", "Access `key` for AWS")
	flag.StringVar(&app.flags.awsSecretKey, "aws_secret_key", "", "Secret `key` for AWS")
	flag.StringVar(&app.flags.awsToken, "aws_token", "", "Temporary access `token` for AWS")
	flag.StringVar(&app.flags.awsRegion, "aws_region", "us-east-1", "AWS `region`")
	return app
}

// AWSSession returns an AWS session.
func (app *App) AWSSession() (*session.Session, error) {
	app.awsSessionOnce.Do(func() {
		awsConfig, err := awsutil.Config(app.flags.awsRegion, app.flags.awsAccessKey, app.flags.awsSecretKey, app.flags.awsToken)
		if err != nil {
			app.awsSessionErr = err
			return
		}
		app.awsSession = session.New(awsConfig)
	})
	return app.awsSession, app.awsSessionErr
}
