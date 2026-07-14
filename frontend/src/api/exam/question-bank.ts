import { get, post } from '@/utils/request'
import type { ApiResponse, ExamRAGDiagnosticCase, ExamRAGDiagnosticResult, QuestionBank, QuestionDetail } from '@/types/exam'

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

export function listQuestionDetails(bankId: string) {
  return get(`/api/v1/exam/question-banks/${bankId}/questions`) as unknown as Promise<ApiResponse<QuestionDetail[]>>
}

export function runQuestionBankRAGDiagnostic(bankId: string, data?: { knowledge_base_ids?: string[]; cases?: ExamRAGDiagnosticCase[] }) {
  return post(`/api/v1/exam/question-banks/${bankId}/rag-diagnostics`, data || {}) as unknown as Promise<ApiResponse<ExamRAGDiagnosticResult>>
}

export { listQuestionGroupDetails } from './question-group'
