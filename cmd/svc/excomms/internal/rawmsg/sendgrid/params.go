package sendgrid

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/storage"
)

type attachmentInfo struct {
	Filename string `json:"filename"`
	Type     string `json:"type"`
}

func ParamsFromRequest(r *http.Request, store storage.Store) (*rawmsg.SendGridIncomingEmail, map[uint64]*models.Media, error) {
	sgi := &rawmsg.SendGridIncomingEmail{
		Headers:      r.FormValue("headers"),
		Text:         r.FormValue("text"),
		HTML:         r.FormValue("html"),
		Sender:       r.FormValue("from"),
		Recipient:    r.FormValue("to"),
		CC:           r.FormValue("cc"),
		Subject:      r.FormValue("subject"),
		DKIM:         r.FormValue("dkim"),
		SPF:          r.FormValue("spf"),
		SMTPEnvelope: r.FormValue("envelope"),
		Charsets:     r.FormValue("charsets"),
		SpamScore:    r.FormValue("spam_score"),
		SpamReport:   r.FormValue("spam_report"),
	}

	media := make(map[uint64]*models.Media)
	if r.FormValue("attachments") != "" {

		attachmentInfoString := r.FormValue("attachment-info")
		var attachmentInfoJSON map[string]*attachmentInfo
		if attachmentInfoString != "" {
			if err := json.Unmarshal([]byte(attachmentInfoString), &attachmentInfoJSON); err != nil {
				return nil, nil, errors.Trace(err)
			}
		}

		numAttachments, err := strconv.Atoi(r.FormValue("attachments"))
		if err != nil {
			return nil, nil, errors.Trace(err)
		}
		sgi.NumAttachments = uint32(numAttachments)

		sgi.Attachments = make([]*rawmsg.SendGridIncomingEmail_Attachment, numAttachments)
		for i := 0; i < numAttachments; i++ {
			key := fmt.Sprintf("attachment%d", i+1)
			fileHandle, _, err := r.FormFile(key)
			if err != nil {
				return nil, nil, errors.Trace(err)
			}
			sgi.Attachments[i] = &rawmsg.SendGridIncomingEmail_Attachment{
				Filename: attachmentInfoJSON[key].Filename,
				Type:     attachmentInfoJSON[key].Type,
			}

			size, err := common.SeekerSize(fileHandle)
			if err != nil {
				return nil, nil, errors.Trace(err)
			}

			// upload the file to S3
			id, err := idgen.NewID()
			url, err := store.PutReader(strconv.FormatInt(int64(id), 10), fileHandle, size, sgi.Attachments[i].Type, map[string]string{
				"X-Amz-Meta-Original-Name": sgi.Attachments[i].Filename,
			})
			if err != nil {
				return nil, nil, errors.Trace(err)
			}
			media[id] = &models.Media{
				ID:   id,
				Type: sgi.Attachments[i].Type,
				URL:  url,
				Name: ptr.String(sgi.Attachments[i].Filename),
			}
			sgi.Attachments[i].ID = id
		}
	}

	return sgi, media, nil
}