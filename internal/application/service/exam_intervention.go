package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

const interventionCandidateLimit = 100

type examInterventionService struct {
	analyticsService interfaces.ExamAnalyticsService
	practiceService  interfaces.ExamPracticeService
	assignmentRepo   interfaces.ExamAssignmentRepository
}

type examPublishedGroupLookup interface {
	ListPublishedGroupsByClass(
		ctx context.Context,
		tenantID uint64,
		classID string,
	) ([]*types.QuestionGroup, error)
}

func NewExamInterventionService(
	analyticsService interfaces.ExamAnalyticsService,
	practiceService interfaces.ExamPracticeService,
	assignmentRepo interfaces.ExamAssignmentRepository,
) interfaces.ExamInterventionService {
	return &examInterventionService{
		analyticsService: analyticsService,
		practiceService:  practiceService,
		assignmentRepo:   assignmentRepo,
	}
}

func (s *examInterventionService) RecommendClassPractice(
	ctx context.Context,
	tenantID uint64,
	userID string,
	classID string,
	req types.ExamPracticeRecommendationRequest,
) (*types.ExamPracticeRecommendationResult, error) {
	analytics, err := s.analyticsService.GetClassAnalytics(ctx, tenantID, userID, strings.TrimSpace(classID))
	if err != nil {
		return nil, err
	}
	signals := s.classInterventionSignals(ctx, tenantID, userID, analytics.FrequentWrongQuestions)
	candidates, err := s.listInterventionCandidates(ctx, tenantID, userID, "")
	if err != nil {
		return nil, err
	}
	lookup, ok := s.assignmentRepo.(examPublishedGroupLookup)
	if !ok {
		return nil, fmt.Errorf("exam assignment repository does not support published group lookup")
	}
	publishedGroups, err := lookup.ListPublishedGroupsByClass(ctx, tenantID, analytics.Class.ID)
	if err != nil {
		return nil, err
	}
	assigned := interventionAssignedGroups(candidates, publishedGroups)
	return buildInterventionResult(types.ExamPracticeRecommendationScopeClass, analytics.Class, candidates, signals, assigned, req), nil
}

func (s *examInterventionService) RecommendStudentPractice(
	ctx context.Context,
	tenantID uint64,
	userID string,
	req types.ExamPracticeRecommendationRequest,
) (*types.ExamPracticeRecommendationResult, error) {
	wrong, err := s.practiceService.ListWrongQuestions(ctx, tenantID, userID, types.ListWrongQuestionsFilter{
		SpaceID: strings.TrimSpace(req.SpaceID),
		Limit:   interventionCandidateLimit,
	})
	if err != nil {
		return nil, err
	}
	signals := studentInterventionSignals(wrong)
	candidates, err := s.listInterventionCandidates(ctx, tenantID, userID, req.SpaceID)
	if err != nil {
		return nil, err
	}
	return buildInterventionResult(types.ExamPracticeRecommendationScopeStudent, nil, candidates, signals, nil, req), nil
}

func (s *examInterventionService) listInterventionCandidates(
	ctx context.Context,
	tenantID uint64,
	userID string,
	spaceID string,
) ([]*types.QuestionGroupPracticeSummary, error) {
	return s.practiceService.ListQuestionGroups(ctx, tenantID, userID, types.ListPracticeQuestionGroupsFilter{
		SpaceID: strings.TrimSpace(spaceID),
		Limit:   interventionCandidateLimit,
	})
}

func (s *examInterventionService) classInterventionSignals(
	ctx context.Context,
	tenantID uint64,
	userID string,
	wrong []*types.ExamClassFrequentWrongQuestion,
) []interventionSignal {
	out := make([]interventionSignal, 0, len(wrong))
	for _, item := range wrong {
		if item == nil {
			continue
		}
		signal := interventionSignal{evidence: classInterventionEvidence(item)}
		if detail, err := s.practiceService.GetQuestionGroupDetail(ctx, tenantID, userID, item.GroupID); err == nil && detail != nil {
			signal.attachGroup(detail.Group)
		}
		out = append(out, signal)
	}
	return out
}

func studentInterventionSignals(wrong []*types.WrongQuestionItem) []interventionSignal {
	out := make([]interventionSignal, 0, len(wrong))
	for _, item := range wrong {
		if item == nil || item.Answer == nil || item.Answer.ReviewStatus == types.PracticeAnswerReviewStatusMastered {
			continue
		}
		signal := interventionSignal{evidence: studentInterventionEvidence(item)}
		signal.attachGroup(item.Group)
		out = append(out, signal)
	}
	return out
}

func classInterventionEvidence(item *types.ExamClassFrequentWrongQuestion) types.ExamPracticeDiagnosisEvidence {
	return types.ExamPracticeDiagnosisEvidence{
		QuestionID: item.QuestionID, GroupID: item.GroupID, QuestionNo: item.QuestionNo,
		Stem: item.Stem, WrongRate: item.WrongRate, AffectedStudentCount: item.AffectedStudentCount,
	}
}

func studentInterventionEvidence(item *types.WrongQuestionItem) types.ExamPracticeDiagnosisEvidence {
	return types.ExamPracticeDiagnosisEvidence{
		QuestionID: item.Answer.QuestionID, GroupID: interventionWrongGroupID(item),
		QuestionNo: item.Answer.QuestionNo, Stem: analyticsSnapshotText(item.Answer.QuestionSnapshot, "stem"),
		WrongRate: 1, AffectedStudentCount: 1, ReviewStatus: item.Answer.ReviewStatus,
	}
}

func interventionWrongGroupID(item *types.WrongQuestionItem) string {
	if item.Group != nil {
		return item.Group.ID
	}
	if item.Attempt != nil {
		return item.Attempt.GroupID
	}
	return ""
}

func interventionAssignedGroups(
	candidates []*types.QuestionGroupPracticeSummary,
	publishedGroups []*types.QuestionGroup,
) map[string]bool {
	out := make(map[string]bool, len(publishedGroups))
	candidateIDsByIdentity := make(map[string][]string, len(candidates))
	for _, candidate := range candidates {
		if candidate != nil && candidate.Group != nil && candidate.Group.ID != "" {
			identity := interventionGroupIdentity(candidate.Group)
			candidateIDsByIdentity[identity] = append(candidateIDsByIdentity[identity], candidate.Group.ID)
		}
	}
	for _, group := range publishedGroups {
		if group == nil || group.ID == "" {
			continue
		}
		out[group.ID] = true
		for _, candidateID := range candidateIDsByIdentity[interventionGroupIdentity(group)] {
			out[candidateID] = true
		}
	}
	return out
}

func interventionGroupIdentity(group *types.QuestionGroup) string {
	if group == nil {
		return ""
	}
	subjectID, sourceYear := "", ""
	if group.SubjectID != nil {
		subjectID = *group.SubjectID
	}
	if group.SourceYear != nil {
		sourceYear = fmt.Sprint(*group.SourceYear)
	}
	return fmt.Sprintf(
		"%q|%q|%q|%q|%q|%q|%x|%x|%q|%q|%q|%d",
		group.DomainID,
		subjectID,
		group.GroupType,
		group.Title,
		group.MaterialText,
		group.MaterialFormat,
		[]byte(group.AssetRefs),
		[]byte(group.SourceChunkIDs),
		sourceYear,
		group.SourceRegion,
		group.PaperType,
		group.SortOrder,
	)
}
