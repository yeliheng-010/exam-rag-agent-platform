package service

import (
	"context"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
)

func (w *stubExamQuestionWriter) CreateQuestionBank(context.Context, *types.QuestionBank) error {
	return nil
}

func (w *stubExamQuestionWriter) ListQuestionBanks(context.Context, uint64, []string) ([]*types.QuestionBank, error) {
	return nil, nil
}

func (w *stubExamQuestionWriter) GetQuestionBankByIDAndTenant(context.Context, string, uint64) (*types.QuestionBank, error) {
	return nil, repository.ErrQuestionBankNotFound
}

func (w *stubExamQuestionWriter) CreateQuestionDetail(_ context.Context, detail *types.QuestionDetail) error {
	w.created = append(w.created, detail)
	return nil
}

func (w *stubExamQuestionWriter) ListQuestionDetailsByBank(context.Context, uint64, string) ([]*types.QuestionDetail, error) {
	return w.created, nil
}

func (w *stubExamQuestionWriter) GetQuestionDetailByIDAndTenant(_ context.Context, _ uint64, questionID string) (*types.QuestionDetail, error) {
	for _, detail := range w.created {
		if detail.Question != nil && detail.Question.ID == questionID {
			return detail, nil
		}
	}
	return nil, repository.ErrQuestionNotFound
}

func (w *stubExamQuestionWriter) CreateQuestionGroupDetail(context.Context, *types.QuestionGroupDetail) error {
	return nil
}

func (w *stubExamQuestionWriter) ListQuestionGroupDetailsByBank(context.Context, uint64, string) ([]*types.QuestionGroupDetail, error) {
	return nil, nil
}

func (w *stubExamQuestionWriter) GetQuestionGroupDetailByIDAndTenant(context.Context, uint64, string) (*types.QuestionGroupDetail, error) {
	return nil, repository.ErrQuestionNotFound
}
