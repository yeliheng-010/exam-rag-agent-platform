package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
)

type examAgentEvaluationSnapshotRunner interface {
	CreateRunFromSnapshot(
		ctx context.Context,
		tenantID uint64,
		userID string,
		bankID string,
		snapshot *types.ExamAgentEvaluationRequestSnapshot,
	) (*types.ExamRAGEvaluationRun, error)
}

func (s *examEvaluationCenterService) ListEvaluationSets(
	ctx context.Context,
	tenantID uint64,
	userID string,
	filter types.ExamEvaluationCenterFilter,
) ([]*types.ExamEvaluationSet, error) {
	filter, err := normalizeEvaluationCenterFilter(filter)
	if err != nil {
		return nil, err
	}
	bankIDs, banks, err := s.resolveCenterBankScope(ctx, tenantID, userID, filter.BankID)
	if err != nil {
		return nil, err
	}
	sets, err := s.repo.ListEvaluationSets(ctx, tenantID, bankIDs, filter)
	if err != nil {
		return nil, err
	}
	bankNames := make(map[string]string, len(banks))
	for _, bank := range banks {
		if bank != nil {
			bankNames[bank.ID] = bank.Name
		}
	}
	for _, set := range sets {
		versions, versionErr := s.repo.ListEvaluationSetVersions(ctx, tenantID, set.ID)
		if versionErr != nil {
			return nil, versionErr
		}
		set.QuestionBankName = bankNames[set.QuestionBankID]
		set.AgentName = evaluationSetAgentName(versions)
		set.Versions = compactEvaluationSetVersions(versions)
	}
	return sets, nil
}

func (s *examEvaluationCenterService) CreateEvaluationSet(
	ctx context.Context,
	tenantID uint64,
	userID string,
	req *types.CreateExamEvaluationSetRequest,
) (*types.ExamEvaluationSet, error) {
	if req == nil || strings.TrimSpace(req.Name) == "" || len([]rune(strings.TrimSpace(req.Name))) > 160 {
		return nil, ErrExamInvalidRequest
	}
	run, err := s.authorizedCompletedCenterRun(ctx, tenantID, userID, req.RunID)
	if err != nil {
		return nil, err
	}
	if len(run.RequestSnapshot) == 0 {
		return nil, ErrExamInvalidRequest
	}
	now := time.Now().UTC()
	set := &types.ExamEvaluationSet{
		ID: uuid.NewString(), TenantID: tenantID, QuestionBankID: run.QuestionBankID,
		EvaluationKind: run.EvaluationKind, AgentID: run.AgentID,
		Name: strings.TrimSpace(req.Name), Description: strings.TrimSpace(req.Description),
		CurrentVersion: 1, CreatedBy: userID, Status: types.ExamEvaluationSetStatusActive,
		CreatedAt: now, UpdatedAt: now,
	}
	version := newEvaluationSetVersion(set.ID, tenantID, 1, run, userID, now)
	if err := s.repo.CreateEvaluationSet(ctx, set, version); err != nil {
		return nil, err
	}
	set.Versions = []*types.ExamEvaluationSetVersion{compactEvaluationSetVersion(version)}
	set.AgentName = evaluationRunAgentName(run)
	return set, nil
}

func (s *examEvaluationCenterService) CreateEvaluationSetVersion(
	ctx context.Context,
	tenantID uint64,
	userID string,
	setID string,
	req *types.CreateExamEvaluationSetVersionRequest,
) (*types.ExamEvaluationSetVersion, error) {
	if req == nil || strings.TrimSpace(setID) == "" || strings.TrimSpace(req.RunID) == "" {
		return nil, ErrExamInvalidRequest
	}
	set, err := s.authorizedEvaluationSet(ctx, tenantID, userID, setID)
	if err != nil {
		return nil, err
	}
	run, err := s.authorizedCompletedCenterRun(ctx, tenantID, userID, req.RunID)
	if err != nil {
		return nil, err
	}
	if !evaluationSetRunCompatible(set, run) || len(run.RequestSnapshot) == 0 {
		return nil, ErrExamInvalidRequest
	}
	version := newEvaluationSetVersion(set.ID, tenantID, 0, run, userID, time.Now().UTC())
	created, err := s.repo.CreateEvaluationSetVersion(ctx, tenantID, set.ID, version)
	if err != nil {
		return nil, err
	}
	return compactEvaluationSetVersion(created), nil
}

