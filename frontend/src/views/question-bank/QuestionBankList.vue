<template>
  <div class="exam-page">
    <div class="exam-header">
      <div>
        <h2>题库中心</h2>
        <p>管理高考、雅思题库资产，后续试卷解析会把结构化题目写入这里。</p>
      </div>
      <div class="header-actions">
        <t-select v-model="selectedSpaceId" clearable placeholder="全部空间" class="space-filter" @change="loadBanks">
          <t-option v-for="space in spaces" :key="space.id" :value="space.id" :label="spaceLabel(space)" />
        </t-select>
        <t-button theme="primary" @click="openCreateDialog">
          <template #icon><t-icon name="add" /></template>
          新建题库
        </t-button>
      </div>
    </div>

    <t-loading :loading="loading">
      <div class="summary-grid">
        <div class="summary-item">
          <span>空间数</span>
          <strong>{{ spaces.length }}</strong>
        </div>
        <div class="summary-item">
          <span>题库数</span>
          <strong>{{ questionBanks.length }}</strong>
        </div>
        <div class="summary-item">
          <span>考试域</span>
          <strong>{{ domains.length }}</strong>
        </div>
      </div>

      <t-table
        row-key="id"
        :data="questionBanks"
        :columns="columns"
        :pagination="{ defaultPageSize: 10, showJumper: true }"
        hover
      >
        <template #name="{ row }">
          <div class="bank-name">
            <strong>{{ row.name }}</strong>
            <span>{{ row.description || '暂无说明' }}</span>
          </div>
        </template>
        <template #space="{ row }">
          {{ spaceName(row.space_id) }}
        </template>
        <template #domain="{ row }">
          {{ domainName(row.domain_id) }}
        </template>
        <template #subject="{ row }">
          {{ subjectName(row.subject_id) }}
        </template>
        <template #review_status="{ row }">
          <t-tag :theme="reviewTag(row.review_status)" variant="light">{{ reviewLabel(row.review_status) }}</t-tag>
        </template>
        <template #operation="{ row }">
          <t-button variant="text" size="small" @click="router.push(`/platform/question-banks/${row.id}`)">详情</t-button>
        </template>
      </t-table>
    </t-loading>

    <t-dialog
      v-model:visible="createVisible"
      header="新建题库"
      :confirm-btn="{ content: '创建', loading: creating }"
      @confirm="submitCreate"
    >
      <t-form ref="formRef" :data="form" :rules="rules" label-align="top">
        <t-form-item label="题库名称" name="name">
          <t-input v-model="form.name" placeholder="例如：2026 高考英语真题精讲" :maxlength="100" />
        </t-form-item>
        <t-form-item label="所属空间" name="space_id">
          <t-select v-model="form.space_id" placeholder="选择个人或班级空间">
            <t-option v-for="space in spaces" :key="space.id" :value="space.id" :label="spaceLabel(space)" />
          </t-select>
        </t-form-item>
        <t-form-item label="考试域" name="domain_id">
          <t-select v-model="form.domain_id" placeholder="选择考试域" @change="form.subject_id = undefined">
            <t-option v-for="domain in domains" :key="domain.id" :value="domain.id" :label="domain.name" />
          </t-select>
        </t-form-item>
        <t-form-item label="科目 / 模块" name="subject_id">
          <t-select v-model="form.subject_id" clearable placeholder="可选">
            <t-option v-for="subject in currentSubjects" :key="subject.id" :value="subject.id" :label="subject.name" />
          </t-select>
        </t-form-item>
        <t-form-item label="说明" name="description">
          <t-textarea v-model="form.description" placeholder="记录来源、年份、用途或审核范围" :maxlength="500" />
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import type { FormInstanceFunctions, FormRule } from 'tdesign-vue-next'
import { listExamDomains, listExamSubjects } from '@/api/exam/domain'
import { ensurePersonalExamSpace, listExamSpaces } from '@/api/exam/space'
import { createQuestionBank, listQuestionBanks } from '@/api/exam/question-bank'
import type { ExamDomain, ExamSpace, ExamSubject, QuestionBank, ReviewStatus } from '@/types/exam'

const router = useRouter()
const loading = ref(false)
const creating = ref(false)
const createVisible = ref(false)
const selectedSpaceId = ref<string>()
const spaces = ref<ExamSpace[]>([])
const domains = ref<ExamDomain[]>([])
const subjectsByDomain = ref<Record<string, ExamSubject[]>>({})
const questionBanks = ref<QuestionBank[]>([])
const formRef = ref<FormInstanceFunctions>()

const form = ref({
  name: '',
  description: '',
  space_id: '',
  domain_id: '',
  subject_id: undefined as string | undefined,
})

