import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  approveQuestionGroupDraft,
  extractQuestionGroupDrafts,
  listQuestionGroupDrafts,
  rejectQuestionGroupDraft,
  updateQuestionGroupDraft,
  type ListQuestionGroupDraftsResult,
} from '@/api/exam/question-group-draft'
import type {
  ExamQuestionGroupDraft,
  ExamQuestionGroupDraftAsset,
} from '@/types/exam'
import {
  draftToForm,
  emptyForm,
  emptyQuestion,
  groupDraftStatusLabel,
  groupDraftStatusTheme,
  normalizeQuestion,
  splitChunkText,
} from './questionGroupDraftReviewUtils'
import type { GroupDraftForm } from './questionGroupDraftReviewUtils'

export { groupDraftStatusLabel, groupDraftStatusTheme }

export function useQuestionGroupDraftReview() {
  const route = useRoute()
  const router = useRouter()
  const loading = ref(false)
  const extracting = ref(false)
  const saving = ref(false)
  const approving = ref(false)
  const rejecting = ref(false)
  const data = ref<ListQuestionGroupDraftsResult | null>(null)
  const drafts = ref<ExamQuestionGroupDraft[]>([])
  const selectedDraft = ref<ExamQuestionGroupDraft | null>(null)
  const selectedQuestionIndex = ref(0)
  const form = ref<GroupDraftForm>(emptyForm())
  const answerText = ref('{}')
  const questionChunkText = ref('')
  const groupChunkText = ref('')
  const assetText = ref('[]')

  const headerText = computed(() => {
    if (!data.value) return '校对题组草稿'
    return `任务 ${data.value.task.id}`
  })
  const selectedQuestion = computed(() => form.value.questions[selectedQuestionIndex.value] || null)
  const canEditSelected = computed(() => selectedDraft.value?.status === 'pending_review')

  const loadDrafts = async () => {
    const taskId = String(route.params.taskId || '')
    if (!taskId) return
    loading.value = true
    try {
      const res = await listQuestionGroupDrafts(taskId)
      applyResult(res.data)
    } catch (error: any) {
      MessagePlugin.error(error?.message || '题组草稿加载失败')
    } finally {
      loading.value = false
    }
  }

  const extractDrafts = async (force = false) => {
    const taskId = String(route.params.taskId || '')
    if (!taskId) return
    extracting.value = true
    try {
      const res = await extractQuestionGroupDrafts(taskId, force)
      MessagePlugin.success('题组抽取完成')
      applyResult(res.data)
    } catch (error: any) {
      MessagePlugin.error(error?.message || '题组抽取失败')
    } finally {
      extracting.value = false
    }
  }

  const selectDraft = (draft: ExamQuestionGroupDraft) => {
    selectedDraft.value = draft
    form.value = draftToForm(draft)
    selectedQuestionIndex.value = 0
    groupChunkText.value = form.value.source_chunk_ids.join('\n')
    assetText.value = JSON.stringify(form.value.assets || [], null, 2)
    hydrateQuestionText()
  }

  const selectQuestion = (index: number) => {
    flushQuestionText()
    selectedQuestionIndex.value = index
    hydrateQuestionText()
  }

  const addQuestion = () => {
    flushQuestionText()
    form.value.questions.push(emptyQuestion(form.value.questions.length + 1))
    selectQuestion(form.value.questions.length - 1)
  }

  const removeQuestion = (index: number) => {
    if (form.value.questions.length <= 1) {
      MessagePlugin.warning('至少保留一道小题')
      return
    }
    form.value.questions.splice(index, 1)
    selectedQuestionIndex.value = Math.min(selectedQuestionIndex.value, form.value.questions.length - 1)
    hydrateQuestionText()
  }

  const addOption = () => {
    const question = selectedQuestion.value
    if (!question) return
    const nextKey = String.fromCharCode(65 + question.options.length)
    question.options.push({ key: nextKey, content: '' })
  }

  const removeOption = (index: number) => {
    selectedQuestion.value?.options.splice(index, 1)
  }

  const saveDraft = async () => {
    if (!selectedDraft.value) return false
    const payload = buildSavePayload()
    if (!payload) return false
    saving.value = true
    try {
      const res = await updateQuestionGroupDraft(selectedDraft.value.id, payload)
      MessagePlugin.success('题组草稿已保存')
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
      const res = await approveQuestionGroupDraft(selectedDraft.value.id)
      MessagePlugin.success('题组已写入正式题库')
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
      const res = await rejectQuestionGroupDraft(selectedDraft.value.id)
      MessagePlugin.success('题组草稿已驳回')
      replaceDraft(res.data)
      await loadDrafts()
    } catch (error: any) {
      MessagePlugin.error(error?.message || '驳回失败')
    } finally {
      rejecting.value = false
    }
  }

  const applyResult = (result: ListQuestionGroupDraftsResult) => {
    data.value = result
    drafts.value = result?.drafts || []
    const current = selectedDraft.value
    const next = drafts.value.find(item => item.id === current?.id) || drafts.value[0] || null
    if (next) selectDraft(next)
    else {
      selectedDraft.value = null
      form.value = emptyForm()
    }
  }

  const replaceDraft = (draft: ExamQuestionGroupDraft) => {
    const index = drafts.value.findIndex(item => item.id === draft.id)
    if (index >= 0) drafts.value.splice(index, 1, draft)
  }

  const buildSavePayload = () => {
    flushQuestionText()
    let assets: ExamQuestionGroupDraftAsset[]
    try {
      assets = JSON.parse(assetText.value || '[]')
    } catch {
      MessagePlugin.error('资源 JSON 格式不正确')
      return null
    }
    if (!form.value.group_type.trim()) {
      MessagePlugin.error('题组类型不能为空')
      return null
    }
    const questions = form.value.questions.map(normalizeQuestion).filter((item) => item.stem)
    if (!questions.length) {
      MessagePlugin.error('至少需要一道有效小题')
      return null
    }
    return {
      group_type: form.value.group_type.trim(),
      title: form.value.title.trim(),
      material_text: form.value.material_text.trim(),
      material_format: form.value.material_format || 'plain_text',
      questions,
      assets,
      source_chunk_ids: splitChunkText(groupChunkText.value),
    }
  }

  const hydrateQuestionText = () => {
    const question = selectedQuestion.value
    answerText.value = JSON.stringify(question?.answer || {}, null, 2)
    questionChunkText.value = (question?.source_chunk_ids || []).join('\n')
  }

  const flushQuestionText = () => {
    const question = selectedQuestion.value
    if (!question) return
    try {
      question.answer = JSON.parse(answerText.value || '{}')
    } catch {
      question.answer = {}
    }
    question.source_chunk_ids = splitChunkText(questionChunkText.value)
  }

  return {
    router,
    loading,
    extracting,
    saving,
    approving,
    rejecting,
    data,
    drafts,
    selectedDraft,
    selectedQuestionIndex,
    selectedQuestion,
    form,
    answerText,
    questionChunkText,
    groupChunkText,
    assetText,
    headerText,
    canEditSelected,
    loadDrafts,
    extractDrafts,
    selectDraft,
    selectQuestion,
    addQuestion,
    removeQuestion,
    addOption,
    removeOption,
    saveDraft,
    approveDraft,
    rejectDraft,
  }
}
