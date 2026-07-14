package service

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Tencent/WeKnora/internal/types"
)

const formulaHeavyReferenceThreshold = 3

var mathSubquestionPattern = regexp.MustCompile(`[\(（](\d+)[\)）]`)

func evaluateQuestionGroupDraftQuality(draft *types.ExamQuestionGroupDraft) types.ExamQuestionGroupQualityReport {
	report := types.ExamQuestionGroupQualityReport{Issues: []types.ExamQuestionGroupQualityIssue{}}
	if draft == nil || draft.StrategyCode != gaokaoMathQuestionGroupStrategy {
		return report
	}
	questions, questionErr := draftGroupQuestions(draft)
	assets, assetErr := draftGroupAssets(draft)
	if questionErr != nil || assetErr != nil {
		addQualityIssue(&report, "invalid_draft_json", types.ExamQualitySeverityError, "草稿结构无法读取，请重新抽取或编辑", "")
		return finalizeQualityReport(report)
	}
	for _, question := range questions {
		evaluateMathQuestionQuality(&report, question, draft.MaterialText, assets)
	}
	evaluateMissingSubquestions(&report, questions)
	return finalizeQualityReport(report)
}

func evaluateMathQuestionQuality(
	report *types.ExamQuestionGroupQualityReport,
	question types.ExamQuestionGroupDraftQuestionCandidate,
	materialText string,
	assets []types.ExamQuestionGroupDraftAssetCandidate,
) {
	questionNo := strings.TrimSpace(question.QuestionNo)
	if isChoiceQuestion(question.QuestionTypeCode) {
		if len(question.Options) < 2 {
			addQualityIssue(report, "missing_options", types.ExamQualitySeverityError, "选择题选项少于两个，请补全后再批准", questionNo)
		}
		for _, option := range question.Options {
			if strings.TrimSpace(option.Content) == "" {
				addQualityIssue(report, "empty_option", types.ExamQualitySeverityError, "选择题存在空选项", questionNo)
				break
			}
		}
	}
	if !hasMeaningfulAnswer(question.Answer) {
		addQualityIssue(report, "missing_answer", types.ExamQualitySeverityError, "题目缺少有效答案", questionNo)
	}
	stem := strings.TrimSpace(question.Stem)
	if utf8.RuneCountInString(stem) < 8 {
		addQualityIssue(report, "short_stem", types.ExamQualitySeverityWarning, "题干过短，可能被概括或截断", questionNo)
	}
	if strings.TrimSpace(question.Explanation) == "" {
		addQualityIssue(report, "missing_explanation", types.ExamQualitySeverityWarning, "题目缺少解析", questionNo)
	}
	if referencesFigure(stem+"\n"+materialText) && !hasUsableFigureAsset(assets) {
		addQualityIssue(report, "missing_figure_asset", types.ExamQualitySeverityError, "题干引用图形，但草稿没有可用图片资源", questionNo)
	}
}

func hasMeaningfulAnswer(answer types.JSONMap) bool {
	if len(answer) == 0 {
		return false
	}
	for _, key := range []string{"value", "values", "text", "answer"} {
		value, exists := answer[key]
		if !exists {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return true
			}
		case []any:
			for _, item := range typed {
				if strings.TrimSpace(fmt.Sprint(item)) != "" {
					return true
				}
			}
		case nil:
			continue
		default:
			return true
		}
	}
	return false
}

func evaluateMissingSubquestions(report *types.ExamQuestionGroupQualityReport, questions []types.ExamQuestionGroupDraftQuestionCandidate) {
	indices := make(map[int]bool)
	maxIndex := 0
	for _, question := range questions {
		matches := mathSubquestionPattern.FindStringSubmatch(question.QuestionNo)
		if len(matches) < 2 {
			continue
		}
		var index int
		_, _ = fmt.Sscanf(matches[1], "%d", &index)
		if index > 0 {
			indices[index] = true
			maxIndex = max(maxIndex, index)
		}
	}
	for index := 1; index <= maxIndex; index++ {
		if !indices[index] {
			addQualityIssue(report, "missing_subquestion", types.ExamQualitySeverityError, fmt.Sprintf("缺少第 (%d) 小问", index), "")
		}
	}
}

func referencesFigure(content string) bool {
	for _, marker := range []string{"如图", "下图", "图中", "图示", "图形"} {
		if strings.Contains(content, marker) {
			return true
		}
	}
	return false
}

func hasUsableFigureAsset(assets []types.ExamQuestionGroupDraftAssetCandidate) bool {
	for _, asset := range assets {
		kind := strings.ToLower(strings.TrimSpace(asset.AssetType))
		if (kind == "image" || kind == "figure") && strings.TrimSpace(asset.StorageURI) != "" {
			return true
		}
	}
	return false
}

func addQualityIssue(report *types.ExamQuestionGroupQualityReport, code string, severity types.ExamQualitySeverity, message string, questionNo string) {
	report.Issues = append(report.Issues, types.ExamQuestionGroupQualityIssue{
		Code: code, Severity: severity, Message: message, QuestionNo: questionNo,
	})
}

func finalizeQualityReport(report types.ExamQuestionGroupQualityReport) types.ExamQuestionGroupQualityReport {
	for _, issue := range report.Issues {
		if issue.Severity == types.ExamQualitySeverityError {
			report.ErrorCount++
		} else {
			report.WarningCount++
		}
	}
	report.Blocking = report.ErrorCount > 0
	return report
}

func summarizeQuestionGroupQuality(drafts []*types.ExamQuestionGroupDraft) types.ExamStructuringQualitySummary {
	summary := types.ExamStructuringQualitySummary{TotalDrafts: len(drafts)}
	for _, draft := range drafts {
		report := evaluateQuestionGroupDraftQuality(draft)
		draft.QualityReport = report
		summary.ErrorCount += report.ErrorCount
		summary.WarningCount += report.WarningCount
		if report.ErrorCount > 0 {
			summary.DraftsWithErrors++
		}
		if report.WarningCount > 0 {
			summary.DraftsWithWarning++
		}
	}
	return summary
}

func preflightQuestionGroupChunks(chunks []*types.Chunk) []types.ExamStructuringWarning {
	references := 0
	for _, chunk := range chunks {
		if chunk == nil {
			continue
		}
		content := strings.ToLower(chunk.Content)
		references += strings.Count(content, "x-wmf/")
		references += strings.Count(content, "image/x-wmf")
		references += strings.Count(content, ".wmf")
	}
	if references < formulaHeavyReferenceThreshold {
		return []types.ExamStructuringWarning{}
	}
	warnings := []types.ExamStructuringWarning{{
		Code:           "formula_heavy_docx",
		Severity:       types.ExamQualitySeverityWarning,
		Message:        "文档包含较多 WMF/公式图片，纯文本解析可能丢失公式或图形",
		Recommendation: "优先上传带文本层的 PDF；配置 VLM 后重新解析图像内容",
		ReferenceCount: references,
	}}
	sort.SliceStable(warnings, func(i, j int) bool { return warnings[i].Code < warnings[j].Code })
	return warnings
}
