import { get, patch, post } from '@/utils/request'
import type {
  ApiResponse,
  ExamQuestionGroupDraft,
  ExamQuestionGroupDraftAsset,
  ExamQuestionGroupDraftQuestion,
  ExamQuestionGroupDraftStats,
  ExamStructuringTask,
  QuestionGroupDetail,
} from '@/types/exam'

export interface ListQuestionGroupDraftsResult {
  task: ExamStructuringTask
  drafts: ExamQuestionGroupDraft[]
  stats: ExamQuestionGroupDraftStats
}

export interface UpdateQuestionGroupDraftPayload {
  group_type: string
  title: string
  material_text: string
  material_format: string
  questions: ExamQuestionGroupDraftQuestion[]
  assets: ExamQuestionGroupDraftAsset[]
  source_chunk_ids: string[]
}

export function extractQuestionGroupDrafts(taskId: string, force = false) {
  return post(`/api/v1/exam/structuring-tasks/${taskId}/group-extract`, { force }) as unknown as Promise<ApiResponse<ListQuestionGroupDraftsResult>>
}

export function listQuestionGroupDrafts(taskId: string) {
  return get(`/api/v1/exam/structuring-tasks/${taskId}/group-drafts`) as unknown as Promise<ApiResponse<ListQuestionGroupDraftsResult>>
}

export function updateQuestionGroupDraft(draftId: string, data: UpdateQuestionGroupDraftPayload) {
  return patch(`/api/v1/exam/question-group-drafts/${draftId}`, data) as unknown as Promise<ApiResponse<ExamQuestionGroupDraft>>
}

export function approveQuestionGroupDraft(draftId: string) {
  return post(`/api/v1/exam/question-group-drafts/${draftId}/approve`, {}) as unknown as Promise<ApiResponse<{ draft: ExamQuestionGroupDraft; group: QuestionGroupDetail }>>
}

export function rejectQuestionGroupDraft(draftId: string) {
  return post(`/api/v1/exam/question-group-drafts/${draftId}/reject`, {}) as unknown as Promise<ApiResponse<ExamQuestionGroupDraft>>
}
