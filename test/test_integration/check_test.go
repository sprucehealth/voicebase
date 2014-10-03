package test_integration

import "testing"

func TestEmpty(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
}
