// Package config implements command line argument and config file parsing.
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
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/BurntSushi/toml"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/s3"
	flags "github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/jessevdk/go-flags"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/golog"
)

type DosespotConfig struct {
	ClinicID     int64  `long:"clinic_id" description:"Clinic Id for dosespot"`
	ClinicKey    string `long:"clinic_key" description:"Clinic Key for dosespot"`
	ProxyID      int64  `long:"proxy_id" description:"Proxy Id for dosespot"`
	UserID       int64  `long:"user_id" description:"User Id for dosespot"`
	SOAPEndpoint string `long:"soap_endpoint" description:"SOAP endpoint"`
	APIEndpoint  string `long:"api_endpoint" description:"API endpoint where soap actions are defined"`
}

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

type MinimumAppVersionConfigs map[string]*MinimumAppVersionConfig

type MinimumAppVersionConfig struct {
	AppVersion  *common.Version `long:"minimum_app_version" description:"Minimum app version that is supported"`
	AppStoreURL string          `long:"app_store_url" description:"App Store URL to download the latest version of the app"`
}

func (m MinimumAppVersionConfigs) Get(configName string) (*MinimumAppVersionConfig, error) {
	minimumAppVersionConfig, ok := m[configName]
	if !ok {
		return nil, fmt.Errorf("Unable to find minimumAppStoreConfig for configName %s", configName)
	}
	return minimumAppVersionConfig, nil
}

type BaseConfig struct {
	AppName      string `long:"app_name" description:"Application name (required)"`
	AWSRegion    string `long:"aws_region" description:"AWS region"`
	AWSSecretKey string `long:"aws_secret_key" description:"AWS secret key"`
	AWSAccessKey string `long:"aws_access_key" description:"AWS access key id"`
	ConfigPath   string `short:"c" long:"config" description:"Path to config file. If not set then stderr is used."`
	Environment  string `short:"e" long:"env" description:"Current environment (dev, stage, prod)"`
	Syslog       bool   `long:"syslog" description:"Log to syslog"`
	Stats        *Stats `group:"Stats" toml:"stats"`

	Version bool `long:"version" description:"Show version and exit" toml:"-"`

	awsConfig   *aws.Config
	awsAuthOnce sync.Once
}

// These variables are set during linking
var (
	GitBranch       string
	GitRevision     string
	BuildTime       string
	BuildNumber     string // CI build number
	MigrationNumber string // The database needs to match this migration number for this build
)

// VersionInfo is a set of build version variables set during linking
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

func (c *BaseConfig) AWS() *aws.Config {
	c.awsAuthOnce.Do(func() {
		var cred *credentials.Credentials
		if c.AWSAccessKey != "" && c.AWSSecretKey != "" {
			cred = credentials.NewStaticCredentials(c.AWSAccessKey, c.AWSSecretKey, "")
		} else {
			cred = credentials.NewEnvCredentials()
			if v, err := cred.Get(); err != nil || v.AccessKeyID == "" || v.SecretAccessKey == "" {
				cred = credentials.NewEC2RoleCredentials(&http.Client{Timeout: 2 * time.Second}, "", time.Minute*5)
			}
		}

		region := c.AWSRegion
		if region == "" {
			az, err := awsutil.GetMetadata(awsutil.MetadataAvailabilityZone)
			if err != nil {
				golog.Fatalf("config: no region provided and failed to get from instance metadata: %s", err)
			}
			region = az[:len(az)-1]
		}

		c.awsConfig = &aws.Config{
			Credentials: cred,
			Region:      region,
		}
	})
	// Return a copy
	cnf := *c.awsConfig
	return &cnf
}

func (c *BaseConfig) OpenURI(uri string) (io.ReadCloser, error) {
	var rd io.ReadCloser
	if strings.Contains(uri, "://") {
		ur, err := url.Parse(uri)
		if err != nil {
			return nil, err
		}
		if ur.Scheme == "s3" {
			out, err := s3.New(c.AWS()).GetObject(&s3.GetObjectInput{
				Bucket: &ur.Host,
				Key:    &ur.Path,
			})
			if err != nil {
				return nil, err
			}
			rd = out.Body
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
	rd, err := c.OpenURI(uri)
	if err != nil {
		return nil, err
	}
	defer rd.Close()
	return ioutil.ReadAll(rd)
}

func LoadConfigFile(configURL string, config interface{}, awsConfig func() *aws.Config) error {
	if configURL == "" {
		return nil
	}

	var rd io.ReadCloser
	if strings.Contains(configURL, "://") {
		ur, err := url.Parse(configURL)
		if err != nil {
			return fmt.Errorf("config: failed to parse config url %s: %+v", configURL, err)
		}
		if ur.Scheme == "s3" {
			obj, err := s3.New(awsConfig()).GetObject(&s3.GetObjectInput{
				Bucket: &ur.Host,
				Key:    &ur.Path,
			})
			if err != nil {
				return fmt.Errorf("config: failed to get config from s3 %s: %+v", configURL, err)
			}
			rd = obj.Body
		} else {
			if res, err := http.Get(configURL); err != nil {
				return fmt.Errorf("config: failed to fetch config from URL %s: %+v", configURL, err)
			} else if res.StatusCode != 200 {
				return fmt.Errorf("config: failed to fetch config from URL %s: status code %d", configURL, res.StatusCode)
			} else {
				rd = res.Body
			}
		}
	} else {
		fi, err := os.Open(configURL)
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

	if err := LoadConfigFile(baseConfig.ConfigPath, config, baseConfig.AWS); err != nil {
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

// SetupLogging configures golog and the stdlib log package
func (c *BaseConfig) SetupLogging() {
	log.SetFlags(log.Lshortfile)
	if c.Syslog {
		if h, err := golog.SyslogHandler(c.AppName, golog.LogfmtFormatter()); err != nil {
			log.Fatal(err)
		} else {
			golog.Default().SetHandler(h)
		}
	}
	log.SetOutput(golog.Writer)
}
