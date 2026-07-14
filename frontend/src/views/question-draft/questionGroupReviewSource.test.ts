import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import test from 'node:test'

const router = readFileSync(new URL('../../router/index.ts', import.meta.url), 'utf8')
const classDetail = readFileSync(new URL('../classes/ClassDetail.vue', import.meta.url), 'utf8')

const groupDraftApiUrl = new URL('../../api/exam/question-group-draft.ts', import.meta.url)
const groupApiUrl = new URL('../../api/exam/question-group.ts', import.meta.url)
const groupReviewUrl = new URL('./QuestionGroupDraftReview.vue', import.meta.url)
const groupReviewComposableUrl = new URL('./useQuestionGroupDraftReview.ts', import.meta.url)

test('wires class structuring tasks to question group review', () => {
  assert.equal(existsSync(groupDraftApiUrl), true)
  assert.equal(existsSync(groupReviewUrl), true)
  assert.match(router, /name:\s*["']questionGroupDraftReview["']/)
  assert.match(router, /QuestionGroupDraftReview\.vue/)
  assert.match(classDetail, /extractQuestionGroupDrafts/)
  assert.match(classDetail, /question-group-drafts\/\$\{row\.id\}/)
})

test('declares asynchronous question group frontend APIs', () => {
  assert.equal(existsSync(groupApiUrl), true)
  const groupApi = readFileSync(groupApiUrl, 'utf8')
  const groupDraftApi = readFileSync(groupDraftApiUrl, 'utf8')
  const groupReviewComposable = readFileSync(groupReviewComposableUrl, 'utf8')

  assert.match(groupApi, /question-banks\/\$\{bankId\}\/question-groups/)
  assert.match(groupDraftApi, /group-extract/)
  assert.match(groupDraftApi, /question-group-drafts\/\$\{draftId\}\/approve/)
  assert.doesNotMatch(groupDraftApi, /GROUP_EXTRACTION_TIMEOUT_MS/)
  assert.match(groupReviewComposable, /createQuestionGroupExtractionPoller/)
  assert.match(groupReviewComposable, /loadDrafts\(true\)/)
})
