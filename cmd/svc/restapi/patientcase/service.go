// Package patientcase - this package name is a little bit akward. In my opinion it should be patient/case, however case is a reserved word
package patientcase

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/patient_case/model"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
)

// Service describes the methods required to proprly expose the case service functionality
type Service interface {
	ChangeCareProvider(caseID, desiredProviderID, changeAuthorProviderID int64) error
	ElligibleCareProvidersForCase(caseID int64) ([]*common.Doctor, error)
}

type svcDAL interface {
	api.Transactor
	AddDoctorToPatientCase(doctorID, caseID int64) error
	DoctorIDsEligibleInState(careProvidingStateID int64) ([]int64, error)
	Doctors(id []int64) ([]*common.Doctor, error)
	GetActiveCareTeamMemberForCase(role string, patientCaseID int64) (*common.CareProviderAssignment, error)
	GetCareProvidingStateID(stateAbbreviation, pathwayTag string) (int64, error)
	GetDoctorFromID(doctorID int64) (*common.Doctor, error)
	GetPatientCaseFromID(patientCaseID int64) (*common.PatientCase, error)
	GetPatientCaseCareProviderAssignment(providerID, caseID int64) (*common.PatientCaseCareProviderAssignment, error)
	InsertPatientCaseNote(n *model.PatientCaseNote) (int64, error)
	PatientLocation(patientID common.PatientID) (zipcode string, state string, err error)
	UpdatePatientCaseCareProviderAssignment(id common.PatientCaseCareProviderAssignmentID, u *common.PatientCaseCareProviderAssignmentUpdate) (int64, error)
}

type service struct {
	svcDAL svcDAL
}

// NewService returns an initialized instance of service
func NewService(svcDAL svcDAL) Service {
	return &service{svcDAL: svcDAL}
}

func (s *service) ChangeCareProvider(caseID, desiredProviderID, changeAuthorProviderID int64) error {
	oldProvider, err := s.svcDAL.GetActiveCareTeamMemberForCase(api.RoleDoctor, caseID)
	if api.IsErrNotFound(err) {
		oldProvider = nil
	} else if err != nil {
		return errors.Trace(err)
	}

	newProvider, err := s.svcDAL.GetDoctorFromID(desiredProviderID)
	if err != nil {
		return errors.Trace(err)
	}

	if oldProvider != nil && oldProvider.ProviderID == desiredProviderID {
		golog.Warningf("Request made to change provider for case %d to provider ID %d. This doctor is already the current active provider, ignoring.")
		return nil
	}

	return errors.Trace(s.svcDAL.Transact(func(svcDAL api.DataAPI) error {
		if oldProvider != nil {
			pa, err := svcDAL.GetPatientCaseCareProviderAssignment(oldProvider.ProviderID, caseID)
			if err != nil {
				return errors.Trace(err)
			}
			aff, err := svcDAL.UpdatePatientCaseCareProviderAssignment(pa.ID, &common.PatientCaseCareProviderAssignmentUpdate{Status: ptr.String(api.StatusInactive)})
			if err != nil {
				return errors.Trace(err)
			} else if aff != 1 {
				return fmt.Errorf("Expected a single record to be updated but got %d", aff)
			}
		}

		if err := svcDAL.AddDoctorToPatientCase(desiredProviderID, caseID); err != nil {
			return errors.Trace(err)
		}

		_, err := svcDAL.InsertPatientCaseNote(&model.PatientCaseNote{
			CaseID:         caseID,
			AuthorDoctorID: changeAuthorProviderID,
			NoteText:       fmt.Sprintf("This case has been assigned to %s %s", newProvider.FirstName, newProvider.LastName),
		})
		return errors.Trace(err)
	}))
}

// ElligibleCareProviders returns a list of care providers elligible to take the provided case ID
func (s *service) ElligibleCareProvidersForCase(caseID int64) ([]*common.Doctor, error) {
	currentProvider, err := s.svcDAL.GetActiveCareTeamMemberForCase(api.RoleDoctor, caseID)
	if api.IsErrNotFound(err) {
		currentProvider = nil
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	patientCase, err := s.svcDAL.GetPatientCaseFromID(caseID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	_, stateAbbr, err := s.svcDAL.PatientLocation(patientCase.PatientID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	cpsID, err := s.svcDAL.GetCareProvidingStateID(stateAbbr, patientCase.PathwayTag)
	if err != nil {
		return nil, errors.Trace(err)
	}

	doctorIDs, err := s.svcDAL.DoctorIDsEligibleInState(cpsID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Filter out the existing provider if they are present in the list
	if currentProvider != nil {
		filteredIDs := make([]int64, 0, len(doctorIDs))
		for _, v := range doctorIDs {
			if v == currentProvider.ProviderID {
				continue
			}
			filteredIDs = append(filteredIDs, v)
		}
		doctorIDs = filteredIDs
	}

	doctors, err := s.svcDAL.Doctors(doctorIDs)
	return doctors, errors.Trace(err)
}
