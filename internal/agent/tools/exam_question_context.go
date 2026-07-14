package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/examrag"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

var examQuestionContextTool = BaseTool{
	name: ToolExamQuestionContext,
	description: `Return structured exam question-group context for exam RAG tasks.

Use this tool after a search result exposes chunk IDs for an exam paper, or when an exact question group ID is known.
It returns the passage/material, questions, options, answers, explanations, and evidence chunk IDs as one model-ready context block.

Prefer this over generic chunk reading when the user asks about:
- an exam reading passage, cloze, writing prompt, or grouped problem
- the original passage/material of a question group
- answers, options, and explanations for exam questions

If you only have a natural-language query and no chunk_id/group_id yet, this tool can perform a lightweight scoped retrieval first, then map matched chunks back to structured exam question groups.`,
	schema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "query": {
      "type": "string",
      "description": "User question or focused exam query."
    },
    "knowledge_base_ids": {
      "type": "array",
      "description": "Optional accessible KB IDs used to resolve tenant scope.",
      "items": { "type": "string" },
      "minItems": 0,
      "maxItems": 10
    },
    "chunk_ids": {
      "type": "array",
      "description": "Chunk IDs from search results that should map back to an exam question group.",
      "items": { "type": "string" },
      "minItems": 0,
      "maxItems": 30
    },
    "group_id": {
      "type": "string",
      "description": "Exact exam question group ID when already known."
    }
  },
  "required": ["query"]
}`),
}

type ExamQuestionContextInput struct {
	Query            string   `json:"query"`
	KnowledgeBaseIDs []string `json:"knowledge_base_ids,omitempty"`
	ChunkIDs         []string `json:"chunk_ids,omitempty"`
	GroupID          string   `json:"group_id,omitempty"`
}

type ExamQuestionContextTool struct {
	BaseTool
	questionRepo         interfaces.ExamQuestionRepository
	knowledgeBaseService interfaces.KnowledgeBaseService
	searchTargets        types.SearchTargets
}

func NewExamQuestionContextTool(
	questionRepo interfaces.ExamQuestionRepository,
	knowledgeBaseService interfaces.KnowledgeBaseService,
	searchTargets types.SearchTargets,
) *ExamQuestionContextTool {
	return &ExamQuestionContextTool{
		BaseTool:             examQuestionContextTool,
		questionRepo:         questionRepo,
		knowledgeBaseService: knowledgeBaseService,
		searchTargets:        searchTargets,
	}
}

func (t *ExamQuestionContextTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	var input ExamQuestionContextInput
	if err := json.Unmarshal(args, &input); err != nil {
		return failedExamContextResult(fmt.Sprintf("Failed to parse args: %v", err)), err
	}
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return failedExamContextResult("query is required"), fmt.Errorf("missing query")
	}

	resolved, err := t.contextResolver().Resolve(ctx, examrag.ExamQuestionContextResolveRequest{
		Query:            query,
		KnowledgeBaseIDs: input.KnowledgeBaseIDs,
		ChunkIDs:         input.ChunkIDs,
		GroupID:          input.GroupID,
	})
	if err != nil {
		return failedExamContextResult(err.Error()), err
	}
	if resolved == nil || len(resolved.CandidateChunkIDs) == 0 && strings.TrimSpace(input.GroupID) == "" {
		return failedExamContextResult("no candidate exam chunks found for query"), nil
	}
	if resolved.Detail == nil || resolved.Detail.Group == nil {
		return failedExamContextResult("exam question context not found"), nil
	}
	if resolved.Bundle == nil {
		return failedExamContextResult("exam question context is empty"), nil
	}
	return formatExamQuestionContextResult(query, resolved), nil
}

func (t *ExamQuestionContextTool) contextResolver() *examrag.ExamQuestionContextResolver {
	return examrag.NewExamQuestionContextResolver(examrag.ExamQuestionContextResolverConfig{
		QuestionRepo:         t.questionRepo,
		KnowledgeBaseService: t.knowledgeBaseService,
		SearchTargets:        t.searchTargets,
	})
}

func formatExamQuestionContextResult(
	query string,
	resolved *examrag.ExamQuestionContextResolveResult,
) *types.ToolResult {
	detail := resolved.Detail
	bundle := resolved.Bundle
	group := detail.Group
	sourceIDs := append([]string{}, bundle.SourceChunkIDs...)
	var out strings.Builder
	out.WriteString(fmt.Sprintf(
		"<exam_question_context group_id=\"%s\" group_type=\"%s\" label=\"%s\">\n",
		xmlEscape(group.ID),
		xmlEscape(group.GroupType),
		xmlEscape(bundle.Label),
	))
	out.WriteString(fmt.Sprintf("<query>%s</query>\n", xmlEscape(query)))
	out.WriteString(fmt.Sprintf("<source_chunk_ids>%s</source_chunk_ids>\n", xmlEscape(strings.Join(sourceIDs, ","))))
	out.WriteString(fmt.Sprintf("<content>%s</content>\n", xmlEscape(bundle.Content)))
	out.WriteString("</exam_question_context>")

	return &types.ToolResult{
		Success: true,
		Output:  out.String(),
		Data: map[string]interface{}{
			"display_type":        ToolExamQuestionContext,
			"query":               query,
			"group_id":            group.ID,
			"group_type":          group.GroupType,
			"group_title":         group.Title,
			"source_chunk_ids":    sourceIDs,
			"retrieved_chunk_ids": append([]string{}, resolved.RetrievedChunkIDs...),
			"candidate_chunk_ids": append([]string{}, resolved.CandidateChunkIDs...),
			"structured_context":  bundle.Content,
		},
	}
}

func failedExamContextResult(message string) *types.ToolResult {
	return &types.ToolResult{Success: false, Error: message}
}
