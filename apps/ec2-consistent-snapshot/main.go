package main

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/libs/aws"
	"github.com/sprucehealth/backend/libs/aws/ec2"
	"github.com/sprucehealth/backend/libs/cmd/cryptsetup"
	"github.com/sprucehealth/backend/libs/cmd/dmsetup"
	"github.com/sprucehealth/backend/libs/cmd/lvm"
	"github.com/sprucehealth/backend/libs/cmd/mount"
	"github.com/sprucehealth/backend/libs/cmd/xfs"

	"github.com/sprucehealth/backend/third_party/github.com/go-sql-driver/mysql"
)

type freezeCmd interface {
	Freeze(path string) error
	Thaw(path string) error
}

var config = struct {
	Verbose bool
	Tags    map[string]string
	// AWS
	AWSRole    string
	AWSKeys    aws.Keys
	Region     string
	InstanceID string
	// FS Freeze
	MountPath  string
	FSType     string
	Devices    []string // Used to lookup which EBS volumes to snapshot
	EBSVolumes []string // IDs for the EBS volumes
	// MySQL
	Config     string
	Host       string
	Port       int
	Socket     string
	Username   string
	Password   string
	CACert     string
	ClientCert string
	ClientKey  string

	freezeCmd     freezeCmd
	db            *sql.DB
	awsAuth       aws.Auth
	ec2           *ec2.EC2
	ebsVolumeInfo []*ec2.Volume
}{
	Host:     "127.0.0.1",
	Port:     3306,
	Socket:   "/var/run/mysqld/mysqld.sock",
	Username: "root",
}

var cnfSearchPath = []string{
	"~/.my.cnf",
	"/etc/mysql/my.cnf",
}

type stringListFlag struct {
	Values *[]string
}

func (sl stringListFlag) String() string {
	return strings.Join(*sl.Values, ",")
}

func (sl stringListFlag) Set(s string) error {
	*sl.Values = append(*sl.Values, s)
	return nil
}

type mapFlag struct {
	Values *map[string]string
}

func (mf mapFlag) String() string {
	return fmt.Sprintf("%+v", *mf.Values)
}

func (mf mapFlag) Set(s string) error {
	idx := strings.Index(s, "=")
	if idx <= 0 {
		return fmt.Errorf("tag arguments must be name=value")
	}
	if *mf.Values == nil {
		*mf.Values = make(map[string]string)
	}
	(*mf.Values)[s[:idx]] = s[idx+1:]
	return nil
}

func init() {
	flag.StringVar(&config.MountPath, "fs", config.MountPath, "Path to filesystem to freeze")
	flag.StringVar(&config.FSType, "fs.type", config.FSType, "Filesystem type (support: xfs)")
	flag.StringVar(&config.AWSRole, "role", config.AWSRole, "AWS Role")
	flag.StringVar(&config.Region, "region", config.Region, "EC2 Region")
	flag.StringVar(&config.Config, "mysql.config", config.Config, "Path to my.cnf")
	flag.StringVar(&config.Host, "mysql.host", config.Host, "MySQL host")
	flag.IntVar(&config.Port, "mysql.port", config.Port, "MySQL port")
	flag.StringVar(&config.Username, "mysql.user", config.Username, "MySQL username")
	flag.StringVar(&config.Password, "mysql.password", config.Password, "MySQL password")
	flag.Var(mapFlag{Values: &config.Tags}, "tag", "Additional tags (e.g. -tag name=value)")
	flag.BoolVar(&config.Verbose, "v", config.Verbose, "Verbose output")
}

