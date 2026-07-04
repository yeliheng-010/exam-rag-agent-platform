package searchutil

import (
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestShouldEnrichExamQuestionContext(t *testing.T) {
	t.Parallel()

	if !ShouldEnrichExamQuestionContext("第一篇阅读的原文和答案是什么") {
		t.Fatal("expected first reading answer query to trigger enrichment")
	}
	if ShouldEnrichExamQuestionContext("总结这份文档") {
		t.Fatal("generic summary query should not trigger enrichment")
	}
}

func TestBuildExamQuestionContextBundle_firstReadingWithDistantAnswer(t *testing.T) {
	t.Parallel()

	chunks := []*types.Chunk{
		textChunk("intro", 10, "阅读下列短文，从每题所给的A、B、C、D四个选项中，选出最佳选项。"),
		textChunk("a1", 11, "**A** **SoFi Stadium Events This Month** SoFi Stadium is the go-to destination."),
		textChunk("a2", 12, "Upcoming Football Events Los Angeles Rams v Dallas Cowboys."),
		textChunk("a3", 17, "21. Which team will play the most games? A. Dallas Cowboys. B. Los Angeles Rams."),
		textChunk("a4", 18, "23. What do you need to do if you want to park at the stadium? C. Obtain a parking pass. **B**"),
		textChunk("b1", 19, "A new passage starts here."),
		textChunk("ans", 80, "**A** 【21~23题答案】 【答案】21. B 22. A 23. C **B** 【24~27题答案】 【答案】24. A"),
		{ID: "parent", ChunkIndex: 11, ChunkType: types.ChunkTypeParentText, Content: "parent should be ignored"},
	}

	bundle := BuildExamQuestionContextBundle("第一篇阅读的原文和答案是什么", chunks)
	if bundle == nil {
		t.Fatal("expected bundle")
	}
	if bundle.Label != "A" {
		t.Fatalf("label = %s, want A", bundle.Label)
	}
	if len(bundle.BodyChunks) != 4 {
		t.Fatalf("body chunk count = %d, want 4", len(bundle.BodyChunks))
	}
	if !strings.Contains(bundle.Content, "SoFi Stadium Events This Month") {
		t.Fatalf("bundle missing passage body:\n%s", bundle.Content)
	}
	if !strings.Contains(bundle.Content, "21. B 22. A 23. C") {
		t.Fatalf("bundle missing answer key:\n%s", bundle.Content)
	}
	if strings.Contains(bundle.Content, "24. A") {
		t.Fatalf("bundle should trim next passage answer:\n%s", bundle.Content)
	}
	if strings.Contains(bundle.Content, "parent should be ignored") {
		t.Fatalf("bundle included parent chunk:\n%s", bundle.Content)
	}
}

func textChunk(id string, index int, content string) *types.Chunk {
	return &types.Chunk{
		ID:         id,
		ChunkIndex: index,
		ChunkType:  types.ChunkTypeText,
		Content:    content,
		StartAt:    index * 100,
		EndAt:      index*100 + len([]rune(content)),
	}
}
