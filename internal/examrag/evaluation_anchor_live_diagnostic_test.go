package examrag

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestLiveEvaluationAnchorDiagnostic(t *testing.T) {
	if os.Getenv("WEKNORA_LIVE_ANCHOR_DIAGNOSTIC") == "" {
		t.Skip("set WEKNORA_LIVE_ANCHOR_DIAGNOSTIC=1 to run")
	}
	dsn := os.Getenv("WEKNORA_LIVE_DATABASE_DSN")
	if dsn == "" {
		t.Skip("set WEKNORA_LIVE_DATABASE_DSN to run")
	}
	runID := os.Getenv("WEKNORA_LIVE_EVALUATION_RUN_ID")
	if runID == "" {
		t.Skip("set WEKNORA_LIVE_EVALUATION_RUN_ID to run")
	}
	db, err := gorm.Open(postgres.Open(dsn))
	if err != nil {
		t.Fatal(err)
	}
	var snapshot []byte
	if err := db.Raw(
		"SELECT result_snapshot FROM exam_rag_evaluation_runs WHERE id = ?", runID,
	).Row().Scan(&snapshot); err != nil {
		t.Fatal(err)
	}
	var result types.ExamRAGDiagnosticResult
	if err := json.Unmarshal(snapshot, &result); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewExamQuestionRepository(db)
	candidateRepo := repo.(evaluationQuestionGroupSourceCandidateRepository)
	for _, item := range result.Summary.Results {
		if item.ContextSource != types.ExamRAGContextSourceNone {
			continue
		}
		logLiveEvaluationCandidateScores(t, db, candidateRepo, result.KnowledgeBaseIDs, item)
	}
}

func logLiveEvaluationCandidateScores(
	t *testing.T,
	db *gorm.DB,
	repo evaluationQuestionGroupSourceCandidateRepository,
	knowledgeBaseIDs []string,
	item types.ExamRAGDiagnosticResultItem,
) {
	t.Helper()
	var chunks []*types.Chunk
	if err := db.Where("id IN ?", item.RetrievedChunkIDs).Find(&chunks).Error; err != nil {
		t.Fatal(err)
	}
	contents := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		contents = append(contents, chunk.Content)
	}
	anchors := append(append([]string{}, item.MatchedRetrievalPhrases...), item.MissingRetrievalPhrases...)
	candidates, err := repo.ListEvaluationQuestionGroupCandidatesForSourcePhrases(
		context.Background(), 10000, knowledgeBaseIDs, anchors,
	)
	if err != nil {
		t.Fatal(err)
	}
	unique := deduplicateEquivalentEvaluationCandidates(candidates)
	querySet := evaluationAnchorSetWithLength([]string{item.Query}, evaluationSourceQueryAnchorRuneLength)
	scores := make([]evaluationAnchorScore, 0, len(unique))
	for _, candidate := range unique {
		score := countSharedEvaluationAnchors(
			evaluationAnchorSetWithLength(evaluationCandidateQueryTexts(candidate), evaluationSourceQueryAnchorRuneLength),
			querySet,
		)
		score += evaluationQuestionNumberBonus * evaluationQuestionNumberMatches(item.Query, candidate)
		scores = append(scores, evaluationAnchorScore{detail: candidate, score: score})
	}
	sortEvaluationAnchorScores(scores)
	top := make([]string, 0, 3)
	for index := 0; index < len(scores) && index < 3; index++ {
		top = append(top, fmt.Sprintf("%s:%s=%d", scores[index].detail.Group.ID, scores[index].detail.Group.Title, scores[index].score))
	}
	matched, confidence := matchEvaluationQuestionGroupWithAnchors(item.Query, anchors, contents, candidates)
	matchedID := "none"
	if matched != nil {
		matchedID = matched.Group.ID
	}
	t.Logf(
		"case=%s phrase_in_retrieval=%t candidates=%d unique=%d top=[%s] matched=%s confidence=%.3f",
		item.Name, evaluationSourcePhraseMatchesContents(anchors, contents), len(candidates), len(unique),
		strings.Join(top, ", "), matchedID, confidence,
	)
}
