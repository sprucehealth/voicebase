package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jeromer/syslogparser"
	"github.com/mcuadros/go-syslog"
)

// http://tools.ietf.org/html/rfc5424

var flagCloudTrail = flag.Bool("cloudtrail", false, "Enable CloudTrail log indexing")

type Facility int

const (
	Kernel Facility = iota
	User
	Mail
	Daemon
	Auth
	Syslog
	LPR
	News
	UUCP
	CRON
	AuthPriv
	FTP
	NTP
	LogAudit
	LogAlert
	ClockDaemon
	Local0
	Local1
	Local2
	Local3
	Local4
	Local5
	Local6
	Local7
)

var FacilityNames = map[Facility]string{
	Kernel:      "KERN",
	User:        "USER",
	Mail:        "MAIL",
	Daemon:      "DAEMON",
	Auth:        "AUTH",
	Syslog:      "SYSLOG",
	LPR:         "LPR",
	News:        "NEWS",
	UUCP:        "UUCP",
	CRON:        "CRON",
	AuthPriv:    "AUTHPRIV",
	FTP:         "FTP",
	NTP:         "NTP",
	LogAudit:    "AUDIT",
	LogAlert:    "ALERT",
	ClockDaemon: "CLOCK",
	Local0:      "LOCAL0",
	Local1:      "LOCAL1",
	Local2:      "LOCAL2",
	Local3:      "LOCAL3",
	Local4:      "LOCAL4",
	Local5:      "LOCAL5",
	Local6:      "LOCAL6",
	Local7:      "LOCAL7",
}

func (f Facility) String() string {
	if n := FacilityNames[f]; n != "" {
		return n
	}
	return strconv.Itoa(int(f))
}

type Severity int

const (
	Emergency Severity = iota
	Alert
	Critical
	Error
	Warning
	Notice
	Info
	Debug
)

var SeverityNames = map[Severity]string{
	Emergency: "EMERG",
	Alert:     "ALERT",
	Critical:  "CRIT",
	Error:     "ERR",
	Warning:   "WARNING",
	Notice:    "NOTICE",
	Info:      "INFO",
	Debug:     "DEBUG",
}

func (s Severity) String() string {
	if n := SeverityNames[s]; n != "" {
		return n
	}
	return strconv.Itoa(int(s))
}

type Entry struct {
	Time           time.Time
	Facility       Facility
	Severity       Severity
	Hostname       string
	AppName        string
	ProcId         string
	MsgId          string
	StructuredData string
	Message        string
}

type handler struct {
	es       *ElasticSearch
	appTypes map[string]string // App name to doc _type map
	jsonApps map[string]bool   // App names that output JSON
}

func (h *handler) Handle(parts syslogparser.LogParts) {
	ent := Entry{
		Hostname:       parts["hostname"].(string),
		AppName:        parts["app_name"].(string),
		ProcId:         parts["proc_id"].(string),
		MsgId:          parts["msg_id"].(string),
		StructuredData: parts["structured_data"].(string),
		Message:        parts["message"].(string),
		Severity:       Severity(parts["severity"].(int)),
		Facility:       Facility(parts["facility"].(int)),
		Time:           parts["timestamp"].(time.Time).UTC(),
	}
	var fields map[string]interface{}

	isJson, ok := h.jsonApps[ent.AppName]
	if !ok {
		// Attempt to decode the message as JSON and remember if it worked
		isJson = json.Unmarshal([]byte(ent.Message), &fields) == nil
		h.jsonApps[ent.AppName] = isJson
		if isJson {
			log.Printf("white listing JSON app %s\n", ent.AppName)
		}
	} else if isJson {
		if err := json.Unmarshal([]byte(ent.Message), &fields); err != nil {
			log.Printf("Failed to parse JSON, black listing %s\n", ent.AppName)
			isJson = false
		}
	}

	if !isJson {
		fields = map[string]interface{}{
			"@message": strings.TrimSpace(ent.Message), // The parser seems to leave a leading space
		}
	}

	// Used by Kibana
	fields["@timestamp"] = ent.Time.Format(time.RFC3339)
	fields["@version"] = "1"

	host := ent.Hostname

	// Normalize internal EC2 host names
	if strings.HasSuffix(host, ".ec2.internal") {
		host = host[:len(host)-13]
	}

	fields["@host"] = host
	fields["@app"] = ent.AppName
	fields["@proc"] = ent.ProcId
	fields["@severity"] = ent.Severity.String()
	fields["@facility"] = ent.Facility.String()

	// For now ignore the _ts field
	if _, ok := fields["_ts"]; ok {
		delete(fields, "_ts")
	}

	var idx string
	if s, ok := fields["_index"].(string); ok {
		idx = s
	}
	if idx != "" {
		delete(fields, "_index")
	} else {
		idx = fmt.Sprintf("log-%s", ent.Time.Format("2006.01.02"))
	}

	var doctype string
	if s, ok := fields["_type"].(string); ok {
		doctype = s
		delete(fields, "_type")
	}
	if doctype == "" {
		doctype = h.appTypes[ent.AppName]
		if doctype == "" {
			doctype = "syslog"
			if isJson {
				doctype = ent.AppName
			}
		}
	}

	if err := h.es.Index(idx, doctype, fields, ent.Time); err != nil {
		log.Printf("Failed to index %s: %+v\n", ent.AppName, err)
	}
}

func main() {
	flag.Parse()

	es := &ElasticSearch{
		Endpoint: "http://127.0.0.1:9200",
	}

	if *flagCloudTrail {
		if err := startCloudTrailIndexer(es); err != nil {
			log.Fatal(err)
		}
	}

	hand := &handler{
		es:       es,
		appTypes: map[string]string{},
		jsonApps: map[string]bool{
			"dhclient": false,
			"kernel":   false,
			"rsyslogd": false,
			"sshd":     false,
			"sudo":     false,

			"mysql-audit": true,
			"deploy":      true,
			"restapi":     true,
		},
	}

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5423)
	server.SetHandler(hand)
	if err := server.ListenTCP("127.0.0.1:1514"); err != nil {
		log.Fatal(err)
	}
	if err := server.Boot(); err != nil {
		log.Fatal(err)
	}

	server.Wait()
}
