package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/subosito/twilio"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/surescripts/pharmacy"
)

type twilioConfig struct {
	AccountSid string `long:"twilio_account_sid" description:"Twilio AccountSid"`
	AuthToken  string `long:"twilio_auth_token" description:"Twilio AuthToken"`
	FromNumber string `long:"twilio_from_number" description:"Twilio From Number for Messages"`

	client *twilio.Client
}

func (c *twilioConfig) Client() (*twilio.Client, error) {
	if c.client != nil {
		return c.client, nil
	}
	if c == nil {
		return nil, fmt.Errorf("Twilio config does not exist")
	}
	if c.AccountSid == "" {
		return nil, fmt.Errorf("Twilio.AccountSid not set")
	}
	if c.AuthToken == "" {
		return nil, fmt.Errorf("Twilio.AuthToken not set")
	}
	c.client = twilio.NewClient(c.AccountSid, c.AuthToken, nil)
	return c.client, nil
}

type stripeConfig struct {
	SecretKey      string `description:"Secrey Key for stripe"`
	PublishableKey string `description:"Publishable Key for stripe"`
}

type smartyStreetsConfig struct {
	AuthID    string `description:"Auth id for smarty streets"`
	AuthToken string `description:"Auth token for smarty streets"`
}

type analyticsConfig struct {
	LogPath   string `description:"Path to store analytics logs"`
	MaxEvents int    `description:"Max number of events per log file before rotating"`
	MaxAge    int    `description:"Max age of a log file in seconds before rotating"`
}

type supportConfig struct {
	TechnicalSupportEmail string `description:"Email address for technical support"`
	CustomerSupportEmail  string `description:"Customer support email address"`
}

type storageConfig struct {
	Type string
	// S3
	Region        string
	Bucket        string
	Prefix        string
	LatchedExpire bool `description:"If enabled then the signed URL is kept consistent within the expire time"`
}

type authTokenConfig struct {
	ExpireDuration int `description:"Expiration time in seconds for the auth token"`
	RenewDuration  int `description:"Time left below which to renew the auth token"`
}

type consulConfig struct {
	ConsulAddress   string `description:"Consul HTTP API host:port"`
	ConsulServiceID string `description:"Service ID for Consul. Only needed when running more than one instance on a host."`
}

type memcachedClusterConfig struct {
	DiscoveryHost     string   `description:"ElastiCache discovery host"`
	DiscoveryInterval int      `description:"Discovery interval in seconds"`
	Hosts             []string `description:"List of hosts when not using discovery"`
}

type rateLimiterConfig struct {
	Max    int `description:"Max number of actions in the given time period"`
	Period int `description:"Time period in seconds"`
}

type launchPromoConfig struct {
	StartDate time.Time `description:"time from which launch promo is to begin"`
}

