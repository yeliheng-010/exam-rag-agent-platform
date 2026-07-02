import { get, post } from '@/utils/request'
import type { ApiResponse, ExamMaterialType, ExamSpaceResource, ExamResourceType } from '@/types/exam'

export interface BindKnowledgeBaseResourcePayload {
  space_id: string
  domain_id: string
  subject_id?: string
  material_type?: ExamMaterialType
}

export interface ListExamResourcesParams {
  resource_type?: ExamResourceType
  space_id?: string
  domain_id?: string
  subject_id?: string
  material_type?: ExamMaterialType
}

export function bindKnowledgeBaseResource(kbId: string, data: BindKnowledgeBaseResourcePayload) {
  return post(`/api/v1/exam/resources/knowledge-bases/${kbId}/bind`, data) as unknown as Promise<ApiResponse<ExamSpaceResource>>
}

export function getKnowledgeBaseExamBinding(kbId: string) {
  return get(`/api/v1/exam/resources/knowledge-bases/${kbId}`) as unknown as Promise<ApiResponse<ExamSpaceResource>>
}

export function listExamResources(params?: ListExamResourcesParams) {
  const query = new URLSearchParams()
  if (params?.resource_type) query.set('resource_type', params.resource_type)
  if (params?.space_id) query.set('space_id', params.space_id)
  if (params?.domain_id) query.set('domain_id', params.domain_id)
  if (params?.subject_id) query.set('subject_id', params.subject_id)
  if (params?.material_type) query.set('material_type', params.material_type)
  const qs = query.toString()
  return get(`/api/v1/exam/resources${qs ? `?${qs}` : ''}`) as unknown as Promise<ApiResponse<ExamSpaceResource[]>>
}
