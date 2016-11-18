package main

import (
	"bufio"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/patientsync"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
)

const configPath = "~/.baymax.conf"

type config struct {
	TLS                 bool
	CACertPath          string
	DBHost              string
	DBPort              int
	DBUsername          string
	DBPassword          string
	DBTLS               string
	AuthAddr            string
	DirectoryAddr       string
	ExCommsAddr         string
	SettingsAddr        string
	ThreadingAddr       string
	LayoutAddr          string
	InviteAddr          string
	PatientSyncAddr     string
	NotificationsSQSURL string
	InviteAPIDomain     string
	KMSKeyARN           string
	Env                 string
}

func (c *config) clientTLSConfig() *tls.Config {
	if !c.TLS {
		return nil
	}
	tlsConf := &tls.Config{}
	if c.CACertPath != "" {
		ca, err := boot.CAFromFile(c.CACertPath)
		if err != nil {
			golog.Fatalf("Failed to load CA from %s: %s", c.CACertPath, err)
		}
		tlsConf.RootCAs = ca
	}
	return tlsConf
}

func (c *config) authClient() (auth.AuthClient, error) {
	conn, err := boot.DialGRPC("baymaxadmin", c.AuthAddr, c.clientTLSConfig())
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to auth service: %s", err)
	}
	return auth.NewAuthClient(conn), nil
}

func (c *config) directoryClient() (directory.DirectoryClient, error) {
	conn, err := boot.DialGRPC("baymaxadmin", c.DirectoryAddr, c.clientTLSConfig())
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to directory service: %s", err)
	}
	return directory.NewDirectoryClient(conn), nil
}

func (c *config) exCommsClient() (excomms.ExCommsClient, error) {
	conn, err := boot.DialGRPC("baymaxadmin", c.ExCommsAddr, c.clientTLSConfig())
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to excomms service: %s", err)
	}
	return excomms.NewExCommsClient(conn), nil
}

func (c *config) settingsClient() (settings.SettingsClient, error) {
	conn, err := boot.DialGRPC("baymaxadmin", c.SettingsAddr, c.clientTLSConfig())
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to settings service: %s", err)
	}
	return settings.NewSettingsClient(conn), nil
}

func (c *config) threadingClient() (threading.ThreadsClient, error) {
	conn, err := boot.DialGRPC("baymaxadmin", c.ThreadingAddr, c.clientTLSConfig())
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to threading service: %s", err)
	}
	return threading.NewThreadsClient(conn), nil
}

func (c *config) layoutClient() (layout.LayoutClient, error) {
	conn, err := boot.DialGRPC("baymaxadmin", c.LayoutAddr, c.clientTLSConfig())
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to layout service: %s", err)
	}
	return layout.NewLayoutClient(conn), nil
}

func (c *config) inviteClient() (invite.InviteClient, error) {
	conn, err := boot.DialGRPC("baymaxadmin", c.InviteAddr, c.clientTLSConfig())
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to invite service: %s", err)
	}
	return invite.NewInviteClient(conn), nil
}

func (c *config) patientSyncClient() (patientsync.PatientSyncClient, error) {
	conn, err := boot.DialGRPC("baymaxadmin", c.PatientSyncAddr, c.clientTLSConfig())
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to patientsync service: %s", err)
	}
	return patientsync.NewPatientSyncClient(conn), nil
}

func (c *config) awsSession() (*session.Session, error) {
	awsConfig, err := awsutil.Config("us-east-1", "", "", "")
	if err != nil {
		return nil, err
	}
	return session.New(awsConfig), nil
}

