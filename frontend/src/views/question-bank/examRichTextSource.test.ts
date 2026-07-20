import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const read = (relative: string) => readFileSync(new URL(relative, import.meta.url), 'utf8')
const detail = read('./QuestionBankDetail.vue')
const detailStyles = read('./QuestionBankDetail.less')
const richText = read('./ExamRichText.vue')

test('question bank review renders every structured text field as rich text', () => {
  assert.match(detail, /ExamRichText/)
  assert.match(detail, /:content="group\.group\.material_text"/)
  assert.match(detail, /:content="item\.question\.stem"/)
  assert.match(detail, /:content="option\.content"/)
  assert.match(detail, /:content="answerSummary\(item\)"/)
  assert.match(detail, /:content="explanationSummary\(item\)"/)
})

test('exam rich text uses sanitized KaTeX markdown and hydrates protected images', () => {
  assert.match(richText, /renderChatMarkdown/)
  assert.match(richText, /sanitizeMarkdownHTML/)
  assert.match(richText, /createSafeImage/)
  assert.match(richText, /hydrateProtectedFileImages/)
  assert.match(richText, /katex\/dist\/katex\.min\.css/)
})

test('exam rich text supports a phrasing-safe root element inside option buttons', () => {
  assert.match(richText, /<component/)
  assert.match(richText, /:is="as"/)
  assert.match(richText, /as\?:\s*['"]div['"]\s*\|\s*['"]span['"]/)
  assert.match(richText, /as:\s*['"]div['"]/)
})

test('inline exam math stays inside narrow question and option columns', () => {
  assert.match(richText, /\.exam-rich-text--inline\s*\{[\s\S]*?max-width:\s*100%/)
  assert.match(richText, /\.exam-rich-text--inline\s*\{[\s\S]*?overflow-x:\s*auto/)
  assert.match(detailStyles, /question-option__content[\s\S]*?\.katex[\s\S]*?font-size:\s*0\.78em/)
})
