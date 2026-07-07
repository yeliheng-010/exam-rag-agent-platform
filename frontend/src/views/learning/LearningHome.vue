<template>
  <div class="exam-page">
    <div class="exam-header">
      <div>
        <h2>学习中心</h2>
        <p>统一查看个人空间、班级、题库和后续练习任务。</p>
      </div>
      <div class="header-actions">
        <t-button v-if="canUseQuestionBanks" variant="outline" @click="router.push('/platform/question-banks')">
          <template #icon><t-icon name="folder" /></template>
          题库中心
        </t-button>
        <t-button theme="primary" @click="router.push('/platform/classes')">
          <template #icon><t-icon name="usergroup" /></template>
          班级中心
        </t-button>
      </div>
    </div>

    <t-loading :loading="loading">
      <div class="metric-grid">
        <div class="metric-card">
          <span class="metric-label">个人学习空间</span>
          <strong>{{ personalSpace?.name || '待创建' }}</strong>
          <p>{{ personalSpace ? '已为当前账号准备私有学习资料承载区。' : '进入页面后会自动创建个人空间。' }}</p>
        </div>
        <div class="metric-card">
          <span class="metric-label">加入班级</span>
          <strong>{{ classes.length }}</strong>
          <p>班级空间用于老师、学生之间的数据隔离和资源协作。</p>
        </div>
        <div class="metric-card">
          <span class="metric-label">班级知识库</span>
          <strong>{{ classResourceCards.length }}</strong>
          <p>老师授权到班级空间的资料，可直接进入基础 RAG 对话。</p>
        </div>
        <div class="metric-card">
          <span class="metric-label">考试域</span>
          <strong>{{ domains.length }}</strong>
          <p>第一阶段预置高考与雅思，后续可扩展四六级、考研等。</p>
        </div>
      </div>

      <div class="workbench-grid">
        <section class="panel">
          <div class="panel-title">
            <div>
              <h3>考试域与模块</h3>
              <p>后续文档解析、切片策略、题目结构化都会绑定到考试域。</p>
            </div>
            <t-button v-if="authStore.hasRole('admin')" variant="text" @click="router.push('/platform/exam-config')">配置</t-button>
          </div>
          <div class="domain-list">
            <div v-for="domain in domains" :key="domain.id" class="domain-row">
              <div>
                <strong>{{ domain.name }}</strong>
                <span>{{ domain.description || domain.code }}</span>
              </div>
              <t-tag theme="success" variant="light">{{ domain.code }}</t-tag>
            </div>
            <t-empty v-if="!domains.length && !loading" size="small" description="暂无考试域" />
          </div>
        </section>

        <section class="panel">
          <div class="panel-title">
            <div>
              <h3>我的班级</h3>
              <p>班级是平台的数据隔离单元，也是老师管理学生的入口。</p>
            </div>
            <t-button variant="text" @click="router.push('/platform/classes')">查看</t-button>
          </div>
          <div class="compact-list">
            <div v-for="item in classes.slice(0, 5)" :key="item.id" class="compact-row" @click="router.push(`/platform/classes/${item.id}`)">
              <span>{{ item.name }}</span>
              <t-tag variant="light">{{ domainName(item.domain_id) }}</t-tag>
            </div>
            <t-empty v-if="!classes.length && !loading" size="small" description="暂无班级" />
          </div>
        </section>
      </div>

      <section class="panel resource-panel">
        <div class="panel-title">
          <div>
            <h3>班级知识库</h3>
            <p>选择老师配置到班级空间的知识库，进入基础问答时会自动带上资料范围。</p>
          </div>
          <t-button variant="text" :loading="resourcesLoading" @click="loadClassResources">刷新</t-button>
        </div>
        <div v-if="classResourceCards.length" class="resource-grid">
          <div v-for="item in classResourceCards" :key="item.id" class="resource-card">
            <div class="resource-card-main">
              <div>
                <strong>{{ item.kbName }}</strong>
                <span>{{ item.className }} · {{ domainName(item.domain_id) }}</span>
              </div>
              <t-tag variant="light">{{ materialTypeLabel(item.material_type) }}</t-tag>
            </div>
            <div class="resource-card-footer">
              <span>{{ reviewLabel(item.review_status) }}</span>
              <t-button size="small" theme="primary" @click="startChat(item.resource_id)">开始对话</t-button>
            </div>
          </div>
        </div>
        <t-empty v-else-if="!resourcesLoading" size="small" description="暂无可用班级知识库" />
      </section>

      <section class="panel practice-panel">
        <div class="panel-title">
          <div>
            <h3>题组练习</h3>
            <p>按老师确认后的正式题组进行练习，提交后查看答案、解析和证据。</p>
          </div>
          <t-button variant="text" :loading="practiceLoading" @click="loadPracticeGroups">刷新</t-button>
        </div>
        <div v-if="practiceGroups.length" class="practice-grid">
          <div v-for="item in practiceGroups" :key="item.group.id" class="practice-card">
            <div class="practice-card__main">
              <div>
                <strong>{{ practiceTitle(item) }}</strong>
                <span>{{ item.bank_name || '题库' }} · {{ groupTypeLabel(item.group.group_type) }} · {{ item.question_count }} 题</span>
              </div>
              <t-tag variant="light">{{ practiceProgress(item) }}</t-tag>
            </div>
            <div class="practice-card__material">
              {{ item.group.material_text || item.group.title || '已确认题组' }}
            </div>
            <div class="practice-card__footer">
              <span>{{ domainName(item.group.domain_id) }}</span>
              <t-button size="small" theme="primary" @click="goPractice(item.group.id)">开始练习</t-button>
            </div>
          </div>
        </div>
        <t-empty v-else-if="!practiceLoading" size="small" description="暂无可练题组" />
      </section>

      <section class="panel">
        <div class="panel-title">
          <div>
            <h3>最近题库</h3>
            <p>题库资产由班主任或助教维护，学生侧先以班级知识库对话为主。</p>
          </div>
          <t-button v-if="canUseQuestionBanks" variant="text" @click="router.push('/platform/question-banks')">进入题库</t-button>
        </div>
        <t-table
          v-if="canUseQuestionBanks"
          row-key="id"
          :data="questionBanks.slice(0, 6)"
          :columns="questionBankColumns"
          :pagination="undefined"
          hover
        >
          <template #domain="{ row }">
            {{ domainName(row.domain_id) }}
          </template>
          <template #review_status="{ row }">
            <t-tag :theme="reviewTag(row.review_status)" variant="light">{{ reviewLabel(row.review_status) }}</t-tag>
          </template>
          <template #operation="{ row }">
            <t-button variant="text" size="small" @click="router.push(`/platform/question-banks/${row.id}`)">详情</t-button>
          </template>
        </t-table>
        <t-empty v-else size="small" description="题库中心由班主任或助教维护" />
      </section>
    </t-loading>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import { listExamDomains } from '@/api/exam/domain'
