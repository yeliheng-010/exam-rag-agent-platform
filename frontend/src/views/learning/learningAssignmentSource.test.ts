import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const learningHome = readFileSync(new URL('./LearningHome.vue', import.meta.url), 'utf8')
const assignmentApi = readFileSync(new URL('../../api/exam/assignment.ts', import.meta.url), 'utf8')
const practiceView = readFileSync(new URL('../practice/QuestionGroupPractice.vue', import.meta.url), 'utf8')

test('learning center shows class assignment cards before free practice', () => {
  assert.match(learningHome, /listMyExamAssignments/)
  assert.match(learningHome, /classAssignments/)
  assert.match(learningHome, /班级任务/)
  assert.match(learningHome, /createAssignmentAttempt/)
  assert.match(learningHome, /goAssignmentPractice/)
  assert.match(learningHome, /attempt_id=\$\{attemptId\}/)
  assert.match(practiceView, /route\.query\.attempt_id/)
  assert.match(practiceView, /getPracticeAttempt/)
})

test('assignment API exposes student task attempt creation', () => {
  assert.match(assignmentApi, /createAssignmentAttempt/)
  assert.match(assignmentApi, /ExamAssignmentSummary/)
  assert.match(assignmentApi, /CreatePracticeAttemptResult/)
})
