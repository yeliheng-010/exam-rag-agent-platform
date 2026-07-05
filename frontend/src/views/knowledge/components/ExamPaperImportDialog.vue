<template>
  <t-dialog
    v-model:visible="dialogVisible"
    header="导入为试卷"
    width="560px"
    :confirm-btn="{ content: '开始抽题', loading: importing }"
    :cancel-btn="{ content: '取消' }"
    :close-on-overlay-click="!importing"
    @confirm="submitImport"
  >
    <t-loading :loading="loading">
      <div class="exam-import-body">
        <div class="doc-summary">
          <t-icon name="file" />
          <div>
            <strong>{{ documentTitle }}</strong>
            <span>{{ parseStatusLabel }}</span>
          </div>
        </div>

        <t-form ref="formRef" :data="form" :rules="rules" label-align="top">
          <t-form-item label="所属空间" name="space_id">
            <t-select v-model="form.space_id" placeholder="选择班级或个人空间">
              <t-option v-for="space in spaces" :key="space.id" :value="space.id" :label="spaceLabel(space)" />
            </t-select>
          </t-form-item>

          <div class="form-grid">
            <t-form-item label="考试域" name="domain_id">
              <t-select v-model="form.domain_id" placeholder="选择考试域">
                <t-option v-for="domain in domains" :key="domain.id" :value="domain.id" :label="domain.name" />
              </t-select>
            </t-form-item>
            <t-form-item label="科目 / 模块" name="subject_id">
              <t-select v-model="form.subject_id" clearable placeholder="可选">
                <t-option v-for="subject in currentSubjects" :key="subject.id" :value="subject.id" :label="subject.name" />
              </t-select>
            </t-form-item>
          </div>

          <t-form-item label="目标题库" name="question_bank_id">
            <t-select
              v-model="form.question_bank_id"
              clearable
              placeholder="不选择则自动创建题库"
              @change="markQuestionBankChoiceManual"
            >
              <t-option v-for="bank in matchingBanks" :key="bank.id" :value="bank.id" :label="bank.name" />
            </t-select>
          </t-form-item>

          <t-alert theme="info" :message="selectedBankNotice" />

          <t-form-item label="试卷标题" name="title">
            <t-input v-model="form.title" :maxlength="255" />
          </t-form-item>

          <div class="form-grid">
            <t-form-item label="年份">
              <t-input-number v-model="form.source_year" :min="1900" :max="2100" theme="column" />
            </t-form-item>
            <t-form-item label="地区 / 来源">
              <t-input v-model="form.source_region" placeholder="例如：全国Ⅰ卷" :maxlength="128" />
            </t-form-item>
          </div>

          <t-form-item label="试卷类型">
            <t-input v-model="form.paper_type" placeholder="例如：高考英语真题" :maxlength="128" />
          </t-form-item>
        </t-form>
      </div>
    </t-loading>
  </t-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import type { FormInstanceFunctions, FormRule } from 'tdesign-vue-next'
import { listExamDomains, listExamSubjects } from '@/api/exam/domain'
import { registerExamMaterial } from '@/api/exam/material'
import { extractQuestionDrafts } from '@/api/exam/question-draft'
import { listQuestionBanks } from '@/api/exam/question-bank'
import { ensurePersonalExamSpace, listExamSpaces } from '@/api/exam/space'
import type { ExamDomain, ExamSpace, ExamSubject, QuestionBank } from '@/types/exam'

interface KnowledgeForExamImport {
  id: string
  knowledge_base_id?: string
  file_name?: string
  title?: string
  parse_status?: string
}

const props = defineProps<{
  visible: boolean
  kbId: string
  knowledge: KnowledgeForExamImport | null
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'imported', taskId: string): void
}>()

const router = useRouter()
const formRef = ref<FormInstanceFunctions>()
const loading = ref(false)
const importing = ref(false)
const questionBankChoiceManual = ref(false)
const spaces = ref<ExamSpace[]>([])
const domains = ref<ExamDomain[]>([])
const subjectsByDomain = ref<Record<string, ExamSubject[]>>({})
const questionBanks = ref<QuestionBank[]>([])

