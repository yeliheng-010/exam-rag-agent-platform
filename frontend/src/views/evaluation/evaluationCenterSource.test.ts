import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const read = (relative: string) => readFileSync(new URL(relative, import.meta.url), 'utf8')
const api = read('../../api/exam/evaluation-center.ts')
const router = read('../../router/index.ts')
const menuStore = read('../../stores/menu.ts')
const menu = read('../../components/menu.vue')
const page = [read('./EvaluationCenter.vue'), read('./EvaluationRunPanel.vue'), read('./EvaluationSetPanel.vue')].join('\n')
const style = read('./EvaluationCenter.less')
const ragDetail = read('../question-bank/QuestionBankRAGObservability.vue')
const agentDetail = read('../question-bank/QuestionBankAgentEvaluation.vue')

test('registers platform evaluation APIs and contributor navigation', () => {
  for (const path of ['/evaluation-center', '/evaluation-center/baseline', '/evaluation-sets', '/versions', '/runs']) {
    assert.match(api, new RegExp(path.replace('/', '\\/')))
  }
  assert.match(router, /path:\s*["']evaluations["']/)
  assert.match(router, /name:\s*["']evaluationCenter["']/)
  assert.match(router, /EvaluationCenter\.vue/)
  assert.match(router, /evaluationCenter[\s\S]*?minRole:\s*'contributor'/)
  assert.match(menuStore, /menu\.evaluations[\s\S]*?path:\s*'evaluations'[\s\S]*?minRole:\s*'contributor'/)
  assert.match(menu, /path === 'evaluations'/)
})

test('provides complete regression and versioned set workflows', () => {
  for (const label of ['回归运行', '评测集', '仅看回归', '设为基线', '保存为评测集', '新增版本', '运行此版本']) {
    assert.match(page, new RegExp(label))
  }
  for (const token of ['setInterval', 'clearInterval', 'queued', 'running', 'failed', 'completed']) {
    assert.match(page, new RegExp(token))
  }
  assert.match(page, /evaluationDrilldownPath/)
  assert.match(page, /compatibleEvaluationRuns/)
  assert.match(ragDetail, /route\.query\.run_id/)
  assert.match(agentDetail, /route\.query\.run_id/)
})

test('uses a dense desktop workspace and mobile stacked rows without overflow', () => {
  assert.match(style, /grid-template-columns:\s*minmax\(0,\s*1fr\)/)
  assert.match(style, /@media \(max-width:\s*720px\)/)
  assert.match(style, /\.desktop-only\s*\{[\s\S]*?display:\s*none/)
  assert.match(style, /\.mobile-only\s*\{[\s\S]*?display:\s*block/)
  assert.match(style, /overflow-wrap:\s*anywhere/)
  assert.match(style, /min-width:\s*0/)
  assert.doesNotMatch(style, /font-size:\s*clamp\(/)
})