import { ensurePersonalExamSpace } from '@/api/exam/space'
import { listExamClasses } from '@/api/exam/class'
import { listQuestionBanks } from '@/api/exam/question-bank'
import { listExamResources } from '@/api/exam/resource'
import { listPracticeQuestionGroups } from '@/api/exam/practice'
import { listKnowledgeBases } from '@/api/knowledge-base'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import type { KnowledgeBaseInfo } from '@/api/auth'
import type { ExamClass, ExamDomain, ExamMaterialType, ExamSpace, ExamSpaceResource, QuestionBank, QuestionGroupPracticeSummary, QuestionGroupType, ReviewStatus } from '@/types/exam'

const router = useRouter()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const loading = ref(false)
const resourcesLoading = ref(false)
const practiceLoading = ref(false)
const domains = ref<ExamDomain[]>([])
const personalSpace = ref<ExamSpace | null>(null)
const classes = ref<ExamClass[]>([])
const questionBanks = ref<QuestionBank[]>([])
const classResources = ref<ExamSpaceResource[]>([])
const knowledgeBases = ref<KnowledgeBaseInfo[]>([])
const practiceGroups = ref<QuestionGroupPracticeSummary[]>([])
const canUseQuestionBanks = computed(() => authStore.hasRole('contributor'))

const questionBankColumns = computed(() => [
  { colKey: 'name', title: '题库名称', ellipsis: true },
  { colKey: 'domain', title: '考试域', width: 120 },
  { colKey: 'review_status', title: '状态', width: 120 },
  { colKey: 'operation', title: '操作', width: 96 },
])

const classResourceCards = computed(() => {
  const classBySpace = new Map(classes.value.map(item => [item.space_id, item]))
  return classResources.value
    .filter(item => classBySpace.has(item.space_id))
    .map(item => {
      const classInfo = classBySpace.get(item.space_id)
      const kb = knowledgeBases.value.find(kbItem => kbItem.id === item.resource_id)
      return {
        ...item,
        className: classInfo?.name || '班级',
        kbName: kb?.name || '未知知识库',
      }
    })
})

const domainName = (domainId?: string) => {
  if (!domainId) return '未绑定'
  return domains.value.find(item => item.id === domainId)?.name || '未知考试域'
}

const materialTypeLabel = (type: ExamMaterialType) => {
  const map: Record<ExamMaterialType, string> = {
    learning_material: '学习资料',
    exam_paper: '试卷',
    answer_key: '答案',
    explanation: '解析',
  }
  return map[type] || type
}

