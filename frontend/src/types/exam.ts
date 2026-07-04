export type ExamSpaceType = 'public' | 'personal' | 'class'
export type ExamClassRole = 'teacher' | 'assistant' | 'student'
export type ExamClassMemberStatus = 'pending' | 'active' | 'removed'
export type ReviewStatus = 'private' | 'pending' | 'approved' | 'rejected'
export type ExamResourceType = 'knowledge_base'
export type ExamMaterialType = 'learning_material' | 'exam_paper' | 'answer_key' | 'explanation'
export type ExamMaterialIngestStatus = 'pending' | 'processing' | 'completed' | 'failed' | 'cancelled' | 'unknown'
export type ExamStructuringTaskStatus = 'pending' | 'ready_for_review' | 'blocked' | 'completed' | 'failed'
export type ExamStructuringStrategy = 'manual_review'
export type ExamTeacherApplicationStatus = 'pending' | 'approved' | 'rejected'

export interface ExamDomain {
  id: string
  code: 'gaokao' | 'ielts' | string
  name: string
  description: string
  status: string
  created_at: string
  updated_at: string
}

export interface ExamSubject {
  id: string
  domain_id: string
  code: string
  name: string
  sort_order: number
  status: string
  created_at?: string
  updated_at?: string
}

export interface ExamSpace {
  id: string
  tenant_id: number
  owner_user_id?: string
  space_type: ExamSpaceType
  name: string
  description: string
  status: string
  created_at: string
  updated_at: string
}

export interface ExamClass {
  id: string
  tenant_id: number
  owner_user_id: string
  space_id: string
  domain_id?: string
  name: string
  description: string
  invite_code?: string
  invite_code_expires_at?: string
  member_limit: number
  status: string
  created_at: string
  updated_at: string
}

export interface ExamClassMember {
  id: string
  class_id: string
  user_id: string
  tenant_id: number
  role: ExamClassRole
  status: ExamClassMemberStatus
  joined_at: string
  created_at: string
  updated_at: string
}

export interface ExamTeacherApplication {
  id: string
  tenant_id: number
  user_id: string
  status: ExamTeacherApplicationStatus
  reason: string
  reviewer_id?: string
  review_note: string
  reviewed_at?: string
  created_at: string
  updated_at: string
  user_email?: string
  username?: string
  reviewer_name?: string
}

export interface QuestionBank {
  id: string
  tenant_id: number
  space_id: string
  domain_id: string
  subject_id?: string
  name: string
  description: string
  source_type: string
  review_status: ReviewStatus
  status: string
  created_by_user_id: string
  created_at: string
  updated_at: string
}

export interface ExamSpaceResource {
  id: string
  tenant_id: number
  space_id: string
  resource_type: ExamResourceType
  resource_id: string
  domain_id: string
  subject_id?: string
  material_type: ExamMaterialType
  review_status: ReviewStatus
  status: string
  created_by_user_id: string
  created_at: string
  updated_at: string
}

export interface ExamMaterial {
  id: string
  tenant_id: number
  space_id: string
  knowledge_base_id: string
  knowledge_id: string
  domain_id: string
  subject_id?: string
  material_type: ExamMaterialType
  title: string
  description: string
  source_year?: number
  source_region: string
  paper_type: string
  ingest_status: ExamMaterialIngestStatus
  review_status: ReviewStatus
  status: string
  created_by_user_id: string
  created_at: string
  updated_at: string
}

export interface ExamStructuringTask {
  id: string
  tenant_id: number
  material_id: string
  space_id: string
  question_bank_id: string
  status: ExamStructuringTaskStatus
  strategy: ExamStructuringStrategy
  source_chunk_count: number
  structured_question_count: number
  review_required: boolean
  error_message: string
  created_by_user_id: string
  created_at: string
  updated_at: string
}

export interface ExamMaterialRegistrationResult {
  material: ExamMaterial
  structuring_task?: ExamStructuringTask
}

export interface ApiResponse<T> {
  success: boolean
  data: T
  message?: string
}
