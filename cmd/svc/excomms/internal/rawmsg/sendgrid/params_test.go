package sendgrid

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/test"
)

func TestAttachmentUpload(t *testing.T) {

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	test.OK(t, w.WriteField("attachment-info", `{"attachment1":{"filename":"Photo on 12-30-15 at 12.10 PM.jpg","name":"Photo on 12-30-15 at 12.10 PM.jpg","type":"image/jpeg"}}`))
	test.OK(t, w.WriteField("attachments", "1"))

	imgFile, err := w.CreateFormFile("attachment1", "Photo on 12-30-15 at 12.10 PM.jpg")
	test.OK(t, err)

	_, err = imgFile.Write([]byte("1234"))
	test.OK(t, err)

	test.OK(t, w.Close())

	r, err := http.NewRequest("POST", "http://test.com", &b)
	test.OK(t, err)
	r.Header.Set("Content-Type", w.FormDataContentType())

	testObjects := make(map[string]*storage.TestObject)
	store := storage.NewTestStore(testObjects)
	sgi, media, err := ParamsFromRequest(r, store)
	test.OK(t, err)
	test.Equals(t, 1, len(media))
	test.Equals(t, 1, len(sgi.Attachments))
	test.Equals(t, "Photo on 12-30-15 at 12.10 PM.jpg", sgi.Attachments[0].Filename)
	test.Equals(t, 1, len(testObjects))
}
