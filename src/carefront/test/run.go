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
	successfulTestDirs := make([]string, 0)
	failedTestDirs := make([]string, 0)
	errors := make(map[string]error)
	for _, f := range files {
		if f.IsDir() && strings.HasPrefix(f.Name(), "test_") {
			testDir := path.Join(testPath, f.Name())
			successfulTestDirs = append(successfulTestDirs, testDir)
			args := []string{"go", "test", "-v", "-race", "-test.timeout=50m", testDir}
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				failedTestDirs = append(failedTestDirs, testDir)
				errors[testDir] = err
			}
		}
	}

	fmt.Printf("Following test packages successfully ran:\n%s\n", strings.Join(successfulTestDirs, "\n"))
	if len(failedTestDirs) > 0 {
		fmt.Println("Following test packages had failed tests:")
		for _, failedTestDir := range failedTestDirs {
			fmt.Printf("FAIL %s error: %s\n", failedTestDir, errors[failedTestDir])
		}

		// ensure to indicate the existence of failed test runs
		os.Exit(1)
	}

}
