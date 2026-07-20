import { get, post, put } from '@/utils/request'
import type {
  ApiResponse,
  ExamEvaluationCenterQuery,
  ExamEvaluationCenterResult,
  ExamEvaluationRunSummary,
  ExamEvaluationSet,
  ExamEvaluationSetVersion,
} from '@/types/exam'

const evaluationQuery = (params: ExamEvaluationCenterQuery) => {
  const query = new URLSearchParams()
  query.set('kind', params.kind)
  if (params.bank_id) query.set('bank_id', params.bank_id)
  if (params.agent_id) query.set('agent_id', params.agent_id)
  if (params.status) query.set('status', params.status)
  if (params.regression) query.set('regression', params.regression)
  query.set('limit', String(params.limit || 50))
  return query.toString()
}

export function getExamEvaluationCenter(params: ExamEvaluationCenterQuery) {
  return get(`/api/v1/exam/evaluation-center?${evaluationQuery(params)}`) as unknown as Promise<ApiResponse<ExamEvaluationCenterResult>>
}

export function setExamEvaluationBaseline(runId: string) {
  return put('/api/v1/exam/evaluation-center/baseline', { run_id: runId }) as unknown as Promise<ApiResponse<ExamEvaluationRunSummary>>
}

export function listExamEvaluationSets(params: ExamEvaluationCenterQuery) {
  return get(`/api/v1/exam/evaluation-sets?${evaluationQuery(params)}`) as unknown as Promise<ApiResponse<ExamEvaluationSet[]>>
}

export function createExamEvaluationSet(data: { run_id: string; name: string; description?: string }) {
  return post('/api/v1/exam/evaluation-sets', data) as unknown as Promise<ApiResponse<ExamEvaluationSet>>
}

export function createExamEvaluationSetVersion(setId: string, runId: string) {
  return post(`/api/v1/exam/evaluation-sets/${setId}/versions`, { run_id: runId }) as unknown as Promise<ApiResponse<ExamEvaluationSetVersion>>
}

export function runExamEvaluationSet(setId: string, version?: number) {
  return post(`/api/v1/exam/evaluation-sets/${setId}/runs`, version ? { version } : {}) as unknown as Promise<ApiResponse<ExamEvaluationRunSummary>>
}