const form = ref({
  space_id: '',
  domain_id: '',
  subject_id: undefined as string | undefined,
  question_bank_id: undefined as string | undefined,
  title: '',
  source_year: undefined as number | undefined,
  source_region: '',
  paper_type: '',
})

const rules: Record<string, FormRule[]> = {
  space_id: [{ required: true, message: '请选择所属空间', type: 'error' }],
  domain_id: [{ required: true, message: '请选择考试域', type: 'error' }],
  title: [{ required: true, message: '请输入试卷标题', type: 'error' }],
}

const dialogVisible = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value),
})

const documentTitle = computed(() => props.knowledge?.file_name || props.knowledge?.title || '未命名文档')
const parseStatusLabel = computed(() => props.knowledge?.parse_status === 'completed' ? '解析已完成，可进入抽题' : '文档解析未完成')
const currentSubjects = computed(() => subjectsByDomain.value[form.value.domain_id] || [])
const matchingBanks = computed(() => questionBanks.value.filter((bank) => {
  if (bank.domain_id !== form.value.domain_id) return false
  if (form.value.subject_id && bank.subject_id && bank.subject_id !== form.value.subject_id) return false
  return true
}))
const selectedQuestionBank = computed(() => matchingBanks.value.find(bank => bank.id === form.value.question_bank_id))
const selectedSpace = computed(() => spaces.value.find(space => space.id === form.value.space_id))
const selectedBankNotice = computed(() => {
  const spaceName = selectedSpace.value?.name || '当前空间'
  if (selectedQuestionBank.value) {
    return `将写入题库「${selectedQuestionBank.value.name}」｜空间：${spaceName}`
  }
  return `未选择题库，将在「${spaceName}」自动创建新题库`
})

const spaceLabel = (space: ExamSpace) => {
  const typeMap: Record<string, string> = { personal: '个人', class: '班级', public: '公共' }
  return `${space.name} · ${typeMap[space.space_type] || space.space_type}`
}

const loadSubjects = async (domainId: string) => {
  if (!domainId || subjectsByDomain.value[domainId]) return
  const res = await listExamSubjects(domainId)
  subjectsByDomain.value = { ...subjectsByDomain.value, [domainId]: res.data || [] }
}

const loadBanks = async (spaceId: string) => {
  if (!spaceId) {
    questionBanks.value = []
    return
  }
  const res = await listQuestionBanks({ space_id: spaceId })
  questionBanks.value = res.data || []
}

const resetFormDefaults = async () => {
  const title = documentTitle.value
  const domain = pickDomain(domains.value, title)
  await loadSubjects(domain?.id || '')
  const subject = pickSubject(subjectsByDomain.value[domain?.id || ''] || [], title)
  const space = pickSpace(spaces.value)
  form.value = {
    space_id: space?.id || '',
    domain_id: domain?.id || '',
    subject_id: subject?.id,
    question_bank_id: undefined,
    title,
    source_year: inferYear(title),
    source_region: title.includes('全国') ? '全国Ⅰ卷' : '',
    paper_type: title.includes('高考') ? '高考英语真题' : '',
  }
  if (form.value.space_id) await loadBanks(form.value.space_id)
  reselectQuestionBank(true)
}

const loadData = async () => {
  loading.value = true
  try {
    await ensurePersonalExamSpace()
    const [spaceRes, domainRes] = await Promise.all([listExamSpaces(), listExamDomains()])
    spaces.value = spaceRes.data || []
    domains.value = domainRes.data || []
    await resetFormDefaults()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '考试导入信息加载失败')
  } finally {
    loading.value = false
  }
}

