import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const dialog = readFileSync(new URL('./ExamPaperImportDialog.vue', import.meta.url), 'utf8')
const knowledgeBase = readFileSync(new URL('../KnowledgeBase.vue', import.meta.url), 'utf8')
const documentList = readFileSync(new URL('./DocumentListView.vue', import.meta.url), 'utf8')
const questionDraftApi = readFileSync(new URL('../../../api/exam/question-draft.ts', import.meta.url), 'utf8')

test('wires knowledge documents into the exam paper import flow', () => {
  assert.match(knowledgeBase, /ExamPaperImportDialog/)
  assert.match(knowledgeBase, /openExamPaperImport/)
  assert.match(knowledgeBase, /exam-paper/)
  assert.match(documentList, /exam-paper/)
  assert.match(documentList, /导入为试卷/)
})

test('registers material, extracts drafts, then opens review page', () => {
  assert.match(dialog, /registerExamMaterial/)
  assert.match(dialog, /extractQuestionDrafts/)
  assert.match(dialog, /create_task:\s*true/)
  assert.match(dialog, /material_type:\s*'exam_paper'/)
  assert.match(dialog, /structuring-tasks\/\$\{taskId\}\/review/)
  assert.match(questionDraftApi, /QUESTION_DRAFT_EXTRACTION_TIMEOUT_MS/)
  assert.match(questionDraftApi, /timeout:\s*QUESTION_DRAFT_EXTRACTION_TIMEOUT_MS/)
})

test('makes the target question bank explicit and reselects it from paper title', () => {
  assert.match(dialog, /selectedBankNotice/)
  assert.match(dialog, /将写入题库/)
  assert.match(dialog, /自动创建新题库/)
  assert.match(dialog, /reselectQuestionBank/)
  assert.match(dialog, /questionBankMatchScore/)
  assert.doesNotMatch(dialog, /const pickQuestionBank = \(items: QuestionBank\[\]\) => items\[0\]/)
})
