package server

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/care/internal/client"
	"github.com/sprucehealth/backend/cmd/svc/care/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	caresettings "github.com/sprucehealth/backend/cmd/svc/care/settings"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/dosespot"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/settings"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var grpcErrorf = grpc.Errorf

type server struct {
	layoutStore layout.Storage
	dal         dal.DAL
	layout      layout.LayoutClient
	settings    settings.SettingsClient
	media       media.MediaClient
	doseSpot    dosespot.API
	clk         clock.Clock
}

func New(dal dal.DAL, layoutClient layout.LayoutClient, settingsClient settings.SettingsClient, mediaClient media.MediaClient, layoutStore layout.Storage, doseSpotClient dosespot.API, clk clock.Clock) care.CareServer {
	return &server{
		layoutStore: layoutStore,
		dal:         dal,
		layout:      layoutClient,
		settings:    settingsClient,
		media:       mediaClient,
		doseSpot:    doseSpotClient,
		clk:         clk,
	}
}

func (s *server) CreateVisit(ctx context.Context, in *care.CreateVisitRequest) (*care.CreateVisitResponse, error) {
	if in.EntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "entity_id required")
	} else if in.LayoutVersionID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "layout_version_id required")
	} else if in.Name == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "name required")
	} else if in.OrganizationID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "organization_id required")
	} else if in.CreatorID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "creator_id required")
	}

	visitToCreate := &models.Visit{
		Name:            in.Name,
		LayoutVersionID: in.LayoutVersionID,
		EntityID:        in.EntityID,
		OrganizationID:  in.OrganizationID,
		CreatorID:       in.CreatorID,
	}

	_, err := s.dal.CreateVisit(ctx, visitToCreate)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	optionalTriageValue, err := settings.GetBooleanValue(ctx, s.settings, &settings.GetValuesRequest{
		NodeID: in.OrganizationID,
		Keys: []*settings.ConfigKey{
			{
				Key: caresettings.ConfigKeyOptionalTriage,
			},
		},
	})
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &care.CreateVisitResponse{
		Visit: transformVisitToResponse(visitToCreate, optionalTriageValue),
	}, nil
}

func (s *server) GetVisit(ctx context.Context, in *care.GetVisitRequest) (*care.GetVisitResponse, error) {
	if in.ID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "id required")
	}

	visitID, err := models.ParseVisitID(in.ID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "unable to parse visit id: %s", err.Error())
	}

	v, err := s.dal.Visit(ctx, visitID)
	if err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "visit %s not found", visitID)
		}
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	optionalTriageValue, err := settings.GetBooleanValue(ctx, s.settings, &settings.GetValuesRequest{
		NodeID: v.OrganizationID,
		Keys: []*settings.ConfigKey{
			{
				Key: caresettings.ConfigKeyOptionalTriage,
			},
		},
	})
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &care.GetVisitResponse{
		Visit: transformVisitToResponse(v, optionalTriageValue),
	}, nil
}

func (s *server) SubmitVisit(ctx context.Context, in *care.SubmitVisitRequest) (*care.SubmitVisitResponse, error) {
	if in.VisitID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "visit_id is required")
	}

	visitID, err := models.ParseVisitID(in.VisitID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "invalid visit id %s: %s", in.VisitID, err)
	}

	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		rowsUpdated, err := dl.UpdateVisit(ctx, visitID, &dal.VisitUpdate{
			Submitted:     ptr.Bool(true),
			SubmittedTime: ptr.Time(time.Now()),
		})
		if err != nil {
			return err
		} else if rowsUpdated > 1 {
			return fmt.Errorf("expected just 1 row to be updated for visit %s but %d rows were updated", visitID, rowsUpdated)
		}

		return nil
	}); err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &care.SubmitVisitResponse{}, nil
}

func (s *server) TriageVisit(ctx context.Context, in *care.TriageVisitRequest) (*care.TriageVisitResponse, error) {
	if in.VisitID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "visit_id is required")
	}

	visitID, err := models.ParseVisitID(in.VisitID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "invalid visit id %s: %s", in.VisitID, err)
	}

	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		rowsUpdated, err := dl.UpdateVisit(ctx, visitID, &dal.VisitUpdate{
			Triaged:     ptr.Bool(true),
			TriagedTime: ptr.Time(s.clk.Now()),
		})
		if err != nil {
			return err
		} else if rowsUpdated > 1 {
			return fmt.Errorf("expected just 1 row to be updated for visit %s but %d rows were updated.", visitID, rowsUpdated)
		}
		return nil
	}); err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &care.TriageVisitResponse{}, nil
}