func readMySQLConfig(path string) error {
	fi, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fi.Close()

	cnf, err := parseConfig(fi)
	if err != nil {
		return err
	}

	for _, secName := range []string{"client", "ec2-consistent-snapshot"} {
		if sec := cnf[secName]; sec != nil {
			if port, err := strconv.Atoi(sec["port"]); err == nil {
				config.Port = port
			}
			if s := sec["host"]; s != "" {
				config.Host = s
			}
			if s := sec["socket"]; s != "" {
				config.Socket = s
			}
			if s := sec["ssl-ca"]; s != "" {
				config.CACert = s
			}
			if s := sec["ssl-cert"]; s != "" {
				config.ClientCert = s
			}
			if s := sec["ssl-key"]; s != "" {
				config.ClientKey = s
			}
			if s := sec["user"]; s != "" {
				config.Username = s
			}
			if s := sec["password"]; s != "" {
				config.Password = s
			}
		}
	}

	if config.MountPath == "" {
		if sec := cnf["mysqld"]; sec != nil {
			if s := sec["datadir"]; s != "" {
				mounts, err := mount.Default.GetMounts()
				if err != nil {
					log.Printf("Failed to get mounts: %+v", err)
				} else {
					longest := ""
					for path := range mounts {
						if path != "/" && strings.HasPrefix(s, path) && len(path) > len(longest) {
							longest = path
						}
					}
					config.MountPath = longest
				}
			}
		}
	}

	return nil
}

func mysqlConfig() {
	for _, path := range cnfSearchPath {
		if path[0] == '~' {
			path = os.Getenv("HOME") + path[1:]
		}
		readMySQLConfig(path) // Ignore error. TODO: could make sure it's "file not found"
	}

	if config.Config != "" {
		if config.Config[0] == '~' {
			config.Config = os.Getenv("HOME") + config.Config[1:]
		}
		if err := readMySQLConfig(config.Config); err != nil {
			log.Fatal(err)
		}
	}

	if config.Username == "" {
		config.Username = os.Getenv("MYSQL_USERNAME")
	}
	if config.Password == "" {
		config.Password = os.Getenv("MYSQL_PASSWORD")
	}
}

func info(st string, args ...interface{}) {
	if config.Verbose {
		fmt.Printf(st, args...)
	}
}

