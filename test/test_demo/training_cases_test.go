package test_demo

import (
	"bytes"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/demo"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestTrainingCase(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// Upload the latest versions of the review and intake
	latestIntakeVersion := determineLatestVersionedFile("intake", t)
	latestReviewVersion := determineLatestVersionedFile("review", t)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", latestIntakeVersion, "../../info_intake/"+latestIntakeVersion, t)
	test_integration.AddFileToMultipartWriter(writer, "review", latestReviewVersion, "../../info_intake/"+latestReviewVersion, t)
	test_integration.AddFieldToMultipartWriter(writer, "patient_app_version", "1.0.0", t)
	test_integration.AddFieldToMultipartWriter(writer, "doctor_app_version", "1.0.0", t)
	test_integration.AddFieldToMultipartWriter(writer, "platform", "iOS", t)
	err := writer.Close()
	test.OK(t, err)
	admin := test_integration.CreateRandomAdmin(t, testData)
	resp, err := testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// have the demo worker run ones to create the training cases
	demo.StartWorker(testData.DataApi, "www.spruce.local", testData.APIServer.URL, "us-east-1", 24*60*60)

	// wait until the training cases have been created
	time.Sleep(time.Second)

}

func determineLatestVersionedFile(prefix string, t *testing.T) string {
	files, err := ioutil.ReadDir("../../info_intake/")
	test.OK(t, err)

	var fileNamesToCompare []string
	for _, f := range files {
		if strings.HasPrefix(f.Name(), prefix) {
			fileNamesToCompare = append(fileNamesToCompare, f.Name())
		}
	}

	if len(fileNamesToCompare) > 0 {
		sort.Strings(fileNamesToCompare)
	} else {
		t.Fatalf("File with prefix %s not found", prefix)
	}

	return fileNamesToCompare[len(fileNamesToCompare)-1]
}
