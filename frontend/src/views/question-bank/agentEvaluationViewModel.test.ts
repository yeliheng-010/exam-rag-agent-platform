import assert from 'node:assert/strict'
import test from 'node:test'
import type { ExamAgentEvaluationRun } from '../../types/exam.ts'
import {
  buildAgentEvaluationCases,
  createAgentEvaluationScenario,
  formatAgentDuration,
  formatAgentRate,
  getAgentRunMetrics,
  getSafeAgentEvaluationTools,
  resolveAgentModelName,
  runStatusLabel,
} from './agentEvaluationViewModel.ts'

const completedRun = (): ExamAgentEvaluationRun => ({
  id: 'run-1',
  tenant_id: 10000,
  question_bank_id: 'bank-1',
  evaluation_kind: 'agent',
  agent_id: 'agent-1',
  created_by: 'teacher-1',
  status: 'completed',
  progress: { completed_cases: 2, total_cases: 2 },
  request_snapshot: {
    agent: {
      id: 'agent-1', name: '诊断助手', model_id: 'model-1',
      allowed_tools: ['exam_class_diagnosis'], knowledge_bases: ['kb-1'], config: {},
    },
    cases: [],
  },
  result_snapshot: {
    question_bank_id: 'bank-1',
    agent: {
      id: 'agent-1', name: '诊断助手', model_id: 'model-1',
      allowed_tools: ['exam_class_diagnosis'], knowledge_bases: ['kb-1'], config: {},
    },
    summary: {
      total: 2, passed: 1, failed: 1, pass_rate: 0.5,
      tool_sequence_rate: 1, tool_arguments_rate: 0.75, evidence_rate: 0.5,
      citation_rate: 0.5, answer_rate: 1, groundedness_rate: 0.5,
      average_duration_ms: 125.4,
    },
    results: [],
  },
  error_message: '',
  created_at: '2026-07-15T10:00:00Z',
  updated_at: '2026-07-15T10:00:01Z',
})

test('builds deterministic Agent cases from structured drafts', () => {
  const scenario = createAgentEvaluationScenario(1)
  scenario.name = '班级诊断'
  scenario.input = '诊断 class-1'
  scenario.expectedToolCalls = [{ name: 'exam_class_diagnosis', argumentsJSON: '{"class_id":"class-1"}' }]
  scenario.evidencePhrases = '第21题\n高频错题'

  assert.deepEqual(buildAgentEvaluationCases([scenario]), [{
    name: '班级诊断', input: '诊断 class-1',
    expected_tool_calls: [{ name: 'exam_class_diagnosis', arguments: { class_id: 'class-1' } }],
    expected_evidence_phrases: ['第21题', '高频错题'],
    expected_citations: [], expected_answer_phrases: [], grounded_phrases: [],
  }])
})

test('rejects invalid expected tool argument JSON', () => {
  const scenario = createAgentEvaluationScenario(1)
  scenario.expectedToolCalls = [{ name: 'exam_question_context', argumentsJSON: '{invalid' }]
  assert.throws(() => buildAgentEvaluationCases([scenario]), /场景 1.*JSON/)
})

test('formats status and persisted Agent metrics', () => {
  assert.equal(runStatusLabel('running'), '运行中')
  assert.equal(formatAgentRate(0.875), '87.5%')
  assert.equal(formatAgentDuration(125.4), '125 ms')
  assert.deepEqual(getAgentRunMetrics(completedRun()).map(item => item.value), [
    '50%', '100%', '75%', '50%', '50%', '125 ms',
  ])
})

test('keeps only supported read-only Agent tools', () => {
  assert.deepEqual(
    getSafeAgentEvaluationTools([
      'exam_class_diagnosis', 'wiki_write_page', 'data_analysis', 'data_schema',
      'wiki_read_source_doc', 'exam_question_context',
    ]),
    ['exam_class_diagnosis', 'exam_question_context'],
  )
})

test('uses the production default tool contract when allowed tools are empty', () => {
  const tools = getSafeAgentEvaluationTools([])
  assert.ok(tools.includes('exam_class_diagnosis'))
  assert.ok(tools.includes('exam_question_context'))
  assert.ok(tools.includes('knowledge_search'))
  assert.ok(!tools.includes('data_analysis'))
})

test('shows the configured model name instead of its storage id', () => {
  const models = [
    { id: 'model-1', name: 'gemma3:12b-tools' },
    { id: 'model-2', name: 'internal-name', display_name: 'Gemma 3 Tools' },
  ]

  assert.equal(resolveAgentModelName('model-1', models), 'gemma3:12b-tools')
  assert.equal(resolveAgentModelName('model-2', models), 'Gemma 3 Tools')
  assert.equal(resolveAgentModelName('missing-model', models), 'missing-model')
  assert.equal(resolveAgentModelName('', models), '-')
})
