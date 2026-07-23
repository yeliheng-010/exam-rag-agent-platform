package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

func TestExamRAGDiagnosticUsesDefaultCasesAndQuestionBankSpaceResources(t *testing.T) {
	questionSvc := &stubExamRAGDiagnosticQuestionService{
		bank: &types.QuestionBank{
			ID:        "bank-english",
			TenantID:  10000,
			SpaceID:   "space-class",
			DomainID:  "gaokao",
			SubjectID: stringPtr("english"),
			Status:    "active",
		},
	}
	resourceSvc := &stubExamRAGDiagnosticResourceService{
		readable: map[string]bool{"kb-english": true},
		resources: []*types.ExamSpaceResource{
			{
				TenantID:     10000,
				SpaceID:      "space-class",
				ResourceType: types.ExamResourceTypeKnowledgeBase,
				ResourceID:   "kb-english",
				DomainID:     "gaokao",
				SubjectID:    stringPtr("english"),
				Status:       types.ExamSpaceResourceStatusActive,
			},
		},
	}
	kbSvc := &stubExamRAGDiagnosticKBService{
		kbs: map[string]*types.KnowledgeBase{
			"kb-english": {ID: "kb-english", TenantID: 10000},
		},
		results: []*types.SearchResult{
			{ID: "chunk-a1", SubChunkID: []string{"chunk-21", "chunk-22", "chunk-23"}},
		},
	}
	questionRepo := &stubExamRAGDiagnosticQuestionRepo{detail: newExamRAGDiagnosticReadingGroup()}
	svc := NewExamRAGDiagnosticService(questionSvc, resourceSvc, questionRepo, kbSvc)

	result, err := svc.EvaluateQuestionBank(context.Background(), 10000, "teacher-1", "bank-english", &types.RunExamRAGDiagnosticRequest{})

	if err != nil {
		t.Fatalf("EvaluateQuestionBank returned error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected diagnostic result")
	}
	if !result.UsedDefaultCases {
		t.Fatalf("expected default cases to be used")
	}
	if got := strings.Join(result.KnowledgeBaseIDs, ","); got != "kb-english" {
		t.Fatalf("knowledge base ids = %q, want kb-english", got)
	}
	if result.Summary.Total < 4 || result.Summary.Passed != result.Summary.Total {
		t.Fatalf("summary = %#v", result.Summary)
	}
	if kbSvc.calls != result.Summary.Total {
		t.Fatalf("HybridSearch calls = %d, want %d", kbSvc.calls, result.Summary.Total)
	}
	if resourceSvc.gotFilter.SpaceID != "space-class" || resourceSvc.gotFilter.DomainID != "gaokao" || resourceSvc.gotFilter.SubjectID != "english" {
		t.Fatalf("resource filter = %#v", resourceSvc.gotFilter)
	}
}

func TestExamRAGDiagnosticRejectsExplicitUnreadableKnowledgeBase(t *testing.T) {
	questionSvc := &stubExamRAGDiagnosticQuestionService{
		bank: &types.QuestionBank{
			ID:       "bank-english",
			TenantID: 10000,
			SpaceID:  "space-class",
			DomainID: "gaokao",
			Status:   "active",
		},
	}
	resourceSvc := &stubExamRAGDiagnosticResourceService{
		readable: map[string]bool{"kb-secret": false},
	}
	kbSvc := &stubExamRAGDiagnosticKBService{
		kbs: map[string]*types.KnowledgeBase{
			"kb-secret": {ID: "kb-secret", TenantID: 10000},
		},
	}
	svc := NewExamRAGDiagnosticService(questionSvc, resourceSvc, &stubExamRAGDiagnosticQuestionRepo{}, kbSvc)

	_, err := svc.EvaluateQuestionBank(context.Background(), 10000, "teacher-1", "bank-english", &types.RunExamRAGDiagnosticRequest{
		KnowledgeBaseIDs: []string{"kb-secret"},
		Cases: []types.ExamRAGDiagnosticCase{
			{Name: "custom", Query: "custom query", RequiredPhrases: []string{"anything"}},
		},
	})

	if !errors.Is(err, ErrExamPermissionDenied) {
		t.Fatalf("EvaluateQuestionBank error = %v, want ErrExamPermissionDenied", err)
	}
}

