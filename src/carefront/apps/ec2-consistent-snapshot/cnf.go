package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var (
	reSection = regexp.MustCompile(`^\s*\[([^\]]+)\]`)
	reValue   = regexp.MustCompile(`^\s*([a-zA-Z_-]+)\s*=\s*([^#]+)\s*(#.*)?`)
)

// A poor man's parser for the MySQL config file format. Return a map from section to a map of key, value pairs.
func ParseConfig(rd io.Reader) (map[string]map[string]string, error) {
	bufr := bufio.NewReader(rd)
	cnf := make(map[string]map[string]string)
	var sectionName string
	var section map[string]string
	lineNum := 0
	for {
		lineNum++
		line, err := bufr.ReadString('\n')
		if err == io.EOF {
			return cnf, nil
		} else if err != nil {
			return cnf, err
		}

		if m := reSection.FindStringSubmatch(line); len(m) > 0 {
			sectionName = strings.ToLower(m[1])
			section = cnf[sectionName]
			if section == nil {
				section = make(map[string]string)
				cnf[sectionName] = section
			}
		} else if m := reValue.FindStringSubmatch(line); len(m) > 0 {
			key := m[1]
			value := strings.TrimSpace(m[2])
			if section == nil {
				return cnf, fmt.Errorf("Key %s found outside of a section at line %d", key, lineNum)
			}
			section[key] = value
		}
	}
}
