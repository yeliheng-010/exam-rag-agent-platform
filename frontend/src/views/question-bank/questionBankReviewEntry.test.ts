import assert from 'node:assert/strict'
import test from 'node:test'
import { loadQuestionBankReviewEntry } from './questionBankReviewEntry.ts'

const task = (id: string, bankId: string, updatedAt: string) => ({
  id,
  question_bank_id: bankId,
  updated_at: updatedAt,
})

test('selects the latest structuring task for the current question bank', async () => {
  const result = await loadQuestionBankReviewEntry('bank-1', 'space-1', {
    listTasks: async () => ({
      data: [
        task('other-bank', 'bank-2', '2026-07-10T12:00:00Z'),
        task('older', 'bank-1', '2026-07-10T10:00:00Z'),
        task('latest', 'bank-1', '2026-07-10T11:00:00Z'),
      ],
    }),
    listDrafts: async () => ({
      data: {
        stats: { total: 1, pending_review: 1, approved: 0, rejected: 0 },
        drafts: [],
      },
    }),
  })

  assert.equal(result.task?.id, 'latest')
})

test('keeps the review task visible when draft statistics fail to load', async () => {
  const result = await loadQuestionBankReviewEntry('bank-1', 'space-1', {
    listTasks: async () => ({ data: [task('latest', 'bank-1', '2026-07-10T11:00:00Z')] }),
    listDrafts: async () => Promise.reject(new Error('draft endpoint unavailable')),
  })

  assert.equal(result.task?.id, 'latest')
  assert.equal(result.stats, null)
  assert.equal(result.pendingErrorCount, 0)
})

test('counts quality errors from pending review drafts only', async () => {
  const result = await loadQuestionBankReviewEntry('bank-1', 'space-1', {
    listTasks: async () => ({ data: [task('latest', 'bank-1', '2026-07-10T11:00:00Z')] }),
    listDrafts: async () => ({
      data: {
        stats: { total: 3, pending_review: 1, approved: 1, rejected: 1 },
        drafts: [
          { status: 'pending_review', quality_report: { error_count: 2 } },
          { status: 'approved', quality_report: { error_count: 5 } },
          { status: 'rejected', quality_report: { error_count: 7 } },
        ],
      },
    }),
  })

  assert.equal(result.pendingErrorCount, 2)
})
