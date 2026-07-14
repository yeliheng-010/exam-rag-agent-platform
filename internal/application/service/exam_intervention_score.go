package service

import (
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

const defaultInterventionLimit = 5
const maxInterventionLimit = 10

type interventionSignal struct {
	evidence  types.ExamPracticeDiagnosisEvidence
	domainID  string
	subjectID string
	groupType string
}

func (s *interventionSignal) attachGroup(group *types.QuestionGroup) {
	if group == nil {
		return
	}
	s.evidence.GroupID = firstAnalyticsText(s.evidence.GroupID, group.ID)
	s.domainID = strings.TrimSpace(group.DomainID)
	if group.SubjectID != nil {
		s.subjectID = strings.TrimSpace(*group.SubjectID)
	}
	s.groupType = strings.TrimSpace(group.GroupType)
}

func buildInterventionResult(
	scope types.ExamPracticeRecommendationScope,
	class *types.ExamClass,
	candidates []*types.QuestionGroupPracticeSummary,
	signals []interventionSignal,
	assigned map[string]bool,
	req types.ExamPracticeRecommendationRequest,
) *types.ExamPracticeRecommendationResult {
	preferredSpaceID := ""
	if class != nil {
		preferredSpaceID = class.SpaceID
	}
	candidates = dedupeInterventionCandidates(candidates, preferredSpaceID)
	result := &types.ExamPracticeRecommendationResult{
		Scope: scope, Class: class, Diagnosis: interventionEvidence(signals),
		Recommendations: scoreInterventionCandidates(scope, candidates, signals, assigned, req),
		Warnings:        []string{},
	}
	if len(signals) == 0 {
		result.Warnings = append(result.Warnings, "no_active_wrong_questions")
	}
	if len(result.Recommendations) == 0 {
		result.Warnings = append(result.Warnings, "no_recommendation_candidates")
	}
	return result
}

func dedupeInterventionCandidates(
	candidates []*types.QuestionGroupPracticeSummary,
	preferredSpaceID string,
) []*types.QuestionGroupPracticeSummary {
	out := make([]*types.QuestionGroupPracticeSummary, 0, len(candidates))
	indexByIdentity := make(map[string]int, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil || candidate.Group == nil {
			continue
		}
		identity := interventionGroupIdentity(candidate.Group)
		index, exists := indexByIdentity[identity]
		if !exists {
			indexByIdentity[identity] = len(out)
			out = append(out, candidate)
			continue
		}
		if preferredSpaceID != "" && candidate.Group.SpaceID == preferredSpaceID && out[index].Group.SpaceID != preferredSpaceID {
			out[index] = candidate
		}
	}
	return out
}

func scoreInterventionCandidates(
	scope types.ExamPracticeRecommendationScope,
	candidates []*types.QuestionGroupPracticeSummary,
	signals []interventionSignal,
	assigned map[string]bool,
	req types.ExamPracticeRecommendationRequest,
) []*types.ExamPracticeRecommendation {
	out := make([]*types.ExamPracticeRecommendation, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil || candidate.Group == nil {
			continue
		}
		isAssigned := assigned[candidate.Group.ID]
		if isAssigned && !req.IncludeAssigned {
			continue
		}
		out = append(out, scoreInterventionCandidate(scope, candidate, signals, isAssigned))
	}
	sortInterventionRecommendations(out)
	limit := normalizeInterventionLimit(req.Limit)
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func scoreInterventionCandidate(
	scope types.ExamPracticeRecommendationScope,
	candidate *types.QuestionGroupPracticeSummary,
	signals []interventionSignal,
	assigned bool,
) *types.ExamPracticeRecommendation {
	recommendation := &types.ExamPracticeRecommendation{
		Group: candidate, Reasons: []types.ExamPracticeRecommendationReason{},
		Evidence: []types.ExamPracticeDiagnosisEvidence{}, Assigned: assigned,
	}
	reasonScores := map[types.ExamPracticeRecommendationReasonCode]int{}
	for _, signal := range signals {
		applyInterventionSignal(recommendation, reasonScores, candidate.Group, signal)
	}
	if scope == types.ExamPracticeRecommendationScopeStudent {
		applyInterventionAttemptSignal(reasonScores, candidate.LastAttempt)
	}
	if len(reasonScores) == 0 {
		reasonScores[types.ExamPracticeRecommendationReasonSupplemental] = 1
	}
	finalizeInterventionReasons(recommendation, reasonScores)
	return recommendation
}

func applyInterventionSignal(
	recommendation *types.ExamPracticeRecommendation,
	reasons map[types.ExamPracticeRecommendationReasonCode]int,
	group *types.QuestionGroup,
	signal interventionSignal,
) {
	matched := false
	matched = addInterventionReason(reasons, group.ID == signal.evidence.GroupID && group.ID != "", types.ExamPracticeRecommendationReasonTargetedReview, 100) || matched
	matched = addInterventionReason(reasons, sameInterventionSubject(group, signal), types.ExamPracticeRecommendationReasonSameSubject, 40) || matched
	matched = addInterventionReason(reasons, group.GroupType != "" && group.GroupType == signal.groupType, types.ExamPracticeRecommendationReasonSameGroupType, 25) || matched
	matched = addInterventionReason(reasons, group.DomainID != "" && group.DomainID == signal.domainID, types.ExamPracticeRecommendationReasonSameDomain, 15) || matched
	if matched && len(recommendation.Evidence) < 3 {
		recommendation.Evidence = append(recommendation.Evidence, signal.evidence)
	}
}

func sameInterventionSubject(group *types.QuestionGroup, signal interventionSignal) bool {
	return group.SubjectID != nil && strings.TrimSpace(*group.SubjectID) != "" && strings.TrimSpace(*group.SubjectID) == signal.subjectID
}

func addInterventionReason(
	reasons map[types.ExamPracticeRecommendationReasonCode]int,
	condition bool,
	code types.ExamPracticeRecommendationReasonCode,
	score int,
) bool {
	if condition {
		reasons[code] = score
	}
	return condition
}

func applyInterventionAttemptSignal(
	reasons map[types.ExamPracticeRecommendationReasonCode]int,
	attempt *types.ExamPracticeAttempt,
) {
	if attempt == nil || attempt.QuestionCount <= 0 {
		return
	}
	rate := float64(attempt.CorrectCount) / float64(attempt.QuestionCount)
	if rate < 0.6 {
		reasons[types.ExamPracticeRecommendationReasonLowAccuracyRetry] = 20
	}
}

func finalizeInterventionReasons(
	recommendation *types.ExamPracticeRecommendation,
	reasons map[types.ExamPracticeRecommendationReasonCode]int,
) {
	for code, score := range reasons {
		recommendation.Score += score
		recommendation.Reasons = append(recommendation.Reasons, types.ExamPracticeRecommendationReason{Code: code, Score: score})
	}
	sort.SliceStable(recommendation.Reasons, func(i, j int) bool {
		if recommendation.Reasons[i].Score != recommendation.Reasons[j].Score {
			return recommendation.Reasons[i].Score > recommendation.Reasons[j].Score
		}
		return recommendation.Reasons[i].Code < recommendation.Reasons[j].Code
	})
}

func sortInterventionRecommendations(items []*types.ExamPracticeRecommendation) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		left, right := items[i].Group.Group, items[j].Group.Group
		if !left.UpdatedAt.Equal(right.UpdatedAt) {
			return left.UpdatedAt.After(right.UpdatedAt)
		}
		return left.ID < right.ID
	})
}

func interventionEvidence(signals []interventionSignal) []types.ExamPracticeDiagnosisEvidence {
	out := make([]types.ExamPracticeDiagnosisEvidence, 0, len(signals))
	for _, signal := range signals {
		out = append(out, signal.evidence)
	}
	return out
}

func normalizeInterventionLimit(limit int) int {
	if limit <= 0 {
		return defaultInterventionLimit
	}
	if limit > maxInterventionLimit {
		return maxInterventionLimit
	}
	return limit
}
