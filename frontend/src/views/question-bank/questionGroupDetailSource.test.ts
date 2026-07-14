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

test('question bank detail supports in-group question pagination', () => {
  assert.match(questionBankDetail, /activeQuestionIndexes/)
  assert.match(questionBankDetail, /visibleQuestions/)
  assert.match(questionBankDetail, /activeQuestionIndex/)
  assert.match(questionBankDetail, /moveQuestion/)
  assert.match(questionBankDetail, /setActiveQuestionIndex/)
  assert.match(questionBankDetail, /question-review__toolbar/)
  assert.match(questionBankDetail, /question-review__tab/)
})

test('question bank detail supports top-level question group pagination', () => {
  assert.match(questionBankDetail, /activeGroupIndex/)
  assert.match(questionBankDetail, /currentGroupIndex/)
  assert.match(questionBankDetail, /visibleQuestionGroups/)
  assert.match(questionBankDetail, /moveGroup/)
  assert.match(questionBankDetail, /setActiveGroupIndex/)
  assert.match(questionBankDetail, /groupTabTitle/)
  assert.match(questionBankDetail, /question-group-review__toolbar/)
  assert.match(questionBankDetail, /question-group-review__tab/)
})

test('question bank detail exposes the latest structuring task review entry', () => {
  assert.match(questionBankDetail, /listExamStructuringTasks/)
  assert.match(questionBankDetail, /listQuestionGroupDrafts/)
  assert.match(questionBankDetail, /loadQuestionBankReviewEntry/)
  assert.match(questionBankDetail, /latestReviewTask/)
  assert.match(questionBankDetail, /reviewDraftStats/)
  assert.match(questionBankDetail, /reviewDraftErrorCount/)
  assert.match(questionBankDetail, /void loadReviewEntry\(bankId, res\.data\.space_id\)/)
  assert.doesNotMatch(questionBankDetail, /Promise\.all\(\[\s*listQuestionGroupDetails\(bankId\),\s*loadReviewEntry/)
  assert.doesNotMatch(questionBankDetail, /latestReviewTask\.progress\?\.quality_summary\?\.error_count/)
  assert.match(questionBankDetail, /进入题组审核/)
  assert.match(questionBankDetail, /question-group-drafts\/\$\{latestReviewTask\.value\.id\}/)
})
