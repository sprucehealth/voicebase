package main

import (
	"fmt"
	"os"
)

func main() {
	currentWorkingDirectory, err := os.Getwd()
	fmt.Println(currentWorkingDirectory)
}