func devMap(dev string) string {
	if len(dev) == 8 && strings.HasPrefix(dev, "/dev/sd") {
		return "/dev/xvd" + string(dev[len(dev)-1])
	}
	return dev
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	// MySQL config

	mysqlConfig()

	if config.MountPath == "" {
		log.Fatalf("Missing required option -fs")
	}
	info("Mount path: %s\n", config.MountPath)

	mounts, err := mount.Default.GetMounts()
	if err != nil {
		log.Fatalf("Failed to get mounts: %+v", err)
	}
	mnt := mounts[config.MountPath]
	if mnt == nil {
		log.Fatalf("Mount not found for path %s", config.MountPath)
	}

	if config.FSType == "" {
		switch mnt.Type {
		default:
			log.Fatalf("Don't know how to freeze filesystem of type %s", mnt.Type)
		case "xfs":
			config.freezeCmd = xfs.Default
		}
	}

	// Resolve devices from the mount point. It may be LUKS and/or LVM.
	if len(config.Devices) == 0 {
		device := mnt.Device
		for {
			dev, err := dmsetup.Default.DMInfo(device)
			if err != nil {
				// not LUKS/LVM
				config.Devices = []string{device}
				break
			} else if strings.HasPrefix(dev.UUID, "CRYPT-LUKS") {
				cs, err := cryptsetup.Default.Status(device)
				if err != nil {
					log.Fatalf("cryptsetup status filed: %+v", err)
				}
				device = cs.Device
			} else if strings.HasPrefix(dev.UUID, "LVM-") {
				info, err := lvm.Default.LVS(device)
				if err != nil {
					log.Fatalf("lvs failed: %+v", err)
				}
				config.Devices = info.Devices
				break
			} else {
				config.Devices = []string{device}
				break
			}
		}
	}
	info("Devices: %s\n", strings.Join(config.Devices, " "))

	if config.AWSRole != "" {
		if config.AWSRole == "*" {
			config.AWSRole = ""
		}
		cred, err := aws.CredentialsForRole(config.AWSRole)
		if err != nil {
			log.Fatal(err)
		}
		config.awsAuth = cred
	} else {
		if keys := aws.KeysFromEnvironment(); keys.AccessKey == "" || keys.SecretKey == "" {
			if cred, err := aws.CredentialsForRole(""); err == nil {
				config.awsAuth = cred
			} else {
				log.Fatal("Missing AWS_ACCESS_KEY or AWS_SECRET_KEY")
			}
		} else {
			config.awsAuth = keys
		}
	}

	if config.Region == "" {
		az, err := aws.GetMetadata(aws.MetadataAvailabilityZone)
		if err != nil {
			log.Fatalf("no region specified and failed to get from instance metadata: %+v", err)
		}
		config.Region = az[:len(az)-1]
		info("Region: %s\n", config.Region)
	}

	config.ec2 = &ec2.EC2{
		Region: aws.Regions[config.Region],
		Client: &aws.Client{Auth: config.awsAuth},
	}

	if config.InstanceID == "" {
		var err error
		config.InstanceID, err = aws.GetMetadata(aws.MetadataInstanceID)
		if err != nil {
			log.Fatalf("Failed to get instance ID: %+v", err)
		}
		info("InstanceID: %s\n", config.InstanceID)
	}

	// Lookup EBS volumes for devices
	if len(config.EBSVolumes) == 0 {
		vol, err := config.ec2.DescribeVolumes(nil, map[string][]string{
			"attachment.instance-id": []string{config.InstanceID},
		})
		if err != nil {
			log.Fatalf("Failed to get volumes: %+v", err)
		}
		config.EBSVolumes = make([]string, len(config.Devices))
		config.ebsVolumeInfo = make([]*ec2.Volume, len(config.Devices))
		count := len(config.Devices)
		info("Attached volumes:\n")
		for _, v := range vol {
			if v.Attachment != nil {
				info("\t%s %s %s\n", v.VolumeID, v.Attachment.Device, v.Attachment.Status)
				for j, d := range config.Devices {
					if d == devMap(v.Attachment.Device) {
						config.EBSVolumes[j] = v.VolumeID
						config.ebsVolumeInfo[j] = v
						count--
						break
					}
				}
			}
		}
		if count != 0 {
			log.Fatalf("Only found %d volumes out of an expected %d", len(config.Devices)-count, len(config.Devices))
		}
	} else {
		vol, err := config.ec2.DescribeVolumes(config.EBSVolumes, nil)
		if err != nil {
			log.Fatalf("Failed to get volumes: %+v", err)
		}
		if len(vol) != len(config.EBSVolumes) {
			log.Fatalf("Not all volumes found")
		}
		config.ebsVolumeInfo = make([]*ec2.Volume, len(config.EBSVolumes))
		for i, v := range vol {
			if config.EBSVolumes[i] != v.VolumeID {
				log.Fatalf("VolumeID mismatch")
			}
			config.ebsVolumeInfo[i] = v
		}
	}

	enableTLS := config.CACert != "" && config.ClientCert != "" && config.ClientKey != ""
	if enableTLS {
		rootCertPool := x509.NewCertPool()
		pem, err := ioutil.ReadFile(config.CACert)
		if err != nil {
			log.Fatal(err)
		}
		if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
			log.Fatal("Failed to append PEM.")
		}
		clientCert := make([]tls.Certificate, 0, 1)
		certs, err := tls.LoadX509KeyPair(config.ClientCert, config.ClientKey)
		if err != nil {
			log.Fatal(err)
		}
		clientCert = append(clientCert, certs)
		mysql.RegisterTLSConfig("custom", &tls.Config{
			RootCAs:            rootCertPool,
			Certificates:       clientCert,
			InsecureSkipVerify: true,
		})
	}

	tlsOpt := ""
	if enableTLS {
		tlsOpt = "?tls=custom"
	}
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s%s", config.Username, config.Password, config.Host, config.Port, "mysql", tlsOpt))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// db.SetMaxOpenConns(1)
	// db.SetMaxIdleConns(1)

	// test the connection to the database by running a ping against it
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	config.db = db

	if err := doIt(); err != nil {
		log.Fatal(err)
	}
}

