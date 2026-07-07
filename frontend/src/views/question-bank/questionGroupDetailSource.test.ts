import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const questionBankApi = readFileSync(new URL('../../api/exam/question-bank.ts', import.meta.url), 'utf8')
const questionBankDetail = readFileSync(new URL('./QuestionBankDetail.vue', import.meta.url), 'utf8')

test('question bank detail consumes official question group API', () => {
  assert.match(questionBankApi, /export\s+\{\s*listQuestionGroupDetails\s*\}\s+from\s+['"]\.\/question-group['"]/)
  assert.match(questionBankDetail, /listQuestionGroupDetails/)
  assert.doesNotMatch(questionBankDetail, /listQuestionDetails/)
  assert.match(questionBankDetail, /QuestionGroupDetail/)
  assert.match(questionBankDetail, /questionGroups/)
})

test('question bank detail renders group cards with nested questions', () => {
  assert.match(questionBankDetail, /question-group-card/)
  assert.match(questionBankDetail, /question-group-material/)
  assert.match(questionBankDetail, /toggleMaterial/)
  assert.match(questionBankDetail, /group\.assets/)
  assert.match(questionBankDetail, /group\.questions/)
})

test('question bank detail renders full question review content', () => {
  assert.match(questionBankDetail, /item\.options/)
  assert.match(questionBankDetail, /question-option/)
  assert.match(questionBankDetail, /isCorrectOption/)
  assert.match(questionBankDetail, /explanationSummary/)
})
