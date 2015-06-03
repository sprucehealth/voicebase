package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func merge(path string, counts map[string]int) string {
	rd, err := os.Open(path)
	if err != nil {
		log.Fatalf("Failed to open %s: %s", path, err)
	}
	defer rd.Close()

	// First line is "mode: foo", where foo is "set", "count", or "atomic".
	// Rest of file is in the format
	//	encoding/base64/base64.go:34.44,37.40 3 1
	// where the fields are: name.go:line.column,line.column numberOfStatements count

	scanner := bufio.NewScanner(rd)
	// Skip the first line which is "mode: set" (or count, atomic)
	if !scanner.Scan() {
		log.Fatalf("Empty file %s", path)
	}
	mode := scanner.Text()
	for scanner.Scan() {
		line := scanner.Text()
		ix := strings.LastIndex(line, " ")
		if ix <= 0 {
			log.Fatalf("Invalid format for %s: missing space in line", path)
		}
		block := line[:ix]
		count, err := strconv.Atoi(line[ix+1:])
		if err != nil {
			log.Fatalf("Invalid format for %s: %s", path, err)
		}
		counts[block] += count
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Failed scanning %s: %s", path, err)
	}
	return mode
}

func main() {
	log.SetFlags(log.Lshortfile)

	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: covermerge outputpath rootpath")
		os.Exit(1)
	}

	outfilepath := os.Args[1]
	rootpath := os.Args[2]

	var mode string
	counts := make(map[string]int)
	filepath.Walk(rootpath, func(path string, info os.FileInfo, err error) error {
		if info.Name() != "cover.out" {
			return nil
		}
		fmt.Printf("Merging %s...\n", path)
		m := merge(path, counts)
		if mode == "" {
			mode = m
		} else if m != mode {
			log.Fatalf("Mode mismatch for %s. Wanted %s got %s", path, mode, m)
		}
		return nil
	})

	// Sort the output
	output := make([]string, 0, len(counts))
	for block, n := range counts {
		output = append(output, block+" "+strconv.Itoa(n)+"\n")
	}
	sort.Strings(output)

	fi, err := os.Create(outfilepath)
	if err != nil {
		log.Fatalf("Failed to create output file '%s': %s", outfilepath, err)
	}
	defer fi.Close()
	if _, err := fmt.Fprintln(fi, mode); err != nil {
		log.Fatal(err)
	}
	for _, line := range output {
		if _, err := fi.Write([]byte(line)); err != nil {
			log.Fatal(err)
		}
	}
}
