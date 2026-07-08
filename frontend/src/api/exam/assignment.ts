import { get, post } from '@/utils/request'
import type {
  ApiResponse,
  CreatePracticeAttemptResult,
  ExamAssignmentProgressSummary,
  ExamAssignmentSummary,
} from '@/types/exam'

export interface CreateClassAssignmentPayload {
  group_id: string
  title?: string
  instructions?: string
  due_at?: string
}

export interface ListExamAssignmentsParams {
  limit?: number
}

function buildAssignmentQuery(params?: ListExamAssignmentsParams) {
  const query = new URLSearchParams()
  if (params?.limit) query.set('limit', String(params.limit))
  const qs = query.toString()
  return qs ? `?${qs}` : ''
}

export function listClassAssignments(classId: string, params?: ListExamAssignmentsParams) {
  return get(`/api/v1/exam/classes/${classId}/assignments${buildAssignmentQuery(params)}`) as unknown as Promise<ApiResponse<ExamAssignmentSummary[]>>
}

export function listMyExamAssignments(params?: ListExamAssignmentsParams) {
  return get(`/api/v1/exam/assignments${buildAssignmentQuery(params)}`) as unknown as Promise<ApiResponse<ExamAssignmentSummary[]>>
}

export function createClassAssignment(classId: string, data: CreateClassAssignmentPayload) {
  return post(`/api/v1/exam/classes/${classId}/assignments`, data) as unknown as Promise<ApiResponse<ExamAssignmentSummary>>
}

export function createAssignmentAttempt(assignmentId: string) {
  return post(`/api/v1/exam/assignments/${assignmentId}/attempts`, {}) as unknown as Promise<ApiResponse<CreatePracticeAttemptResult>>
}

export function getClassAssignmentProgress(classId: string, assignmentId: string) {
  return get(`/api/v1/exam/classes/${classId}/assignments/${assignmentId}/progress`) as unknown as Promise<ApiResponse<ExamAssignmentProgressSummary>>
}
