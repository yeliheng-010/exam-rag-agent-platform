import assert from 'node:assert/strict'
import test from 'node:test'
import type { ExamRAGEvaluationRun, SearchTrace } from '../../types/exam.ts'
import {
  compareRunConfigurations,
	compareRunMetrics,
  formatRAGDuration,
  formatRAGRate,
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
  assert.deepEqual(getRunMetrics(run(8, 0.5)).map(item => item.value), ['50%', '100%', '50%', '50%', '125 ms'])
})

test('compares two immutable request snapshots', () => {
  const rows = compareRunConfigurations(run(8, 0.5), run(12, 0.75))
  assert.equal(rows.find(row => row.key === 'match_count')?.changed, true)
  assert.equal(rows.find(row => row.key === 'vector_threshold')?.changed, false)
})

test('compares persisted metrics without inferring causality', () => {
	const rows = compareRunMetrics(run(8, 0.5), run(12, 0.75))
	assert.deepEqual(rows.find(row => row.key === 'hit_rate'), {
		key: 'hit_rate', label: '总体通过率', first: '50%', second: '75%', changed: true,
	})
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
