export type ExamSpaceType = 'public' | 'personal' | 'class'
export type ExamClassRole = 'teacher' | 'assistant' | 'student'
export type ReviewStatus = 'private' | 'pending' | 'approved' | 'rejected'
export type ExamResourceType = 'knowledge_base'
export type ExamMaterialType = 'learning_material' | 'exam_paper' | 'answer_key' | 'explanation'

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

export interface ApiResponse<T> {
  success: boolean
  data: T
  message?: string
}
