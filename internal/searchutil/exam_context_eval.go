package searchutil

import (
	"context"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

// ExamContextEvalCase describes one deterministic answerability check for an
// exam question context bundle.
type ExamContextEvalCase struct {
	Name            string
	Query           string
	RequiredPhrases []string
}

type ExamContextEvalResult struct {
	Name           string
	Query          string
	Passed         bool
	Score          float64
	MatchedPhrases []string
	MissingPhrases []string
	ContextLabel   string
	SourceChunkIDs []string
}

type ExamContextEvalSummary struct {
	Total   int
	Passed  int
	HitRate float64
	Results []ExamContextEvalResult
}

type ExamContextRetrievalEvalCase struct {
	ExamContextEvalCase
	ExpectedChunkIDs []string
}

type ExamContextBundleResolver func(
	ctx context.Context,
	query string,
) (*ExamQuestionContextBundle, []string, error)

type ExamContextRetrievalEvalResult struct {
	Name              string
	Query             string
	Passed            bool
	RetrievalPassed   bool
	AnswerPassed      bool
	RetrievalScore    float64
	AnswerScore       float64
	MatchedChunkIDs   []string
	MissingChunkIDs   []string
	RetrievedChunkIDs []string
	MatchedPhrases    []string
	MissingPhrases    []string
	ContextLabel      string
	SourceChunkIDs    []string
	Error             string
}

type ExamContextRetrievalEvalSummary struct {
	Total            int
	Passed           int
	RetrievalPassed  int
	AnswerPassed     int
	HitRate          float64
	RetrievalHitRate float64
	AnswerHitRate    float64
	Results          []ExamContextRetrievalEvalResult
}

func EvaluateStructuredExamQuestionContext(
	detail *types.QuestionGroupDetail,
	cases []ExamContextEvalCase,
) ExamContextEvalSummary {
	results := make([]ExamContextEvalResult, 0, len(cases))
	passed := 0
	for _, evalCase := range cases {
		result := evaluateStructuredExamQuestionContextCase(detail, evalCase)
		if result.Passed {
			passed++
		}
		results = append(results, result)
	}
	return ExamContextEvalSummary{
		Total:   len(cases),
		Passed:  passed,
		HitRate: examEvalHitRate(passed, len(cases)),
		Results: results,
	}
}

func EvaluateExamContextRetrieval(
	ctx context.Context,
	cases []ExamContextRetrievalEvalCase,
	resolver ExamContextBundleResolver,
) ExamContextRetrievalEvalSummary {
	results := make([]ExamContextRetrievalEvalResult, 0, len(cases))
	var passed, retrievalPassed, answerPassed int
	for _, evalCase := range cases {
		result := evaluateExamContextRetrievalCase(ctx, evalCase, resolver)
		if result.Passed {
			passed++
		}
		if result.RetrievalPassed {
			retrievalPassed++
		}
		if result.AnswerPassed {
			answerPassed++
		}
		results = append(results, result)
	}
	total := len(cases)
	return ExamContextRetrievalEvalSummary{
		Total:            total,
		Passed:           passed,
		RetrievalPassed:  retrievalPassed,
		AnswerPassed:     answerPassed,
		HitRate:          examEvalHitRate(passed, total),
		RetrievalHitRate: examEvalHitRate(retrievalPassed, total),
		AnswerHitRate:    examEvalHitRate(answerPassed, total),
		Results:          results,
	}
}

func evaluateExamContextRetrievalCase(
	ctx context.Context,
	evalCase ExamContextRetrievalEvalCase,
	resolver ExamContextBundleResolver,
) ExamContextRetrievalEvalResult {
	result := ExamContextRetrievalEvalResult{
		Name:  evalCase.Name,
		Query: evalCase.Query,
	}
	if resolver == nil {
		result.Error = "exam context resolver is nil"
		result.MissingChunkIDs = cleanEvalPhrases(evalCase.ExpectedChunkIDs)
		result.MissingPhrases = cleanEvalPhrases(evalCase.RequiredPhrases)
		return result
	}
	bundle, retrievedChunkIDs, err := resolver(ctx, evalCase.Query)
	result.RetrievedChunkIDs = cleanEvalPhrases(retrievedChunkIDs)
	if err != nil {
		result.Error = err.Error()
		result.MissingChunkIDs = missingEvalItems(evalCase.ExpectedChunkIDs, result.RetrievedChunkIDs)
		result.MissingPhrases = cleanEvalPhrases(evalCase.RequiredPhrases)
		return result
	}

	result.MatchedChunkIDs, result.MissingChunkIDs = matchEvalItems(
		result.RetrievedChunkIDs,
		evalCase.ExpectedChunkIDs,
	)
	result.RetrievalScore = examEvalScore(len(result.MatchedChunkIDs), len(cleanEvalPhrases(evalCase.ExpectedChunkIDs)))
	result.RetrievalPassed = len(result.MissingChunkIDs) == 0

	if bundle == nil {
		result.MissingPhrases = cleanEvalPhrases(evalCase.RequiredPhrases)
		return result
	}
	result.ContextLabel = bundle.Label
	result.SourceChunkIDs = append([]string{}, bundle.SourceChunkIDs...)
	result.MatchedPhrases, result.MissingPhrases = matchExamEvalPhrases(
		bundle.Content,
		evalCase.RequiredPhrases,
	)
	result.AnswerScore = examEvalScore(len(result.MatchedPhrases), len(cleanEvalPhrases(evalCase.RequiredPhrases)))
	result.AnswerPassed = len(result.MissingPhrases) == 0
	result.Passed = result.RetrievalPassed && result.AnswerPassed
	return result
}

func evaluateStructuredExamQuestionContextCase(
	detail *types.QuestionGroupDetail,
	evalCase ExamContextEvalCase,
) ExamContextEvalResult {
	result := ExamContextEvalResult{Name: evalCase.Name, Query: evalCase.Query}
	bundle := BuildStructuredExamQuestionContextBundle(evalCase.Query, detail)
	if bundle == nil {
		result.MissingPhrases = cleanEvalPhrases(evalCase.RequiredPhrases)
		return result
	}
	result.ContextLabel = bundle.Label
	result.SourceChunkIDs = append([]string{}, bundle.SourceChunkIDs...)
	result.MatchedPhrases, result.MissingPhrases = matchExamEvalPhrases(
		bundle.Content,
		evalCase.RequiredPhrases,
	)
	result.Score = examEvalScore(len(result.MatchedPhrases), len(evalCase.RequiredPhrases))
	result.Passed = len(result.MissingPhrases) == 0
	return result
}

func matchExamEvalPhrases(content string, phrases []string) ([]string, []string) {
	var matched []string
	var missing []string
	for _, phrase := range cleanEvalPhrases(phrases) {
		if strings.Contains(content, phrase) {
			matched = append(matched, phrase)
		} else {
			missing = append(missing, phrase)
		}
	}
	return matched, missing
}

func matchEvalItems(haystack []string, needles []string) ([]string, []string) {
	available := make(map[string]bool)
	for _, item := range cleanEvalPhrases(haystack) {
		available[item] = true
	}
	var matched []string
	var missing []string
	for _, item := range cleanEvalPhrases(needles) {
		if available[item] {
			matched = append(matched, item)
		} else {
			missing = append(missing, item)
		}
	}
	return matched, missing
}

func missingEvalItems(expected []string, actual []string) []string {
	_, missing := matchEvalItems(actual, expected)
	return missing
}

func cleanEvalPhrases(phrases []string) []string {
	out := make([]string, 0, len(phrases))
	seen := make(map[string]bool, len(phrases))
	for _, phrase := range phrases {
		phrase = strings.TrimSpace(phrase)
		if phrase == "" || seen[phrase] {
			continue
		}
		seen[phrase] = true
		out = append(out, phrase)
	}
	return out
}

func examEvalScore(matched int, total int) float64 {
	if total == 0 {
		return 1
	}
	return float64(matched) / float64(total)
}

func examEvalHitRate(passed int, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(passed) / float64(total)
}
