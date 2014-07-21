/*
	Package config implements command line argument and config file parsing.
*/
package config

import (
	"expvar"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/aws"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"

	"github.com/sprucehealth/backend/third_party/github.com/BurntSushi/toml"
	flags "github.com/sprucehealth/backend/third_party/github.com/jessevdk/go-flags"
	goamz "github.com/sprucehealth/backend/third_party/launchpad.net/goamz/aws"
	"github.com/sprucehealth/backend/third_party/launchpad.net/goamz/s3"
)

type NotificationConfig struct {
	SNSApplicationEndpoint string          `long:"sns_application_endpoint" description:"SNS Application endpoint for push notification"`
	IsApnsSandbox          bool            `long:"apns_sandbox"`
	Platform               common.Platform `long:"platform"`
	URLScheme              string          `long:"url_scheme" description:"URL scheme to include in communication for deep linking into app"`
}

func DetermineNotificationConfigName(platform common.Platform, appType, appEnvironment string) string {
	return fmt.Sprintf("%s-%s-%s", platform.String(), appType, appEnvironment)
}

type NotificationConfigs map[string]*NotificationConfig

func (n NotificationConfigs) Get(configName string) (*NotificationConfig, error) {
	notificationConfig, ok := n[configName]
	if !ok {
		return nil, fmt.Errorf("Unable to find notificationConfig for config name %s", configName)
	}
	return notificationConfig, nil
}

type BaseConfig struct {
	AppName      string `long:"app_name" description:"Application name (required)"`
	AWSRegion    string `long:"aws_region" description:"AWS region"`
	AWSRole      string `long:"aws_role" description:"AWS role for fetching temporary credentials"`
	AWSSecretKey string `long:"aws_secret_key" description:"AWS secret key"`
	AWSAccessKey string `long:"aws_access_key" description:"AWS access key id"`
	ConfigPath   string `short:"c" long:"config" description:"Path to config file. If not set then stderr is used."`
	Environment  string `short:"e" long:"env" description:"Current environment (dev, stage, prod)"`
	Syslog       bool   `long:"syslog" description:"Log to syslog"`
	Stats        *Stats `group:"Stats" toml:"stats"`
	AlertEmail   string `long:"alert_email" description:"Email address to which to send panics"`

	Version bool `long:"version" description:"Show version and exit" toml:"-"`

	awsAuth     aws.Auth
	awsAuthOnce sync.Once
}

var (
	GitBranch       string
	GitRevision     string
	BuildTime       string
	BuildNumber     string // Travis-CI build
	MigrationNumber string // The database needs to match this migration number for this build
)

var VersionInfo map[string]string

func init() {
	if MigrationNumber == "" {
		// Should only be unset for local builds so try to find latest migration in source tree
		if files, err := filepath.Glob(path.Join(path.Dir(os.Args[0]), "../../mysql/migration-*.sql")); err == nil {
			maxMigration := 0
			for _, name := range files {
				name = path.Base(name)[10:]
				if idx := strings.IndexByte(name, '.'); idx >= 0 {
					name = name[:idx]
					if num, err := strconv.Atoi(name); err == nil && num > maxMigration {
						maxMigration = num
					}
				}
			}
			if maxMigration != 0 {
				MigrationNumber = strconv.Itoa(maxMigration)
			}
		}
	}

	VersionInfo = map[string]string{
		"GitBranch":       GitBranch,
		"GitRevision":     GitRevision,
		"BuildTime":       BuildTime,
		"BuildNumber":     BuildNumber,
		"MigrationNumber": MigrationNumber,
		"GoVersion":       runtime.Version(),
	}

	expvar.Publish("version", expvar.Func(func() interface{} {
		return VersionInfo
	}))
}

var validEnvironments = map[string]bool{
	"prod":    true,
	"staging": true,
	"dev":     true,
	"test":    true,
	"demo":    true,
}

func (c *BaseConfig) AWSAuth() (auth aws.Auth, err error) {
	c.awsAuthOnce.Do(func() {
		if c.AWSRole != "" {
			c.awsAuth, err = aws.CredentialsForRole(c.AWSRole)
		} else {
			keys := aws.KeysFromEnvironment()
			if c.AWSAccessKey != "" && c.AWSSecretKey != "" {
				keys.AccessKey = c.AWSAccessKey
				keys.SecretKey = c.AWSSecretKey
			} else {
				c.AWSAccessKey = keys.AccessKey
				c.AWSSecretKey = keys.SecretKey
			}
			c.awsAuth = keys
		}
	})
	auth = c.awsAuth
	return
}

func (c *BaseConfig) OpenURI(uri string) (io.ReadCloser, error) {
	var rd io.ReadCloser
	if strings.Contains(uri, "://") {
		ur, err := url.Parse(uri)
		if err != nil {
			return nil, err
		}
		if ur.Scheme == "s3" {
			awsAuth, err := c.AWSAuth()
			if err != nil {
				return nil, err
			}
			s3 := s3.New(common.AWSAuthAdapter(awsAuth), goamz.Regions[c.AWSRegion])
			rd, _, err = s3.Bucket(ur.Host).GetReader(ur.Path)
			if err != nil {
				return nil, err
			}
		} else {
			if res, err := http.Get(uri); err != nil {
				return nil, err
			} else if res.StatusCode != 200 {
				return nil, fmt.Errorf("config: failed to fetch URI %s: status code %d", uri, res.StatusCode)
			} else {
				rd = res.Body
			}
		}
	} else {
		fi, err := os.Open(uri)
		if err != nil {
			return nil, err
		}
		rd = fi
	}
	return rd, nil
}

