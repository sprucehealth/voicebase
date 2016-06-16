package test_integration

import "testing"

func TestEmpty(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)
}