const rules: Record<string, FormRule[]> = {
  name: [{ required: true, message: '请输入题库名称', type: 'error' }],
  space_id: [{ required: true, message: '请选择所属空间', type: 'error' }],
  domain_id: [{ required: true, message: '请选择考试域', type: 'error' }],
}

const columns = computed(() => [
  { colKey: 'name', title: '题库', minWidth: 240 },
  { colKey: 'space', title: '空间', width: 180 },
  { colKey: 'domain', title: '考试域', width: 120 },
  { colKey: 'subject', title: '科目 / 模块', width: 130 },
  { colKey: 'review_status', title: '审核状态', width: 120 },
  { colKey: 'operation', title: '操作', width: 96 },
])

const currentSubjects = computed(() => {
  if (!form.value.domain_id) return []
  return subjectsByDomain.value[form.value.domain_id] || []
})

const spaceLabel = (space: ExamSpace) => {
  const typeMap: Record<string, string> = {
    personal: '个人',
    class: '班级',
    public: '公共',
  }
  return `${space.name} · ${typeMap[space.space_type] || space.space_type}`
}

const spaceName = (spaceId: string) => spaces.value.find(item => item.id === spaceId)?.name || '未知空间'
const domainName = (domainId: string) => domains.value.find(item => item.id === domainId)?.name || '未知考试域'
const subjectName = (subjectId?: string) => {
  if (!subjectId) return '未指定'
  return Object.values(subjectsByDomain.value).flat().find(item => item.id === subjectId)?.name || '未知模块'
}

const reviewLabel = (status: ReviewStatus) => {
  const map: Record<ReviewStatus, string> = {
    private: '私有',
    pending: '待审核',
    approved: '已公开',
    rejected: '已驳回',
  }
  return map[status] || status
}

const reviewTag = (status: ReviewStatus) => {
  if (status === 'approved') return 'success'
  if (status === 'pending') return 'warning'
  if (status === 'rejected') return 'danger'
  return 'default'
}

const loadSubjects = async (domainId: string) => {
  if (!domainId || subjectsByDomain.value[domainId]) return
  const res = await listExamSubjects(domainId)
  subjectsByDomain.value = {
    ...subjectsByDomain.value,
    [domainId]: res.data || [],
  }
}

const loadBanks = async () => {
  try {
    const res = await listQuestionBanks({ space_id: selectedSpaceId.value })
    questionBanks.value = res.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '题库列表加载失败')
  }
}

const loadData = async () => {
  loading.value = true
  try {
    await ensurePersonalExamSpace()
    const [spaceRes, domainRes] = await Promise.all([listExamSpaces(), listExamDomains()])
    spaces.value = spaceRes.data || []
    domains.value = domainRes.data || []
    await Promise.all(domains.value.map(item => loadSubjects(item.id)))
    await loadBanks()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '题库中心加载失败')
  } finally {
    loading.value = false
  }
}

const openCreateDialog = () => {
  const defaultDomain = domains.value[0]
  form.value = {
    name: '',
    description: '',
    space_id: selectedSpaceId.value || spaces.value[0]?.id || '',
    domain_id: defaultDomain?.id || '',
    subject_id: undefined,
  }
  createVisible.value = true
}

const submitCreate = async () => {
  const result = await formRef.value?.validate()
  if (result !== true) return

  creating.value = true
  try {
    const res = await createQuestionBank({
      ...form.value,
      subject_id: form.value.subject_id || undefined,
    })
    MessagePlugin.success('题库已创建')
    createVisible.value = false
    await loadBanks()
    if (res.data?.id) {
      router.push(`/platform/question-banks/${res.data.id}`)
    }
  } catch (error: any) {
    MessagePlugin.error(error?.message || '创建题库失败')
  } finally {
    creating.value = false
  }
}

watch(() => form.value.domain_id, (domainId) => {
  if (domainId) {
    void loadSubjects(domainId)
  }
})

onMounted(loadData)
</script>

<style lang="less" scoped>
.exam-page {
  flex: 1;
  overflow-y: auto;
  padding: 28px 32px;
  background: var(--td-bg-color-container);
}

.exam-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;

  h2 {
    margin: 0;
    font-size: 22px;
    line-height: 30px;
    font-weight: 600;
  }

  p {
    margin: 6px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 14px;
    line-height: 22px;
  }
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.space-filter {
  width: 240px;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 14px;
}

.summary-item {
  padding: 14px 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;

  span {
    display: block;
    margin-bottom: 8px;
    color: var(--td-text-color-secondary);
    font-size: 12px;
  }

  strong {
    font-size: 22px;
    line-height: 30px;
    font-weight: 600;
  }
}

.bank-name {
  display: flex;
  flex-direction: column;
  gap: 2px;

  strong {
    font-size: 14px;
    font-weight: 600;
  }

  span {
    color: var(--td-text-color-secondary);
    font-size: 12px;
  }
}
</style>
