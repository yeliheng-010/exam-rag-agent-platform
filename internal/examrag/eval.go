package examrag

import (
	"context"

	"github.com/Tencent/WeKnora/internal/searchutil"
)

type ExamContextEvalCase = searchutil.ExamContextEvalCase
type ExamContextRetrievalEvalSummary = searchutil.ExamContextRetrievalEvalSummary

type ExamContextRetrievalEvalCase struct {
	Name                     string
	Query                    string
	RequiredPhrases          []string
	ExpectedChunkIDs         []string
	RequiredRetrievalPhrases []string
}

type ExamQuestionContextEvalRequest struct {
	TenantID         uint64
	KnowledgeBaseIDs []string
	Cases            []ExamContextRetrievalEvalCase
}

func (r *ExamQuestionContextResolver) EvaluateRetrieval(
	ctx context.Context,
	req ExamQuestionContextEvalRequest,
) ExamContextRetrievalEvalSummary {
	var resolver searchutil.ExamContextBundleResolver
	if r != nil {
		resolver = r.EvalResolver(req.TenantID, req.KnowledgeBaseIDs)
	}
	return searchutil.EvaluateExamContextRetrieval(ctx, toSearchutilEvalCases(req.Cases), resolver)
}

func toSearchutilEvalCases(cases []ExamContextRetrievalEvalCase) []searchutil.ExamContextRetrievalEvalCase {
	out := make([]searchutil.ExamContextRetrievalEvalCase, 0, len(cases))
	for _, evalCase := range cases {
		out = append(out, searchutil.ExamContextRetrievalEvalCase{
			ExamContextEvalCase: searchutil.ExamContextEvalCase{
				Name:            evalCase.Name,
				Query:           evalCase.Query,
				RequiredPhrases: evalCase.RequiredPhrases,
			},
			ExpectedChunkIDs:         evalCase.ExpectedChunkIDs,
			RequiredRetrievalPhrases: evalCase.RequiredRetrievalPhrases,
		})
	}
	return out
}
