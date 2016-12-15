package server

import (
	"testing"

	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestCloneAttachments(t *testing.T) {
	t.Run("Error-OwnerID", func(t *testing.T) {
		st := newServerTest(t)
		defer st.Finish()
		testCloneAttachments(t, st, &threading.CloneAttachmentsRequest{}, nil, grpc.Errorf(codes.InvalidArgument, "OwnerID required"))
	})
	t.Run("Error-NoAttachments", func(t *testing.T) {
		st := newServerTest(t)
		defer st.Finish()
		testCloneAttachments(t, st, &threading.CloneAttachmentsRequest{
			OwnerID: "ownerID",
		}, nil, grpc.Errorf(codes.InvalidArgument, "At least 1 attachment is required"))
	})
	t.Run("Success-Image", func(t *testing.T) {
		st := newServerTest(t)
		defer st.Finish()
		st.mediaClient.Expect(mock.NewExpectation(st.mediaClient.CloneMedia, &media.CloneMediaRequest{
			OwnerType: media.MediaOwnerType_ACCOUNT,
			OwnerID:   "ownerID",
			MediaID:   "MediaID",
		}).WithReturns(&media.CloneMediaResponse{
			MediaInfo: &media.MediaInfo{
				ID: "ClonedMediaID",
			},
		}, nil))
		testCloneAttachments(t, st, &threading.CloneAttachmentsRequest{
			OwnerID:   "ownerID",
			OwnerType: threading.CLONED_ATTACHMENT_OWNER_ACCOUNT,
			Attachments: []*threading.Attachment{
				{
					Title:     "Title",
					URL:       "MediaID",
					UserTitle: "UserTitle",
					ContentID: "MediaID",
					Data: &threading.Attachment_Image{
						Image: &threading.ImageAttachment{
							MediaID: "MediaID",
						},
					},
				},
			},
		}, &threading.CloneAttachmentsResponse{
			Attachments: []*threading.Attachment{
				{
					Title:     "Title",
					URL:       "ClonedMediaID",
					UserTitle: "UserTitle",
					ContentID: "ClonedMediaID",
					Data: &threading.Attachment_Image{
						Image: &threading.ImageAttachment{
							MediaID: "ClonedMediaID",
						},
					},
				},
			},
		}, nil)
	})
	t.Run("Success-Audio", func(t *testing.T) {
		st := newServerTest(t)
		defer st.Finish()
		st.mediaClient.Expect(mock.NewExpectation(st.mediaClient.CloneMedia, &media.CloneMediaRequest{
			OwnerType: media.MediaOwnerType_ACCOUNT,
			OwnerID:   "ownerID",
			MediaID:   "MediaID",
		}).WithReturns(&media.CloneMediaResponse{
			MediaInfo: &media.MediaInfo{
				ID: "ClonedMediaID",
			},
		}, nil))
		testCloneAttachments(t, st, &threading.CloneAttachmentsRequest{
			OwnerID:   "ownerID",
			OwnerType: threading.CLONED_ATTACHMENT_OWNER_ACCOUNT,
			Attachments: []*threading.Attachment{
				{
					Title:     "Title",
					URL:       "MediaID",
					UserTitle: "UserTitle",
					ContentID: "MediaID",
					Data: &threading.Attachment_Audio{
						Audio: &threading.AudioAttachment{
							MediaID: "MediaID",
						},
					},
				},
			},
		}, &threading.CloneAttachmentsResponse{
			Attachments: []*threading.Attachment{
				{
					Title:     "Title",
					URL:       "ClonedMediaID",
					UserTitle: "UserTitle",
					ContentID: "ClonedMediaID",
					Data: &threading.Attachment_Audio{
						Audio: &threading.AudioAttachment{
							MediaID: "ClonedMediaID",
						},
					},
				},
			},
		}, nil)
	})
	t.Run("Success-Document", func(t *testing.T) {
		st := newServerTest(t)
		defer st.Finish()
		st.mediaClient.Expect(mock.NewExpectation(st.mediaClient.CloneMedia, &media.CloneMediaRequest{
			OwnerType: media.MediaOwnerType_ACCOUNT,
			OwnerID:   "ownerID",
			MediaID:   "MediaID",
		}).WithReturns(&media.CloneMediaResponse{
			MediaInfo: &media.MediaInfo{
				Name: "ClonedTitle",
				ID:   "ClonedMediaID",
			},
		}, nil))
		testCloneAttachments(t, st, &threading.CloneAttachmentsRequest{
			OwnerID:   "ownerID",
			OwnerType: threading.CLONED_ATTACHMENT_OWNER_ACCOUNT,
			Attachments: []*threading.Attachment{
				{
					Title:     "Title",
					URL:       "MediaID",
					UserTitle: "UserTitle",
					ContentID: "MediaID",
					Data: &threading.Attachment_Document{
						Document: &threading.DocumentAttachment{
							MediaID: "MediaID",
						},
					},
				},
			},
		}, &threading.CloneAttachmentsResponse{
			Attachments: []*threading.Attachment{
				{
					Title:     "ClonedTitle",
					URL:       "ClonedMediaID",
					UserTitle: "UserTitle",
					ContentID: "ClonedMediaID",
					Data: &threading.Attachment_Document{
						Document: &threading.DocumentAttachment{
							Name:    "ClonedTitle",
							MediaID: "ClonedMediaID",
						},
					},
				},
			},
		}, nil)
	})
	t.Run("Success-Visit", func(t *testing.T) {
		st := newServerTest(t)
		defer st.Finish()
		st.careClient.EXPECT().GetVisit(st.ctx, &care.GetVisitRequest{
			ID: "VisitID",
		}).Return(&care.GetVisitResponse{
			Visit: &care.Visit{
				ID:              "VisitID",
				LayoutVersionID: "LayoutVersionID",
			},
		}, nil)
		st.layoutClient.EXPECT().GetVisitLayoutByVersion(st.ctx, &layout.GetVisitLayoutByVersionRequest{
			VisitLayoutVersionID: "LayoutVersionID",
		}).Return(&layout.GetVisitLayoutByVersionResponse{
			VisitLayout: &layout.VisitLayout{
				ID: "ClonedVisitID",
			},
		}, nil)
		testCloneAttachments(t, st, &threading.CloneAttachmentsRequest{
			OwnerID:   "ownerID",
			OwnerType: threading.CLONED_ATTACHMENT_OWNER_ACCOUNT,
			Attachments: []*threading.Attachment{
				{
					Data: &threading.Attachment_Visit{
						Visit: &threading.VisitAttachment{
							VisitID: "VisitID",
						},
					},
				},
			},
		}, &threading.CloneAttachmentsResponse{
			Attachments: []*threading.Attachment{
				{
					ContentID: "ClonedVisitID",
					Data: &threading.Attachment_Visit{
						Visit: &threading.VisitAttachment{
							VisitID: "VisitID",
						},
					},
				},
			},
		}, nil)
	})
	t.Run("Success-Payment", func(t *testing.T) {
		st := newServerTest(t)
		defer st.Finish()
		st.paymentsClient.Expect(mock.NewExpectation(st.paymentsClient.Payment, &payments.PaymentRequest{
			PaymentID: "PaymentID",
		}).WithReturns(&payments.PaymentResponse{
			Payment: &payments.Payment{
				RequestingEntityID: "RequestingEntityID",
				Amount:             1,
				Currency:           "Currency",
			},
		}, nil))
		st.paymentsClient.Expect(mock.NewExpectation(st.paymentsClient.CreatePayment, &payments.CreatePaymentRequest{
			RequestingEntityID: "RequestingEntityID",
			Amount:             1,
			Currency:           "Currency",
		}).WithReturns(&payments.CreatePaymentResponse{
			Payment: &payments.Payment{
				ID: "ClonedPaymentID",
			},
		}, nil))
		testCloneAttachments(t, st, &threading.CloneAttachmentsRequest{
			OwnerID:   "ownerID",
			OwnerType: threading.CLONED_ATTACHMENT_OWNER_ACCOUNT,
			Attachments: []*threading.Attachment{
				{
					Data: &threading.Attachment_PaymentRequest{
						PaymentRequest: &threading.PaymentRequestAttachment{
							PaymentID: "PaymentID",
						},
					},
				},
			},
		}, &threading.CloneAttachmentsResponse{
			Attachments: []*threading.Attachment{
				{
					ContentID: "ClonedPaymentID",
					Data: &threading.Attachment_PaymentRequest{
						PaymentRequest: &threading.PaymentRequestAttachment{
							PaymentID: "ClonedPaymentID",
						},
					},
				},
			},
		}, nil)
	})
	t.Run("Success-CarePlan", func(t *testing.T) {
		st := newServerTest(t)
		defer st.Finish()
		st.careClient.EXPECT().CarePlan(st.ctx, &care.CarePlanRequest{
			ID: "CarePlanID",
		}).Return(&care.CarePlanResponse{
			CarePlan: &care.CarePlan{
				ID:           "CarePlanID",
				Name:         "Name",
				Instructions: []*care.CarePlanInstruction{},
				Treatments:   []*care.CarePlanTreatment{},
			},
		}, nil)
		st.careClient.EXPECT().CreateCarePlan(st.ctx, &care.CreateCarePlanRequest{
			Name:         "Name",
			CreatorID:    "ownerID",
			Instructions: []*care.CarePlanInstruction{},
			Treatments:   []*care.CarePlanTreatment{},
		}).Return(&care.CreateCarePlanResponse{
			CarePlan: &care.CarePlan{
				ID:   "ClonedCarePlanID",
				Name: "Name",
			},
		}, nil)
		testCloneAttachments(t, st, &threading.CloneAttachmentsRequest{
			OwnerID:   "ownerID",
			OwnerType: threading.CLONED_ATTACHMENT_OWNER_ACCOUNT,
			Attachments: []*threading.Attachment{
				{
					Data: &threading.Attachment_CarePlan{
						CarePlan: &threading.CarePlanAttachment{
							CarePlanID: "CarePlanID",
						},
					},
				},
			},
		}, &threading.CloneAttachmentsResponse{
			Attachments: []*threading.Attachment{
				{
					ContentID: "ClonedCarePlanID",
					Data: &threading.Attachment_CarePlan{
						CarePlan: &threading.CarePlanAttachment{
							CarePlanID:   "ClonedCarePlanID",
							CarePlanName: "Name",
						},
					},
				},
			},
		}, nil)
	})
}

func testCloneAttachments(
	t *testing.T,
	st *serverTest,
	in *threading.CloneAttachmentsRequest,
	exp *threading.CloneAttachmentsResponse,
	expErr error) {
	resp, err := st.server.CloneAttachments(st.ctx, in)
	test.Equals(t, expErr, err)
	test.Equals(t, exp, resp)
}
