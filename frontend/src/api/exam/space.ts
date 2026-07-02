import { get, post } from '@/utils/request'
import type { ApiResponse, ExamSpace } from '@/types/exam'

export function listExamSpaces() {
  return get('/api/v1/exam/spaces') as unknown as Promise<ApiResponse<ExamSpace[]>>
}

export function ensurePersonalExamSpace() {
  return post('/api/v1/exam/spaces/personal/ensure', {}) as unknown as Promise<ApiResponse<ExamSpace>>
}
