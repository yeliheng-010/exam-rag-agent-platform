import { get } from '@/utils/request'
import type { ApiResponse, ExamPracticeRecommendationResult } from '@/types/exam'

export interface PracticeRecommendationParams {
  space_id?: string
  limit?: number
  include_assigned?: boolean
}

function buildRecommendationQuery(params?: PracticeRecommendationParams) {
  const query = new URLSearchParams()
  if (params?.space_id) query.set('space_id', params.space_id)
  if (params?.limit) query.set('limit', String(params.limit))
  if (params?.include_assigned) query.set('include_assigned', 'true')
  const qs = query.toString()
  return qs ? `?${qs}` : ''
}

export function getClassPracticeRecommendations(classId: string, params?: PracticeRecommendationParams) {
  return get(`/api/v1/exam/classes/${classId}/practice-recommendations${buildRecommendationQuery(params)}`) as unknown as Promise<ApiResponse<ExamPracticeRecommendationResult>>
}

export function getStudentPracticeRecommendations(params?: PracticeRecommendationParams) {
  return get(`/api/v1/exam/practice/recommendations${buildRecommendationQuery(params)}`) as unknown as Promise<ApiResponse<ExamPracticeRecommendationResult>>
}
