import assert from 'node:assert/strict'
import test from 'node:test'
import {
  hasBlockingDraftQuality,
  normalizeExtractionPercent,
  shouldPollQuestionGroupExtraction,
} from './questionGroupExtractionProgress.ts'

test('polls only while the extraction worker is running', () => {
  assert.equal(shouldPollQuestionGroupExtraction('extracting'), true)
  assert.equal(shouldPollQuestionGroupExtraction('reviewing'), false)
  assert.equal(shouldPollQuestionGroupExtraction('failed'), false)
})
test('normalizes persisted extraction progress for the progress control', () => {
  assert.equal(normalizeExtractionPercent(undefined), 0)
  assert.equal(normalizeExtractionPercent(-1), 0)
  assert.equal(normalizeExtractionPercent(37), 37)
  assert.equal(normalizeExtractionPercent(120), 100)
})

test('blocks approval only when the latest quality report is blocking', () => {
  assert.equal(hasBlockingDraftQuality(undefined), false)
  assert.equal(hasBlockingDraftQuality({ blocking: false }), false)
  assert.equal(hasBlockingDraftQuality({ blocking: true }), true)
})