func (s *server) CreateVisitAnswers(ctx context.Context, in *care.CreateVisitAnswersRequest) (*care.CreateVisitAnswersResponse, error) {
	if in.VisitID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "visit_id is required")
	} else if in.ActorEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "actory_entity_id is required")
	} else if in.AnswersJSON == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "answers_json is required")
	}

	visitID, err := models.ParseVisitID(in.VisitID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "unable to parse visit_id %s: %s", in.VisitID, err)
	}

	visitAnswers, err := client.Decode(in.AnswersJSON)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	visitAnswers.DeleteNilAnswers()

	// ensure that no answer in the clear answers array is also an answer mentioned in the answer dictionary
	for _, questionID := range visitAnswers.ClearAnswers {
		if _, ok := visitAnswers.Answers[questionID]; ok {
			return nil, grpcErrorf(care.ErrorInvalidAnswer, "question %s specified in list to clear answers for as well as in dictionary with answer", questionID)
		}
	}

	visit, err := s.dal.Visit(ctx, visitID)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	visitLayoutVersionRes, err := s.layout.GetVisitLayoutVersion(ctx, &layout.GetVisitLayoutVersionRequest{
		ID: visit.LayoutVersionID,
	})
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	intake, err := s.layoutStore.GetIntake(visitLayoutVersionRes.VisitLayoutVersion.IntakeLayoutLocation)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	questionInIntakeMap := make(map[string]*layout.Question)
	for _, section := range intake.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				questionInIntakeMap[question.ID] = question
			}
		}
	}

	// validate each incoming answer against the question in the intake
	for questionID, answer := range visitAnswers.Answers {
		question, ok := questionInIntakeMap[questionID]
		if !ok {
			return nil, grpcErrorf(codes.InvalidArgument, "question %s not in visit intake for %s", questionID, visit.ID)
		}

		if err := answer.Validate(question); err != nil {
			return nil, grpcErrorf(care.ErrorInvalidAnswer, "invalid answer to question in visit %s : %s", visit.ID, err)
		}
	}

	// transform the incoming answers to the internal models and store
	transformedAnswers := make([]*models.Answer, 0, len(visitAnswers.Answers))
	for questionID, answer := range visitAnswers.Answers {

		transformedAnswer, err := transformAnswerToModel(questionID, answer, s.media)
		if err != nil {
			return nil, grpcErrorf(codes.Internal, err.Error())
		}
		transformedAnswers = append(transformedAnswers, transformedAnswer)
	}

	// store the incoming answers
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		for _, answer := range transformedAnswers {
			if err := dl.CreateVisitAnswer(ctx, visitID, in.ActorEntityID, answer); err != nil {
				return errors.Trace(err)
			}

			rowsDeleted, err := dl.DeleteVisitAnswers(ctx, visitID, visitAnswers.ClearAnswers)
			if err != nil {
				return errors.Trace(err)
			} else if rowsDeleted > int64(len(visitAnswers.ClearAnswers)) {
				return errors.Trace(fmt.Errorf("more rows attempted to be deleted (%d) than anticpated (%d)", rowsDeleted, len(visitAnswers.ClearAnswers)))
			}

		}
		return nil
	}); err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &care.CreateVisitAnswersResponse{}, nil
}

