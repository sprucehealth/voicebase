package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
)

const configPath = "~/.baymax.conf"

type config struct {
	DBHost        string
	DBPort        int
	DBUsername    string
	DBPassword    string
	AuthAddr      string
	DirectoryAddr string
	ExCommsAddr   string
	SettingsAddr  string
	ThreadingAddr string
	LayoutAddr    string
	Env           string
}

func (c *config) authClient() (auth.AuthClient, error) {
	if c.AuthAddr == "" {
		return nil, errors.New("Auth service address required")
	}
	conn, err := grpc.Dial(c.AuthAddr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to auth service: %s", err)
	}
	return auth.NewAuthClient(conn), nil
}

func (c *config) directoryClient() (directory.DirectoryClient, error) {
	if c.DirectoryAddr == "" {
		return nil, errors.New("Directory service address required")
	}
	conn, err := grpc.Dial(c.DirectoryAddr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to directory service: %s", err)
	}
	return directory.NewDirectoryClient(conn), nil
}

func (c *config) exCommsClient() (excomms.ExCommsClient, error) {
	if c.ExCommsAddr == "" {
		return nil, errors.New("ExComms service address required")
	}
	conn, err := grpc.Dial(c.ExCommsAddr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to excomms service: %s", err)
	}
	return excomms.NewExCommsClient(conn), nil
}

func (c *config) settingsClient() (settings.SettingsClient, error) {
	if c.SettingsAddr == "" {
		return nil, errors.New("Settings service address required")
	}
	conn, err := grpc.Dial(c.SettingsAddr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to settings service: %s", err)
	}
	return settings.NewSettingsClient(conn), nil
}

func (c *config) threadingClient() (threading.ThreadsClient, error) {
	if c.ThreadingAddr == "" {
		return nil, errors.New("Threading service address required")
	}
	conn, err := grpc.Dial(c.ThreadingAddr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to threading service: %s", err)
	}
	return threading.NewThreadsClient(conn), nil
}

func (c *config) layoutClient() (layout.LayoutClient, error) {
	if c.LayoutAddr == "" {
		return nil, errors.New("Layout service address required")
	}

	conn, err := grpc.Dial(c.LayoutAddr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to layout service: %s", err)
	}
	return layout.NewLayoutClient(conn), nil
}

func (c *config) directoryDB() (*sql.DB, error) {
	return c.db("directory")
}

func (c *config) threadingDB() (*sql.DB, error) {
	return c.db("threading")
}

func (c *config) db(name string) (*sql.DB, error) {
	if c.DBHost == "" {
		return nil, errors.New("DBHost not set")
	}
	if c.DBUsername == "" {
		return nil, errors.New("DBUsername not set")
	}
	if c.DBPort == 0 {
		c.DBPort = 3306
	}
	return dbutil.ConnectMySQL(&dbutil.DBConfig{
		Name:     name,
		Host:     c.DBHost,
		Port:     c.DBPort,
		User:     c.DBUsername,
		Password: c.DBPassword,
	})
}

func loadConfig() *config {
	path, err := interpolatePath(configPath)
	if err != nil {
		golog.Fatalf("Invalid config path %s: %s", configPath, err)
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &config{}
		}
		golog.Fatalf("Failed to read %s: %T", path, err)
	}
	var c config
	if err := json.Unmarshal(b, &c); err != nil {
		golog.Fatalf("Failed to parse %s: %s", path, err)
	}
	return &c
}

type command interface {
	run(args []string) error
}

type commandNew func(*config) (command, error)

var commands = map[string]commandNew{
	"account":             newAccountCmd,
	"decodeid":            newDecodeIDCmd,
	"deletecontact":       newDeleteContactCmd,
	"encodeid":            newEncodeIDCmd,
	"entity":              newEntityCmd,
	"moveentity":          newMoveEntityCmd,
	"getsetting":          newGetSettingCmd,
	"setsetting":          newSetSettingCmd,
	"thread":              newThreadCmd,
	"changeorgemail":      newChangeOrgEmailCmd,
	"blockaccount":        newBlockAccountCmd,
	"updateentity":        newUpdateEntityCmd,
	"setgreeting":         newSetGreetingCmd,
	"migratesetupthreads": newMigrateSetupThreadsCmd,
	"uploadlayout":        newUploadLayoutCmd,
}

func main() {
	golog.Default().SetLevel(golog.INFO)

	cnf := loadConfig()

	flag.StringVar(&cnf.DBHost, "db_host", cnf.DBHost, "mysql database `host`")
	flag.IntVar(&cnf.DBPort, "db_port", cnf.DBPort, "mysql database `port`")
	flag.StringVar(&cnf.DBUsername, "db_username", cnf.DBUsername, "mysql database `username`")
	flag.StringVar(&cnf.DBPassword, "db_password", cnf.DBPassword, "mysql database `password`")
	flag.StringVar(&cnf.DirectoryAddr, "directory_addr", cnf.DirectoryAddr, "`host:port` of directory service")
	flag.StringVar(&cnf.ThreadingAddr, "threading_addr", cnf.ThreadingAddr, "`host:port` of treading service")
	flag.StringVar(&cnf.LayoutAddr, "layout_addr", cnf.LayoutAddr, "`host:port` of layout service")
	flag.StringVar(&cnf.Env, "env", cnf.Env, "environment that baymaxadmin is running against")

	flag.Parse()

	cmd := flag.Arg(0)

	for name, cfn := range commands {
		if name == cmd {
			c, err := cfn(cnf)
			if err != nil {
				golog.Fatalf(err.Error())
			}
			if err := c.run(flag.Args()[1:]); err != nil {
				fmt.Fprintf(os.Stderr, "FAILED: %s\n", err)
				os.Exit(2)
			}
			os.Exit(0)
		}
	}

	if cmd != "" {
		fmt.Printf("Unknown command '%s'\n", cmd)
	}

	fmt.Printf("Available commands:\n")
	cmdList := make([]string, 0, len(commands))
	for name := range commands {
		cmdList = append(cmdList, name)
	}
	sort.Strings(cmdList)
	for _, name := range cmdList {
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