func (c *BaseConfig) ReadURI(uri string) ([]byte, error) {
	if rd, err := c.OpenURI(uri); err != nil {
		return nil, err
	} else {
		defer rd.Close()
		return ioutil.ReadAll(rd)
	}
}

func LoadConfigFile(configUrl string, config interface{}, awsAuther func() (aws.Auth, error)) error {
	if configUrl == "" {
		return nil
	}

	var rd io.ReadCloser
	if strings.Contains(configUrl, "://") {
		awsAuth, err := awsAuther()
		if err != nil {
			return fmt.Errorf("config: failed to get AWS auth: %+v", err)
		}
		ur, err := url.Parse(configUrl)
		if err != nil {
			return fmt.Errorf("config: failed to parse config url %s: %+v", configUrl, err)
		}
		if ur.Scheme == "s3" {
			s3 := s3.New(common.AWSAuthAdapter(awsAuth), goamz.USEast)
			rd, _, err = s3.Bucket(ur.Host).GetReader(ur.Path)
			if err != nil {
				return fmt.Errorf("config: failed to get config from s3 %s: %+v", configUrl, err)
			}
		} else {
			if res, err := http.Get(configUrl); err != nil {
				return fmt.Errorf("config: failed to fetch config from URL %s: %+v", configUrl, err)
			} else if res.StatusCode != 200 {
				return fmt.Errorf("config: failed to fetch config from URL %s: status code %d", configUrl, res.StatusCode)
			} else {
				rd = res.Body
			}
		}
	} else {
		fi, err := os.Open(configUrl)
		if err != nil {
			return fmt.Errorf("config: failed to open config file: %+v", err)
		}
		rd = fi
	}
	defer rd.Close()
	by, err := ioutil.ReadAll(rd)
	if err != nil {
		return fmt.Errorf("config: failed top read config file: %+v", err)
	}
	st := string(by)
	if _, err := toml.Decode(st, config); err != nil {
		return fmt.Errorf("config: failed to parse config file: %+v", err)
	}
	v := reflect.ValueOf(config).Elem()
	fv := v.FieldByName("BaseConfig")
	if fv.IsValid() && fv.Kind() == reflect.Ptr {
		if _, err := toml.Decode(st, fv.Interface()); err != nil {
			return fmt.Errorf("config: failed to parse config file: %+v", err)
		}
	}
	return nil
}

func Parse(config interface{}) ([]string, error) {
	return ParseArgs(config, os.Args[1:])
}

func ParseArgs(config interface{}, args []string) ([]string, error) {
	if args == nil {
		args = os.Args[1:]
	}

	v := reflect.ValueOf(config).Elem()
	fv := v.FieldByName("BaseConfig")
	if !fv.IsValid() || fv.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("config: config struct must contain a pointer to BaseConfig")
	}
	var baseConfig *BaseConfig
	if fv.IsNil() {
		baseConfig = &BaseConfig{}
		fv.Set(reflect.ValueOf(baseConfig))
	} else {
		baseConfig = fv.Interface().(*BaseConfig)
	}

	parser := flags.NewParser(baseConfig, flags.PrintErrors|flags.PassDoubleDash|flags.IgnoreUnknown)
	_, err := parser.ParseArgs(args)
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			return nil, fmt.Errorf("config: failed to parse flags: %+v", err)
		}
		os.Exit(1)
	}

	if baseConfig.Version {
		for k, v := range VersionInfo {
			fmt.Printf("%s: %s\n", k, v)
		}
		os.Exit(0)
	}

	if err := LoadConfigFile(baseConfig.ConfigPath, config, baseConfig.AWSAuth); err != nil {
		return nil, err
	}

	// Make sure command line overrides config
	extraArgs, err := flags.ParseArgs(config, args)
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			os.Exit(1)
		}
		return nil, err
	}

	if baseConfig.AWSRegion == "" {
		az, err := aws.GetMetadata(aws.MetadataAvailabilityZone)
		if err != nil {
			return nil, fmt.Errorf("config: no region specified and failed to get from instance metadata: %+v", err)
		}
		baseConfig.AWSRegion = az[:len(az)-1]
		return nil, fmt.Errorf("config: got region from metadata: %s", baseConfig.AWSRegion)
	}

	if baseConfig.AppName == "" {
		fmt.Fprintf(os.Stderr, "Missing required app_name config value.\n")
		os.Exit(1)
	}
	if !validEnvironments[baseConfig.Environment] {
		fmt.Fprintf(os.Stderr, "flag --env is required and must be one of prod, staging, or dev\n")
		os.Exit(1)
	}

	return extraArgs, nil
}

func (c *BaseConfig) SetupLogging() {
	log.SetFlags(log.Lshortfile)
	if c.Syslog {
		if h, err := golog.SyslogHandler(c.AppName, golog.LogfmtFormatter()); err != nil {
			log.Fatal(err)
		} else {
			if c.AlertEmail != "" {
				golog.Default().SetHandler(panicLogHandler(c, h))
			} else {
				golog.Default().SetHandler(h)
			}
		}
	} else {
		if c.AlertEmail != "" {
			golog.Default().SetHandler(panicLogHandler(c, golog.DefaultHandler))
		}
	}
	log.SetOutput(golog.Writer)
}

func panicLogHandler(conf *BaseConfig, next golog.Handler) golog.Handler {
	return golog.HandlerFunc(func(e *golog.Entry) error {
		if e.Lvl == golog.CRIT {
			go func() {
				body := fmt.Sprintf("%s\n%s\n", e.Msg, golog.FormatContext(e.Ctx, '\n'))
				dispatch.Default.Publish(&PanicEvent{
					AppName:     conf.AppName,
					Environment: conf.Environment,
					Body:        body,
				})
			}()
		}
		return next.Log(e)
	})
}
