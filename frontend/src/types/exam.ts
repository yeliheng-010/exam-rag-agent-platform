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
export type ExamStructuringPhase = 'queued' | 'preflight' | 'extracting' | 'quality_check' | 'completed' | 'failed'
export type ExamQualitySeverity = 'error' | 'warning'
export type QuestionGroupType = 'reading_passage' | 'math_problem' | 'single_question' | string
export type ExamPracticeAttemptStatus = 'in_progress' | 'completed'
export type PracticeAnswerReviewStatus = 'unreviewed' | 'reviewing' | 'mastered'
export type ExamAssignmentStatus = 'published' | 'archived'
export type ExamAssignmentProgressStatus = 'not_started' | 'in_progress' | 'completed'

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
  display_id?: string
  display_name?: string
}

export interface ExamClassAssignment {
  id: string
  tenant_id: number
  class_id: string
  space_id: string
  question_bank_id: string
  group_id: string
  title: string
  instructions: string
  status: ExamAssignmentStatus
  due_at?: string
  created_by_user_id: string
  created_at: string
  updated_at: string
}

export interface ExamClassAnalyticsMember {
  member: ExamClassMember
  assignment_count: number
  started_count: number
  completed_count: number
  completion_rate: number
  average_correct_rate: number
  last_activity_at?: string
}

export interface ExamClassAnalyticsAssignment {
  assignment: ExamClassAssignment
  started_count: number
  completed_count: number
  completion_rate: number
  average_correct_rate: number
}

export interface ExamClassFrequentWrongQuestion {
  question_id: string
  group_id: string
  assignment_id: string
  question_no: string
  stem: string
  answer_count: number
  wrong_count: number
  wrong_rate: number
  affected_student_count: number
}

export interface ExamClassAnalyticsSummary {
  class: ExamClass
  total_students: number
  assignment_count: number
  total_assignment_slots: number
  started_count: number
  completed_count: number
  completion_rate: number
  average_correct_rate: number
  members: ExamClassAnalyticsMember[]
  assignments: ExamClassAnalyticsAssignment[]
  frequent_wrong_questions: ExamClassFrequentWrongQuestion[]
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
  progress: ExamStructuringProgress
  created_by_user_id: string
  created_at: string
  updated_at: string
}

export interface ExamStructuringWarning {
  code: string
  severity: ExamQualitySeverity
  message: string
  recommendation?: string
  reference_count?: number
}

export interface ExamQuestionGroupQualityIssue {
  code: string
  severity: ExamQualitySeverity
  message: string
  question_no?: string
}

export interface ExamQuestionGroupQualityReport {
  blocking: boolean
  error_count: number
  warning_count: number
  issues: ExamQuestionGroupQualityIssue[]
}

export interface ExamStructuringQualitySummary {
  total_drafts: number
  drafts_with_errors: number
  drafts_with_warnings: number
  error_count: number
  warning_count: number
}

