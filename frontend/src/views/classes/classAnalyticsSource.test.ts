import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import test from 'node:test'

const classDetail = readFileSync(new URL('./ClassDetail.vue', import.meta.url), 'utf8')
const analyticsApiUrl = new URL('../../api/exam/analytics.ts', import.meta.url)
const analyticsApi = existsSync(analyticsApiUrl) ? readFileSync(analyticsApiUrl, 'utf8') : ''
const examTypes = readFileSync(new URL('../../types/exam.ts', import.meta.url), 'utf8')

test('declares class analytics frontend API', () => {
  assert.equal(existsSync(analyticsApiUrl), true)
  assert.match(analyticsApi, /getClassAnalytics/)
  assert.match(analyticsApi, /exam\/classes\/\$\{classId\}\/analytics/)
})

test('class detail renders real analytics tab instead of placeholder', () => {
  assert.match(classDetail, /value="analytics"/)
  assert.match(classDetail, /getClassAnalytics/)
  assert.match(classDetail, /classAnalytics/)
  assert.match(classDetail, /analyticsLoading/)
  assert.match(classDetail, /学生维度/)
  assert.match(classDetail, /任务维度/)
  assert.doesNotMatch(classDetail, /value:\s*['"]analytics['"][\s\S]{0,180}分析指标将在学习记录接入后生成/)
})

test('exam types include class analytics aggregates', () => {
  assert.match(examTypes, /ExamClassAnalyticsSummary/)
  assert.match(examTypes, /ExamClassAnalyticsMember/)
  assert.match(examTypes, /ExamClassAnalyticsAssignment/)
  assert.match(examTypes, /total_assignment_slots:\s*number/)
  assert.match(examTypes, /average_correct_rate:\s*number/)
})
