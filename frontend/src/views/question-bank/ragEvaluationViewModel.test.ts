import assert from 'node:assert/strict'
import test from 'node:test'
import type { ExamRAGEvaluationRun, SearchTrace } from '../../types/exam.ts'
import {
  compareRunConfigurations,
	compareChunkingSnapshots,
	compareRunMetrics,
	formatAssociationConfidence,
	formatContextSource,
	formatFirstRelevantRank,
  formatRAGDuration,
  formatRAGRate,
  formatRankedRate,
	formatOptionalRankedRate,
  getChunkingSnapshotRows,
  getRunMetrics,
  getTraceStages,
  runStatusLabel,
	visibleTraceCandidates,
} from './ragEvaluationViewModel.ts'

const run = (matchCount: number, hitRate: number): ExamRAGEvaluationRun => ({
  id: `run-${matchCount}`,
  tenant_id: 10000,
  question_bank_id: 'bank-1',
  created_by: 'teacher-1',
  status: 'completed',
  progress: { completed_cases: 2, total_cases: 2 },
  request_snapshot: {
    knowledge_base_ids: ['kb-1'],
    cases: [],
    match_count: matchCount,
    vector_threshold: 0.5,
    keyword_threshold: 0.3,
		chunking_snapshots: [{
			knowledge_base_id: 'kb-1',
			config: {
				strategy: 'auto', chunk_size: 512, chunk_overlap: 80,
				enable_parent_child: true, parent_chunk_size: 4096,
				child_chunk_size: 384, token_limit: 0, languages: ['en'],
			},
			knowledge_count: 2, text_chunk_count: 3, parent_chunk_count: 1,
			min_chars: 10, p50_chars: 50, p90_chars: 120, max_chars: 140,
			tiny_chunk_rate: 1 / 3, oversize_rate: 1 / 3, parent_coverage: 2 / 3,
			actual_tier_counts: { heading: 1, heuristic: 2 }, unknown_tier_count: 0,
		}],
  },
  result_snapshot: {
    question_bank: null,
    knowledge_base_ids: ['kb-1'],
    used_default_cases: true,
    cases: [],
    summary: {
      total: 2,
      passed: 1,
      retrieval_passed: 2,
      answer_passed: 1,
      hit_rate: hitRate,
      retrieval_hit_rate: 1,
      answer_hit_rate: 0.5,
      recall_at_k: 0.75,
			candidate_recall: 1,
			candidate_hit_rate: 0.8,
      mean_reciprocal_rank: 0.5,
      ranked_case_count: 2,
      structured_resolution_rate: 0.5,
      structured_resolved: 1,
      average_duration_ms: 124.5,
      failed_case_count: 0,
      results: [],
    },
  },
  error_message: '',
  created_at: '2026-07-14T10:00:00Z',
  updated_at: '2026-07-14T10:00:01Z',
})

test('formats metrics and status for display', () => {
  assert.equal(formatRAGRate(0.875), '87.5%')
  assert.equal(formatRAGDuration(124.5), '125 ms')
  assert.equal(runStatusLabel('running'), '运行中')
  assert.deepEqual(getRunMetrics(run(8, 0.5)).map(item => item.value), [
		'50%', '100%', '75%', '50%', '80%', '100%', '50%', '50%', '125 ms',
	])
})

test('shows candidate metrics as unrecorded for legacy snapshots', () => {
	const legacy = run(8, 0.5)
	delete legacy.result_snapshot!.summary.candidate_recall
	delete legacy.result_snapshot!.summary.candidate_hit_rate

	assert.equal(formatOptionalRankedRate(undefined, 2), '未记录')
	assert.deepEqual(getRunMetrics(legacy).slice(4, 6).map(item => item.value), ['未记录', '未记录'])
})

test('formats exact, dynamic, and missing context associations', () => {
	assert.equal(formatContextSource('structured_question_group'), '正式引用关联')
	assert.equal(formatContextSource('evaluation_anchor_match'), '评测动态关联')
	assert.equal(formatContextSource('none'), '未关联')
	assert.equal(formatAssociationConfidence('evaluation_anchor_match', 0.875), '87.5%')
	assert.equal(formatAssociationConfidence('structured_question_group', 0), '-')
})