export interface ExamStructuringProgress {
  phase?: ExamStructuringPhase
  total_batches: number
  completed_batches: number
  current_batch: number
  failed_batch?: number
  percent: number
  message?: string
  warnings: ExamStructuringWarning[]
  quality_summary: ExamStructuringQualitySummary
  started_at?: string
  finished_at?: string
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
  quality_report: ExamQuestionGroupQualityReport
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

export interface ExamRAGDiagnosticCase {
  name: string
  query: string
  required_phrases: string[]
  expected_chunk_ids: string[]
}

export interface ExamRAGDiagnosticResultItem {
  name: string
  query: string
  passed: boolean
  retrieval_passed: boolean
  answer_passed: boolean
  retrieval_score: number
  answer_score: number
  matched_chunk_ids: string[]
  missing_chunk_ids: string[]
  retrieved_chunk_ids: string[]
  matched_phrases: string[]
  missing_phrases: string[]
  context_label: string
  source_chunk_ids: string[]
	 candidate_chunk_ids: string[]
	 context_source: 'none' | 'structured_question_group'
	 group_id: string
	 search_traces: SearchTrace[]
	 duration_ms: number
	 reciprocal_rank: number
  error: string
}

export interface ExamRAGDiagnosticSummary {
  total: number
  passed: number
  retrieval_passed: number
  answer_passed: number
  hit_rate: number
  retrieval_hit_rate: number
  answer_hit_rate: number
	 recall_at_k: number
	 mean_reciprocal_rank: number
	 ranked_case_count: number
	 structured_resolution_rate: number
	 structured_resolved: number
	 average_duration_ms: number
	 failed_case_count: number
  results: ExamRAGDiagnosticResultItem[]
}

export interface ExamRAGDiagnosticResult {
	question_bank: QuestionBank | null
  knowledge_base_ids: string[]
  used_default_cases: boolean
  summary: ExamRAGDiagnosticSummary
  cases: ExamRAGDiagnosticCase[]
}

export interface SearchTraceCandidate {
	chunk_id: string
	score: number
	rank: number
	vector_rank?: number
	keyword_rank?: number
}

export interface SearchTraceParameters {
	match_count: number
	vector_threshold: number
	keyword_threshold: number
	vector_enabled: boolean
	keyword_enabled: boolean
	rrf_k: number
	rrf_vector_weight: number
	rrf_keyword_weight: number
}

export interface SearchTrace {
	query: string
	knowledge_base_id: string
	knowledge_base_ids: string[]
	parameters: SearchTraceParameters
	embedding_model_id: string
	embedding_dimensions: number
	vector_candidates: SearchTraceCandidate[]
	keyword_candidates: SearchTraceCandidate[]
	fusion_method: 'none' | 'rrf' | 'vector_only' | 'keyword_only'
	fusion_candidates: SearchTraceCandidate[]
	final_chunks: SearchTraceCandidate[]
	duration_ms: number
}

export type ExamRAGEvaluationRunStatus = 'queued' | 'running' | 'completed' | 'failed'

export interface RunExamRAGEvaluationPayload {
	knowledge_base_ids?: string[]
	cases?: ExamRAGDiagnosticCase[]
	match_count?: number
	vector_threshold?: number
	keyword_threshold?: number
}

export interface ExamRAGEvaluationProgress {
	completed_cases: number
	total_cases: number
}

export interface ExamRAGEvaluationRun {
	id: string
	tenant_id: number
	question_bank_id: string
	created_by: string
	status: ExamRAGEvaluationRunStatus
	progress: ExamRAGEvaluationProgress
	request_snapshot: Required<RunExamRAGEvaluationPayload>
	result_snapshot?: ExamRAGDiagnosticResult
	error_message: string
	started_at?: string
	completed_at?: string
	created_at: string
	updated_at: string
}

export interface ExamPracticeAttempt {
  id: string
  tenant_id: number
  user_id: string
  space_id: string
  question_bank_id: string
  group_id: string
  assignment_id?: string
  status: ExamPracticeAttemptStatus
  question_count: number
  answered_count: number
  correct_count: number
  started_at: string
  completed_at?: string
  created_at: string
  updated_at: string
}

export interface ExamAssignmentSummary {
  assignment: ExamClassAssignment
  group?: QuestionGroup
  bank_name: string
  question_count: number
  last_attempt?: ExamPracticeAttempt
}

export interface ExamAssignmentMemberProgress {
  member: ExamClassMember
  attempt?: ExamPracticeAttempt
  status: ExamAssignmentProgressStatus
  correct_rate: number
}

export interface ExamAssignmentProgressSummary {
  assignment: ExamClassAssignment
  total_students: number
  started_count: number
  completed_count: number
  average_correct_rate: number
  members: ExamAssignmentMemberProgress[]
}

export interface ExamPracticeAnswer {
  id: string
  tenant_id: number
  attempt_id: string
  question_id: string
  question_no: string
  answer_text: string
  is_correct: boolean
  correct_answer: string
  question_snapshot: Record<string, any>
  answer_snapshot: Array<Record<string, any>>
  explanation_snapshot: Array<Record<string, any>>
  review_status: PracticeAnswerReviewStatus
  review_note: string
  reviewed_at?: string
  answered_at: string
  created_at: string
  updated_at: string
}

export interface QuestionGroupPracticeSummary {
  group: QuestionGroup
  bank_name: string
  question_count: number
  last_attempt?: ExamPracticeAttempt
  assets?: QuestionGroupAsset[]
}

export type ExamPracticeRecommendationScope = 'class' | 'student'
export type ExamPracticeRecommendationReasonCode =
  | 'targeted_review'
  | 'same_subject'
  | 'same_group_type'
  | 'same_domain'
  | 'low_accuracy_retry'
  | 'supplemental_practice'

export interface ExamPracticeDiagnosisEvidence {
  question_id: string
  group_id: string
  question_no: string
  stem: string
  wrong_rate: number
  affected_student_count: number
  review_status?: PracticeAnswerReviewStatus
}

export interface ExamPracticeRecommendationReason {
  code: ExamPracticeRecommendationReasonCode
  score: number
}

export interface ExamPracticeRecommendation {
  group: QuestionGroupPracticeSummary
  score: number
  reasons: ExamPracticeRecommendationReason[]
  evidence: ExamPracticeDiagnosisEvidence[]
  assigned: boolean
}

export interface ExamPracticeRecommendationResult {
  scope: ExamPracticeRecommendationScope
  class?: ExamClass
  diagnosis: ExamPracticeDiagnosisEvidence[]
  recommendations: ExamPracticeRecommendation[]
  warnings: string[]
}

export interface PracticeAttemptSummary {
  attempt: ExamPracticeAttempt
  group?: QuestionGroup
  bank_name: string
}

export interface PracticeAttemptDetail {
  attempt: ExamPracticeAttempt
  group: QuestionGroupDetail
  answers: ExamPracticeAnswer[]
}

export interface WrongQuestionItem {
  attempt: ExamPracticeAttempt
  answer: ExamPracticeAnswer
  group?: QuestionGroup
  bank_name: string
}

export interface CreatePracticeAttemptResult {
  attempt: ExamPracticeAttempt
  group: QuestionGroupDetail
}

export interface PracticeAnswerResult {
  attempt: ExamPracticeAttempt
  answer: ExamPracticeAnswer
  correct_answers: string[]
  explanations: Array<{ id: string; explanation_text: string; source_type: string }>
  chunk_refs: Array<{ question_id: string; chunk_id: string; ref_type: string; confidence: number }>
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
