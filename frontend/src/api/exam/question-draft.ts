import { get, patch, post } from '@/utils/request'
import type {
  ApiResponse,
  ExamQuestionDraft,
  ExamQuestionDraftOption,
  ExamQuestionDraftStats,
  ExamStructuringTask,
  QuestionDetail,
} from '@/types/exam'

export interface ListDraftsResult {
  task: ExamStructuringTask
  drafts: ExamQuestionDraft[]
  stats: ExamQuestionDraftStats
}

export function extractQuestionDrafts(taskId: string, force = false) {
  return post(`/api/v1/exam/structuring-tasks/${taskId}/extract`, { force }) as unknown as Promise<ApiResponse<ListDraftsResult>>
}

export function listQuestionDrafts(taskId: string) {
  return get(`/api/v1/exam/structuring-tasks/${taskId}/drafts`) as unknown as Promise<ApiResponse<ListDraftsResult>>
}

export interface UpdateQuestionDraftPayload {
  question_no: string
  question_type_code: string
  stem: string
  options: ExamQuestionDraftOption[]
  answer: Record<string, any>
  explanation: string
  difficulty: string
  source_chunk_ids: string[]
}

export function updateQuestionDraft(draftId: string, data: UpdateQuestionDraftPayload) {
  return patch(`/api/v1/exam/question-drafts/${draftId}`, data) as unknown as Promise<ApiResponse<ExamQuestionDraft>>
}

export function approveQuestionDraft(draftId: string) {
  return post(`/api/v1/exam/question-drafts/${draftId}/approve`, {}) as unknown as Promise<ApiResponse<{ draft: ExamQuestionDraft; question: QuestionDetail }>>
}

export function rejectQuestionDraft(draftId: string) {
  return post(`/api/v1/exam/question-drafts/${draftId}/reject`, {}) as unknown as Promise<ApiResponse<ExamQuestionDraft>>
}
