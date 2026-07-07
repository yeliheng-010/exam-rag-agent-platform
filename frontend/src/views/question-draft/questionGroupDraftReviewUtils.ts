import type {
  ExamQuestionDraftOption,
  ExamQuestionGroupDraft,
  ExamQuestionGroupDraftAsset,
  ExamQuestionGroupDraftQuestion,
  ExamQuestionGroupDraftStatus,
} from '@/types/exam'

export interface GroupDraftForm {
  group_type: string
  title: string
  material_text: string
  material_format: string
  questions: ExamQuestionGroupDraftQuestion[]
  assets: ExamQuestionGroupDraftAsset[]
  source_chunk_ids: string[]
}

export const groupDraftStatusLabel = (status: ExamQuestionGroupDraftStatus) => {
  const map: Record<ExamQuestionGroupDraftStatus, string> = {
    pending_review: '待校对',
    approved: '已入库',
    rejected: '已驳回',
  }
  return map[status] || status
}

export const groupDraftStatusTheme = (status: ExamQuestionGroupDraftStatus) => {
  if (status === 'approved') return 'success'
  if (status === 'rejected') return 'danger'
  return 'warning'
}

export function emptyForm(): GroupDraftForm {
  return { group_type: 'single_question', title: '', material_text: '', material_format: 'plain_text', questions: [], assets: [], source_chunk_ids: [] }
}

export function draftToForm(draft: ExamQuestionGroupDraft): GroupDraftForm {
  return {
    group_type: draft.group_type || 'single_question',
    title: draft.title || '',
    material_text: draft.material_text || '',
    material_format: draft.material_format || 'plain_text',
    questions: (draft.questions_json || []).map(item => ({
      ...item,
      options: [...(item.options || [])],
      answer: { ...(item.answer || {}) },
      source_chunk_ids: [...(item.source_chunk_ids || [])],
    })),
    assets: (draft.assets_json || []).map(item => ({ ...item })),
    source_chunk_ids: [...(draft.source_chunk_ids || [])],
  }
}

export function emptyQuestion(order: number): ExamQuestionGroupDraftQuestion {
  return { question_no: '', question_type_code: 'unknown', stem: '', options: [], answer: {}, explanation: '', difficulty: 'unknown', confidence: 0, order_in_group: order, source_chunk_ids: [] }
}

export function normalizeQuestion(question: ExamQuestionGroupDraftQuestion): ExamQuestionGroupDraftQuestion {
  return {
    ...question,
    question_no: question.question_no.trim(),
    question_type_code: question.question_type_code.trim() || 'unknown',
    stem: question.stem.trim(),
    options: normalizeOptions(question.options || []),
    explanation: question.explanation.trim(),
    difficulty: question.difficulty || 'unknown',
    order_in_group: question.order_in_group || 1,
    source_chunk_ids: question.source_chunk_ids || [],
  }
}

export function splitChunkText(value: string) {
  return value.split(/\r?\n|,/).map(item => item.trim()).filter(Boolean)
}

function normalizeOptions(options: ExamQuestionDraftOption[]) {
  return options.map(item => ({ key: item.key.trim(), content: item.content.trim() })).filter(item => item.key || item.content)
}
