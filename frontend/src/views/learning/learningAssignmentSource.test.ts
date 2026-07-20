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

test('student assignment entry continues existing attempts and blocks expired starts', () => {
  assert.match(learningHome, /existingAssignmentAttemptID/)
  assert.match(learningHome, /canCreateAssignmentAttempt/)
  assert.match(learningHome, /assignmentLifecycleState/)
  assert.match(learningHome, /已截止/)
  assert.match(
    learningHome,
    /existingAttemptId[\s\S]{0,500}attempt_id=\$\{existingAttemptId\}[\s\S]{0,300}return[\s\S]{0,300}canCreateAssignmentAttempt[\s\S]{0,300}createAssignmentAttempt/,
  )
})

test('learning center locates a notification assignment without starting it', () => {
  assert.match(learningHome, /useRoute/)
  assert.match(learningHome, /route\.query\.assignment_id/)
  assert.match(learningHome, /assignment-card--highlighted/)
  assert.match(learningHome, /scrollIntoView\(\{\s*behavior:\s*'smooth',\s*block:\s*'center'/)
  assert.match(learningHome, /setTimeout/)
  assert.match(learningHome, /clearTimeout/)
  assert.match(learningHome, /onUnmounted/)
  assert.match(learningHome, /watch\(\(\) => route\.query\.assignment_id/)
})