func doIt() (err error) {
	var binlogName string
	var binlogPos int64
	if binlogName, binlogPos, err = lockDB(); err != nil {
		err = fmt.Errorf("failed to lock DB: %s", err.Error())
		return
	}
	defer func() {
		e := unlockDB()
		// Don't overwrite other errors
		if err == nil {
			err = e
		} else if e != nil {
			log.Printf("Failed to unlock DB: %s", e.Error())
		}
	}()

	if err = config.freezeCmd.Freeze(config.MountPath); err != nil {
		err = fmt.Errorf("failed to freeze filesystem: %s", err.Error())
		return
	}
	defer func() {
		e := config.freezeCmd.Thaw(config.MountPath)
		// Don't overwrite other errors
		if err == nil {
			err = e
		} else if e != nil {
			log.Printf("Failed to thaw filesystem: %s", e.Error())
		}
	}()

	err = snapshotEBS(binlogName, binlogPos)
	return
}

func lockDB() (string, int64, error) {
	fmt.Println("Locking database...")

	// Don't pass FLUSH TABLES statements on to replication slaves
	// as this can interfere with long-running queries on the slaves.
	if _, err := config.db.Exec("SET SQL_LOG_BIN=0"); err != nil {
		return "", 0, err
	}

	fmt.Println("Flushing tables without lock...")
	// Try a flush first without locking so the later flush with lock
	// goes faster.  This may not be needed as it seems to interfere with
	// some statements anyway.
	if _, err := config.db.Exec("FLUSH LOCAL TABLES"); err != nil {
		return "", 0, err
	}

	fmt.Println("Aquiring lock...")
	// Get a lock on the entire database
	if _, err := config.db.Exec("FLUSH LOCAL TABLES WITH READ LOCK"); err != nil {
		return "", 0, err
	}

	// This might be a slave database already
	// my $slave_status = $mysql_dbh->selectrow_hashref(q{ SHOW SLAVE STATUS });
	// $mysql_logfile           = $slave_status->{Slave_IO_State}
	//                          ? $slave_status->{Master_Log_File}
	//                          : undef;
	// $mysql_position          = $slave_status->{Read_Master_Log_Pos};
	// $mysql_binlog_do_db      = $slave_status->{Replicate_Do_DB};
	// $mysql_binlog_ignore_db  = $slave_status->{Replicate_Ignore_DB};

	fmt.Println("Getting master status...")
	// or this might be the master
	// File | Position | Binlog_Do_DB | Binlog_Ignore_DB | Executed_Gtid_Set
	var binlogFile, binlogDoDB, binlogIgnoreDB, executedGtidSet string
	var binlogPos int64
	if err := config.db.QueryRow("SHOW MASTER STATUS").Scan(&binlogFile, &binlogPos, &binlogDoDB, &binlogIgnoreDB, &executedGtidSet); err != nil {
		return "", 0, err
	}

	fmt.Printf("File=%s Position=%d Binlog_Do_DB=%s Binlog_Ignore_DB=%s Executed_Gtid_Set=%s\n", binlogFile, binlogPos, binlogDoDB, binlogIgnoreDB, executedGtidSet)

	if _, err := config.db.Exec("SET SQL_LOG_BIN=1"); err != nil {
		return binlogFile, binlogPos, err
	}

	return binlogFile, binlogPos, nil
}

func unlockDB() error {
	fmt.Println("Unlocking tables...")
	_, err := config.db.Exec("UNLOCK TABLES")
	return err
}

func snapshotEBS(binlogName string, binlogPos int64) error {
	timestamp := time.Now().Format(time.RFC3339)
	for _, vol := range config.ebsVolumeInfo {
		fmt.Printf("Snapshotting %s (%s)...", vol.VolumeID, vol.Tags["Name"])
		res, err := config.ec2.CreateSnapshot(vol.VolumeID, fmt.Sprintf("%s %s", vol.Tags["Group"], timestamp))
		if err != nil {
			log.Fatalf("Failed to create snapshot of %s: %+v", vol.VolumeID, err)
		}
		fmt.Printf(" %s %s\n", res.SnapshotID, res.Status)
		tags := vol.Tags
		tags["BinlogName"] = binlogName
		tags["BinlogPos"] = strconv.FormatInt(binlogPos, 10)
		for n, v := range config.Tags {
			tags[n] = v
		}
		if err := config.ec2.CreateTags([]string{res.SnapshotID}, tags); err != nil {
			log.Printf("Failed to tag snapshot %s: %+v", res.SnapshotID, err)
		}
	}
	return nil
}
