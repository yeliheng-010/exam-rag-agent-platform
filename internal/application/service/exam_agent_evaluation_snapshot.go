package service

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
)

func (s *examAgentEvaluationService) CreateRunFromSnapshot(
	ctx context.Context,
	tenantID uint64,
	userID string,
	bankID string,
	snapshot *types.ExamAgentEvaluationRequestSnapshot,
) (*types.ExamRAGEvaluationRun, error) {
	bankID = strings.TrimSpace(bankID)
	userID = strings.TrimSpace(userID)
	if tenantID == 0 || userID == "" || bankID == "" || snapshot == nil || snapshot.Agent.ID == "" {
		return nil, ErrExamInvalidRequest
	}
	bank, err := s.questionService.GetQuestionBank(ctx, tenantID, userID, bankID)
	if err != nil {
		return nil, err
	}
	agentCtx := context.WithValue(ctx, types.TenantIDContextKey, tenantID)
	agentCtx = context.WithValue(agentCtx, types.UserIDContextKey, userID)
	current, err := s.agentService.GetAgentByID(agentCtx, strings.TrimSpace(snapshot.Agent.ID))
	if err != nil {
		return nil, err
	}
	if bank == nil || current == nil || current.TenantID != tenantID || current.Config.AgentMode != types.AgentModeSmartReasoning {
		return nil, ErrExamInvalidRequest
	}
	stored, err := validatedAgentEvaluationSnapshot(snapshot)
	if err != nil {
		return nil, err
	}
	run, err := newExamAgentEvaluationRunFromSnapshot(tenantID, userID, bank.ID, stored)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateRun(ctx, run); err != nil {
		return nil, err
	}
	if err := s.enqueueRun(run.ID); err != nil {
		return nil, s.failQueuedRun(ctx, run, err)
	}
	return run, nil
}

func validatedAgentEvaluationSnapshot(
	snapshot *types.ExamAgentEvaluationRequestSnapshot,
) (*types.ExamAgentEvaluationRequestSnapshot, error) {
	config := snapshot.Agent.Config
	if config.AgentMode != types.AgentModeSmartReasoning ||
		snapshot.Agent.ModelID != config.ModelID ||
		!reflect.DeepEqual(snapshot.Agent.AllowedTools, config.AllowedTools) ||
		!reflect.DeepEqual(snapshot.Agent.KnowledgeBases, config.KnowledgeBases) ||
		!reflect.DeepEqual(safeAgentEvaluationConfig(config), config) {
		return nil, ErrExamInvalidRequest
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	var stored types.ExamAgentEvaluationRequestSnapshot
	if err := json.Unmarshal(raw, &stored); err != nil {
		return nil, err
	}
	stored.Cases, err = normalizeAgentEvaluationCases(stored.Cases, stored.Agent.Config.AllowedTools)
	if err != nil {
		return nil, err
	}
	return &stored, nil
}

func newExamAgentEvaluationRunFromSnapshot(
	tenantID uint64,
	userID string,
	bankID string,
	snapshot *types.ExamAgentEvaluationRequestSnapshot,
) (*types.ExamRAGEvaluationRun, error) {
	request, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	progress, err := json.Marshal(types.ExamRAGEvaluationProgress{TotalCases: len(snapshot.Cases)})
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	return &types.ExamRAGEvaluationRun{
		ID: uuid.NewString(), TenantID: tenantID, QuestionBankID: bankID,
		EvaluationKind: types.ExamEvaluationKindAgent, AgentID: snapshot.Agent.ID,
		CreatedBy: userID, Status: types.ExamRAGEvaluationRunStatusQueued,
		Progress: progress, RequestSnapshot: request, CreatedAt: now, UpdatedAt: now,
	}, nil
}
