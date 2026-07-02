import { get, post } from '@/utils/request'
import type { ApiResponse, ExamClass, ExamClassMember } from '@/types/exam'

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

export function requestJoinExamClass(inviteCode: string) {
  return post('/api/v1/exam/classes/join', { invite_code: inviteCode }) as unknown as Promise<ApiResponse<ExamClassMember>>
}

export function getExamClass(classId: string) {
  return get(`/api/v1/exam/classes/${classId}`) as unknown as Promise<ApiResponse<ExamClass>>
}

export function listExamClassMembers(classId: string) {
  return get(`/api/v1/exam/classes/${classId}/members`) as unknown as Promise<ApiResponse<ExamClassMember[]>>
}

export function approveExamClassMember(classId: string, userId: string) {
  return post(`/api/v1/exam/classes/${classId}/members/${userId}/approve`, {}) as unknown as Promise<ApiResponse<ExamClassMember>>
}

export function rejectExamClassMember(classId: string, userId: string) {
  return post(`/api/v1/exam/classes/${classId}/members/${userId}/reject`, {}) as unknown as Promise<ApiResponse<ExamClassMember>>
}