type mainConfig struct {
	*config.BaseConfig
	ProxyProtocol                bool                             `long:"proxy_protocol" description:"Enable if behind a proxy that uses the PROXY protocol"`
	ListenAddr                   string                           `short:"l" long:"listen" description:"Address and port on which to listen (e.g. 127.0.0.1:8080)"`
	TLSListenAddr                string                           `long:"tls_listen" description:"Address and port on which to listen (e.g. 127.0.0.1:8080)"`
	TLSCert                      string                           `long:"tls_cert" description:"Path of SSL certificate"`
	TLSKey                       string                           `long:"tls_key" description:"Path of SSL private key"`
	APIDomain                    string                           `long:"api_domain" description:"Domain of REST API"`
	WebDomain                    string                           `long:"www_domain" description:"Domain of website"`
	APICDNDomain                 string                           `description:"Domain of CDN fronted REST API"`
	InfoAddr                     string                           `long:"info_addr" description:"Address to listen on for the info server"`
	DB                           *config.DB                       `group:"Database" toml:"database"`
	AnalyticsDB                  *config.DB                       `group:"AnalyticsDatabase" toml:"AnalyticsDatabase"`
	MaxInMemoryForPhotoMB        int64                            `long:"max_in_memory_photo" description:"Amount of data in MB to be held in memory when parsing multipart form data"`
	CaseBucket                   string                           `long:"case_bucket" description:"S3 Bucket name for case information"`
	Debug                        bool                             `long:"debug" description:"Enable debugging"`
	DoseSpotUserID               string                           `long:"dose_spot_user_id" description:"DoseSpot user ID for eRx integration"`
	NoServices                   bool                             `long:"noservices" description:"Disable connecting to remote services"`
	ERxRouting                   bool                             `long:"erx_routing" description:"Disable sending of prescriptions electronically"`
	ERxRoutingQueue              string                           `long:"erx_routing_queue" description:"ERx Routing Queue"`
	ERxStatusQueue               string                           `long:"erx_status_queue" description:"Erx queue name"`
	MedicalRecordQueue           string                           `long:"medical_record_queue" description:"Queue name for background generation of medical record"`
	VisitQueue                   string                           `long:"visit_queue" description:"Queue name for background charging and routing of patient visits"`
	VisitWorkerTimePeriodSeconds int                              `long:"visit_worker_time_period" description:"Time period between worker checking for messages in queue"`
	JBCQMinutesThreshold         int                              `long:"jbcq_minutes_threshold" description:"Threshold of inactivity between activities"`
	NumDoctorSelection           int                              `long:"num_doctor_selection" description:"number of doctors to return to select from"`
	OnboardingURLExpires         int64                            `long:"onboarding_url_expire_duration" description:"duration for which an onboarding url will stay valid"`
	RegularAuth                  *authTokenConfig                 `group:"regular_auth" toml:"regular_auth"`
	ExtendedAuth                 *authTokenConfig                 `group:"extended_auth" toml:"extended_auth"`
	StaticContentBaseURL         string                           `long:"static_content_base_url" description:"URL from which to serve static content"`
	Twilio                       *twilioConfig                    `group:"Twilio" toml:"twilio"`
	DoseSpot                     *config.DosespotConfig           `group:"Dosespot" toml:"dosespot"`
	Consul                       *consulConfig                    `group:"Consul" toml:"consul"`
	SmartyStreets                *smartyStreetsConfig             `group:"smarty_streets" toml:"smarty_streets"`
	TestStripe                   *stripeConfig                    `group:"test_stripe" toml:"test_stripe"`
	Stripe                       *stripeConfig                    `group:"stripe" toml:"stripe"`
	MinimumAppVersionConfigs     *config.MinimumAppVersionConfigs `group:"minimum_app_version"  toml:"minimum_app_version"`
	IOSDeeplinkScheme            string                           `long:"ios_deeplink_scheme" description:"Scheme for iOS deep-links (e.g. spruce://)"`
	NotifiyConfigs               *config.NotificationConfigs      `group:"notification" toml:"notification"`
	Analytics                    *analyticsConfig                 `group:"Analytics" toml:"analytics"`
	Support                      *supportConfig                   `group:"support" toml:"support"`
	Email                        *email.Config                    `group:"email" toml:"email"`
	PharmacyDB                   *pharmacy.Config                 `group:"pharmacy_database" toml:"pharmacy_database"`
	DiagnosisDB                  *config.DB                       `group:"diagnosis_database" toml:"diagnosis_database"`
	EventsDB                     *config.DB                       `group:"events_database" toml:"events_database"`
	Storage                      map[string]*storageConfig        `group:"storage" toml:"storage"`
	StaticResourceURL            string                           `long:"static_url" description:"URL prefix for static resources"`
	WebPassword                  string                           `long:"web_password" description:"Password to access website"`
	TwoFactorExpiration          int                              `description:"Time to live of two factor auth token in seconds"`
	OfficeNotifySNSTopic         string                           `description:"SNS Topic to send submitted visit notifications"`
	ExperimentID                 map[string]string                `description:"Google Analytics Experiment IDs"`
	CompressResponse             bool                             `description:"Compress the HTTP response"`
	LaunchPromo                  *launchPromoConfig               `group:"LaunchPromo" toml:"launch_promo" description:"Config to run launch promo"`
	Memcached                    map[string]*memcachedClusterConfig
	RateLimiters                 map[string]*rateLimiterConfig
	// Secret keys used for generating signatures
	SecretSignatureKeys []string
}

