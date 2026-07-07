export type ExamSpaceType = 'public' | 'personal' | 'class'
export type ExamClassRole = 'teacher' | 'assistant' | 'student'
export type ExamClassMemberStatus = 'pending' | 'active' | 'removed'
export type ReviewStatus = 'private' | 'pending' | 'approved' | 'rejected'
export type ExamResourceType = 'knowledge_base'
export type ExamMaterialType = 'learning_material' | 'exam_paper' | 'answer_key' | 'explanation'
export type ExamMaterialIngestStatus = 'pending' | 'processing' | 'completed' | 'failed' | 'cancelled' | 'unknown'
export type ExamStructuringTaskStatus = 'pending' | 'ready_for_review' | 'blocked' | 'extracting' | 'reviewing' | 'completed' | 'failed'
export type ExamStructuringStrategy = 'manual_review'
export type ExamTeacherApplicationStatus = 'pending' | 'approved' | 'rejected'
export type ExamQuestionDraftStatus = 'pending_review' | 'approved' | 'rejected'
export type ExamQuestionGroupDraftStatus = 'pending_review' | 'approved' | 'rejected'
export type QuestionGroupType = 'reading_passage' | 'math_problem' | 'single_question' | string

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

export interface ExamQuestionDraftOption {
  key: string
  content: string
}

export interface ExamQuestionDraft {
  id: string
  tenant_id: number
  space_id: string
  task_id: string
  material_id: string
  question_bank_id: string
  domain_id: string
  subject_id?: string
  source_chunk_ids: string[]
  question_no: string
  question_type_code: string
  stem: string
  options_json: ExamQuestionDraftOption[]
  answer_json: Record<string, any>
  explanation: string
  difficulty: string
  confidence: number
  status: ExamQuestionDraftStatus
  raw_model_output?: string
  error_message: string
  approved_question_id: string
  reviewed_by_user_id?: string
  reviewed_at?: string
  created_at: string
  updated_at: string
}

export interface ExamQuestionGroupDraftQuestion {
  question_no: string
  question_type_code: string
  stem: string
  options: ExamQuestionDraftOption[]
  answer: Record<string, any>
  explanation: string
  evidence?: Array<Record<string, any>>
  metadata?: Record<string, any>
  difficulty: string
  confidence: number
  order_in_group: number
  source_chunk_ids?: string[]
}

export interface ExamQuestionGroupDraftAsset {
  asset_type: string
  storage_uri: string
  alt_text: string
  source_chunk_id: string
  bbox?: Record<string, any>
  metadata?: Record<string, any>
  sort_order?: number
}

export interface ExamQuestionGroupDraft {
  id: string
  tenant_id: number
  space_id: string
  task_id: string
  material_id: string
  question_bank_id: string
  domain_id: string
  subject_id?: string
  group_type: QuestionGroupType
  title: string
  material_text: string
  material_format: string
  questions_json: ExamQuestionGroupDraftQuestion[]
  assets_json: ExamQuestionGroupDraftAsset[]
  source_chunk_ids: string[]
  strategy_code: string
  confidence: number
  status: ExamQuestionGroupDraftStatus
  raw_model_output?: string
  error_message: string
  approved_group_id: string
  reviewed_by_user_id?: string
  reviewed_at?: string
  created_at: string
  updated_at: string
}

export interface ExamQuestionDraftStats {
  total: number
  pending_review: number
  approved: number
  rejected: number
}

export interface ExamQuestionGroupDraftStats {
  total: number
  pending_review: number
  approved: number
  rejected: number
}

export interface QuestionGroupAsset {
  id: string
  tenant_id: number
  group_id: string
  asset_type: string
  storage_uri: string
  alt_text: string
  source_chunk_id: string
  bbox: Record<string, any>
  metadata: Record<string, any>
  sort_order: number
  created_at: string
}

export interface QuestionGroup {
  id: string
  tenant_id: number
  space_id: string
  question_bank_id: string
  domain_id: string
  subject_id?: string
  group_type: QuestionGroupType
  title: string
  material_text: string
  material_format: string
  asset_refs?: Array<Record<string, any>>
  source_chunk_ids: string[]
  source_year?: number
  source_region: string
  paper_type: string
  sort_order: number
  review_status: ReviewStatus
  status: string
  created_by_user_id: string
  created_at: string
  updated_at: string
}

export interface QuestionDetail {
  question: {
    id: string
    question_bank_id: string
    domain_id: string
    subject_id?: string
    group_id?: string
    question_no?: string
    order_in_group?: number
    question_metadata?: Record<string, any>
    stem: string
    difficulty: string
    review_status: ReviewStatus
    status: string
    created_at?: string
  }
  options: Array<{ id: string; option_key: string; content: string; sort_order: number }>
  answers: Array<{ id: string; answer_text: string; is_correct: boolean }>
  explanations: Array<{ id: string; explanation_text: string; source_type: string }>
  chunk_refs: Array<{ question_id: string; chunk_id: string; ref_type: string; confidence: number }>
}

export interface QuestionGroupDetail {
  group: QuestionGroup
  assets: QuestionGroupAsset[]
  questions: QuestionDetail[]
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
