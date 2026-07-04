import { get, post } from '@/utils/request'
import type {
  ApiResponse,
  ExamMaterial,
  ExamMaterialRegistrationResult,
  ExamMaterialType,
  ExamStructuringTask,
  ExamStructuringTaskStatus,
  ExamStructuringStrategy,
} from '@/types/exam'

export interface RegisterExamMaterialPayload {
  space_id: string
  knowledge_base_id: string
  knowledge_id: string
  domain_id: string
  subject_id?: string
  material_type?: ExamMaterialType
  title?: string
  description?: string
  source_year?: number
  source_region?: string
  paper_type?: string
  question_bank_id?: string
  create_task?: boolean
}

export interface ListExamMaterialsParams {
  space_id?: string
  domain_id?: string
  subject_id?: string
  material_type?: ExamMaterialType
}

export interface CreateExamStructuringTaskPayload {
  question_bank_id?: string
  strategy?: ExamStructuringStrategy
}

export interface ListExamStructuringTasksParams {
  space_id?: string
  material_id?: string
  status?: ExamStructuringTaskStatus
}

export function registerExamMaterial(data: RegisterExamMaterialPayload) {
  return post('/api/v1/exam/materials', data) as unknown as Promise<ApiResponse<ExamMaterialRegistrationResult>>
}

export function listExamMaterials(params?: ListExamMaterialsParams) {
  const query = new URLSearchParams()
  if (params?.space_id) query.set('space_id', params.space_id)
  if (params?.domain_id) query.set('domain_id', params.domain_id)
  if (params?.subject_id) query.set('subject_id', params.subject_id)
  if (params?.material_type) query.set('material_type', params.material_type)
  const qs = query.toString()
  return get(`/api/v1/exam/materials${qs ? `?${qs}` : ''}`) as unknown as Promise<ApiResponse<ExamMaterial[]>>
}

export function createExamStructuringTask(materialId: string, data: CreateExamStructuringTaskPayload) {
  return post(`/api/v1/exam/materials/${materialId}/structuring-tasks`, data) as unknown as Promise<ApiResponse<ExamStructuringTask>>
}

export function listExamStructuringTasks(params?: ListExamStructuringTasksParams) {
  const query = new URLSearchParams()
  if (params?.space_id) query.set('space_id', params.space_id)
  if (params?.material_id) query.set('material_id', params.material_id)
  if (params?.status) query.set('status', params.status)
  const qs = query.toString()
  return get(`/api/v1/exam/structuring-tasks${qs ? `?${qs}` : ''}`) as unknown as Promise<ApiResponse<ExamStructuringTask[]>>
}