func (s *examEvaluationCenterService) RunEvaluationSet(
	ctx context.Context,
	tenantID uint64,
	userID string,
	setID string,
	req *types.RunExamEvaluationSetRequest,
) (*types.ExamRAGEvaluationRun, error) {
	set, err := s.authorizedEvaluationSet(ctx, tenantID, userID, setID)
	if err != nil {
		return nil, err
	}
	versionNumber := set.CurrentVersion
	if req != nil && req.Version > 0 {
		versionNumber = req.Version
	}
	version, err := s.repo.GetEvaluationSetVersion(ctx, tenantID, set.ID, versionNumber)
	if err != nil {
		return nil, ErrExamNotFound
	}
	run, err := s.runEvaluationDefinition(ctx, tenantID, userID, set, version)
	if err != nil {
		return nil, err
	}
	if err := s.repo.AttachRunEvaluationSet(ctx, tenantID, run.ID, set.ID, version.Version); err != nil {
		return nil, err
	}
	run.EvaluationSetID = set.ID
	run.EvaluationSetVersion = version.Version
	return run, nil
}

func (s *examEvaluationCenterService) runEvaluationDefinition(
	ctx context.Context,
	tenantID uint64,
	userID string,
	set *types.ExamEvaluationSet,
	version *types.ExamEvaluationSetVersion,
) (*types.ExamRAGEvaluationRun, error) {
	if set.EvaluationKind == types.ExamEvaluationKindRAG {
		if s.ragRunner == nil {
			return nil, ErrExamInvalidRequest
		}
		var request types.RunExamRAGDiagnosticRequest
		if json.Unmarshal(version.DefinitionSnapshot, &request) != nil {
			return nil, ErrExamInvalidRequest
		}
		return s.ragRunner.CreateRun(ctx, tenantID, userID, set.QuestionBankID, &request)
	}
	if set.EvaluationKind != types.ExamEvaluationKindAgent || s.agentRunner == nil {
		return nil, ErrExamInvalidRequest
	}
	var snapshot types.ExamAgentEvaluationRequestSnapshot
	if json.Unmarshal(version.DefinitionSnapshot, &snapshot) != nil || snapshot.Agent.ID == "" {
		return nil, ErrExamInvalidRequest
	}
	snapshotRunner, ok := s.agentRunner.(examAgentEvaluationSnapshotRunner)
	if !ok {
		return nil, ErrExamInvalidRequest
	}
	return snapshotRunner.CreateRunFromSnapshot(ctx, tenantID, userID, set.QuestionBankID, &snapshot)
}

func (s *examEvaluationCenterService) authorizedEvaluationSet(
	ctx context.Context,
	tenantID uint64,
	userID string,
	setID string,
) (*types.ExamEvaluationSet, error) {
	set, err := s.repo.GetEvaluationSet(ctx, tenantID, strings.TrimSpace(setID))
	if err != nil {
		return nil, ErrExamNotFound
	}
	if _, err := s.questions.GetQuestionBank(ctx, tenantID, userID, set.QuestionBankID); err != nil {
		return nil, err
	}
	return set, nil
}

func (s *examEvaluationCenterService) authorizedCompletedCenterRun(
	ctx context.Context,
	tenantID uint64,
	userID string,
	runID string,
) (*types.ExamRAGEvaluationRun, error) {
	run, err := s.authorizedCenterRun(ctx, tenantID, userID, runID)
	if err != nil {
		return nil, err
	}
	if run.Status != types.ExamRAGEvaluationRunStatusCompleted {
		return nil, ErrExamInvalidRequest
	}
	return run, nil
}

func newEvaluationSetVersion(
	setID string,
	tenantID uint64,
	version int,
	run *types.ExamRAGEvaluationRun,
	userID string,
	createdAt time.Time,
) *types.ExamEvaluationSetVersion {
	return &types.ExamEvaluationSetVersion{
		ID: uuid.NewString(), TenantID: tenantID, EvaluationSetID: setID, Version: version,
		SourceRunID: run.ID, DefinitionSnapshot: append(types.JSON(nil), run.RequestSnapshot...),
		CreatedBy: userID, CreatedAt: createdAt,
	}
}

func evaluationSetRunCompatible(set *types.ExamEvaluationSet, run *types.ExamRAGEvaluationRun) bool {
	return set != nil && run != nil && set.QuestionBankID == run.QuestionBankID &&
		set.EvaluationKind == run.EvaluationKind && strings.TrimSpace(set.AgentID) == strings.TrimSpace(run.AgentID)
}

func evaluationSetAgentName(versions []*types.ExamEvaluationSetVersion) string {
	if len(versions) == 0 || versions[0] == nil {
		return ""
	}
	var snapshot types.ExamAgentEvaluationRequestSnapshot
	if json.Unmarshal(versions[0].DefinitionSnapshot, &snapshot) == nil {
		return snapshot.Agent.Name
	}
	return ""
}

func compactEvaluationSetVersions(versions []*types.ExamEvaluationSetVersion) []*types.ExamEvaluationSetVersion {
	result := make([]*types.ExamEvaluationSetVersion, 0, len(versions))
	for _, version := range versions {
		result = append(result, compactEvaluationSetVersion(version))
	}
	return result
}

func compactEvaluationSetVersion(version *types.ExamEvaluationSetVersion) *types.ExamEvaluationSetVersion {
	if version == nil {
		return nil
	}
	clone := *version
	clone.DefinitionSnapshot = nil
	return &clone
}
