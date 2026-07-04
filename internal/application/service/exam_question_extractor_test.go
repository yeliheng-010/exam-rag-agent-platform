package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestBuildQuestionExtractionPromptKeepsLateAnswerChunks(t *testing.T) {
	chunks := make([]*types.Chunk, 0, 9)
	for i := 0; i < 8; i++ {
		chunks = append(chunks, &types.Chunk{
			ID:         fmt.Sprintf("body-%d", i),
			ChunkIndex: i,
			Content:    strings.Repeat("阅读正文内容。", 450),
		})
	}
	chunks = append(chunks, &types.Chunk{
		ID:         "answer-80",
		ChunkIndex: 80,
		Content:    "【21~23题答案】\n【答案】21. B 22. A 23. C",
	})

	prompt := buildQuestionExtractionPrompt(
		&types.ExamMaterial{Title: "2026 高考英语", DomainID: "gaokao"},
		&types.ExamStructuringTask{QuestionBankID: "bank-1"},
		chunks,
	)

	if !strings.Contains(prompt, "--- chunk_id: answer-80 | chunk_index: 80 ---") {
		t.Fatalf("prompt should keep late answer chunk, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "21. B 22. A 23. C") {
		t.Fatalf("prompt should include answer content, got:\n%s", prompt)
	}
}

func TestBuildQuestionExtractionPromptFormatsSubjectID(t *testing.T) {
	subjectID := "english"
	prompt := buildQuestionExtractionPrompt(
		&types.ExamMaterial{Title: "2026 高考英语", DomainID: "gaokao", SubjectID: &subjectID},
		&types.ExamStructuringTask{QuestionBankID: "bank-1"},
		[]*types.Chunk{{ID: "chunk-1", ChunkIndex: 1, Content: "21. What is asked?\nA. One\nB. Two"}},
	)

	if !strings.Contains(prompt, "科目：english") {
		t.Fatalf("prompt should include dereferenced subject id, got:\n%s", prompt)
	}
	if strings.Contains(prompt, "科目：0x") {
		t.Fatalf("prompt should not include pointer address, got:\n%s", prompt)
	}
}

func TestBuildQuestionExtractionPromptFocusesFirstReadingGroup(t *testing.T) {
	chunks := []*types.Chunk{
		{ID: "listening-1", ChunkIndex: 1, Content: "第一部分 听力\n" + strings.Repeat("listening question ", 200)},
		{ID: "reading-a-start", ChunkIndex: 10, Content: "第一部分 阅读理解\n阅读下列短文，从每题所给的A、B、C、D四个选项中选出最佳选项。\n**A**\nSoFi Stadium Events This Month"},
		{ID: "reading-a-body", ChunkIndex: 11, Content: "21. Which team will play the most games?\nA. Dallas Cowboys. B. Los Angeles Rams.\n22. Which hotel is nearest?"},
		{ID: "reading-a-end", ChunkIndex: 12, Content: "23. What do you need to do if you want to park?\nA. Call staff. B. Prepay. C. Obtain a parking pass.\n**B**"},
		{ID: "reading-b-body", ChunkIndex: 13, Content: "Passage B body should wait for a later extraction batch."},
		{ID: "answer-80", ChunkIndex: 80, Content: "参考答案\n21. B 22. A 23. C"},
	}

	prompt := buildQuestionExtractionPrompt(
		&types.ExamMaterial{Title: "2026 高考英语", DomainID: "gaokao"},
		&types.ExamStructuringTask{QuestionBankID: "bank-1"},
		chunks,
	)

	if strings.Contains(prompt, "listening question") {
		t.Fatalf("prompt should skip pre-reading chunks, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "SoFi Stadium Events This Month") {
		t.Fatalf("prompt should include first reading passage, got:\n%s", prompt)
	}
	if strings.Contains(prompt, "Passage B body should wait") {
		t.Fatalf("prompt should stop before passage B body, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "21. B 22. A 23. C") {
		t.Fatalf("prompt should keep answer chunk, got:\n%s", prompt)
	}
}
