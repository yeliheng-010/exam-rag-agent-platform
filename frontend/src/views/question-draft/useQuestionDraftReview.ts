import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  approveQuestionDraft,
  listQuestionDrafts,
  rejectQuestionDraft,
  updateQuestionDraft,
  type ListDraftsResult,
} from '@/api/exam/question-draft'
import type { ExamQuestionDraft, ExamQuestionDraftOption, ExamQuestionDraftStatus } from '@/types/exam'

interface DraftForm {
  question_no: string
  question_type_code: string
  stem: string
  options_json: ExamQuestionDraftOption[]
  answer_json: Record<string, any>
  explanation: string
  difficulty: string
  source_chunk_ids: string[]
}

export function useQuestionDraftReview() {
  const route = useRoute()
  const router = useRouter()
  const loading = ref(false)
  const saving = ref(false)
  const approving = ref(false)
  const rejecting = ref(false)
  const data = ref<ListDraftsResult | null>(null)
  const drafts = ref<ExamQuestionDraft[]>([])
  const selectedDraft = ref<ExamQuestionDraft | null>(null)
  const answerText = ref('{}')
  const sourceChunkText = ref('')
  const form = ref<DraftForm>(emptyForm())

  const headerText = computed(() => {
    if (!data.value) return '校对 LLM 抽取的题目草稿'
    return `任务 ${data.value.task.id}`
  })

  const canEditSelected = computed(() => selectedDraft.value?.status === 'pending_review')

  const loadDrafts = async () => {
    const taskId = String(route.params.taskId || '')
    if (!taskId) return
    loading.value = true
    try {
      const res = await listQuestionDrafts(taskId)
      data.value = res.data
      drafts.value = res.data?.drafts || []
      const current = selectedDraft.value
      const next = drafts.value.find(item => item.id === current?.id) || drafts.value[0] || null
      if (next) {
        selectDraft(next)
      } else {
        selectedDraft.value = null
        form.value = emptyForm()
      }
    } catch (error: any) {
      MessagePlugin.error(error?.message || '草稿加载失败')
    } finally {
      loading.value = false
    }
  }

  const selectDraft = (draft: ExamQuestionDraft) => {
    selectedDraft.value = draft
    form.value = draftToForm(draft)
    answerText.value = JSON.stringify(form.value.answer_json || {}, null, 2)
    sourceChunkText.value = form.value.source_chunk_ids.join('\n')
  }

  const addOption = () => {
    const nextKey = String.fromCharCode(65 + form.value.options_json.length)
    form.value.options_json.push({ key: nextKey, content: '' })
  }

  const removeOption = (index: number) => {
    form.value.options_json.splice(index, 1)
  }

  const saveDraft = async () => {
    if (!selectedDraft.value) return false
    const payload = buildSavePayload(form.value, answerText.value, sourceChunkText.value)
    if (!payload) return false
    saving.value = true
    try {
      const res = await updateQuestionDraft(selectedDraft.value.id, payload)
      MessagePlugin.success('草稿已保存')
      replaceDraft(res.data)
      selectDraft(res.data)
      return true
    } catch (error: any) {
      MessagePlugin.error(error?.message || '保存失败')
      return false
    } finally {
      saving.value = false
    }
  }

  const approveDraft = async () => {
    if (!selectedDraft.value) return
    const saved = await saveDraft()
    if (!saved) return
    approving.value = true
    try {
      const res = await approveQuestionDraft(selectedDraft.value.id)
      MessagePlugin.success('题目已写入正式题库')
      replaceDraft(res.data.draft)
      await loadDrafts()
    } catch (error: any) {
      MessagePlugin.error(error?.message || '确认入库失败')
    } finally {
      approving.value = false
    }
  }

  const rejectDraft = async () => {
    if (!selectedDraft.value) return
    rejecting.value = true
    try {
      const res = await rejectQuestionDraft(selectedDraft.value.id)
      MessagePlugin.success('草稿已驳回')
      replaceDraft(res.data)
      await loadDrafts()
    } catch (error: any) {
      MessagePlugin.error(error?.message || '驳回失败')
    } finally {
      rejecting.value = false
    }
  }

  const replaceDraft = (draft: ExamQuestionDraft) => {
    const index = drafts.value.findIndex(item => item.id === draft.id)
    if (index >= 0) {
      drafts.value.splice(index, 1, draft)
    }
  }

  return {
    router,
    loading,
    saving,
    approving,
    rejecting,
    data,
    drafts,
    selectedDraft,
    answerText,
    sourceChunkText,
    form,
    headerText,
    canEditSelected,
    loadDrafts,
    selectDraft,
    addOption,
    removeOption,
    saveDraft,
    approveDraft,
    rejectDraft,
  }
}

export const draftStatusLabel = (status: ExamQuestionDraftStatus) => {
  const map: Record<ExamQuestionDraftStatus, string> = {
    pending_review: '待校对',
    approved: '已入库',
    rejected: '已驳回',
  }
  return map[status] || status
}

export const draftStatusTheme = (status: ExamQuestionDraftStatus) => {
  if (status === 'approved') return 'success'
  if (status === 'rejected') return 'danger'
  return 'warning'
}

function emptyForm(): DraftForm {
  return {
    question_no: '',
    question_type_code: 'unknown',
    stem: '',
    options_json: [],
    answer_json: {},
    explanation: '',
    difficulty: 'unknown',
    source_chunk_ids: [],
  }
}

function draftToForm(draft: ExamQuestionDraft): DraftForm {
  return {
    question_no: draft.question_no,
    question_type_code: draft.question_type_code || 'unknown',
    stem: draft.stem,
    options_json: (draft.options_json || []).map(item => ({ ...item })),
    answer_json: { ...(draft.answer_json || {}) },
    explanation: draft.explanation || '',
    difficulty: draft.difficulty || 'unknown',
    source_chunk_ids: [...(draft.source_chunk_ids || [])],
  }
}

function buildSavePayload(form: DraftForm, answerText: string, sourceChunkText: string) {
  let answer: Record<string, any>
  try {
    answer = JSON.parse(answerText || '{}')
  } catch {
    MessagePlugin.error('答案 JSON 格式不正确')
    return null
  }
  if (!form.stem.trim()) {
    MessagePlugin.error('题干不能为空')
    return null
  }
  return {
    question_no: form.question_no.trim(),
    question_type_code: form.question_type_code.trim() || 'unknown',
    stem: form.stem.trim(),
    options: form.options_json
      .map(item => ({ key: item.key.trim(), content: item.content.trim() }))
      .filter(item => item.key || item.content),
    answer,
    explanation: form.explanation.trim(),
    difficulty: form.difficulty || 'unknown',
    source_chunk_ids: sourceChunkText
      .split(/\r?\n|,/)
      .map(item => item.trim())
      .filter(Boolean),
  }
}
