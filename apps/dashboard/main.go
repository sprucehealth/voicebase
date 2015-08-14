package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"reflect"
	"regexp"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/sprucehealth/backend/libs/geckoboard"
	"github.com/sprucehealth/backend/libs/golog"
	"gopkg.in/yaml.v2"
)

const (
	maxRows = 1000
)

type config struct {
	APIKey      string            `json:"api_key" yaml:"api_key"`
	AnalyticsDB *dbConfig         `json:"analytics_db" yaml:"analytics_db"`
	Queries     map[string]string `json:"queries" yaml:"queries"`
	Widgets     []*widgetConfig   `json:"widgets" yaml:"widgets"`
}

type dbConfig struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Name     string `json:"name" yaml:"name"`
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
}

type widgetConfig struct {
	Name    string        `json:"name" yaml:"name"`
	Keys    []string      `json:"keys" yaml:"keys"`
	Type    string        `json:"type" yaml:"type"`
	Period  time.Duration `json:"period" yaml:"period"`
	Queries []*query      `json:"queries" yaml:"queries"`
	Config  string        `json:"config" yaml:"config"`
}

type query struct {
	Name         string            `json:"name" yaml:"name"`
	Query        string            `json:"query" yaml:"query"`
	Params       []interface{}     `json:"params" yaml:"params"`
	Replacements map[string]string `json:"replacements" yaml:"replacements"`
}

var widgets = map[string]reflect.Type{
	"pie-chart":                 reflect.TypeOf(geckoboard.PieChart{}),
	"bar-chart":                 reflect.TypeOf(geckoboard.BarChart{}),
	"line-chart":                reflect.TypeOf(geckoboard.LineChart{}),
	"number-and-secondary-stat": reflect.TypeOf(geckoboard.NumberAndSecondaryStat{}),
	"map":         reflect.TypeOf(geckoboard.Map{}),
	"list":        reflect.TypeOf(geckoboard.List{}),
	"funnel":      reflect.TypeOf(geckoboard.Funnel{}),
	"text":        reflect.TypeOf(geckoboard.Text{}),
	"leaderboard": reflect.TypeOf(geckoboard.Leaderboard{}),
}

var (
	flagConfig  = flag.String("c", "dashboard.yml", "Path to config file")
	flagVerbose = flag.Bool("v", false, "Verbose output")
	flagWidgets = flag.String("w", "", "Regular expression matching widget names to update")
)

func loadConfig() (*config, error) {
	b, err := ioutil.ReadFile(*flagConfig)
	if err != nil {
		return nil, err
	}
	var conf config
	return &conf, yaml.Unmarshal(b, &conf)
}

func main() {
	flag.Parse()
	if *flagVerbose {
		golog.Default().SetLevel(golog.DEBUG)
	}

	conf, err := loadConfig()
	if err != nil {
		golog.Fatalf("Failed to load config: %s", err)
	}

	db, err := connectToDB(conf)
	if err != nil {
		golog.Fatalf("Failed to connect to analytics DB: %s", err)
	}
	defer db.Close()

	gb := geckoboard.NewClient(conf.APIKey)

	var re *regexp.Regexp
	if *flagWidgets != "" {
		re, err = regexp.Compile(*flagWidgets)
		if err != nil {
			golog.Fatalf("Failed to parse widget name regex: %s", err)
		}
	}

	for _, w := range conf.Widgets {
		if re == nil || re.MatchString(w.Name) {
			processWidget(db, gb, conf, w)
		}
	}
}

func processWidget(db *sql.DB, gb *geckoboard.Client, conf *config, w *widgetConfig) {
	golog.Debugf("Processing widget %s", w.Name)

	typ := widgets[w.Type]
	if typ == nil {
		golog.Errorf("No widget of type %s for %s", w.Type, w.Name)
		return
	}
	widget := reflect.New(typ).Interface().(geckoboard.Widget)
	if len(w.Config) != 0 {
		if err := json.Unmarshal([]byte(w.Config), widget); err != nil {
			golog.Errorf("Failed to decode config for %s: %s", w.Name, err)
			return
		}
	}

	for _, query := range w.Queries {
		q := query.Query
		if q == "" {
			q = conf.Queries[query.Name]
		}
		if q == "" {
			golog.Errorf("Empty query for %s", w.Name)
			continue
		}
		for name, rep := range query.Replacements {
			q = strings.Replace(q, "{"+name+"}", rep, -1)
		}
		cols, rows, err := queryDB(db, q, query.Params...)
		if err != nil {
			golog.Errorf("Failed query for %s: %s", w.Name, err)
			return
		}
		for _, r := range rows {
			if err := widget.AppendData(cols, r); err != nil {
				golog.Errorf("Bad data for %s: %s", w.Name, err)
				return
			}
		}
	}

	for _, key := range w.Keys {
		if err := gb.Push(key, widget); err != nil {
			golog.Errorf("Failed to push data for %s: %s", w.Name, err)
		}
	}
}

func connectToDB(conf *config) (*sql.DB, error) {
	dbArgs := fmt.Sprintf("host=%s port=%d dbname=%s sslmode=%s",
		conf.AnalyticsDB.Host, conf.AnalyticsDB.Port, conf.AnalyticsDB.Name, "require")
	if conf.AnalyticsDB.Username != "" {
		dbArgs += " user=" + conf.AnalyticsDB.Username
	}
	if conf.AnalyticsDB.Password != "" {
		dbArgs += " password=" + conf.AnalyticsDB.Password
	}

	// enableTLS := c.CACert != "" && c.TLSCert != "" && c.TLSKey != ""
	// if !enableTLS && strings.ToLower(c.Host) == "localhost" {
	// 	dbArgs += " sslmode=disable"
	// }

	db, err := sql.Open("postgres", dbArgs)
	if err != nil {
		return nil, err
	}
	// Make sure the database connection is working
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func queryDB(db *sql.DB, query string, params ...interface{}) ([]string, [][]interface{}, error) {
	rows, err := db.Query(query, params...)
	if err != nil {
		// TODO: This is super janky, but there's something either wrong with Redshift, the Postgres driver,
		// or the sql package that causes the next query to fail (causing a panic) following a bad query.
		// To contain this execute a query and recover which seems to fix it. Need to figure out what's going on,
		// but for now this "works"
		func() {
			defer func() {
				_ = recover()
			}()
			var x int
			_ = db.QueryRow("SELECT 1").Scan(&x)
		}()
		return nil, nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}
	valPtrs := make([]interface{}, len(cols))
	var res [][]interface{}
	for rows.Next() {
		// rows.Scan requires ptrs to values so give it pointers to interfaces. This
		// feels terrible and one of the only places one will see pointers to interfaces,
		// but I can't think of a better way to do it.
		vals := make([]interface{}, len(cols))
		for i := 0; i < len(vals); i++ {
			valPtrs[i] = &vals[i]
		}
		if err := rows.Scan(valPtrs...); err != nil {
			return nil, nil, err
		}
		for i, v := range vals {
			switch x := v.(type) {
			case []byte:
				vals[i] = string(x)
			}
		}
		res = append(res, vals)
		if len(res) > maxRows {
			break
		}
	}
	return cols, res, rows.Err()
}
