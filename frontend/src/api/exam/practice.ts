import { get, post } from '@/utils/request'
import type {
  ApiResponse,
  CreatePracticeAttemptResult,
  ExamPracticeAttempt,
  PracticeAnswerResult,
  QuestionGroupDetail,
  QuestionGroupPracticeSummary,
} from '@/types/exam'

export interface ListPracticeQuestionGroupsParams {
  space_id?: string
  domain_id?: string
  subject_id?: string
  limit?: number
}

export interface SubmitPracticeAnswerPayload {
  question_id: string
  answer_text: string
}

export function listPracticeQuestionGroups(params?: ListPracticeQuestionGroupsParams) {
  const query = new URLSearchParams()
  if (params?.space_id) query.set('space_id', params.space_id)
  if (params?.domain_id) query.set('domain_id', params.domain_id)
  if (params?.subject_id) query.set('subject_id', params.subject_id)
  if (params?.limit) query.set('limit', String(params.limit))
  const qs = query.toString()
  return get(`/api/v1/exam/practice/question-groups${qs ? `?${qs}` : ''}`) as unknown as Promise<ApiResponse<QuestionGroupPracticeSummary[]>>
}

export function getPracticeQuestionGroup(groupId: string) {
  return get(`/api/v1/exam/practice/question-groups/${groupId}`) as unknown as Promise<ApiResponse<QuestionGroupDetail>>
}

export function createPracticeAttempt(groupId: string) {
  return post(`/api/v1/exam/practice/question-groups/${groupId}/attempts`, {}) as unknown as Promise<ApiResponse<CreatePracticeAttemptResult>>
}

export function submitPracticeAnswer(attemptId: string, data: SubmitPracticeAnswerPayload) {
  return post(`/api/v1/exam/practice/attempts/${attemptId}/answers`, data) as unknown as Promise<ApiResponse<PracticeAnswerResult>>
}

export function completePracticeAttempt(attemptId: string) {
  return post(`/api/v1/exam/practice/attempts/${attemptId}/complete`, {}) as unknown as Promise<ApiResponse<ExamPracticeAttempt>>
}
