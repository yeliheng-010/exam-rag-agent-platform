import { get, post } from '@/utils/request'
import type { ApiResponse, ExamClass } from '@/types/exam'

export interface CreateExamClassPayload {
  name: string
  description?: string
  domain_id?: string
  member_limit?: number
}

export function listExamClasses() {
  return get('/api/v1/exam/classes') as unknown as Promise<ApiResponse<ExamClass[]>>
}

export function createExamClass(data: CreateExamClassPayload) {
  return post('/api/v1/exam/classes', data) as unknown as Promise<ApiResponse<ExamClass>>
}

export function getExamClass(classId: string) {
  return get(`/api/v1/exam/classes/${classId}`) as unknown as Promise<ApiResponse<ExamClass>>
}
