package server

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/care/internal/client"
	"github.com/sprucehealth/backend/cmd/svc/care/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/layout"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var grpcErrorf = grpc.Errorf

type server struct {
	layoutStore layout.Storage
	dal         dal.DAL
	layout      layout.LayoutClient
}

func New(dal dal.DAL, layoutClient layout.LayoutClient, layoutStore layout.Storage) care.CareServer {
	return &server{
		layoutStore: layoutStore,
		dal:         dal,
		layout:      layoutClient,
	}
}

func (s *server) CreateVisit(ctx context.Context, in *care.CreateVisitRequest) (*care.CreateVisitResponse, error) {
	if in.EntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "entity_id required")
	} else if in.LayoutVersionID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "layout_version_id required")
	} else if in.Name == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "name required")
	}

	visitToCreate := &models.Visit{
		Name:            in.Name,
		LayoutVersionID: in.LayoutVersionID,
		EntityID:        in.EntityID,
	}

	_, err := s.dal.CreateVisit(ctx, visitToCreate)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &care.CreateVisitResponse{
		Visit: transformVisitToResponse(visitToCreate),
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

	return &care.GetVisitResponse{
		Visit: transformVisitToResponse(v),
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

	questionIDToAnswerMap, err := client.Decode(in.AnswersJSON)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
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
	for questionID, answer := range questionIDToAnswerMap {
		question, ok := questionInIntakeMap[questionID]
		if !ok {
			return nil, grpcErrorf(codes.InvalidArgument, "question %s not in visit intake for %s", questionID, visit.ID)
		}

		if err := answer.Validate(question); err != nil {
			return nil, grpcErrorf(care.ErrorInvalidAnswer, "invalid answer to question in visit %s : %s", visit.ID, err)
		}
	}

	// transform the incoming answers to the internal models and store
	transformedAnswers := make([]*models.Answer, 0, len(questionIDToAnswerMap))
	for questionID, answer := range questionIDToAnswerMap {
		transformedAnswer, err := transformAnswerToModel(questionID, answer)
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

	transformedAnswerMap := make(map[string]client.Answer, len(answerMap))
	for questionID, answer := range answerMap {
		transformedAnswerMap[questionID], err = transformAnswerModelToResponse(answer)
		if err != nil {
			return nil, grpcErrorf(codes.Internal, err.Error())
		}
	}

	answerJSONData, err := json.Marshal(transformedAnswerMap)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &care.GetAnswersForVisitResponse{
		AnswersJSON: string(answerJSONData),
	}, nil
}