func TestDefaultExamRAGDiagnosticCasesSelectsMathQualityChecks(t *testing.T) {
	subjectID := "math"
	bank := &types.QuestionBank{DomainID: "gaokao", SubjectID: &subjectID}

	cases := defaultExamRAGDiagnosticCasesForBank(bank)

	if len(cases) < 4 {
		t.Fatalf("math cases = %#v", cases)
	}
	names := make([]string, 0, len(cases))
	for _, item := range cases {
		names = append(names, item.Name)
	}
	joined := strings.Join(names, ",")
	for _, expected := range []string{"gaokao_math_q5_options", "gaokao_math_q14_answer", "gaokao_math_q15_figure"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("math cases %q missing %q", joined, expected)
		}
	}
}

func TestBuildExamRAGDiagnosticCasesNormalizesRetrievalPhraseGold(t *testing.T) {
	t.Parallel()

	cases, usedDefault, err := buildExamRAGDiagnosticCases([]types.ExamRAGDiagnosticCase{
		{
			Name: " stable_case ", Query: " query ",
			RequiredPhrases:          []string{"answer", "answer"},
			ExpectedChunkIDs:         []string{"chunk-1", "chunk-1"},
			RequiredRetrievalPhrases: []string{" source phrase ", "source phrase", ""},
		},
	}, &types.QuestionBank{})

	if err != nil || usedDefault || len(cases) != 1 {
		t.Fatalf("cases=%#v usedDefault=%v err=%v", cases, usedDefault, err)
	}
	if got := strings.Join(cases[0].RequiredRetrievalPhrases, ","); got != "source phrase" {
		t.Fatalf("retrieval phrases = %q", got)
	}
}

type stubExamRAGDiagnosticQuestionService struct {
	interfaces.ExamQuestionService
	bank *types.QuestionBank
}

func (s *stubExamRAGDiagnosticQuestionService) GetQuestionBank(_ context.Context, tenantID uint64, _ string, bankID string) (*types.QuestionBank, error) {
	if s.bank == nil || s.bank.ID != bankID || s.bank.TenantID != tenantID {
		return nil, repository.ErrQuestionBankNotFound
	}
	cp := *s.bank
	if s.bank.SubjectID != nil {
		subjectID := *s.bank.SubjectID
		cp.SubjectID = &subjectID
	}
	return &cp, nil
}

type stubExamRAGDiagnosticResourceService struct {
	interfaces.ExamResourceService
	readable  map[string]bool
	resources []*types.ExamSpaceResource
	gotFilter types.ListExamResourcesFilter
}

func (s *stubExamRAGDiagnosticResourceService) CanReadKnowledgeBase(_ context.Context, _ uint64, _ string, kbID string) (bool, error) {
	return s.readable[kbID], nil
}

func (s *stubExamRAGDiagnosticResourceService) ListResources(_ context.Context, _ uint64, _ string, filter types.ListExamResourcesFilter) ([]*types.ExamSpaceResource, error) {
	s.gotFilter = filter
	out := make([]*types.ExamSpaceResource, 0, len(s.resources))
	for _, resource := range s.resources {
		if resource == nil {
			continue
		}
		if filter.SpaceID != "" && resource.SpaceID != filter.SpaceID {
			continue
		}
		if filter.DomainID != "" && resource.DomainID != filter.DomainID {
			continue
		}
		if filter.SubjectID != "" && (resource.SubjectID == nil || *resource.SubjectID != filter.SubjectID) {
			continue
		}
		if filter.ResourceType != "" && resource.ResourceType != filter.ResourceType {
			continue
		}
		out = append(out, resource)
	}
	return out, nil
}

type stubExamRAGDiagnosticKBService struct {
	interfaces.KnowledgeBaseService
	kbs     map[string]*types.KnowledgeBase
	results []*types.SearchResult
	calls   int
}

