package service

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/response"
	"github.com/sprucehealth/backend/libs/errors"
)

type startOnlineVisitService struct {
	doctorDAL  dal.DoctorDAL
	contentURL string
	webURL     string
}

type StartOnlineVisitPageContext struct {
	DoctorID string
}

func NewForOnlineVisit(doctorDAL dal.DoctorDAL, contentURL, webURL string) PageContentBuilder {
	return &startOnlineVisitService{
		doctorDAL:  doctorDAL,
		contentURL: contentURL,
		webURL:     webURL,
	}
}

func (s *startOnlineVisitService) PageContentForID(ctx interface{}, r *http.Request) (interface{}, error) {
	soc := ctx.(*StartOnlineVisitPageContext)
	doctorID := soc.DoctorID
	// check to ensure that the doctor is a spruce doctor
	doctor, err := s.doctorDAL.Doctor(doctorID)
	if errors.Cause(err) == dal.ErrNoDoctorFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Trace(err)
	} else if !doctor.IsSpruceDoctor {
		return nil, nil
	}

	doctorResponse, err := response.TransformModel(doctor, "", s.contentURL, s.webURL)
	if err != nil {
		return nil, errors.Trace(err)
	}

	states, err := s.doctorDAL.ShortListedStatesForSpruceDoctor(doctorID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	bcs := make([]*response.BreadcrumbList, len(states))
	for i, item := range states {
		bcs[i] = &response.BreadcrumbList{
			Items: spruceDoctorBreadcrumbs(s.webURL, doctorResponse, item),
		}
	}

	return &response.StartOnlineVisitPage{
		HTMLTitle:              fmt.Sprintf("%s | Start an Online Visit | Spruce Health", doctorResponse.LongDisplayName),
		SEODescription:         doctor.SEODescription,
		DoctorShortDisplayName: doctorResponse.ShortDisplayName,
		DoctorLongDisplayName:  doctorResponse.LongDisplayName,
		ProfileImageURL:        doctorResponse.ProfileImageURL,
		DoctorID:               doctor.ID,
		ReferralLink:           doctor.ReferralLink,
		IsMobile:               isMobile(r),
		Breadcrumbs:            bcs,
	}, nil
}