const reviewLabel = (status: ReviewStatus) => {
  const map: Record<ReviewStatus, string> = {
    private: '班级可见',
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

const groupTypeLabel = (type: QuestionGroupType) => {
  const map: Record<string, string> = {
    reading_passage: '阅读',
    math_problem: '数学题',
    single_question: '单题',
    cloze: '完型',
    essay: '作文',
  }
  return map[type] || type || '题组'
}

const practiceTitle = (item: QuestionGroupPracticeSummary) => {
  return item.group.title || groupTypeLabel(item.group.group_type)
}

const practiceProgress = (item: QuestionGroupPracticeSummary) => {
  const attempt = item.last_attempt
  if (!attempt) return '未练习'
  if (attempt.status === 'completed') return `${attempt.correct_count}/${attempt.question_count}`
  return `${attempt.answered_count}/${attempt.question_count}`
}

const goPractice = (groupId: string) => {
  router.push(`/platform/practice/question-groups/${groupId}`)
}

const loadClassResources = async () => {
  resourcesLoading.value = true
  try {
    const [resourceRes, kbRes] = await Promise.all([
      listExamResources({ resource_type: 'knowledge_base' }),
      listKnowledgeBases(),
    ])
    classResources.value = resourceRes.data || []
    knowledgeBases.value = ((kbRes as any).data || []) as KnowledgeBaseInfo[]
  } catch (error: any) {
    MessagePlugin.error(error?.message || '班级知识库加载失败')
  } finally {
    resourcesLoading.value = false
  }
}

const loadPracticeGroups = async () => {
  practiceLoading.value = true
  try {
    const res = await listPracticeQuestionGroups({ limit: 6 })
    practiceGroups.value = res.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '题组练习加载失败')
  } finally {
    practiceLoading.value = false
  }
}

const startChat = (kbId: string) => {
  if (!kbId) return
  settingsStore.selectKnowledgeBases([kbId])
  settingsStore.clearFiles()
  settingsStore.clearTags()
  router.push('/platform/creatChat')
}

const loadData = async () => {
  loading.value = true
  try {
    const [domainRes, spaceRes, classRes, bankRes] = await Promise.all([
      listExamDomains(),
      ensurePersonalExamSpace(),
      listExamClasses(),
      canUseQuestionBanks.value ? listQuestionBanks() : Promise.resolve({ data: [] }),
    ])
    domains.value = domainRes.data || []
    personalSpace.value = spaceRes.data || null
    classes.value = classRes.data || []
    questionBanks.value = bankRes.data || []
    await Promise.all([loadClassResources(), loadPracticeGroups()])
  } catch (error: any) {
    MessagePlugin.error(error?.message || '学习中心加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
</script>

<style lang="less" scoped>
.exam-page {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  padding: 28px 32px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
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
  gap: 10px;
  flex-shrink: 0;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 12px;
}

.metric-card,
.panel {
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.metric-card {
  min-height: 118px;
  padding: 16px;

  .metric-label {
    display: block;
    margin-bottom: 10px;
    color: var(--td-text-color-secondary);
    font-size: 13px;
  }

  strong {
    display: block;
    font-size: 26px;
    line-height: 34px;
    font-weight: 600;
  }

  p {
    margin: 8px 0 0;
    color: var(--td-text-color-placeholder);
    font-size: 12px;
    line-height: 18px;
  }
}

.workbench-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 12px;
}

.panel {
  padding: 16px;
}

.resource-panel {
  margin-bottom: 12px;
}

.practice-panel {
  margin-bottom: 12px;
}

.panel-title {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;

  h3 {
    margin: 0;
    font-size: 16px;
    line-height: 24px;
    font-weight: 600;
  }

  p {
    margin: 4px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 13px;
    line-height: 20px;
  }
}

.domain-list,
.compact-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.domain-row,
.compact-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 48px;
  padding: 10px 12px;
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
}

.domain-row strong,
.compact-row span:first-child {
  display: block;
  font-size: 14px;
  font-weight: 600;
}

.domain-row span {
  display: block;
  margin-top: 2px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.compact-row {
  cursor: pointer;
  transition: background 0.15s ease;

  &:hover {
    background: var(--td-bg-color-container-hover);
  }
}

.resource-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 10px;
}

.practice-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 10px;
}

.resource-card,
.practice-card {
  display: flex;
  min-height: 116px;
  flex-direction: column;
  justify-content: space-between;
  gap: 14px;
  padding: 14px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.resource-card-main,
.resource-card-footer,
.practice-card__main,
.practice-card__footer {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.resource-card-main,
.practice-card__main {
  strong {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 15px;
    line-height: 22px;
    font-weight: 600;
  }

  span {
    display: block;
    margin-top: 4px;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
  }
}

.practice-card__material {
  display: -webkit-box;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  font-size: 12px;
  line-height: 18px;
}

.resource-card-footer,
.practice-card__footer {
  align-items: center;
  padding-top: 12px;
  border-top: 1px solid var(--td-component-stroke);

  span {
    color: var(--td-text-color-placeholder);
    font-size: 12px;
  }
}

@media (max-width: 1180px) {
  .metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .workbench-grid {
    grid-template-columns: 1fr;
  }
}
</style>
