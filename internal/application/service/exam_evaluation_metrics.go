package service

import (
	"encoding/json"
	"math"
	"sort"

	"github.com/Tencent/WeKnora/internal/types"
)

const (
	evaluationQualityRegressionThreshold = 0.01
	evaluationLatencyRegressionRatio     = 0.10
	evaluationLatencyRegressionFloorMS   = 100.0
)

var evaluationMetricLabels = map[string]string{
	"pass_rate": "总体通过率", "retrieval_hit_rate": "召回通过率",
	"answer_hit_rate": "答案覆盖率", "recall_at_k": "Recall@K",
	"mean_reciprocal_rank": "MRR", "structured_resolution_rate": "结构化解析率",
	"tool_sequence_rate": "工具顺序", "tool_arguments_rate": "工具参数",
	"evidence_rate": "证据覆盖", "citation_rate": "引用覆盖",
	"answer_rate": "答案覆盖", "groundedness_rate": "Groundedness",
	"average_duration_ms": "平均耗时",
}

func buildEvaluationCenterItems(
	runs []*types.ExamRAGEvaluationRun,
	baselines []*types.ExamRAGEvaluationRun,
	banks []*types.QuestionBank,
	sets []*types.ExamEvaluationSet,
) []*types.ExamEvaluationCenterItem {
	baselineMap := make(map[string]*types.ExamRAGEvaluationRun, len(baselines))
	for _, baseline := range baselines {
		if baseline != nil {
			baselineMap[evaluationRunScopeKey(baseline)] = baseline
		}
	}
	bankNames := make(map[string]string, len(banks))
	for _, bank := range banks {
		if bank != nil {
			bankNames[bank.ID] = bank.Name
		}
	}
	setNames := make(map[string]string, len(sets))
	for _, set := range sets {
		if set != nil {
			setNames[set.ID] = set.Name
		}
	}
	items := make([]*types.ExamEvaluationCenterItem, 0, len(runs))
	for _, run := range runs {
		if run == nil {
			continue
		}
		item := compareEvaluationRun(run, baselineMap[evaluationRunScopeKey(run)])
		item.QuestionBankName = bankNames[run.QuestionBankID]
		item.AgentName = evaluationRunAgentName(run)
		item.EvaluationSetName = setNames[run.EvaluationSetID]
		item.Run = compactEvaluationCenterRun(run)
		items = append(items, item)
	}
	return items
}

func compareEvaluationRun(
	run *types.ExamRAGEvaluationRun,
	baseline *types.ExamRAGEvaluationRun,
) *types.ExamEvaluationCenterItem {
	item := &types.ExamEvaluationCenterItem{
		Run: run, Metrics: evaluationRunMetrics(run),
		ComparisonStatus:  types.ExamEvaluationComparisonUncompared,
		RegressionReasons: []types.ExamEvaluationRegressionReason{},
	}
	if run.IsBaseline {
		item.BaselineRunID = run.ID
		item.BaselineMetrics = cloneEvaluationMetrics(item.Metrics)
		item.MetricDeltas = zeroEvaluationDeltas(item.Metrics)
		item.ComparisonStatus = types.ExamEvaluationComparisonBaseline
		return item
	}
	if run.Status != types.ExamRAGEvaluationRunStatusCompleted || baseline == nil || len(item.Metrics) == 0 {
		return item
	}
	item.BaselineRunID = baseline.ID
	item.BaselineMetrics = evaluationRunMetrics(baseline)
	if len(item.BaselineMetrics) == 0 {
		return item
	}
	item.MetricDeltas, item.RegressionReasons = compareEvaluationMetrics(item.Metrics, item.BaselineMetrics)
	item.ComparisonStatus = types.ExamEvaluationComparisonStable
	if len(item.RegressionReasons) > 0 {
		item.ComparisonStatus = types.ExamEvaluationComparisonRegressed
	}
	return item
}

func evaluationRunMetrics(run *types.ExamRAGEvaluationRun) map[string]float64 {
	if run == nil || len(run.ResultSnapshot) == 0 {
		return map[string]float64{}
	}
	if run.EvaluationKind == types.ExamEvaluationKindAgent {
		var result types.ExamAgentEvaluationResult
		if json.Unmarshal(run.ResultSnapshot, &result) != nil {
			return map[string]float64{}
		}
		s := result.Summary
		return map[string]float64{
			"pass_rate": s.PassRate, "tool_sequence_rate": s.ToolSequenceRate,
			"tool_arguments_rate": s.ToolArgumentsRate, "evidence_rate": s.EvidenceRate,
			"citation_rate": s.CitationRate, "answer_rate": s.AnswerRate,
			"groundedness_rate": s.GroundednessRate, "average_duration_ms": s.AverageDurationMS,
		}
	}
	var result types.ExamRAGDiagnosticResult
	if json.Unmarshal(run.ResultSnapshot, &result) != nil {
		return map[string]float64{}
	}
	s := result.Summary
	return map[string]float64{
		"pass_rate": s.HitRate, "retrieval_hit_rate": s.RetrievalHitRate,
		"answer_hit_rate": s.AnswerHitRate, "recall_at_k": s.RecallAtK,
		"mean_reciprocal_rank":       s.MeanReciprocalRank,
		"structured_resolution_rate": s.StructuredResolutionRate,
		"average_duration_ms":        s.AverageDurationMS,
	}
}

