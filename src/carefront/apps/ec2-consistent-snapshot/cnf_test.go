package main

import (
	"bytes"
	"testing"
)

func TestConfigParser(t *testing.T) {
	rd := bytes.NewBuffer([]byte(`
		[mysql]
		basic = 123
		with_comment=abc # testing

		[client]
		tabs    		 =    1122
		path-and-dash = /some/path
		# ignore = this
	`))
	cnf, err := parseConfig(rd)
	if err != nil {
		t.Fatal(err)
	}

	sec := cnf["mysql"]
	if sec == nil {
		t.Fatal("Section mysql not parsed")
	}
	if len(sec) != 2 {
		t.Fatalf("Expected 2 items in section 'mysql' found %d: %+v", len(sec), sec)
	}
	if sec["basic"] != "123" {
		t.Fatalf("Expected '123' for mysql.basic instead of '%s'", sec["basic"])
	}
	if sec["with_comment"] != "abc" {
		t.Fatalf("Expected 'abc' for mysql.with_comment instead of '%s'", sec["with_comment"])
	}

	sec = cnf["client"]
	if sec == nil {
		t.Fatal("Section client not parsed")
	}
	if len(sec) != 2 {
		t.Fatalf("Expected 2 items in section 'client' found %d: %+v", len(sec), sec)
	}
	if sec["tabs"] != "1122" {
		t.Fatalf("Expected '1122' for client.tabs instead of '%s'", sec["tabs"])
	}
	if sec["path-and-dash"] != "/some/path" {
		t.Fatalf("Expected '/some/path' for client.path-and-dash instead of '%s'", sec["path-and-dash"])
	}
}
