import { get } from '@/utils/request'
import type { ApiResponse, ExamClassAnalyticsSummary } from '@/types/exam'

export function getClassAnalytics(classId: string) {
  return get(`/api/v1/exam/classes/${classId}/analytics`) as unknown as Promise<ApiResponse<ExamClassAnalyticsSummary>>
}