func (c *config) sqsClient() (sqsiface.SQSAPI, error) {
	if c.KMSKeyARN == "" {
		return nil, errors.New("KMSKeyARN required")
	}
	awsSession, err := c.awsSession()
	if err != nil {
		return nil, errors.Trace(err)
	}
	return awsutil.NewEncryptedSQS(c.KMSKeyARN, kms.New(awsSession), sqs.New(awsSession))
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
		Name:          name,
		Host:          c.DBHost,
		Port:          c.DBPort,
		User:          c.DBUsername,
		Password:      c.DBPassword,
		EnableTLS:     c.DBTLS == "true" || c.DBTLS == "skip-verify",
		SkipVerifyTLS: c.DBTLS == "skip-verify",
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
	"account":                     newAccountCmd,
	"addcontact":                  newAddContactCmd,
	"blockaccount":                newBlockAccountCmd,
	"blocknumber":                 newBlockNumberCmd,
	"changeorgemail":              newChangeOrgEmailCmd,
	"createsupporthread":          newCreateSupportThreadCmd,
	"decodeid":                    newDecodeIDCmd,
	"deletecontact":               newDeleteContactCmd,
	"deletethread":                newDeleteThreadCmd,
	"enableorgcode":               newEnableOrgCodeCmd,
	"enablepackage":               newEnablePackageCmd,
	"encodeid":                    newEncodeIDCmd,
	"entity":                      newEntityCmd,
	"getsetting":                  newGetSettingCmd,
	"initiatesync":                newInitiateSyncCmd,
	"migratebrokenimages":         newMigrateBrokenImagesCmd,
	"migratenotificationsettings": newMigrateNotificationSettingCmd,
	"migratesavedqueries":         newMigrateSavedQueriesCmd,
	"migratevmsettings":           newMigrateVMSettingCmd,
	"moveentity":                  newMoveEntityCmd,
	"rebuildsavedqueries":         newRebuildSavedQueriesCmd,
	"replacesavedqueries":         newReplaceSavedQueriesCmd,
	"setgreeting":                 newSetGreetingCmd,
	"setsetting":                  newSetSettingCmd,
	"thread":                      newThreadCmd,
	"updateentity":                newUpdateEntityCmd,
	"updateverifiedemail":         newUpdateVerifiedEmailCmd,
	"uploadlayout":                newUploadLayoutCmd,
	"zerobadges":                  newZeroBadgesCmd,
	"cleanupspamaccounts":         newCleanupSpamAccountsCmd,
}

func main() {
	golog.Default().SetLevel(golog.INFO)

	cnf := loadConfig()

	flag.StringVar(&cnf.DBHost, "db_host", cnf.DBHost, "mysql database `host`")
	flag.IntVar(&cnf.DBPort, "db_port", cnf.DBPort, "mysql database `port`")
	flag.StringVar(&cnf.DBUsername, "db_username", cnf.DBUsername, "mysql database `username`")
	flag.StringVar(&cnf.DBPassword, "db_password", cnf.DBPassword, "mysql database `password`")
	flag.StringVar(&cnf.DBTLS, "db_tls", cnf.DBTLS, "mysql database TLS setting (skip-verify or true)")
	flag.StringVar(&cnf.AuthAddr, "auth_addr", cnf.AuthAddr, "`host:port` of auth service")
	flag.StringVar(&cnf.DirectoryAddr, "directory_addr", cnf.DirectoryAddr, "`host:port` of directory service")
	flag.StringVar(&cnf.ExCommsAddr, "excomms_addr", cnf.ExCommsAddr, "`host:port` of excomms service")
	flag.StringVar(&cnf.ThreadingAddr, "threading_addr", cnf.ThreadingAddr, "`host:port` of treading service")
	flag.StringVar(&cnf.LayoutAddr, "layout_addr", cnf.LayoutAddr, "`host:port` of layout service")
	flag.StringVar(&cnf.InviteAddr, "invite_addr", cnf.InviteAddr, "`host:port` of invite service")
	flag.StringVar(&cnf.SettingsAddr, "settings_addr", cnf.SettingsAddr, "`host:port` of invite service")
	flag.StringVar(&cnf.Env, "env", cnf.Env, "environment that baymaxadmin is running against")

	flag.Parse()

	cmd := flag.Arg(0)

	environment.SetCurrent(cnf.Env)

	if cnf.AuthAddr == "" {
		cnf.AuthAddr = fmt.Sprintf("_auth._tcp.service.%s-us-east-1.spruce", strings.ToLower(cnf.Env))
	}
	if cnf.DirectoryAddr == "" {
		cnf.DirectoryAddr = fmt.Sprintf("_directory._tcp.service.%s-us-east-1.spruce", strings.ToLower(cnf.Env))
	}
	if cnf.ExCommsAddr == "" {
		cnf.ExCommsAddr = fmt.Sprintf("_excomms._tcp.service.%s-us-east-1.spruce", strings.ToLower(cnf.Env))
	}
	if cnf.ThreadingAddr == "" {
		cnf.ThreadingAddr = fmt.Sprintf("_threading._tcp.service.%s-us-east-1.spruce", strings.ToLower(cnf.Env))
	}
	if cnf.LayoutAddr == "" {
		cnf.LayoutAddr = fmt.Sprintf("_layout._tcp.service.%s-us-east-1.spruce", strings.ToLower(cnf.Env))
	}
	if cnf.InviteAddr == "" {
		cnf.InviteAddr = fmt.Sprintf("_invite._tcp.service.%s-us-east-1.spruce", strings.ToLower(cnf.Env))
	}
	if cnf.SettingsAddr == "" {
		cnf.SettingsAddr = fmt.Sprintf("_settings._tcp.service.%s-us-east-1.spruce", strings.ToLower(cnf.Env))
	}
	if cnf.PatientSyncAddr == "" {
		cnf.PatientSyncAddr = fmt.Sprintf("_patientsync._tcp.service.%s-us-east-1.spruce", strings.ToLower(cnf.Env))
	}

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