const submitImport = async () => {
  if (!props.knowledge?.id || importing.value) return
  if (props.knowledge.parse_status !== 'completed') {
    MessagePlugin.warning('文档解析完成后才能导入试卷')
    return
  }
  const result = await formRef.value?.validate()
  if (result !== true) return

  importing.value = true
  try {
    const res = await registerExamMaterial({
      space_id: form.value.space_id,
      knowledge_base_id: props.knowledge.knowledge_base_id || props.kbId,
      knowledge_id: props.knowledge.id,
      domain_id: form.value.domain_id,
      subject_id: form.value.subject_id || undefined,
      material_type: 'exam_paper',
      title: form.value.title,
      source_year: form.value.source_year,
      source_region: form.value.source_region,
      paper_type: form.value.paper_type,
      question_bank_id: form.value.question_bank_id || undefined,
      create_task: true,
    })
    const taskId = res.data?.structuring_task?.id
    if (!taskId) throw new Error('结构化任务创建失败')
    await extractQuestionDrafts(taskId)
    MessagePlugin.success('试卷已进入题目校对')
    emit('imported', taskId)
    dialogVisible.value = false
    await router.push(`/platform/structuring-tasks/${taskId}/review`)
  } catch (error: any) {
    MessagePlugin.error(error?.message || '试卷导入失败')
  } finally {
    importing.value = false
  }
}

const pickSpace = (items: ExamSpace[]) => items.find(item => item.space_type === 'class') || items[0]
const pickDomain = (items: ExamDomain[], title: string) => {
  const normalized = title.toLowerCase()
  if (normalized.includes('ielts') || title.includes('雅思')) return items.find(item => item.code === 'ielts') || items[0]
  return items.find(item => item.code === 'gaokao') || items[0]
}
const pickSubject = (items: ExamSubject[], title: string) => {
  const normalized = title.toLowerCase()
  if (normalized.includes('english') || title.includes('英语')) return items.find(item => item.code === 'english') || items[0]
  if (normalized.includes('reading')) return items.find(item => item.code === 'reading') || items[0]
  return items[0]
}
const normalizeMatchText = (value: string) => value.toLowerCase().replace(/\s+/g, '')
const questionBankMatchScore = (bank: QuestionBank, title: string) => {
  const normalizedTitle = normalizeMatchText(title)
  const normalizedName = normalizeMatchText(bank.name)
  let score = 0
  if (form.value.subject_id && bank.subject_id === form.value.subject_id) score += 40
  if (normalizedName && normalizedTitle.includes(normalizedName)) score += 30
  if (normalizedTitle && normalizedName.includes(normalizedTitle)) score += 30
  for (const keyword of ['高考', '英语', '雅思', '真题', '全国', '阅读']) {
    if (normalizedTitle.includes(keyword) && normalizedName.includes(keyword)) score += 8
  }
  const year = inferYear(title)
  if (year && normalizedName.includes(String(year))) score += 12
  return score
}
const pickQuestionBank = (items: QuestionBank[], title: string) => {
  if (!items.length) return undefined
  return [...items].sort((a, b) => {
    const scoreDiff = questionBankMatchScore(b, title) - questionBankMatchScore(a, title)
    if (scoreDiff !== 0) return scoreDiff
    return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  })[0]
}
const markQuestionBankChoiceManual = () => {
  questionBankChoiceManual.value = true
}
const reselectQuestionBank = (force = false) => {
  if (!force && questionBankChoiceManual.value && (!form.value.question_bank_id || selectedQuestionBank.value)) {
    return
  }
  form.value.question_bank_id = pickQuestionBank(matchingBanks.value, form.value.title)?.id
  questionBankChoiceManual.value = false
}
const inferYear = (title: string) => {
  const match = title.match(/20\d{2}/)
  return match ? Number(match[0]) : undefined
}

watch(() => props.visible, (visible) => {
  if (visible) void loadData()
})

watch(() => form.value.domain_id, (domainId) => {
  if (domainId) void loadSubjects(domainId)
  questionBankChoiceManual.value = false
  reselectQuestionBank()
})

watch(() => form.value.subject_id, () => {
  questionBankChoiceManual.value = false
  reselectQuestionBank()
})

watch(() => form.value.title, () => {
  reselectQuestionBank()
})

watch(() => form.value.space_id, async (spaceId) => {
  if (!props.visible) return
  questionBankChoiceManual.value = false
  await loadBanks(spaceId)
  reselectQuestionBank()
})
</script>

<style lang="less" scoped>
.exam-import-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.doc-summary {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);

  :deep(.t-icon) {
    color: var(--td-brand-color);
    font-size: 20px;
  }

  div {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  strong,
  span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  span {
    color: var(--td-text-color-secondary);
    font-size: 12px;
  }
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

@media (max-width: 640px) {
  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
