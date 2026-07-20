import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import test from 'node:test'

const router = readFileSync(new URL('../../router/index.ts', import.meta.url), 'utf8')
const learningHome = readFileSync(new URL('../learning/LearningHome.vue', import.meta.url), 'utf8')
const practiceApi = readFileSync(new URL('../../api/exam/practice.ts', import.meta.url), 'utf8')
const practiceViewUrl = new URL('./QuestionGroupPractice.vue', import.meta.url)
const practiceView = readFileSync(practiceViewUrl, 'utf8')
const explanationPrompt = readFileSync(new URL('./explanationPrompt.ts', import.meta.url), 'utf8')

test('wires student question group practice route and learning entry', () => {
  assert.equal(existsSync(practiceViewUrl), true)
  assert.match(router, /name:\s*["']questionGroupPractice["']/)
  assert.match(router, /QuestionGroupPractice\.vue/)
  assert.doesNotMatch(router, /questionGroupPractice[\s\S]{0,180}minRole/)
  assert.match(learningHome, /listPracticeQuestionGroups/)
  assert.match(learningHome, /题组练习/)
  assert.doesNotMatch(learningHome, /canUseQuestionBanks[\s\S]{0,120}listPracticeQuestionGroups/)
})

test('declares practice API endpoints', () => {
  assert.match(practiceApi, /practice\/question-groups/)
  assert.match(practiceApi, /practice\/question-groups\/\$\{groupId\}\/attempts/)
  assert.match(practiceApi, /practice\/attempts\/\$\{attemptId\}\/answers/)
  assert.match(practiceApi, /practice\/attempts\/\$\{attemptId\}\/questions\/\$\{questionId\}\/explanation-context/)
  assert.match(practiceApi, /practice\/attempts\/\$\{attemptId\}\/complete/)
})

test('practice page supports answer submission and AI explanation prefill', () => {
  assert.match(practiceView, /submitPracticeAnswer/)
  assert.match(practiceView, /completePracticeAttempt/)
  assert.match(practiceView, /setPrefillQuery/)
  assert.match(practiceView, /getPracticeExplanationContext/)
  assert.match(practiceView, /reference_answer/)
  assert.match(practiceView, /BUILTIN_QUICK_ANSWER_ID/)
  assert.match(practiceView, /settingsStore\.selectAgent\(BUILTIN_QUICK_ANSWER_ID\)/)
  assert.match(practiceView, /\/platform\/creatChat/)
  assert.match(practiceView, /listExamResources/)
})

test('AI explanation prompt includes supplemental group formula assets', () => {
  assert.match(practiceView, /buildSupplementalAssetPrompt/)
  assert.match(practiceView, /buildGroupReferencedAssetContent/)
  assert.match(practiceView, /group\.value\?\.questions\.flatMap/)
  assert.match(practiceView, /group\.value\?\.assets/)
  assert.match(practiceView, /asset\.storage_uri/)
  assert.match(practiceView, /题面补充公式/)
})

test('AI explanation prompt treats the scoped question-bank answer as authoritative', () => {
  assert.match(practiceView, /buildAuthoritativeReferenceAnswerPrompt/)
  assert.doesNotMatch(practiceView, /buildAuthoritativeReferenceAnswerPrompt\(questionNo\(question\), question\.answers\)/)
  assert.match(explanationPrompt, /【题库参考答案（权威）】/)
  assert.match(explanationPrompt, /只解释和展开参考答案/)
  assert.match(explanationPrompt, /不得声称题目条件不足/)
  assert.match(explanationPrompt, /不得展示与参考答案冲突的试算、自我纠正或失败过程/)
  assert.doesNotMatch(practiceView, /若题面信息仍不足，请明确指出缺失条件/)
})

test('practice page renders all structured question content as rich text', () => {
  assert.match(practiceView, /import ExamRichText from ['"]\.\.\/question-bank\/ExamRichText\.vue['"]/)
  assert.match(practiceView, /:content="group\.group\.material_text \|\| '当前题组没有单独材料。'"/)
  assert.match(practiceView, /:content="currentQuestion\.question\.stem"/)
  assert.match(practiceView, /:content="option\.content"/)
  assert.match(practiceView, /:content="currentResult\.answer\.answer_text \|\| '-'"/)
  assert.match(practiceView, /:content="currentResult\.correct_answers\.join\('，'\) \|\| '-'"/)
  assert.match(practiceView, /:content="item\.explanation_text"/)
  assert.doesNotMatch(practiceView, /\{\{\s*currentQuestion\.question\.stem\s*\}\}/)
  assert.doesNotMatch(practiceView, /\{\{\s*option\.content\s*\}\}/)
})
