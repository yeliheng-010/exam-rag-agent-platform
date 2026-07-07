import { get, post } from '@/utils/request'
import type {
  ApiResponse,
  CreatePracticeAttemptResult,
  ExamPracticeAttempt,
  PracticeAnswerResult,
  PracticeAttemptDetail,
  PracticeAttemptSummary,
  QuestionGroupDetail,
  QuestionGroupPracticeSummary,
  WrongQuestionItem,
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

export interface ListPracticeAttemptsParams {
  space_id?: string
  group_id?: string
  limit?: number
}

export interface ListWrongQuestionsParams {
  space_id?: string
  group_id?: string
  limit?: number
}

function buildPracticeQuery(params?: ListPracticeAttemptsParams | ListWrongQuestionsParams) {
  const query = new URLSearchParams()
  if (params?.space_id) query.set('space_id', params.space_id)
  if (params?.group_id) query.set('group_id', params.group_id)
  if (params?.limit) query.set('limit', String(params.limit))
  const qs = query.toString()
  return qs ? `?${qs}` : ''
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

export function listPracticeAttempts(params?: ListPracticeAttemptsParams) {
  return get(`/api/v1/exam/practice/attempts${buildPracticeQuery(params)}`) as unknown as Promise<ApiResponse<PracticeAttemptSummary[]>>
}

export function getPracticeAttempt(attemptId: string) {
  return get(`/api/v1/exam/practice/attempts/${attemptId}`) as unknown as Promise<ApiResponse<PracticeAttemptDetail>>
}

export function listWrongQuestions(params?: ListWrongQuestionsParams) {
  return get(`/api/v1/exam/practice/wrong-questions${buildPracticeQuery(params)}`) as unknown as Promise<ApiResponse<WrongQuestionItem[]>>
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
