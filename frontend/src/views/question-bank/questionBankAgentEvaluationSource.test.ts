import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const read = (relative: string) => readFileSync(new URL(relative, import.meta.url), 'utf8')
const api = read('../../api/exam/question-bank.ts')
const router = read('../../router/index.ts')
const detail = read('./QuestionBankDetail.vue')
const ragPage = read('./QuestionBankRAGObservability.vue')
const page = read('./QuestionBankAgentEvaluation.vue')
const editor = read('./AgentEvaluationScenarioEditor.vue')
const runDetail = read('./AgentEvaluationRunDetail.vue')
const style = [
  read('./QuestionBankAgentEvaluation.less'),
  read('./AgentEvaluationScenarioEditor.less'),
  read('./AgentEvaluationRunDetail.less'),
].join('\n')
const viewModel = read('./agentEvaluationViewModel.ts')

test('registers Agent evaluation APIs and contributor route', () => {
  for (const token of [
    'createQuestionBankAgentEvaluationRun',
    'listQuestionBankAgentEvaluationRuns',
    'getQuestionBankAgentEvaluationRun',
    '/agent-evaluation-runs',
  ]) assert.match(api, new RegExp(token))
  assert.match(router, /question-banks\/:bankId\/agent-evaluation/)
  assert.match(router, /QuestionBankAgentEvaluation\.vue/)
  assert.match(router, /questionBankAgentEvaluation[\s\S]*?minRole:\s*'contributor'/)
})

test('links RAG, Agent evaluation and question bank context', () => {
  assert.match(detail, /Agent 行为评测/)
  assert.match(detail, /agent-evaluation/)
  assert.match(ragPage, /agent-evaluation/)
  assert.match(page, /rag-observability/)
  assert.match(page, /返回题库/)
})

test('provides structured scenarios and deterministic assertions', () => {
  for (const label of ['场景名称', '用户输入', '预期工具', '参数 JSON', '证据短语', '引用标识', '答案短语', '事实短语']) {
    assert.match(editor, new RegExp(label))
  }
  assert.match(editor, /addScenario/)
  assert.match(editor, /removeScenario/)
  assert.match(viewModel, /JSON\.parse/)
  assert.match(viewModel, /AGENT_EVALUATION_READ_ONLY_TOOLS/)
  assert.doesNotMatch(viewModel, /wiki_write_page/)
})

test('covers async history, polling, metrics and full execution trace', () => {
  for (const token of ['queued', 'running', 'failed', 'completed', 'setInterval', 'clearInterval', 'pollInFlight']) {
    assert.match(page, new RegExp(token))
  }
  for (const label of ['通过率', '工具序列', '参数命中', '证据覆盖', '事实支撑', '平均耗时']) {
    assert.match(page + viewModel, new RegExp(label))
  }
  for (const token of ['actual_tool_calls', 'arguments', 'output_excerpt', 'assertion_failures', 'final_answer']) {
    assert.match(runDetail, new RegExp(token))
  }
})

test('keeps narrow layouts single-column without horizontal overflow', () => {
  assert.match(style, /@media \(max-width: 720px\)/)
  assert.match(style, /grid-template-columns:\s*1fr/)
  assert.match(style, /overflow-wrap:\s*anywhere/)
  assert.match(style, /min-width:\s*0/)
})
