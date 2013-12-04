/*
	Package config implements command line argument and config file parsing.
*/
package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"carefront/libs/aws"
	"carefront/libs/svcreg"
	"carefront/libs/svcreg/zksvcreg"
	"carefront/util"
	"github.com/BurntSushi/toml"
	flags "github.com/jessevdk/go-flags"
	"github.com/samuel/go-zookeeper/zk"
	goamz "launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
)

type BaseConfig struct {
	AWSRegion               string `long:"aws_region" description:"AWS region"`
	AWSRole                 string `long:"aws_role" description:"AWS role for fetching temporary credentials"`
	AWSSecretKey            string `long:"aws_secret_key" description:"AWS secret key"`
	AWSAccessKey            string `long:"aws_access_key" description:"AWS access key id"`
	ConfigPath              string `short:"c" long:"config" description:"Path to config file. If not set then stderr is used."`
	Environment             string `short:"e" long:"env" description:"Current environment (dev, stage, prod)"`
	LogPath                 string `long:"log_path" description:"Path to log file"`
	ZookeeperHosts          string `long:"zk_hosts" description:"Zookeeper host list (e.g. 127.0.0.1:2181,192.168.1.1:2181)"`
	ZookeeperServicesPrefix string `long:"zk_svc_prefix" description:"Zookeeper svc registry prefix" default:"/services"`
	Stats                   *Stats `group:"Stats" toml:"stats"`

	awsAuth     aws.Auth
	awsAuthOnce sync.Once
	zkConn      *zk.Conn
	zkChan      <-chan zk.Event
	zkOnce      sync.Once
	reg         svcreg.Registry
	regOnce     sync.Once
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

func (c *BaseConfig) ZKClient() (conn *zk.Conn, err error) {
	c.zkOnce.Do(func() {
		if c.ZookeeperHosts != "" {
			hosts := strings.Split(c.ZookeeperHosts, ",")
			c.zkConn, c.zkChan, err = zk.Connect(hosts, time.Second*10)
		}
	})
	conn = c.zkConn
	return
}

func (c *BaseConfig) ServiceRegistry() (reg svcreg.Registry, err error) {
	c.regOnce.Do(func() {
		zk, e := c.ZKClient()
		if e != nil {
			err = e
			return
		}
		if zk == nil {
			c.reg = &svcreg.StaticRegistry{}
		} else {
			c.reg, err = zksvcreg.NewServiceRegistry(zk, c.ZookeeperServicesPrefix)
		}
	})
	reg = c.reg
	return
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
			s3 := s3.New(util.AWSAuthAdapter(awsAuth), goamz.USEast)
			rd, err = s3.Bucket(ur.Host).GetReader(ur.Path)
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

	if baseConfig.LogPath != "" {
		// check if the file exists
		_, err := os.Stat(baseConfig.LogPath)
		var file *os.File
		if os.IsNotExist(err) {
			// file doesn't exist so lets create it
			file, err = os.Create(baseConfig.LogPath)
			if err != nil {
				return nil, fmt.Errorf("config: failed to create log: %s", err.Error())
			}
		} else {
			file, err = os.OpenFile(baseConfig.LogPath, os.O_RDWR|os.O_APPEND, 0660)
			if err != nil {
				return nil, fmt.Errorf("config: could not open logfile %s", err.Error())
			}
		}
		log.SetOutput(file)
	}
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)

	return extraArgs, nil
}
