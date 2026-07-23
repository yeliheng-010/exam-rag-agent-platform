import type {
	ExamRAGDiagnosticResultItem,
	ExamRAGEvaluationRun,
	ExamRAGEvaluationRunStatus,
	ExamRAGChunkingSnapshot,
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

export interface RAGChunkingSnapshotRow {
	knowledgeBaseID: string
	requestedStrategy: string
	actualTiers: string
	chunks: string
	sizes: string
	tinyRate: string
	oversizeRate: string
	parentCoverage: string
	unknownTiers: string
}

type RetrievalRankDisplayItem = Pick<
	ExamRAGDiagnosticResultItem,
	| 'first_relevant_rank'
	| 'matched_chunk_ids'
	| 'missing_chunk_ids'
	| 'matched_retrieval_phrases'
	| 'missing_retrieval_phrases'
>

export function formatRAGRate(value?: number): string {
	const percentage = Math.max(0, Number(value || 0)) * 100
	return `${new Intl.NumberFormat('zh-CN', { maximumFractionDigits: 1 }).format(percentage)}%`
}

export function formatRAGDuration(value?: number): string {
	return `${Math.round(Number(value || 0))} ms`
}

export function formatContextSource(source?: ExamRAGDiagnosticResultItem['context_source']): string {
	return {
		structured_question_group: '正式引用关联',
		evaluation_anchor_match: '评测动态关联',
		none: '未关联',
	}[source || 'none']
}

export function formatAssociationConfidence(
	source?: ExamRAGDiagnosticResultItem['context_source'],
	confidence?: number,
): string {
	return source === 'evaluation_anchor_match' ? formatRAGRate(confidence) : '-'
}

export function formatRankedRate(value: number | undefined, rankedCaseCount: number | undefined): string {
	return Number(rankedCaseCount || 0) > 0 ? formatRAGRate(value) : '无检索金标'
}

export function formatOptionalRankedRate(
	value: number | undefined,
	rankedCaseCount: number | undefined,
): string {
	return value === undefined ? '未记录' : formatRankedRate(value, rankedCaseCount)
}

export function caseHasRetrievalGold(item: RetrievalRankDisplayItem): boolean {
	return Boolean(
		item.matched_chunk_ids?.length || item.missing_chunk_ids?.length ||
		item.matched_retrieval_phrases?.length || item.missing_retrieval_phrases?.length,
	)
}

export function formatFirstRelevantRank(item: RetrievalRankDisplayItem): string {
	const rank = Number(item.first_relevant_rank || 0)
	if (rank > 0) return String(rank)
	return caseHasRetrievalGold(item) ? '未命中' : '无检索金标'
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
	const ranked = summary?.ranked_case_count
	return [
		{ label: '总体通过率', value: formatRAGRate(summary?.hit_rate) },
		{ label: 'Top-K 命中率', value: formatRankedRate(summary?.retrieval_hit_rate, ranked) },
		{ label: 'Recall@K', value: formatRankedRate(summary?.recall_at_k, ranked) },
		{ label: 'MRR', value: formatRankedRate(summary?.mean_reciprocal_rank, ranked) },
		{ label: '候选命中率', value: formatOptionalRankedRate(summary?.candidate_hit_rate, ranked) },
		{ label: '候选 Recall', value: formatOptionalRankedRate(summary?.candidate_recall, ranked) },
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
		rankedMetricRow('retrieval_hit_rate', 'Top-K 命中率', first?.retrieval_hit_rate, second?.retrieval_hit_rate, first?.ranked_case_count, second?.ranked_case_count),
		rankedMetricRow('recall_at_k', 'Recall@K', first?.recall_at_k, second?.recall_at_k, first?.ranked_case_count, second?.ranked_case_count),
		rankedMetricRow('mean_reciprocal_rank', 'MRR', first?.mean_reciprocal_rank, second?.mean_reciprocal_rank, first?.ranked_case_count, second?.ranked_case_count),
		optionalRankedMetricRow('candidate_hit_rate', '候选命中率', first?.candidate_hit_rate, second?.candidate_hit_rate, first?.ranked_case_count, second?.ranked_case_count),
		optionalRankedMetricRow('candidate_recall', '候选 Recall', first?.candidate_recall, second?.candidate_recall, first?.ranked_case_count, second?.ranked_case_count),
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

export function getChunkingSnapshotRows(run?: ExamRAGEvaluationRun | null): RAGChunkingSnapshotRow[] {
	return (run?.request_snapshot.chunking_snapshots || []).map(snapshot => ({
		knowledgeBaseID: snapshot.knowledge_base_id,
		requestedStrategy: snapshot.config.strategy || 'legacy',
		actualTiers: formatTierCounts(snapshot.actual_tier_counts),
		chunks: `${snapshot.text_chunk_count} text / ${snapshot.parent_chunk_count} parent`,
		sizes: `P50 ${snapshot.p50_chars} / P90 ${snapshot.p90_chars}`,
		tinyRate: formatRAGRate(snapshot.tiny_chunk_rate),
		oversizeRate: formatRAGRate(snapshot.oversize_rate),
		parentCoverage: formatRAGRate(snapshot.parent_coverage),
		unknownTiers: String(snapshot.unknown_tier_count),
	}))
}

export function compareChunkingSnapshots(
	firstRun: ExamRAGEvaluationRun,
	secondRun: ExamRAGEvaluationRun,
): RAGConfigurationDiff[] {
	const first = snapshotsByKnowledgeBase(firstRun.request_snapshot.chunking_snapshots)
	const second = snapshotsByKnowledgeBase(secondRun.request_snapshot.chunking_snapshots)
	const ids = [...new Set([...first.keys(), ...second.keys()])].sort()
	return ids.flatMap(id => compareKnowledgeBaseSnapshot(id, first.get(id), second.get(id)))
}

function metricRow(key: string, label: string, first?: number, second?: number): RAGConfigurationDiff {
	return { key, label, first: formatRAGRate(first), second: formatRAGRate(second), changed: first !== second }
}

function rankedMetricRow(
	key: string,
	label: string,
	first: number | undefined,
	second: number | undefined,
	firstCount: number | undefined,
	secondCount: number | undefined,
): RAGConfigurationDiff {
	return {
		key, label,
		first: formatRankedRate(first, firstCount),
		second: formatRankedRate(second, secondCount),
		changed: first !== second || firstCount !== secondCount,
	}
}

function optionalRankedMetricRow(
	key: string,
	label: string,
	first: number | undefined,
	second: number | undefined,
	firstCount: number | undefined,
	secondCount: number | undefined,
): RAGConfigurationDiff {
	return {
		key, label,
		first: formatOptionalRankedRate(first, firstCount),
		second: formatOptionalRankedRate(second, secondCount),
		changed: first !== second || (first !== undefined && second !== undefined && firstCount !== secondCount),
	}
}

function formatTierCounts(counts?: Record<string, number>): string {
	const entries = Object.entries(counts || {}).filter(([, count]) => count > 0)
	return entries.length
		? entries.sort(([first], [second]) => first.localeCompare(second)).map(([tier, count]) => `${tier} ${count}`).join(' / ')
		: '未记录'
}

function snapshotsByKnowledgeBase(snapshots?: ExamRAGChunkingSnapshot[]): Map<string, ExamRAGChunkingSnapshot> {
	return new Map((snapshots || []).map(snapshot => [snapshot.knowledge_base_id, snapshot]))
}

function compareKnowledgeBaseSnapshot(
	id: string,
	first?: ExamRAGChunkingSnapshot,
	second?: ExamRAGChunkingSnapshot,
): RAGConfigurationDiff[] {
	const prefix = `chunking:${id}`
	const label = id.slice(0, 8)
	return [
		chunkingDiff(`${prefix}:strategy`, `${label} strategy`, first?.config.strategy, second?.config.strategy),
		chunkingDiff(`${prefix}:chunk_size`, `${label} chunk size`, first?.config.chunk_size, second?.config.chunk_size),
		chunkingDiff(`${prefix}:child_chunk_size`, `${label} child size`, first?.config.child_chunk_size, second?.config.child_chunk_size),
		chunkingDiff(`${prefix}:actual_tiers`, `${label} actual tiers`, formatTierCounts(first?.actual_tier_counts), formatTierCounts(second?.actual_tier_counts)),
		chunkingDiff(`${prefix}:text_chunks`, `${label} text chunks`, first?.text_chunk_count, second?.text_chunk_count),
		chunkingDiff(`${prefix}:p90_chars`, `${label} P90`, first?.p90_chars, second?.p90_chars),
		chunkingDiff(`${prefix}:tiny_rate`, `${label} tiny`, optionalRate(first?.tiny_chunk_rate), optionalRate(second?.tiny_chunk_rate)),
		chunkingDiff(`${prefix}:oversize_rate`, `${label} oversize`, optionalRate(first?.oversize_rate), optionalRate(second?.oversize_rate)),
	]
}

function optionalRate(value?: number): string {
	return value === undefined ? '未记录' : formatRAGRate(value)
}

function chunkingDiff(key: string, label: string, first: unknown, second: unknown): RAGConfigurationDiff {
	const firstValue = first === undefined || first === '' ? '未记录' : String(first)
	const secondValue = second === undefined || second === '' ? '未记录' : String(second)
	return { key, label, first: firstValue, second: secondValue, changed: firstValue !== secondValue }
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
