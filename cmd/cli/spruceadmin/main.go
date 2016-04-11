package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/patientcase"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

const configPath = "~/.spruce.conf"

type conf struct {
	DBHost     string
	DBPort     int
	DBName     string
	DBUsername string
	DBPassword string
}

func (c *conf) validate() error {
	if c.DBHost == "" {
		return errors.New("DB Host required")
	}
	if c.DBName == "" {
		return errors.New("DB Name required")
	}
	if c.DBUsername == "" {
		return errors.New("DB Username required")
	}
	return nil
}

func loadConfig() *conf {
	path, err := interpolatePath(configPath)
	if err != nil {
		golog.Fatalf("Invalid config path %s: %s", configPath, err)
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &conf{}
		}
		golog.Fatalf("Failed to read %s: %T", path, err)
	}
	var c conf
	if err := json.Unmarshal(b, &c); err != nil {
		golog.Fatalf("Failed to parse %s: %s", path, err)
	}
	return &c
}

type command interface {
	run(args []string) error
}

type commandNew func(api.DataAPI, patientcase.Service) (command, error)

var commands = map[string]commandNew{
	"movecase":      newMoveCaseCmd,
	"searchdoctors": newSearchDoctorsCmd,
}

func main() {
	golog.Default().SetLevel(golog.INFO)

	cnf := loadConfig()
	if cnf.DBPort == 0 {
		cnf.DBPort = 3306
	}

	flag.StringVar(&cnf.DBHost, "db_host", cnf.DBHost, "mysql database host")
	flag.IntVar(&cnf.DBPort, "db_port", cnf.DBPort, "mysql database port")
	flag.StringVar(&cnf.DBName, "db_name", cnf.DBName, "mysql database name")
	flag.StringVar(&cnf.DBUsername, "db_username", cnf.DBUsername, "mysql database username")
	flag.StringVar(&cnf.DBPassword, "db_password", cnf.DBPassword, "mysql database password")
	flag.Parse()

	if err := cnf.validate(); err != nil {
		golog.Fatalf(err.Error())
	}

	db, err := dbutil.ConnectMySQL(&dbutil.DBConfig{
		Host:     cnf.DBHost,
		Port:     cnf.DBPort,
		Name:     cnf.DBName,
		User:     cnf.DBUsername,
		Password: cnf.DBPassword,
	})
	if err != nil {
		golog.Fatalf(err.Error())
	}

	cfgStore, err := cfg.NewLocalStore(config.CfgDefs())
	if err != nil {
		golog.Fatalf("Failed to initialize local cfg store: %s", err)
	}

	dataAPI, err := api.NewDataService(db, cfgStore, metrics.NewRegistry())
	if err != nil {
		golog.Fatalf("Unable to initialize data service layer: %s", err)
	}

	svc := patientcase.NewService(dataAPI)

	cmd := flag.Arg(0)

	for name, cfn := range commands {
		if name == cmd {
			c, err := cfn(dataAPI, svc)
			if err != nil {
				golog.Fatalf(err.Error())
			}
			if err := c.run(flag.Args()[1:]); err != nil {
				golog.Fatalf(err.Error())
			}
			os.Exit(0)
		}
	}

	if cmd != "" {
		fmt.Printf("Unknown command '%s'\n", cmd)
	}

	fmt.Printf("Available commands:\n")
	for name := range commands {
		fmt.Printf("\t%s\n", name)
	}
	os.Exit(1)
}

func interpolatePath(p string) (string, error) {
	if p == "" {
		return "", errors.New("empty path")
	}
	if p[0] == '~' {
		p = os.Getenv("HOME") + p[1:]
	}
	return filepath.Abs(p)
}

func prompt(scn *bufio.Scanner, prompt string) string {
	fmt.Print(prompt)
	if !scn.Scan() {
		os.Exit(1)
	}
	return strings.TrimSpace(scn.Text())
}
