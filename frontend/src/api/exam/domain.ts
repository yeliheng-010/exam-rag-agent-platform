import { get } from '@/utils/request'
import type { ApiResponse, ExamDomain, ExamSubject } from '@/types/exam'

export function listExamDomains() {
  return get('/api/v1/exam/domains') as unknown as Promise<ApiResponse<ExamDomain[]>>
}

export function listExamSubjects(domainId: string) {
  return get(`/api/v1/exam/domains/${domainId}/subjects`) as unknown as Promise<ApiResponse<ExamSubject[]>>
}
