interface ReviewTaskLike {
  id: string
  question_bank_id: string
  updated_at: string
}

interface ReviewDraftLike {
  status: string
  quality_report?: { error_count?: number }
}

interface ReviewTaskListResult<TTask> {
  data?: TTask[]
}

interface ReviewDraftListResult<TStats, TDraft> {
  data: {
    stats: TStats
    drafts?: TDraft[]
  }
}

interface ReviewEntryAPI<TTask, TStats, TDraft> {
  listTasks: (params: { space_id: string }) => Promise<ReviewTaskListResult<TTask>>
  listDrafts: (taskId: string) => Promise<ReviewDraftListResult<TStats, TDraft>>
}

export interface QuestionBankReviewEntry<TTask, TStats> {
  task: TTask | null
  stats: TStats | null
  pendingErrorCount: number
}

const latestTaskForBank = <TTask extends ReviewTaskLike>(tasks: TTask[], bankId: string) => {
  return tasks
    .filter(task => task.question_bank_id === bankId)
    .sort((a, b) => Date.parse(b.updated_at) - Date.parse(a.updated_at))[0] || null
}

const pendingReviewErrorCount = <TDraft extends ReviewDraftLike>(drafts: TDraft[]) => {
  return drafts.reduce(
    (total, draft) => draft.status === 'pending_review'
      ? total + (draft.quality_report?.error_count || 0)
      : total,
    0,
  )
}

export const loadQuestionBankReviewEntry = async <
  TTask extends ReviewTaskLike,
  TStats,
  TDraft extends ReviewDraftLike,
>(bankId: string, spaceId: string, api: ReviewEntryAPI<TTask, TStats, TDraft>): Promise<QuestionBankReviewEntry<TTask, TStats>> => {
  const tasksResult = await api.listTasks({ space_id: spaceId })
  const task = latestTaskForBank(tasksResult.data || [], bankId)
  if (!task) return { task: null, stats: null, pendingErrorCount: 0 }

  try {
    const draftsResult = await api.listDrafts(task.id)
    return {
      task,
      stats: draftsResult.data.stats,
      pendingErrorCount: pendingReviewErrorCount(draftsResult.data.drafts || []),
    }
  } catch {
    return { task, stats: null, pendingErrorCount: 0 }
  }
}
