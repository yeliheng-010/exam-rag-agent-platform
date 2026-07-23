export type ExamSpaceType = 'public' | 'personal' | 'class'
export type ExamClassRole = 'teacher' | 'assistant' | 'student'
export type ExamClassStatus = 'active' | 'archived'
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
export type ExamAssignmentStatus = 'published' | 'withdrawn' | 'archived'
export type ExamAssignmentProgressStatus = 'not_started' | 'in_progress' | 'completed'
export type ExamAssignmentNotificationKind = 'published' | 'republished' | 'withdrawn' | 'reminder'
export type ExamEvaluationKind = 'rag' | 'agent'

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
  status: ExamClassStatus
  created_at: string
  updated_at: string
}

export interface UpdateExamClassPayload {
  name: string
  description: string
  member_limit: number
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
  required_retrieval_phrases?: string[]
}

export interface ExamRAGDiagnosticRankedItem {
  rank: number
  knowledge_base_id: string
  local_rank: number
  chunk_ids: string[]
  contents: string[]
}

export interface ExamRAGDiagnosticResultItem {
  name: string
  query: string
  passed: boolean
  retrieval_passed: boolean
  answer_passed: boolean
  retrieval_score: number
  candidate_retrieval_score?: number
  candidate_retrieval_passed?: boolean
  answer_score: number
  matched_chunk_ids: string[]
  missing_chunk_ids: string[]
  retrieved_chunk_ids: string[]
  retrieved_items?: ExamRAGDiagnosticRankedItem[]
  matched_retrieval_phrases?: string[]
  missing_retrieval_phrases?: string[]
  matched_phrases: string[]
  missing_phrases: string[]
  context_label: string
  source_chunk_ids: string[]
  candidate_chunk_ids: string[]
  candidate_items?: ExamRAGDiagnosticRankedItem[]
  context_source: 'none' | 'structured_question_group' | 'evaluation_anchor_match'
  group_id: string
  association_confidence?: number
  search_traces: SearchTrace[]
  duration_ms: number
  reciprocal_rank: number
  first_relevant_rank?: number
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
  candidate_recall?: number
  candidate_hit_rate?: number
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

export interface ChunkingConfigSnapshot {
	strategy: string
	chunk_size: number
	chunk_overlap: number
	enable_parent_child: boolean
	parent_chunk_size: number
	child_chunk_size: number
	token_limit: number
	languages: string[]
}

export interface ExamRAGChunkingSnapshot {
	knowledge_base_id: string
	config: ChunkingConfigSnapshot
	knowledge_count: number
	text_chunk_count: number
	parent_chunk_count: number
	min_chars: number
	p50_chars: number
	p90_chars: number
	max_chars: number
	tiny_chunk_rate: number
	oversize_rate: number
	parent_coverage: number
	actual_tier_counts: Record<string, number>
	unknown_tier_count: number
}

export type ExamRAGEvaluationRequestSnapshot = Required<RunExamRAGEvaluationPayload> & {
	used_default_cases?: boolean
	chunking_snapshots?: ExamRAGChunkingSnapshot[]
}

export interface ExamRAGEvaluationProgress {
	completed_cases: number
	total_cases: number
}

export interface ExamRAGEvaluationRun {
	id: string
	tenant_id: number
	question_bank_id: string
	evaluation_kind?: ExamEvaluationKind
	agent_id?: string
	is_baseline?: boolean
	evaluation_set_id?: string
	evaluation_set_version?: number
	created_by: string
	status: ExamRAGEvaluationRunStatus
	progress: ExamRAGEvaluationProgress
	request_snapshot: ExamRAGEvaluationRequestSnapshot
	result_snapshot?: ExamRAGDiagnosticResult
	error_message: string
	started_at?: string
	completed_at?: string
	created_at: string
	updated_at: string
}

export interface ExamAgentExpectedToolCall {
  name: string
  arguments?: Record<string, unknown>
}

export interface ExamAgentEvaluationCase {
  name: string
  input: string
  expected_tool_calls?: ExamAgentExpectedToolCall[]
  expected_evidence_phrases?: string[]
  expected_citations?: string[]
  expected_answer_phrases?: string[]
  grounded_phrases?: string[]
}

export interface RunExamAgentEvaluationPayload {
  agent_id: string
  cases: ExamAgentEvaluationCase[]
}

export interface ExamAgentSnapshot {
  id: string
  name: string
  model_id: string
  allowed_tools: string[]
  knowledge_bases: string[]
  config: Record<string, unknown>
}

export interface ExamAgentEvaluationRequestSnapshot {
  agent: ExamAgentSnapshot
  cases: ExamAgentEvaluationCase[]
}

export interface ExamAgentActualToolCall {
  name: string
  arguments?: Record<string, unknown>
  success: boolean
  duration_ms: number
  output_excerpt?: string
  error?: string
}

export interface ExamAgentEvaluationCaseResult {
  name: string
  input: string
  passed: boolean
  tool_sequence_score: number
  tool_arguments_score: number
  evidence_score: number
  citation_score: number
  answer_score: number
  groundedness_score: number
  duration_ms: number
  actual_tool_calls: ExamAgentActualToolCall[]
  final_answer: string
  assertion_failures?: string[]
  error?: string
}

export interface ExamAgentEvaluationSummary {
  total: number
  passed: number
  failed: number
  pass_rate: number
  tool_sequence_rate: number
  tool_arguments_rate: number
  evidence_rate: number
  citation_rate: number
  answer_rate: number
  groundedness_rate: number
  average_duration_ms: number
}

export interface ExamAgentEvaluationResult {
  question_bank_id: string
  agent: ExamAgentSnapshot
  summary: ExamAgentEvaluationSummary
  results: ExamAgentEvaluationCaseResult[]
}

export interface ExamAgentEvaluationRun {
  id: string
  tenant_id: number
  question_bank_id: string
  evaluation_kind: 'agent'
  agent_id: string
  is_baseline?: boolean
  evaluation_set_id?: string
  evaluation_set_version?: number
  created_by: string
  status: ExamRAGEvaluationRunStatus
  progress: ExamRAGEvaluationProgress
  request_snapshot: ExamAgentEvaluationRequestSnapshot
  result_snapshot?: ExamAgentEvaluationResult
  error_message: string
  started_at?: string
  completed_at?: string
  created_at: string
  updated_at: string
}

export type ExamEvaluationComparisonStatus = 'baseline' | 'regressed' | 'stable' | 'uncompared'
export type ExamEvaluationRegressionFilter = 'all' | 'regressed' | 'stable' | 'uncompared'

export interface ExamEvaluationRunSummary {
  id: string
  tenant_id: number
  question_bank_id: string
  evaluation_kind: ExamEvaluationKind
  agent_id?: string
  is_baseline: boolean
  evaluation_set_id?: string
  evaluation_set_version?: number
  created_by: string
  status: ExamRAGEvaluationRunStatus
  progress: ExamRAGEvaluationProgress
  request_snapshot: Record<string, unknown>
  result_snapshot?: Record<string, unknown>
  error_message: string
  started_at?: string
  completed_at?: string
  created_at: string
  updated_at: string
}

export interface ExamEvaluationRegressionReason {
  metric: string
  label: string
  baseline: number
  current: number
  delta: number
}

export interface ExamEvaluationCenterItem {
  run: ExamEvaluationRunSummary
  question_bank_name: string
  agent_name?: string
  evaluation_set_name?: string
  metrics: Record<string, number>
  baseline_run_id?: string
  baseline_metrics?: Record<string, number>
  metric_deltas?: Record<string, number>
  comparison_status: ExamEvaluationComparisonStatus
  regression_reasons: ExamEvaluationRegressionReason[]
}

export interface ExamEvaluationCenterSummary {
  total_runs: number
  completed_runs: number
  baseline_runs: number
  regression_runs: number
  average_pass_rate: number
}

export interface ExamEvaluationAgentOption {
  id: string
  name: string
}

export interface ExamEvaluationCenterResult {
  summary: ExamEvaluationCenterSummary
  items: ExamEvaluationCenterItem[]
  question_banks: QuestionBank[]
  agents: ExamEvaluationAgentOption[]
}

export interface ExamEvaluationSetVersion {
  id: string
  tenant_id: number
  evaluation_set_id: string
  version: number
  source_run_id: string
  definition_snapshot?: Record<string, unknown>
  created_by: string
  created_at: string
}

export interface ExamEvaluationSet {
  id: string
  tenant_id: number
  question_bank_id: string
  evaluation_kind: ExamEvaluationKind
  agent_id?: string
  name: string
  description: string
  current_version: number
  created_by: string
  status: 'active'
  created_at: string
  updated_at: string
  question_bank_name?: string
  agent_name?: string
  versions: ExamEvaluationSetVersion[]
}

export interface ExamEvaluationCenterQuery {
  kind: ExamEvaluationKind
  bank_id?: string
  agent_id?: string
  status?: ExamRAGEvaluationRunStatus | ''
  regression?: ExamEvaluationRegressionFilter
  limit?: number
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
  last_reminded_at?: string
  can_remind: boolean
}

export interface ExamAssignmentProgressSummary {
  assignment: ExamClassAssignment
  total_students: number
  started_count: number
  completed_count: number
  average_correct_rate: number
  members: ExamAssignmentMemberProgress[]
}

export interface ExamAssignmentNotification {
  id: string
  tenant_id: number
  class_id: string
  assignment_id: string
  group_id: string
  recipient_user_id: string
  actor_user_id: string
  kind: ExamAssignmentNotificationKind
  title: string
  content: string
  read_at?: string
  created_at: string
}

export interface ExamAssignmentNotificationItem {
  notification: ExamAssignmentNotification
  last_attempt_id?: string
  assignment_status: ExamAssignmentStatus
  assignment_due_at?: string
  can_start: boolean
}

export interface ExamAssignmentNotificationList {
  items: ExamAssignmentNotificationItem[]
  unread_count: number
}

export interface SendExamAssignmentReminderRequest {
  recipient_user_ids: string[]
}

export interface SendExamAssignmentReminderResult {
  sent_count: number
  completed_skipped_count: number
  cooldown_skipped_count: number
  sent_user_ids: string[]
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

export interface PracticeExplanationContext {
  question_id: string
  reference_answer: string
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
