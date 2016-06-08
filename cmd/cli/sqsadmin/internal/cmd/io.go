package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func prompt(scn *bufio.Scanner, prompt string) string {
	fmt.Print(prompt)
	if !scn.Scan() {
		os.Exit(1)
	}
	return strings.TrimSpace(scn.Text())
}

func pprint(fs string, args ...interface{}) {
	fmt.Printf(fs, args...)
}
