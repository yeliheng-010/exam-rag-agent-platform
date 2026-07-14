import type {
	ExamRAGEvaluationRun,
	ExamRAGEvaluationRunStatus,
	SearchTrace,
	SearchTraceCandidate,
} from '../../types/exam.ts'

export interface RAGMetricItem {
	label: string
	value: string
}

export interface RAGConfigurationDiff {
	key: string
	label: string
	first: string
	second: string
	changed: boolean
}

export interface RAGTraceStage {
	key: 'vector' | 'keyword' | 'rrf' | 'final'
	label: string
	candidates: SearchTraceCandidate[]
}

export function formatRAGRate(value?: number): string {
	const percentage = Math.max(0, Number(value || 0)) * 100
	return `${new Intl.NumberFormat('zh-CN', { maximumFractionDigits: 1 }).format(percentage)}%`
}

export function formatRAGDuration(value?: number): string {
	return `${Math.round(Number(value || 0))} ms`
}

export function runStatusLabel(status: ExamRAGEvaluationRunStatus): string {
	return {
		queued: '排队中',
		running: '运行中',
		completed: '已完成',
		failed: '失败',
	}[status]
}

export function getRunMetrics(run?: ExamRAGEvaluationRun | null): RAGMetricItem[] {
	const summary = run?.result_snapshot?.summary
	return [
		{ label: '总体通过率', value: formatRAGRate(summary?.hit_rate) },
		{ label: '召回覆盖', value: formatRAGRate(summary?.retrieval_hit_rate) },
		{ label: '答案覆盖', value: formatRAGRate(summary?.answer_hit_rate) },
		{ label: '结构化解析率', value: formatRAGRate(summary?.structured_resolution_rate) },
		{ label: '平均耗时', value: formatRAGDuration(summary?.average_duration_ms) },
	]
}

export function compareRunConfigurations(
	firstRun: ExamRAGEvaluationRun,
	secondRun: ExamRAGEvaluationRun,
): RAGConfigurationDiff[] {
	const first = firstRun.request_snapshot
	const second = secondRun.request_snapshot
	return [
		configurationRow('match_count', 'MatchCount', first.match_count, second.match_count),
		configurationRow('vector_threshold', '向量阈值', first.vector_threshold, second.vector_threshold),
		configurationRow('keyword_threshold', '关键词阈值', first.keyword_threshold, second.keyword_threshold),
		configurationRow('knowledge_base_ids', '知识库数量', first.knowledge_base_ids.length, second.knowledge_base_ids.length),
	]
}

export function compareRunMetrics(
	firstRun: ExamRAGEvaluationRun,
	secondRun: ExamRAGEvaluationRun,
): RAGConfigurationDiff[] {
	const first = firstRun.result_snapshot?.summary
	const second = secondRun.result_snapshot?.summary
	return [
		metricRow('hit_rate', '总体通过率', first?.hit_rate, second?.hit_rate),
		metricRow('retrieval_hit_rate', '召回覆盖', first?.retrieval_hit_rate, second?.retrieval_hit_rate),
		metricRow('answer_hit_rate', '答案覆盖', first?.answer_hit_rate, second?.answer_hit_rate),
		metricRow('structured_resolution_rate', '结构化解析率', first?.structured_resolution_rate, second?.structured_resolution_rate),
		{
			key: 'average_duration_ms', label: '平均耗时',
			first: formatRAGDuration(first?.average_duration_ms),
			second: formatRAGDuration(second?.average_duration_ms),
			changed: first?.average_duration_ms !== second?.average_duration_ms,
		},
	]
}

function metricRow(key: string, label: string, first?: number, second?: number): RAGConfigurationDiff {
	return { key, label, first: formatRAGRate(first), second: formatRAGRate(second), changed: first !== second }
}

function configurationRow(key: string, label: string, first: number, second: number): RAGConfigurationDiff {
	return {
		key,
		label,
		first: String(first),
		second: String(second),
		changed: first !== second,
	}
}

export function getTraceStages(trace: SearchTrace): RAGTraceStage[] {
	return [
		{ key: 'vector', label: 'Vector', candidates: trace.vector_candidates || [] },
		{ key: 'keyword', label: 'Keyword', candidates: trace.keyword_candidates || [] },
		{ key: 'rrf', label: 'RRF 融合', candidates: trace.fusion_candidates || [] },
		{ key: 'final', label: 'Final chunks', candidates: trace.final_chunks || [] },
	]
}

export function visibleTraceCandidates(
	candidates: SearchTraceCandidate[],
	expanded: boolean,
	limit = 12,
): SearchTraceCandidate[] {
	return expanded ? candidates : candidates.slice(0, limit)
}