var defaultConfig = mainConfig{
	BaseConfig: &config.BaseConfig{
		AppName: "restapi",
	},
	DB: &config.DB{
		Name: "carefront",
		Host: "127.0.0.1",
		Port: 3306,
	},
	Twilio:                &twilioConfig{},
	APIDomain:             "api.sprucehealth.com",
	WebDomain:             "www.sprucehealth.com",
	ListenAddr:            ":8080",
	TLSListenAddr:         ":8443",
	InfoAddr:              ":9000",
	CaseBucket:            "carefront-cases",
	MaxInMemoryForPhotoMB: defaultMaxInMemoryPhotoMB,
	RegularAuth: &authTokenConfig{
		ExpireDuration: 60 * 60 * 24 * 2,
		RenewDuration:  60 * 60 * 36,
	},
	ExtendedAuth: &authTokenConfig{
		ExpireDuration: 60 * 60 * 24 * 30 * 2,
		RenewDuration:  60 * 60 * 24 * 45,
	},
	OnboardingURLExpires: 60 * 60 * 24 * 14,
	IOSDeeplinkScheme:    "spruce",
	Analytics: &analyticsConfig{
		MaxEvents: 100 << 10,
		MaxAge:    10 * 60, // seconds
	},
	TwoFactorExpiration: 10 * 60, // seconds
}

func (c *mainConfig) Validate() {
	var errors []string
	if c.ExperimentID == nil {
		c.ExperimentID = make(map[string]string)
	}
	if len(c.Storage) == 0 {
		errors = append(errors, "No storage configs set")
	}
	if c.Stripe == nil || c.Stripe.SecretKey == "" || c.Stripe.PublishableKey == "" {
		errors = append(errors, "No stripe key set")
	}
	if len(c.SecretSignatureKeys) == 0 {
		errors = append(errors, "No secret signature keys")
	}

	if !c.Debug {
		if c.TLSCert == "" {
			errors = append(errors, "TLSCert not set")
		}
		if c.TLSKey == "" {
			errors = append(errors, "TLSKey not set")
		}
	}
	if c.StaticResourceURL == "" {
		if os.Getenv("GOPATH") == "" {
			errors = append(errors, "StaticResourceURL not set")
		} else {
			// In dev we can use a local file server in the app
			c.StaticResourceURL = fmt.Sprintf("https://%s/static", c.WebDomain)
		}
	} else if n := len(c.StaticResourceURL); c.StaticResourceURL[n-1] == '/' {
		c.StaticResourceURL = c.StaticResourceURL[:n-1]
	}
	c.StaticResourceURL = strings.Replace(c.StaticResourceURL, "{BuildNumber}", config.BuildNumber, -1)
	if len(errors) != 0 {
		fmt.Fprintf(os.Stderr, "Config failed validation:\n")
		for _, e := range errors {
			fmt.Fprintf(os.Stderr, "- %s\n", e)
		}
		os.Exit(1)
	}
	if c.APICDNDomain == "" {
		c.APICDNDomain = c.APIDomain
	}
}

type twilioSMSAPI struct {
	*twilio.Client
}

func (sms *twilioSMSAPI) Send(fromNumber, toNumber, text string) error {
	_, _, err := sms.Client.Messages.SendSMS(fromNumber, toNumber, text)
	return err
}

type loggingSMSAPI struct{}

func (loggingSMSAPI) Send(fromNumber, toNumber, text string) error {
	golog.Infof("SMS: from=%s to=%s text=%s", fromNumber, toNumber, text)
	return nil
}
