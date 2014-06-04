package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

func main() {
	RunTests()
}

// RunTests identifies all test packages under the ./src/carefront/integration folder
// and then iteratively runs through all tests. Any folder listed as test_X is identified to be a
// folder with tests within it. Note that this method is currently setup to run tests from the top most level
// of the repository because thats how Travis runs the tests
func RunTests() {
	testPath := "carefront/test"
	files, _ := ioutil.ReadDir(path.Join(os.Getenv("GOPATH"), "src", testPath))
	testDirs := make([]string, 0)
	for _, f := range files {
		if f.IsDir() && strings.HasPrefix(f.Name(), "test_") {
			testDir := path.Join(testPath, f.Name())
			testDirs = append(testDirs, testDir)
			args := []string{"go", "test", "-v", "-race", "-test.timeout=50m", testDir}
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Println("FAILED to run command successfully: " + err.Error())
				os.Exit(1)
			}
		}
	}
	fmt.Printf("Ran tests under:\n%s", strings.Join(testDirs, "\n"))
}