func (s *stubExamRAGDiagnosticKBService) GetKnowledgeBaseByID(_ context.Context, id string) (*types.KnowledgeBase, error) {
	kb := s.kbs[id]
	if kb == nil {
		return nil, repository.ErrKnowledgeBaseNotFound
	}
	cp := *kb
	return &cp, nil
}

func (s *stubExamRAGDiagnosticKBService) HybridSearch(_ context.Context, _ string, _ types.SearchParams) ([]*types.SearchResult, error) {
	s.calls++
	return s.results, nil
}

type stubExamRAGDiagnosticQuestionRepo struct {
	interfaces.ExamQuestionRepository
	detail *types.QuestionGroupDetail
}

func (r *stubExamRAGDiagnosticQuestionRepo) FindQuestionGroupDetailByChunkIDs(context.Context, uint64, []string) (*types.QuestionGroupDetail, error) {
	return r.detail, nil
}

func newExamRAGDiagnosticReadingGroup() *types.QuestionGroupDetail {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	groupID := "group-reading-a"
	return &types.QuestionGroupDetail{
		Group: &types.QuestionGroup{
			ID:             groupID,
			TenantID:       10000,
			QuestionBankID: "bank-english",
			GroupType:      "reading_passage",
			Title:          "SoFi Stadium Events This Month",
			MaterialText: strings.Join([]string{
				"SoFi Stadium Events This Month",
				"SoFi Stadium is the go-to destination in the heart of Los Angeles for sports fans.",
				"Upcoming Football Events",
				"Nearby Hotels include Sonder.",
				"Parking guests should buy a parking pass before the event.",
			}, "\n"),
			SourceChunkIDs: types.JSON(`["chunk-a1"]`),
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		Questions: []*types.QuestionDetail{
			newExamRAGDiagnosticChoice(groupID, "question-21", "21", 1, "Which team will play the most games?", "B", "Los Angeles Rams", "The schedule names the Rams more than any other team.", "chunk-21", now),
			newExamRAGDiagnosticChoice(groupID, "question-22", "22", 2, "Which hotel is mentioned near the stadium?", "A", "Sonder", "The nearby hotels section explicitly mentions Sonder.", "chunk-22", now),
			newExamRAGDiagnosticChoice(groupID, "question-23", "23", 3, "What should guests buy before parking?", "C", "parking pass", "The parking note tells guests to buy a parking pass before the event.", "chunk-23", now),
		},
	}
}

func newExamRAGDiagnosticChoice(groupID string, questionID string, no string, order int, stem string, answer string, correctOption string, explanation string, chunkID string, now time.Time) *types.QuestionDetail {
	return &types.QuestionDetail{
		Question: &types.Question{
			ID:           questionID,
			TenantID:     10000,
			GroupID:      &groupID,
			QuestionNo:   no,
			OrderInGroup: order,
			Stem:         stem,
			Status:       "active",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		Options: []*types.QuestionOption{
			{ID: questionID + "-a", QuestionID: questionID, OptionKey: "A", Content: "Sonder", SortOrder: 1},
			{ID: questionID + "-b", QuestionID: questionID, OptionKey: "B", Content: "Los Angeles Rams", SortOrder: 2},
			{ID: questionID + "-c", QuestionID: questionID, OptionKey: "C", Content: correctOption, SortOrder: 3},
		},
		Answers: []*types.QuestionAnswer{
			{ID: questionID + "-answer", QuestionID: questionID, AnswerText: answer, IsCorrect: true, CreatedAt: now},
		},
		Explanations: []*types.QuestionExplanation{
			{ID: questionID + "-explanation", QuestionID: questionID, ExplanationText: explanation, SourceType: "structured", CreatedAt: now},
		},
		ChunkRefs: []*types.QuestionChunkRef{
			{QuestionID: questionID, ChunkID: chunkID, RefType: "evidence", Confidence: 0.9, CreatedAt: now},
		},
	}
}

func stringPtr(value string) *string {
	return &value
}
