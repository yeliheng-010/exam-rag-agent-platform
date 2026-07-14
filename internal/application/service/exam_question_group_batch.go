package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

const (
	gaokaoMathQuestionGroupStrategy = "gaokao_math_basic_v1"
	gaokaoMathMaxOutputTokens       = 6144
)

func (e *examQuestionGroupExtractor) extractBatches(
	ctx context.Context,
	chatModel chat.Chat,
	strategy ExamQuestionGroupExtractionStrategy,
	material *types.ExamMaterial,
	task *types.ExamStructuringTask,
	chunks []*types.Chunk,
	observer interfaces.ExamQuestionGroupBatchObserver,
) ([]*types.ExamQuestionGroupDraftCandidate, string, error) {
	batches := buildQuestionGroupExtractionBatches(strategy.Code(), chunks)
	allCandidates := make([]*types.ExamQuestionGroupDraftCandidate, 0)
	rawOutputs := make([]string, 0, len(batches))
	rejectedBatches := make([]string, 0)
	for index, batch := range batches {
		if observer != nil {
			if err := observer(interfaces.ExamQuestionGroupBatchProgress{Total: len(batches), Current: index + 1, Completed: index}); err != nil {
				return nil, strings.Join(rawOutputs, "\n\n--- batch ---\n\n"), err
			}
		}
		candidates, raw, err := e.extractBatch(ctx, chatModel, strategy, material, task, batch, len(batches))
		if err != nil {
			if len(batches) == 1 || !errors.Is(err, errNoValidQuestionGroupDrafts) {
				return nil, strings.Join(rawOutputs, "\n\n--- batch ---\n\n"), fmt.Errorf("extract batch %d/%d: %w", index+1, len(batches), err)
			}
			rawOutputs = append(rawOutputs, raw)
			rejectedBatches = append(rejectedBatches, fmt.Sprintf("batch %d/%d: %v", index+1, len(batches), err))
			if observer != nil {
				if err := observer(interfaces.ExamQuestionGroupBatchProgress{Total: len(batches), Current: index + 1, Completed: index + 1}); err != nil {
					return nil, strings.Join(rawOutputs, "\n\n--- batch ---\n\n"), err
				}
			}
			continue
		}
		rawOutputs = append(rawOutputs, raw)
		allCandidates = append(allCandidates, candidates...)
		if observer != nil {
			if err := observer(interfaces.ExamQuestionGroupBatchProgress{Total: len(batches), Current: index + 1, Completed: index + 1}); err != nil {
				return nil, strings.Join(rawOutputs, "\n\n--- batch ---\n\n"), err
			}
		}
	}
	if len(allCandidates) == 0 {
		return nil, strings.Join(rawOutputs, "\n\n--- batch ---\n\n"), fmt.Errorf("%w: %s", errNoValidQuestionGroupDrafts, strings.Join(rejectedBatches, "; "))
	}
	return mergeQuestionGroupCandidates(allCandidates), strings.Join(rawOutputs, "\n\n--- batch ---\n\n"), nil
}

func (e *examQuestionGroupExtractor) extractBatch(
	ctx context.Context,
	chatModel chat.Chat,
	strategy ExamQuestionGroupExtractionStrategy,
	material *types.ExamMaterial,
	task *types.ExamStructuringTask,
	batch questionGroupExtractionBatch,
	batchCount int,
) ([]*types.ExamQuestionGroupDraftCandidate, string, error) {
	prompt, _, err := strategy.BuildPrompt(QuestionGroupStrategyInput{
		Material: material, Task: task, Chunks: batch.CoreChunks,
		ContextChunks: batch.ContextChunks, AnswerChunks: batch.AnswerChunks,
		TargetQuestionNumbers: batch.TargetQuestionNumbers,
	})
	if err != nil {
		return nil, "", err
	}
	raw, err := callQuestionGroupExtractionModel(ctx, chatModel, prompt, questionGroupMaxTokens(strategy.Code(), batchCount))
	if err != nil {
		return nil, raw, err
	}
	candidates, err := parseQuestionGroupDraftCandidates(raw)
	if err != nil {
		if strategy.Code() != gaokaoMathQuestionGroupStrategy || !errors.Is(err, errNoValidQuestionGroupDrafts) {
			return nil, raw, err
		}
		candidates = nil
	}
	if strategy.Code() == gaokaoMathQuestionGroupStrategy {
		candidates = constrainMathBatchCandidates(candidates, batch)
		candidates = enrichMathBatchCandidates(candidates, batch)
		if len(candidates) == 0 {
			return nil, raw, errNoValidQuestionGroupDrafts
		}
	}
	if err := prepareQuestionGroupCandidates(strategy, candidates, raw, batch.allChunks()); err != nil {
		return nil, raw, err
	}
	return candidates, raw, nil
}

func prepareQuestionGroupCandidates(strategy ExamQuestionGroupExtractionStrategy, candidates []*types.ExamQuestionGroupDraftCandidate, raw string, chunks []*types.Chunk) error {
	for _, candidate := range candidates {
		candidate.StrategyCode = strategy.Code()
		candidate.RawModelOutput = raw
		enrichQuestionGroupDraftCandidateMaterial(candidate, chunks)
		if err := strategy.Validate(candidate); err != nil {
			return err
		}
	}
	return nil
}

func questionGroupMaxTokens(strategyCode string, batchCount int) int {
	if strategyCode == gaokaoMathQuestionGroupStrategy && batchCount > 1 {
		return gaokaoMathMaxOutputTokens
	}
	return 8192
}
