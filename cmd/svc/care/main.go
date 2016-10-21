package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"time"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/care/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/care/internal/server"
	caresettings "github.com/sprucehealth/backend/cmd/svc/care/settings"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/dosespot"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/settings"
)

var config struct {
	dbCACert             string
	dbHost               string
	dbName               string
	dbPassword           string
	dbPort               int
	dbTLS                string
	dbUserName           string
	doseSpotClinicID     int64
	doseSpotClinicKey    string
	doseSpotSOAPEndpoint string
	doseSpotUserID       int64
	layoutAddr           string
	listeningPort        int
	s3Bucket             string
	s3Prefix             string
	settingsAddr         string
	mediaAddr            string
}

func init() {
	flag.StringVar(&config.dbHost, "db_host", "", "database host")
	flag.StringVar(&config.dbPassword, "db_password", "", "database password")
	flag.StringVar(&config.dbName, "db_name", "", "database name")
	flag.StringVar(&config.dbUserName, "db_username", "", "database username")
	flag.IntVar(&config.dbPort, "db_port", 3306, "database port")
	flag.StringVar(&config.dbCACert, "db_ca_cert", "", "Path to database CA certificate")
	flag.StringVar(&config.dbTLS, "db_tls", "", "Enable TLS for database connection (one of 'true', 'false', 'skip-verify'). Ignored if CA cert provided.")
	flag.IntVar(&config.listeningPort, "listening_port", 0, "Port on which visit service should listen")
	flag.StringVar(&config.s3Bucket, "s3_bucket", "", "name of S3 bucket where layouts are stored")
	flag.StringVar(&config.s3Prefix, "s3_prefix", "", "prefix for layouts in s3 bucket")
	flag.StringVar(&config.doseSpotClinicKey, "dosespot_clinic_key", "", "DoseSpot clinic key")
	flag.StringVar(&config.doseSpotSOAPEndpoint, "dosespot_soap_endpoint", "", "DoseSpot SOAP endpoint URL")
	flag.Int64Var(&config.doseSpotClinicID, "dosespot_clinic_id", 0, "DoseSpot clinic ID")
	flag.Int64Var(&config.doseSpotUserID, "dosespot_user_id", 0, "DoseSpot user ID")

	// Services
	flag.StringVar(&config.layoutAddr, "layout_addr", "_layout._tcp.service", "`host:port` to communicate with the layout service")
	flag.StringVar(&config.settingsAddr, "settings_addr", "_settings._tcp.service", "`host:port` of settings service")
	flag.StringVar(&config.mediaAddr, "media_addr", "_media._tcp.service", "`host:port of media service`")
}

func main() {
	svc := boot.NewService("care", nil)

	switch {
	case config.s3Bucket == "":
		golog.Fatalf("s3_bucket required")
	case config.s3Prefix == "":
		golog.Fatalf("s3_prefix required")
	case config.listeningPort == 0:
		golog.Fatalf("listening_port required")
	case config.layoutAddr == "":
		golog.Fatalf("layout_addr required")
	case config.doseSpotClinicKey == "":
		golog.Fatalf("dose_spot_clinic_key required")
	case config.doseSpotSOAPEndpoint == "":
		golog.Fatalf("dosespot_soap_endpoint required")
	case config.doseSpotClinicID == 0:
		golog.Fatalf("dosespot_clinic_id required")
	case config.doseSpotUserID == 0:
		golog.Fatalf("dosespot_user_id required")
	case config.settingsAddr == "":
		golog.Fatalf("settings_addr required")
	}

	db, err := dbutil.ConnectMySQL(&dbutil.DBConfig{
		User:          config.dbUserName,
		Password:      config.dbPassword,
		Host:          config.dbHost,
		Port:          config.dbPort,
		Name:          config.dbName,
		CACert:        config.dbCACert,
		EnableTLS:     config.dbTLS == "true" || config.dbTLS == "skip-verify",
		SkipVerifyTLS: config.dbTLS == "skip-verify",
	})
	if err != nil {
		golog.Fatalf(err.Error())
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.listeningPort))
	if err != nil {
		golog.Fatalf(err.Error())
	}

	conn, err := svc.DialGRPC(config.layoutAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to directory service: %s", err)
	}
	layoutClient := layout.NewLayoutClient(conn)

	conn, err = svc.DialGRPC(config.settingsAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to settings service :%s", err)
	}
	settingsClient := settings.NewSettingsClient(conn)

	conn, err = svc.DialGRPC(config.mediaAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to media service :%s", err)
	}
	mediaClient := media.NewMediaClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = settings.RegisterConfigs(ctx, settingsClient, []*settings.Config{
		caresettings.OptionalTriageConfig,
	})
	if err != nil {
		golog.Fatalf("Unable to register configs with settings service: %s", err.Error())
	}
	cancel()

	doseSpotClient := dosespot.New(config.doseSpotClinicID, config.doseSpotUserID, config.doseSpotClinicKey, config.doseSpotSOAPEndpoint, "http://www.dosespot.com/API/11/", svc.MetricsRegistry.Scope("dosespot"))

	awsSession, err := svc.AWSSession()
	if err != nil {
		golog.Fatalf(err.Error())
	}

	careServer := svc.GRPCServer()
	careService := server.New(dal.New(db), layoutClient, settingsClient, mediaClient, layout.NewStore(storage.NewS3(awsSession, config.s3Bucket, config.s3Prefix)), doseSpotClient, clock.New())

	care.InitMetrics(careServer, svc.MetricsRegistry.Scope("care"))
	care.RegisterCareServer(careServer, careService)

	conc.Go(func() {
		golog.Infof("Starting visit service on port %d", config.listeningPort)
		if err := careServer.Serve(lis); err != nil {
			golog.Errorf(err.Error())
		}
	})

	boot.WaitForTermination()
	svc.Shutdown()
}
