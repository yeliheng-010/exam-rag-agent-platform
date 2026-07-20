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
  assert.match(assignmentApi, /getClassAssignmentProgress/)
  assert.match(assignmentApi, /updateClassAssignment/)
  assert.match(assignmentApi, /withdrawClassAssignment/)
  assert.match(assignmentApi, /republishClassAssignment/)
  assert.match(assignmentApi, /import \{ get, post, put \}/)
  assert.match(assignmentApi, /exam\/classes\/\$\{classId\}\/assignments/)
  assert.match(assignmentApi, /exam\/assignments\/\$\{assignmentId\}\/attempts/)
  assert.match(assignmentApi, /exam\/classes\/\$\{classId\}\/assignments\/\$\{assignmentId\}\/progress/)
  assert.match(assignmentApi, /exam\/classes\/\$\{classId\}\/assignments\/\$\{assignmentId\}\/withdraw/)
  assert.match(assignmentApi, /exam\/classes\/\$\{classId\}\/assignments\/\$\{assignmentId\}\/republish/)
})

test('class detail replaces homework placeholder with assignment workflow', () => {
  assert.match(classDetail, /value="assignments"/)
  assert.match(classDetail, /listClassAssignments/)
  assert.match(classDetail, /createClassAssignment/)
  assert.match(classDetail, /assignmentVisible/)
  assert.match(classDetail, /submitCreateAssignment/)
  assert.match(classDetail, /getClassAssignmentProgress/)
  assert.match(classDetail, /assignmentProgressVisible/)
  assert.match(classDetail, /查看结果/)
  assert.match(classDetail, /练习结果/)
  assert.match(classDetail, /listPracticeQuestionGroups/)
  assert.match(classDetail, /attempt_id=\$\{attemptId\}/)
  assert.doesNotMatch(classDetail, /value:\s*['"]homework['"][\s\S]{0,160}作业闭环将在练习阶段启用/)
})

test('class detail manages assignment lifecycle and preserves existing attempts', () => {
  assert.match(classDetail, /assignmentStatusLabel/)
  assert.match(classDetail, /canEditAssignment/)
  assert.match(classDetail, /canWithdrawAssignment/)
  assert.match(classDetail, /canRepublishAssignment/)
  assert.match(classDetail, /updateClassAssignment/)
  assert.match(classDetail, /withdrawClassAssignment/)
  assert.match(classDetail, /republishClassAssignment/)
  assert.match(classDetail, /DialogPlugin\.confirm/)
  assert.match(classDetail, /学生入口将隐藏，历史答题不会删除/)
  assert.match(classDetail, /existingAssignmentAttemptID/)
  assert.match(classDetail, /canCreateAssignmentAttempt/)
  assert.match(classDetail, /已截止/)
})

test('class detail supports bulk and individual assignment reminders', () => {
  assert.match(classDetail, /sendAssignmentReminders/)
  assert.match(classDetail, /一键催交/)
  assert.match(classDetail, /row\.can_remind/)
  assert.match(classDetail, /row\.status !== 'completed'/)
  assert.match(classDetail, /name="send"/)
  assert.match(classDetail, /24 小时内已催/)
  assert.match(classDetail, /loadAssignmentProgress/)
  assert.match(classDetail, /completed_skipped_count/)
  assert.match(classDetail, /cooldown_skipped_count/)
})

test('exam types include assignment summary and attempt linkage', () => {
  assert.match(examTypes, /ExamClassAssignment/)
  assert.match(examTypes, /ExamAssignmentSummary/)
  assert.match(examTypes, /ExamAssignmentMemberProgress/)
  assert.match(examTypes, /ExamAssignmentProgressSummary/)
  assert.match(examTypes, /assignment_id\?:\s*string/)
})