func (s *server) GetAnswersForVisit(ctx context.Context, in *care.GetAnswersForVisitRequest) (*care.GetAnswersForVisitResponse, error) {
	if in.VisitID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "visit_id required")
	}

	visitID, err := models.ParseVisitID(in.VisitID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "unable to parse visit_id %s : %s", in.VisitID, err)
	}

	visit, err := s.dal.Visit(ctx, visitID)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	visitLayoutVersionRes, err := s.layout.GetVisitLayoutVersion(ctx, &layout.GetVisitLayoutVersionRequest{
		ID: visit.LayoutVersionID,
	})
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	intake, err := s.layoutStore.GetIntake(visitLayoutVersionRes.VisitLayoutVersion.IntakeLayoutLocation)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	// collect all questions in the intake
	var questionIDs []string
	for _, section := range intake.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				questionIDs = append(questionIDs, question.ID)
			}
		}
	}

	answerMap, err := s.dal.VisitAnswers(ctx, visitID, questionIDs)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	if in.SerializedForPatient {
		transformedAnswerMap := make(map[string]client.Answer, len(answerMap))
		for questionID, answer := range answerMap {
			transformedAnswerMap[questionID], err = transformAnswerModelToResponse(answer, s.media)
			if err != nil {
				return nil, grpcErrorf(codes.Internal, err.Error())
			}
		}

		answerJSONData, err := json.Marshal(transformedAnswerMap)
		if err != nil {
			return nil, grpcErrorf(codes.Internal, err.Error())
		}

		return &care.GetAnswersForVisitResponse{
			PatientAnswersJSON: string(answerJSONData),
		}, nil
	}

	transformedAnswerMap := make(map[string]*care.Answer, len(answerMap))
	for questionID, answer := range answerMap {
		transformedAnswerMap[questionID], err = transformAnswerModelToSVCResponse(answer, s.media)
		if err != nil {
			return nil, grpcErrorf(codes.Internal, err.Error())
		}
	}

	return &care.GetAnswersForVisitResponse{
		Answers: transformedAnswerMap,
	}, nil
}

func (s *server) CarePlan(ctx context.Context, in *care.CarePlanRequest) (*care.CarePlanResponse, error) {
	if in.ID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "care plan id is required")
	}
	id, err := models.ParseCarePlanID(in.ID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "care plan id is invalid")
	}
	cp, err := s.dal.CarePlan(ctx, id)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpcErrorf(codes.NotFound, "care plan %s not found", id)
	} else if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	cpr, err := transformCarePlanToResponse(cp)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	return &care.CarePlanResponse{CarePlan: cpr}, nil
}

func (s *server) CreateCarePlan(ctx context.Context, in *care.CreateCarePlanRequest) (*care.CreateCarePlanResponse, error) {
	if in.Name == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "care plan name is required")
	}
	cp := &models.CarePlan{
		Name:         in.Name,
		CreatorID:    in.CreatorID,
		Treatments:   make([]*models.CarePlanTreatment, len(in.Treatments)),
		Instructions: make([]*models.CarePlanInstruction, len(in.Instructions)),
	}
	for i, ins := range in.Instructions {
		cp.Instructions[i] = &models.CarePlanInstruction{Title: ins.Title, Steps: ins.Steps}
	}
	for i, t := range in.Treatments {
		var availability models.TreatmentAvailability
		switch t.Availability {
		case care.CarePlanTreatment_UNKNOWN:
			availability = models.TreatmentAvailabilityUnknown
		case care.CarePlanTreatment_OTC:
			availability = models.TreatmentAvailabilityOTC
		case care.CarePlanTreatment_RX:
			availability = models.TreatmentAvailabilityRx
		default:
			return nil, grpcErrorf(codes.InvalidArgument, "unknown treatment availability '%s'", t.Availability.String())
		}
		cp.Treatments[i] = &models.CarePlanTreatment{
			EPrescribe:           t.EPrescribe,
			Availability:         availability,
			Name:                 t.Name,
			Route:                t.Route,
			Form:                 t.Form,
			MedicationID:         t.MedicationID,
			Dosage:               t.Dosage,
			DispenseType:         t.DispenseType,
			DispenseNumber:       int(t.DispenseNumber),
			Refills:              int(t.Refills),
			SubstitutionsAllowed: t.SubstitutionsAllowed,
			DaysSupply:           int(t.DaysSupply),
			Sig:                  t.Sig,
			PharmacyID:           t.PharmacyID,
			PharmacyInstructions: t.PharmacyInstructions,
		}
	}
	id, err := s.dal.CreateCarePlan(ctx, cp)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	// Re-query to get actual values for timestamps
	cp, err = s.dal.CarePlan(ctx, id)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	cpr, err := transformCarePlanToResponse(cp)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	return &care.CreateCarePlanResponse{CarePlan: cpr}, nil
}

