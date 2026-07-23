import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const read = (relative: string) => readFileSync(new URL(relative, import.meta.url), 'utf8')
const api = read('../../api/exam/question-bank.ts')
const router = read('../../router/index.ts')
const detail = read('./QuestionBankDetail.vue')
const page = read('./QuestionBankRAGObservability.vue')
const style = read('./QuestionBankRAGObservability.less')
const runDetail = read('./RAGEvaluationRunDetail.vue')
const runDetailStyle = read('./RAGEvaluationRunDetail.less')
const trace = read('./RAGEvaluationTrace.vue')
const viewModel = read('./ragEvaluationViewModel.ts')

test('registers persistent evaluation APIs and the nested route', () => {
  assert.match(api, /createQuestionBankRAGEvaluationRun/)
  assert.match(api, /listQuestionBankRAGEvaluationRuns/)
  assert.match(api, /getQuestionBankRAGEvaluationRun/)
  assert.match(router, /question-banks\/:bankId\/rag-observability/)
  assert.match(router, /QuestionBankRAGObservability\.vue/)
})

test('keeps the observability center inside question bank context', () => {
  assert.match(detail, /进入评测中心/)
  assert.match(detail, /rag-observability/)
  assert.match(page, /返回题库/)
  assert.match(page, /运行评测/)
})

test('covers async states, polling, metrics, history and comparison', () => {
  for (const token of ['queued', 'running', 'failed', 'completed', 'setInterval', 'clearInterval']) {
    assert.match(page, new RegExp(token))
  }
  for (const label of ['总体通过率', 'Top-K 命中率', '候选 Recall', '答案覆盖', '结构化解析率', '平均耗时']) {
    assert.match(page + viewModel, new RegExp(label))
  }
  assert.match(page, /selectedComparisonRunIds/)
  assert.match(page, /compareRunConfigurations/)
	assert.match(page + runDetail, /切块快照/)
	assert.match(page + viewModel, /compareChunkingSnapshots/)
	assert.match(runDetail + viewModel, /无检索金标/)
	assert.match(runDetail + viewModel, /未记录/)
	assert.match(style, /grid-template-columns:\s*repeat\(9,/)
})

test('renders all retrieval trace stages and copyable chunk ids', () => {
  for (const token of ['vector', 'keyword', 'RRF', 'final', 'chunk_id', 'copyChunkID']) {
    assert.match(trace + viewModel, new RegExp(token))
  }
  assert.match(trace, /candidate\.vector_rank/)
  assert.match(trace, /candidate\.keyword_rank/)
})

test('wraps long case names before the mobile status tag', () => {
  assert.match(runDetail + runDetailStyle, /\.case-copy\s*\{[\s\S]*?strong\s*\{[^}]*overflow-wrap:\s*anywhere/)
})

test('reserves mobile space for the fixed notification button', () => {
  assert.match(runDetailStyle, /\.case-row summary\s*\{[^}]*padding-right:\s*40px/)
})

test('guards polling from overlap and stale active-run responses', () => {
  assert.match(page, /pollInFlight/)
  assert.match(page, /activeRun\.value\?\.id !== runId/)
})

test('shows the persisted pass rate in each history row', () => {
  assert.match(page, /run-row__rate/)
  assert.match(page, /runPassRate/)
  assert.doesNotMatch(page, /\.run-row__rate\s*\{[^}]*display:\s*none/)
})
