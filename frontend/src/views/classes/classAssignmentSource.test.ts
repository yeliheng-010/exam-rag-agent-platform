import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import test from 'node:test'

const classDetail = readFileSync(new URL('./ClassDetail.vue', import.meta.url), 'utf8')
const assignmentApiUrl = new URL('../../api/exam/assignment.ts', import.meta.url)
const assignmentApi = existsSync(assignmentApiUrl) ? readFileSync(assignmentApiUrl, 'utf8') : ''
const examTypes = readFileSync(new URL('../../types/exam.ts', import.meta.url), 'utf8')

test('declares class assignment frontend API', () => {
  assert.equal(existsSync(assignmentApiUrl), true)
  assert.match(assignmentApi, /listClassAssignments/)
  assert.match(assignmentApi, /listMyExamAssignments/)
  assert.match(assignmentApi, /createClassAssignment/)
  assert.match(assignmentApi, /createAssignmentAttempt/)
  assert.match(assignmentApi, /exam\/classes\/\$\{classId\}\/assignments/)
  assert.match(assignmentApi, /exam\/assignments\/\$\{assignmentId\}\/attempts/)
})

test('class detail replaces homework placeholder with assignment workflow', () => {
  assert.match(classDetail, /value="assignments"/)
  assert.match(classDetail, /listClassAssignments/)
  assert.match(classDetail, /createClassAssignment/)
  assert.match(classDetail, /assignmentVisible/)
  assert.match(classDetail, /submitCreateAssignment/)
  assert.match(classDetail, /listPracticeQuestionGroups/)
  assert.match(classDetail, /attempt_id=\$\{attemptId\}/)
  assert.doesNotMatch(classDetail, /value:\s*['"]homework['"][\s\S]{0,160}作业闭环将在练习阶段启用/)
})

test('exam types include assignment summary and attempt linkage', () => {
  assert.match(examTypes, /ExamClassAssignment/)
  assert.match(examTypes, /ExamAssignmentSummary/)
  assert.match(examTypes, /assignment_id\?:\s*string/)
})