func (s *server) SearchMedications(ctx context.Context, in *care.SearchMedicationsRequest) (*care.SearchMedicationsResponse, error) {
	// TODO: use clinic ID from request
	// TODO: really need to cache this as it does a ton of requests against the DoseSpot API

	// DoseSpot doesn't return results for any searches shorter than 3 characters so don't bother trying and just return an empty list.
	if len(in.Query) < 3 {
		return &care.SearchMedicationsResponse{}, nil
	}

	clinicianID := int64(in.ClinicianID)

	names, err := s.doseSpot.GetDrugNamesForDoctor(clinicianID, in.Query)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	medications := make([]*care.Medication, len(names))
	par := conc.NewParallel()
	for i, name := range names {
		medIndex := i
		medName := name
		par.Go(func() error {
			strengths, err := s.doseSpot.SearchForMedicationStrength(clinicianID, medName)
			if err != nil {
				return errors.Trace(err)
			}
			medStrengths := make([]*care.MedicationStrength, len(strengths))
			var route, form string
			for i, strength := range strengths {
				med, err := s.doseSpot.SelectMedication(clinicianID, medName, strength)
				if err != nil {
					return errors.Trace(err)
				}
				genName, err := dosespot.ParseGenericName(med)
				if err != nil {
					golog.Errorf("Failed to parse generic name '%s': %s", med.GenericProductName, err)
				}
				schedule, _ := strconv.ParseUint(med.Schedule, 10, 64)
				// route and form are the same for each strength, but it's the only place they come down
				route = med.RouteDescription
				form = med.DoseFormDescription
				medStrengths[i] = &care.MedicationStrength{
					OTC:               med.OTC,
					Schedule:          uint32(schedule),
					Strength:          med.StrengthDescription,
					DispenseUnit:      med.DispenseUnitDescription,
					GenericName:       genName,
					LexiGenProductID:  uint64(med.LexiGenProductID),
					LexiDrugSynID:     uint64(med.LexiDrugSynID),
					LexiSynonymTypeID: uint64(med.LexiSynonymTypeID),
					NDC:               med.RepresentativeNDC,
				}
			}
			// Since we lookup the medication by the name use it as the ID and parse out the "(route - form)" from it to generate a user friendly name
			medID := medName
			if i := strings.IndexRune(medName, '('); i > 1 {
				medName = medName[:i-1] // -1 to remove space before '('
			}
			medications[medIndex] = &care.Medication{
				ID:        medID,
				Name:      medName,
				Route:     route,
				Form:      form,
				Strengths: medStrengths,
			}
			return nil
		})
	}
	if err := par.Wait(); err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &care.SearchMedicationsResponse{Medications: medications}, nil
}

func (s *server) SearchSelfReportedMedications(ctx context.Context, in *care.SearchSelfReportedMedicationsRequest) (*care.SearchSelfReportedMedicationsResponse, error) {
	// TODO: Cache results

	// DoseSpot doesn't return results for any searches shorter than 3 characters so don't bother trying and just return an empty list.
	if len(in.Query) < 3 {
		return &care.SearchSelfReportedMedicationsResponse{}, nil
	}

	results, err := s.doseSpot.GetDrugNamesForPatient(in.Query)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &care.SearchSelfReportedMedicationsResponse{
		Results: results,
	}, nil
}

func (s *server) SearchAllergyMedications(ctx context.Context, in *care.SearchAllergyMedicationsRequest) (*care.SearchAllergyMedicationsResponse, error) {
	// TODO: Cache results

	// DoseSpot doesn't return results for any searches shorter than 3 characters so don't bother trying and just return an empty list.
	if len(in.Query) < 3 {
		return &care.SearchAllergyMedicationsResponse{}, nil
	}

	results, err := s.doseSpot.SearchForAllergyRelatedMedications(in.Query)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &care.SearchAllergyMedicationsResponse{
		Results: results,
	}, nil
}
func (s *server) SubmitCarePlan(ctx context.Context, in *care.SubmitCarePlanRequest) (*care.SubmitCarePlanResponse, error) {
	if in.ID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "care plan id is required")
	}
	if in.ParentID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "care plan parent ID is required")
	}
	id, err := models.ParseCarePlanID(in.ID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "care plan id is invalid")
	}
	if err := s.dal.SubmitCarePlan(ctx, id, in.ParentID); errors.Cause(err) == dal.ErrNotFound {
		return nil, grpcErrorf(codes.NotFound, "care plan %s not found", id)
	} else if errors.Cause(err) == dal.ErrAlreadySubmitted {
		return nil, grpcErrorf(codes.AlreadyExists, "care plan %s already submitted", id)
	} else if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	cp, err := s.dal.CarePlan(ctx, id)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpcErrorf(codes.NotFound, "care plan %s not found", id)
	} else if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	cpr, err := transformCarePlanToResponse(cp)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	return &care.SubmitCarePlanResponse{CarePlan: cpr}, nil
}
