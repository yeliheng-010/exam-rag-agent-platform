import assert from 'node:assert/strict'
import test from 'node:test'
import type { ExamEvaluationCenterItem, ExamEvaluationSet } from '../../types/exam.ts'
import {
  compatibleEvaluationRuns,
  evaluationComparisonLabel,
  evaluationDrilldownPath,
  formatEvaluationDelta,
  formatEvaluationDuration,
  formatEvaluationRate,
  getEvaluationPrimaryMetrics,
} from './evaluationCenterViewModel.ts'

const ragItem = (): ExamEvaluationCenterItem => ({
  run: {
    id: 'run-rag', tenant_id: 10000, question_bank_id: 'bank-1', evaluation_kind: 'rag',
    created_by: 'teacher-1', status: 'completed', progress: { completed_cases: 2, total_cases: 2 },
    request_snapshot: {}, result_snapshot: {}, error_message: '', is_baseline: false,
    created_at: '2026-07-15T10:00:00Z', updated_at: '2026-07-15T10:01:00Z',
  },
  question_bank_name: '英语题库', metrics: { pass_rate: 0.875, retrieval_hit_rate: 0.9, average_duration_ms: 125.4 },
  baseline_run_id: 'run-base', baseline_metrics: { pass_rate: 0.9 },
  metric_deltas: { pass_rate: -0.025, average_duration_ms: 25.4 },
  comparison_status: 'regressed', regression_reasons: [],
})

test('formats deterministic rates, latency and signed deltas', () => {
  assert.equal(formatEvaluationRate(0.875), '87.5%')
  assert.equal(formatEvaluationDuration(125.4), '125 ms')
  assert.equal(formatEvaluationDelta(-0.025, 'rate'), '-2.5%')
  assert.equal(formatEvaluationDelta(25.4, 'duration'), '+25 ms')
  assert.equal(evaluationComparisonLabel('regressed'), '发生回归')
})

test('builds primary RAG metrics and correct trace drilldown', () => {
  const item = ragItem()
  assert.deepEqual(getEvaluationPrimaryMetrics(item).map(metric => metric.value), ['87.5%', '90%', '125 ms'])
  assert.equal(evaluationDrilldownPath(item), '/platform/question-banks/bank-1/rag-observability?run_id=run-rag')
  item.run.evaluation_kind = 'agent'
  assert.equal(evaluationDrilldownPath(item), '/platform/question-banks/bank-1/agent-evaluation?run_id=run-rag')
})

test('only offers completed runs compatible with an evaluation set scope', () => {
  const set: ExamEvaluationSet = {
    id: 'set-1', tenant_id: 10000, question_bank_id: 'bank-1', evaluation_kind: 'rag',
    name: '英语回归集', description: '', current_version: 1, created_by: 'teacher-1',
    status: 'active', created_at: '', updated_at: '', versions: [],
  }
  const compatible = ragItem()
  const queued = ragItem()
  queued.run.id = 'queued'
  queued.run.status = 'queued'
  const otherBank = ragItem()
  otherBank.run.id = 'other-bank'
  otherBank.run.question_bank_id = 'bank-2'

  assert.deepEqual(compatibleEvaluationRuns(set, [compatible, queued, otherBank]).map(item => item.run.id), ['run-rag'])
})
