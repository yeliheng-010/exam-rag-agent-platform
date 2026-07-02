import { get, post } from '@/utils/request'
import type { ApiResponse, ExamTeacherApplication } from '@/types/exam'

export interface ApplyTeacherPayload {
  reason?: string
}

export interface ReviewTeacherPayload {
  review_note?: string
}

export function getMyTeacherApplication() {
  return get('/api/v1/exam/teacher-applications/me') as unknown as Promise<ApiResponse<ExamTeacherApplication>>
}

export function applyTeacher(data: ApplyTeacherPayload) {
  return post('/api/v1/exam/teacher-applications', data) as unknown as Promise<ApiResponse<ExamTeacherApplication>>
}

export function listTeacherApplications(status?: string) {
  const qs = status ? `?status=${encodeURIComponent(status)}` : ''
  return get(`/api/v1/exam/admin/teacher-applications${qs}`) as unknown as Promise<ApiResponse<ExamTeacherApplication[]>>
}

export function approveTeacherApplication(applicationId: string, data: ReviewTeacherPayload = {}) {
  return post(`/api/v1/exam/admin/teacher-applications/${applicationId}/approve`, data) as unknown as Promise<ApiResponse<ExamTeacherApplication>>
}

export function rejectTeacherApplication(applicationId: string, data: ReviewTeacherPayload = {}) {
  return post(`/api/v1/exam/admin/teacher-applications/${applicationId}/reject`, data) as unknown as Promise<ApiResponse<ExamTeacherApplication>>
}
