package config

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"

	"carefront/libs/aws"
	"carefront/util"
	"github.com/BurntSushi/toml"
	flags "github.com/jessevdk/go-flags"
	goamz "launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
)

type BaseConfig struct {
	AWSRegion    string `long:"aws_region" description:"AWS region"`
	AWSRole      string `long:"aws_role" description:"AWS role for fetching temporary credentials"`
	AWSSecretKey string `long:"aws_secret_key" description:"AWS secret key"`
	AWSAccessKey string `long:"aws_access_key" description:"AWS access key id"`
	ConfigPath   string `short:"c" long:"config" description:"Path to config file. If not set then stderr is used."`
	LogPath      string `short:"l" long:"log_path" description:"Path to log file"`

	awsAuth aws.Auth
}

func (c *BaseConfig) AWSAuth() (aws.Auth, error) {
	var err error
	if c.awsAuth == nil {
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
	}
	if err != nil {
		c.awsAuth = nil
	}
	return c.awsAuth, err
}

func LoadConfigFile(configUrl string, config interface{}, awsAuther func() (aws.Auth, error)) error {
	if configUrl == "" {
		return nil
	}
	if strings.Contains(configUrl, "://") {
		awsAuth, err := awsAuther()
		if err != nil {
			return fmt.Errorf("config: failed to get AWS auth: %+v", err)
		}
		ur, err := url.Parse(configUrl)
		if err != nil {
			return fmt.Errorf("Failed to parse config url %s: %+v", configUrl, err)
		}
		var rd io.ReadCloser
		if ur.Scheme == "s3" {
			s3 := s3.New(util.AWSAuthAdapter(awsAuth), goamz.USEast)
			rd, err = s3.Bucket(ur.Host).GetReader(ur.Path)
			if err != nil {
				return fmt.Errorf("Failed to get config from s3 %s: %+v", configUrl, err)
			}
		} else {
			if res, err := http.Get(configUrl); err != nil {
				return fmt.Errorf("Failed to fetch config from URL %s: %+v", configUrl, err)
			} else if res.StatusCode != 200 {
				return fmt.Errorf("Failed to fetch config from URL %s: status code %d", configUrl, res.StatusCode)
			} else {
				rd = res.Body
			}
		}
		if _, err := toml.DecodeReader(rd, config); err != nil {
			return fmt.Errorf("Failed to parse config file: %+v", err)
		}
		rd.Close()
	} else if _, err := toml.DecodeFile(configUrl, config); err != nil {
		return fmt.Errorf("Failed to parse config file: %+v", err)
	}
	return nil
}

func ParseFlagsAndConfig(config interface{}, args []string) ([]string, error) {
	if args == nil {
		args = os.Args[1:]
	}
	baseConfig := &BaseConfig{}
	parser := flags.NewParser(baseConfig, flags.PrintErrors|flags.PassDoubleDash|flags.IgnoreUnknown)
	_, err := parser.ParseArgs(args)
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			return nil, fmt.Errorf("config: failed to parse flags: %+v", err)
		}
		os.Exit(1)
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
			return nil, fmt.Errorf("No region specified and failed to get from instance metadata: %+v", err)
		}
		baseConfig.AWSRegion = az[:len(az)-1]
		return nil, fmt.Errorf("Got region from metadata: %s", baseConfig.AWSRegion)
	}

	if baseConfig.LogPath != "" {
		// check if the file exists
		_, err := os.Stat(baseConfig.LogPath)
		var file *os.File
		if os.IsNotExist(err) {
			// file doesn't exist so lets create it
			file, err = os.Create(baseConfig.LogPath)
			if err != nil {
				return nil, fmt.Errorf("Failed to create log: %s", err.Error())
			}
		} else {
			file, err = os.OpenFile(baseConfig.LogPath, os.O_RDWR|os.O_APPEND, 0660)
			if err != nil {
				return nil, fmt.Errorf("Could not open logfile %s", err.Error())
			}
		}
		log.SetOutput(file)
	}
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)

	// If the config struct includes BaseConfig then set the value
	v := reflect.ValueOf(config).Elem()
	fv := v.FieldByName("BaseConfig")
	if fv.IsValid() {
		if fv.Kind() == reflect.Ptr {
			fv.Set(reflect.ValueOf(baseConfig))
		} else {
			fv.Set(reflect.ValueOf(baseConfig).Elem())
		}
	}

	return extraArgs, nil
}