test('labels retrieval metrics when a run has no retrieval gold', () => {
	const noGold = run(8, 0.5)
	noGold.result_snapshot!.summary.ranked_case_count = 0
	assert.equal(formatRankedRate(0.8, 0), '无检索金标')
	assert.deepEqual(getRunMetrics(noGold).slice(1, 4).map(item => item.value), [
		'无检索金标', '无检索金标', '无检索金标',
	])
})

test('distinguishes an unmatched retrieval gold from a case without retrieval gold', () => {
	assert.equal(formatFirstRelevantRank({
		first_relevant_rank: 0,
		matched_chunk_ids: [], missing_chunk_ids: [],
		matched_retrieval_phrases: [], missing_retrieval_phrases: ['expected phrase'],
	}), '未命中')
	assert.equal(formatFirstRelevantRank({
		first_relevant_rank: 0,
		matched_chunk_ids: [], missing_chunk_ids: [],
		matched_retrieval_phrases: [], missing_retrieval_phrases: [],
	}), '无检索金标')
	assert.equal(formatFirstRelevantRank({
		first_relevant_rank: 3,
		matched_chunk_ids: [], missing_chunk_ids: [],
		matched_retrieval_phrases: ['expected phrase'], missing_retrieval_phrases: [],
	}), '3')
})

test('compares two immutable request snapshots', () => {
  const rows = compareRunConfigurations(run(8, 0.5), run(12, 0.75))
  assert.equal(rows.find(row => row.key === 'match_count')?.changed, true)
  assert.equal(rows.find(row => row.key === 'vector_threshold')?.changed, false)
})

test('compares persisted metrics without inferring causality', () => {
	const first = run(8, 0.5)
	const second = run(12, 0.75)
	delete first.result_snapshot!.summary.candidate_recall
	const rows = compareRunMetrics(first, second)
	assert.deepEqual(rows.find(row => row.key === 'hit_rate'), {
		key: 'hit_rate', label: '总体通过率', first: '50%', second: '75%', changed: true,
	})
	assert.deepEqual(rows.find(row => row.key === 'candidate_recall'), {
		key: 'candidate_recall', label: '候选 Recall', first: '未记录', second: '100%', changed: true,
	})
})

test('does not mark two unrecorded candidate metrics as changed', () => {
	const first = run(8, 0.5)
	const second = run(12, 0.75)
	delete first.result_snapshot!.summary.candidate_recall
	delete second.result_snapshot!.summary.candidate_recall
	first.result_snapshot!.summary.ranked_case_count = 1
	second.result_snapshot!.summary.ranked_case_count = 2

	const row = compareRunMetrics(first, second).find(item => item.key === 'candidate_recall')
	assert.equal(row?.changed, false)
})

test('formats and compares immutable chunking snapshots', () => {
	const first = run(8, 0.5)
	const second = run(12, 0.75)
	second.request_snapshot.chunking_snapshots![0].config.strategy = 'heuristic'
	second.request_snapshot.chunking_snapshots![0].p90_chars = 96

	const rows = getChunkingSnapshotRows(first)
	assert.equal(rows[0].requestedStrategy, 'auto')
	assert.equal(rows[0].actualTiers, 'heading 1 / heuristic 2')
	assert.equal(rows[0].sizes, 'P50 50 / P90 120')
	const comparison = compareChunkingSnapshots(first, second)
	assert.equal(comparison.find(row => row.key.endsWith(':strategy'))?.changed, true)
	assert.equal(comparison.find(row => row.key.endsWith(':p90_chars'))?.changed, true)
})

test('returns search trace stages in retrieval order', () => {
  const trace = {
    vector_candidates: [{ chunk_id: 'vector-1', score: 0.9, rank: 1 }],
    keyword_candidates: [{ chunk_id: 'keyword-1', score: 0.8, rank: 1 }],
    fusion_candidates: [{ chunk_id: 'fusion-1', score: 0.7, rank: 1 }],
    final_chunks: [{ chunk_id: 'final-1', score: 0.7, rank: 1 }],
  } as SearchTrace

  assert.deepEqual(getTraceStages(trace).map(stage => stage.key), ['vector', 'keyword', 'rrf', 'final'])
})

test('limits dense candidate stages until explicitly expanded', () => {
	const candidates = Array.from({ length: 20 }, (_, index) => ({
		chunk_id: `chunk-${index}`, score: 1, rank: index + 1,
	}))
	assert.equal(visibleTraceCandidates(candidates, false).length, 12)
	assert.equal(visibleTraceCandidates(candidates, true).length, 20)
})
