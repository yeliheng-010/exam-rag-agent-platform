import { get, post } from '@/utils/request'
import type { ApiResponse, QuestionBank } from '@/types/exam'

export interface CreateQuestionBankPayload {
  space_id: string
  domain_id: string
  subject_id?: string
  name: string
  description?: string
}

export function listQuestionBanks(params?: { space_id?: string }) {
  const query = new URLSearchParams()
  if (params?.space_id) {
    query.set('space_id', params.space_id)
  }
  const qs = query.toString()
  return get(`/api/v1/exam/question-banks${qs ? `?${qs}` : ''}`) as unknown as Promise<ApiResponse<QuestionBank[]>>
}

export function createQuestionBank(data: CreateQuestionBankPayload) {
  return post('/api/v1/exam/question-banks', data) as unknown as Promise<ApiResponse<QuestionBank>>
}

export function getQuestionBank(bankId: string) {
  return get(`/api/v1/exam/question-banks/${bankId}`) as unknown as Promise<ApiResponse<QuestionBank>>
}
