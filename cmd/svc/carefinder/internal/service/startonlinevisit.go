package service

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/response"
	"github.com/sprucehealth/backend/libs/errors"
)

type startOnlineVisitService struct {
	doctorDAL  dal.DoctorDAL
	contentURL string
	webURL     string
}

func NewForOnlineVisit(doctorDAL dal.DoctorDAL, contentURL, webURL string) PageContentBuilder {
	return &startOnlineVisitService{
		doctorDAL:  doctorDAL,
		contentURL: contentURL,
		webURL:     webURL,
	}
}

func (s *startOnlineVisitService) PageContentForID(doctorID string, r *http.Request) (interface{}, error) {
	// check to ensure that the doctor is a spruce doctor
	doctor, err := s.doctorDAL.Doctor(doctorID)
	if errors.Cause(err) == dal.ErrNoDoctorFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Trace(err)
	} else if !doctor.IsSpruceDoctor {
		return nil, nil
	}

	doctorResponse, err := response.TransformModel(doctor, s.contentURL, s.webURL)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &response.StartOnlineVisitPage{
		HTMLTitle:              fmt.Sprintf("%s | Start an Online Visit | Spruce Health", doctorResponse.LongDisplayName),
		DoctorShortDisplayName: doctorResponse.ShortDisplayName,
		ProfileImageURL:        doctorResponse.ProfileImageURL,
		DoctorID:               doctor.ID,
		ReferralLink:           doctor.ReferralLink,
		IsMobile:               strings.Contains(r.UserAgent(), "iPhone") || strings.Contains(r.UserAgent(), "iPod") || strings.Contains(strings.ToLower(r.UserAgent()), "android"),
	}, nil
}