func compareEvaluationMetrics(current, baseline map[string]float64) (map[string]float64, []types.ExamEvaluationRegressionReason) {
	deltas := make(map[string]float64, len(current))
	reasons := make([]types.ExamEvaluationRegressionReason, 0)
	for metric, value := range current {
		base, ok := baseline[metric]
		if !ok {
			continue
		}
		delta := roundEvaluationMetric(value - base)
		deltas[metric] = delta
		regressed := metric != "average_duration_ms" && delta < -evaluationQualityRegressionThreshold
		if metric == "average_duration_ms" {
			regressed = delta > evaluationLatencyRegressionFloorMS && delta > base*evaluationLatencyRegressionRatio
		}
		if regressed {
			reasons = append(reasons, types.ExamEvaluationRegressionReason{
				Metric: metric, Label: evaluationMetricLabels[metric], Baseline: base, Current: value, Delta: delta,
			})
		}
	}
	sort.Slice(reasons, func(i, j int) bool { return reasons[i].Metric < reasons[j].Metric })
	return deltas, reasons
}

func filterEvaluationCenterItems(items []*types.ExamEvaluationCenterItem, filter types.ExamEvaluationRegressionFilter, limit int) []*types.ExamEvaluationCenterItem {
	filtered := make([]*types.ExamEvaluationCenterItem, 0, len(items))
	for _, item := range items {
		match := filter == types.ExamEvaluationRegressionAll ||
			(filter == types.ExamEvaluationRegressionRegressed && item.ComparisonStatus == types.ExamEvaluationComparisonRegressed) ||
			(filter == types.ExamEvaluationRegressionStable && (item.ComparisonStatus == types.ExamEvaluationComparisonStable || item.ComparisonStatus == types.ExamEvaluationComparisonBaseline)) ||
			(filter == types.ExamEvaluationRegressionUncompared && item.ComparisonStatus == types.ExamEvaluationComparisonUncompared)
		if match {
			filtered = append(filtered, item)
		}
		if len(filtered) == limit {
			break
		}
	}
	return filtered
}

func calculateEvaluationCenterSummary(items []*types.ExamEvaluationCenterItem) types.ExamEvaluationCenterSummary {
	summary := types.ExamEvaluationCenterSummary{TotalRuns: len(items)}
	var passTotal float64
	for _, item := range items {
		if item.Run.Status == types.ExamRAGEvaluationRunStatusCompleted {
			summary.CompletedRuns++
			if value, ok := item.Metrics["pass_rate"]; ok {
				passTotal += value
			}
		}
		if item.Run.IsBaseline {
			summary.BaselineRuns++
		}
		if item.ComparisonStatus == types.ExamEvaluationComparisonRegressed {
			summary.RegressionRuns++
		}
	}
	if summary.CompletedRuns > 0 {
		summary.AveragePassRate = passTotal / float64(summary.CompletedRuns)
	}
	return summary
}

func evaluationRunScopeKey(run *types.ExamRAGEvaluationRun) string {
	return run.QuestionBankID + "\x00" + string(run.EvaluationKind) + "\x00" + run.AgentID
}

func evaluationRunAgentName(run *types.ExamRAGEvaluationRun) string {
	if run == nil || run.EvaluationKind != types.ExamEvaluationKindAgent {
		return ""
	}
	var result types.ExamAgentEvaluationResult
	if json.Unmarshal(run.ResultSnapshot, &result) == nil && result.Agent.Name != "" {
		return result.Agent.Name
	}
	var request types.ExamAgentEvaluationRequestSnapshot
	if json.Unmarshal(run.RequestSnapshot, &request) == nil {
		return request.Agent.Name
	}
	return ""
}

func compactEvaluationCenterRun(run *types.ExamRAGEvaluationRun) *types.ExamRAGEvaluationRun {
	clone := *run
	if run.EvaluationKind == types.ExamEvaluationKindAgent {
		clone.RequestSnapshot = compactAgentEvaluationRequestSnapshot(run.RequestSnapshot)
		clone.ResultSnapshot = compactAgentEvaluationResultSnapshot(run.ResultSnapshot)
	} else {
		var request examRAGEvaluationRequestSnapshot
		if json.Unmarshal(run.RequestSnapshot, &request) == nil {
			request.Cases = nil
			clone.RequestSnapshot, _ = json.Marshal(request)
		}
		clone.ResultSnapshot = compactRAGEvaluationResultSnapshot(run.ResultSnapshot)
	}
	return &clone
}

func cloneEvaluationMetrics(input map[string]float64) map[string]float64 {
	result := make(map[string]float64, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func zeroEvaluationDeltas(input map[string]float64) map[string]float64 {
	result := make(map[string]float64, len(input))
	for key := range input {
		result[key] = 0
	}
	return result
}

func roundEvaluationMetric(value float64) float64 {
	return math.Round(value*10000) / 10000
}
