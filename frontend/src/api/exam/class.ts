import { get, post, put } from '@/utils/request'
import type { ApiResponse, ExamClass, ExamClassMember, UpdateExamClassPayload } from '@/types/exam'

export interface CreateExamClassPayload {
  name: string
  description?: string
  domain_id?: string
  member_limit?: number
}

export interface ListExamClassesOptions {
  includeArchived?: boolean
}

export function listExamClasses(options: ListExamClassesOptions = {}) {
  const config = options.includeArchived ? { params: { include_archived: true } } : undefined
  return get('/api/v1/exam/classes', config) as unknown as Promise<ApiResponse<ExamClass[]>>
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

export function updateExamClass(classId: string, data: UpdateExamClassPayload) {
  return put(`/api/v1/exam/classes/${classId}`, data) as unknown as Promise<ApiResponse<ExamClass>>
}

export function archiveExamClass(classId: string) {
  return post(`/api/v1/exam/classes/${classId}/archive`, {}) as unknown as Promise<ApiResponse<ExamClass>>
}

export function restoreExamClass(classId: string) {
  return post(`/api/v1/exam/classes/${classId}/restore`, {}) as unknown as Promise<ApiResponse<ExamClass>>
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
