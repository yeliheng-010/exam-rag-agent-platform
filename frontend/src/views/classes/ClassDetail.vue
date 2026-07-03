<template>
  <div class="exam-page">
    <div class="exam-header">
      <div>
        <t-button variant="text" size="small" @click="router.push('/platform/classes')">
          <template #icon><t-icon name="chevron-left" /></template>
          返回班级
        </t-button>
        <h2>{{ classInfo?.name || '班级详情' }}</h2>
        <p>{{ classInfo?.description || '围绕班级空间组织资料、题库、作业与学习分析。' }}</p>
      </div>
      <t-tag v-if="classInfo" :theme="classInfo.status === 'active' ? 'success' : 'default'" variant="light">
        {{ classInfo.status === 'active' ? '运行中' : '已归档' }}
      </t-tag>
    </div>

    <t-loading :loading="loading">
      <div v-if="classInfo" class="summary-grid">
        <div class="summary-item">
          <span>班级空间</span>
          <strong>{{ classInfo.space_id }}</strong>
        </div>
        <div class="summary-item">
          <span>成员上限</span>
          <strong>{{ classInfo.member_limit }}</strong>
        </div>
        <div class="summary-item">
          <span>邀请码</span>
          <strong>{{ classInfo.invite_code || '待生成' }}</strong>
        </div>
      </div>

      <t-tabs v-model="activeTab" class="detail-tabs">
        <t-tab-panel value="overview" label="概览">
          <div class="tab-panel">
            <h3>第一阶段班级承载能力</h3>
            <p>班级已拥有独立 exam_space，后续知识库、题库、作业、权益都会通过 space_id 进行隔离。</p>
            <div class="flow-grid">
              <div>班级空间</div>
              <div>资料入库</div>
              <div>试卷结构化</div>
              <div>作业与练习</div>
            </div>
          </div>
        </t-tab-panel>
        <t-tab-panel value="members" label="成员">
          <div class="tab-panel">
            <div class="panel-title-row">
              <div>
                <h3>成员审核</h3>
                <p>学生通过邀请码提交加入申请，老师或助教审核通过后才能进入班级。</p>
              </div>
              <t-button v-if="canReviewMembers" variant="outline" :loading="membersLoading" @click="loadMembers">
                <template #icon><t-icon name="refresh" /></template>
                刷新
              </t-button>
            </div>
            <t-alert
              v-if="!canReviewMembers"
              theme="info"
              message="当前账号可查看班级信息，成员审核由班级老师或助教处理。"
            />
            <t-loading v-else :loading="membersLoading">
              <t-table
                row-key="id"
                :data="members"
                :columns="memberColumns"
                :pagination="{ pageSize: 8, total: members.length }"
                size="small"
              >
                <template #role="{ row }">
                  <t-tag variant="light" :theme="roleTheme(row.role)">{{ roleLabel(row.role) }}</t-tag>
                </template>
                <template #status="{ row }">
                  <t-tag variant="light" :theme="row.status === 'pending' ? 'warning' : 'success'">
                    {{ row.status === 'pending' ? '待审核' : '已加入' }}
                  </t-tag>
                </template>
                <template #created_at="{ row }">
                  {{ formatDate(row.created_at) }}
                </template>
                <template #actions="{ row }">
                  <t-space v-if="row.status === 'pending'" size="small">
                    <t-button size="small" theme="primary" :loading="reviewingUserId === row.user_id" @click="approveMember(row.user_id)">通过</t-button>
                    <t-button size="small" theme="danger" variant="outline" :loading="reviewingUserId === row.user_id" @click="rejectMember(row.user_id)">拒绝</t-button>
                  </t-space>
                  <span v-else class="muted-text">无操作</span>
                </template>
              </t-table>
            </t-loading>
          </div>
        </t-tab-panel>
        <t-tab-panel value="resources" label="资料">
          <div class="tab-panel">
            <div class="panel-title-row">
              <div>
                <h3>班级知识库</h3>
                <p>老师将知识库绑定到班级空间后，审核通过的学生可以在学习中心选择这些资料进行基础 RAG 对话。</p>
              </div>
              <t-space size="small">
                <t-button variant="outline" :loading="resourcesLoading" @click="loadResources">
                  <template #icon><t-icon name="refresh" /></template>
                  刷新
                </t-button>
                <t-button v-if="canManageResources" theme="primary" @click="openBindDialog">
                  <template #icon><t-icon name="link" /></template>
                  绑定知识库
                </t-button>
              </t-space>
            </div>
            <t-alert
              v-if="!canManageResources"
              theme="info"
              message="当前账号可使用班级资料进行对话，资料配置由班级老师或助教处理。"
            />
            <t-loading :loading="resourcesLoading">
              <t-table
                row-key="id"
                :data="resources"
                :columns="resourceColumns"
                :pagination="{ pageSize: 8, total: resources.length }"
                size="small"
              >
                <template #resource_id="{ row }">
                  <div class="resource-name-cell">
                    <strong>{{ knowledgeBaseName(row.resource_id) }}</strong>
                    <span>{{ row.resource_id }}</span>
                  </div>
                </template>
                <template #material_type="{ row }">
                  <t-tag variant="light">{{ materialTypeLabel(row.material_type) }}</t-tag>
                </template>
                <template #domain_id="{ row }">
                  {{ domainName(row.domain_id) }}
                </template>
                <template #review_status="{ row }">
                  <t-tag :theme="reviewTag(row.review_status)" variant="light">{{ reviewLabel(row.review_status) }}</t-tag>
                </template>
                <template #updated_at="{ row }">
                  {{ formatDate(row.updated_at) }}
                </template>
                <template #actions="{ row }">
                  <t-button variant="text" size="small" @click="startChat(row.resource_id)">开始对话</t-button>
                </template>
              </t-table>
              <t-empty v-if="!resources.length && !resourcesLoading" size="small" description="暂无班级资料" />
            </t-loading>
          </div>
        </t-tab-panel>
        <t-tab-panel v-for="item in futureTabs" :key="item.value" :value="item.value" :label="item.label">
          <div class="tab-panel">
            <h3>{{ item.label }}</h3>
            <p>{{ item.desc }}</p>
            <t-empty size="small" :description="item.empty" />
          </div>
        </t-tab-panel>
      </t-tabs>
    </t-loading>

    <t-dialog
      v-model:visible="bindVisible"
      header="绑定知识库到班级"
      :confirm-btn="{ content: '绑定', loading: bindingResource }"
      @confirm="submitBindResource"
    >
      <t-form ref="resourceFormRef" :data="bindForm" :rules="bindRules" label-align="top">
        <t-form-item label="知识库" name="kb_id">
          <t-select
            v-model="bindForm.kb_id"
            :loading="knowledgeBasesLoading"
            placeholder="选择一个已创建的知识库"
            clearable
            filterable
          >
            <t-option v-for="kb in knowledgeBases" :key="kb.id" :value="kb.id" :label="kb.name" />
          </t-select>
        </t-form-item>
        <t-form-item label="考试方向" name="domain_id">
          <t-select
            v-model="bindForm.domain_id"
            :loading="domainsLoading"
            placeholder="选择高考或雅思"
            clearable
            @change="handleBindDomainChange"
          >
            <t-option v-for="domain in domains" :key="domain.id" :value="domain.id" :label="domain.name" />
          </t-select>
        </t-form-item>
        <t-form-item label="科目 / 模块" name="subject_id">
          <t-select
            v-model="bindForm.subject_id"
            :disabled="!bindForm.domain_id"
            :loading="subjectsLoading"
            placeholder="可选"
            clearable
          >
            <t-option v-for="subject in subjects" :key="subject.id" :value="subject.id" :label="subject.name" />
          </t-select>
        </t-form-item>
        <t-form-item label="资料类型" name="material_type">
          <t-select v-model="bindForm.material_type">
            <t-option value="learning_material" label="学习资料" />
            <t-option value="exam_paper" label="试卷" />
            <t-option value="answer_key" label="答案" />
            <t-option value="explanation" label="解析" />
          </t-select>
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import type { FormInstanceFunctions, FormRule } from 'tdesign-vue-next'
import { approveExamClassMember, getExamClass, listExamClassMembers, rejectExamClassMember } from '@/api/exam/class'
import { listExamDomains, listExamSubjects } from '@/api/exam/domain'
import { bindKnowledgeBaseResource, listExamResources } from '@/api/exam/resource'
import { listKnowledgeBases } from '@/api/knowledge-base'
import { useSettingsStore } from '@/stores/settings'
import { useAuthStore } from '@/stores/auth'
import type {
  ExamClass,
  ExamClassMember,
  ExamClassRole,
  ExamDomain,
  ExamMaterialType,
  ExamSpaceResource,
  ExamSubject,
  ReviewStatus,
} from '@/types/exam'
import type { KnowledgeBaseInfo } from '@/api/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const loading = ref(false)
const membersLoading = ref(false)
const resourcesLoading = ref(false)
const domainsLoading = ref(false)
const subjectsLoading = ref(false)
const knowledgeBasesLoading = ref(false)
const bindingResource = ref(false)
const reviewingUserId = ref('')
const activeTab = ref('overview')
const classInfo = ref<ExamClass | null>(null)
const members = ref<ExamClassMember[]>([])
const resources = ref<ExamSpaceResource[]>([])
const domains = ref<ExamDomain[]>([])
const subjects = ref<ExamSubject[]>([])
const knowledgeBases = ref<KnowledgeBaseInfo[]>([])
const resourceFormRef = ref<FormInstanceFunctions>()
const bindVisible = ref(false)
const canReviewMembers = computed(() => authStore.hasRole('contributor'))
const canManageResources = computed(() => authStore.hasRole('contributor'))

const bindForm = ref({
  kb_id: '',
  domain_id: '',
  subject_id: '',
  material_type: 'learning_material' as ExamMaterialType,
})

const futureTabs = [
  { value: 'questionSets', label: '题集', desc: '班级题集来自题库筛选、试卷结构化和老师手动组题。', empty: '题集能力将在结构化题库后启用' },
  { value: 'homework', label: '作业', desc: '老师可从题集生成作业，学生答题后进入错题与学习报告。', empty: '作业闭环将在练习阶段启用' },
  { value: 'analytics', label: '分析', desc: '班级分析聚合掌握度、错题分布、任务完成率和资料使用情况。', empty: '分析指标将在学习记录接入后生成' },
  { value: 'entitlements', label: '权益', desc: '高成本解析、Agent 工具调用和班级人数会进入权益校验。', empty: '权益明细将在支付模块接入后显示' },
  { value: 'settings', label: '设置', desc: '班级名称、考试域、成员上限和归档策略在此维护。', empty: '班级设置将在编辑接口接入后启用' },
]

const memberColumns = [
  { colKey: 'user_id', title: '用户 ID', ellipsis: true },
  { colKey: 'role', title: '班级角色', cell: 'role', width: 120 },
  { colKey: 'status', title: '状态', cell: 'status', width: 120 },
  { colKey: 'created_at', title: '申请时间', cell: 'created_at', width: 160 },
  { colKey: 'actions', title: '操作', cell: 'actions', width: 160 },
]

const resourceColumns = [
  { colKey: 'resource_id', title: '知识库', cell: 'resource_id', ellipsis: true },
  { colKey: 'material_type', title: '资料类型', cell: 'material_type', width: 120 },
  { colKey: 'domain_id', title: '考试方向', cell: 'domain_id', width: 120 },
  { colKey: 'review_status', title: '状态', cell: 'review_status', width: 120 },
  { colKey: 'updated_at', title: '更新时间', cell: 'updated_at', width: 160 },
  { colKey: 'actions', title: '操作', cell: 'actions', width: 120 },
]

const bindRules: Record<string, FormRule[]> = {
  kb_id: [{ required: true, message: '请选择知识库', type: 'error' }],
  domain_id: [{ required: true, message: '请选择考试方向', type: 'error' }],
}

const formatDate = (value?: string) => {
  if (!value) return ''
  return new Date(value).toLocaleString()
}

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

const knowledgeBaseName = (kbId: string) => {
  return knowledgeBases.value.find(item => item.id === kbId)?.name || '未知知识库'
}

const roleLabel = (role: ExamClassRole) => {
  if (role === 'teacher') return '老师'
  if (role === 'assistant') return '助教'
  return '学生'
}

const roleTheme = (role: ExamClassRole) => {
  if (role === 'teacher') return 'primary'
  if (role === 'assistant') return 'warning'
  return 'default'
}

const loadData = async () => {
  const classId = String(route.params.classId || '')
  if (!classId) return
  loading.value = true
  try {
    const res = await getExamClass(classId)
    classInfo.value = res.data
  } catch (error: any) {
    MessagePlugin.error(error?.message || '班级详情加载失败')
  } finally {
    loading.value = false
  }
}

const loadDomains = async () => {
  domainsLoading.value = true
  try {
    const res = await listExamDomains()
    domains.value = res.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '考试方向加载失败')
  } finally {
    domainsLoading.value = false
  }
}

const loadSubjects = async (domainId: string) => {
  subjects.value = []
  if (!domainId) return
  subjectsLoading.value = true
  try {
    const res = await listExamSubjects(domainId)
    subjects.value = res.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '科目列表加载失败')
  } finally {
    subjectsLoading.value = false
  }
}

const loadKnowledgeBases = async () => {
  knowledgeBasesLoading.value = true
  try {
    const res: any = await listKnowledgeBases()
    knowledgeBases.value = res.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '知识库列表加载失败')
  } finally {
    knowledgeBasesLoading.value = false
  }
}

const loadMembers = async () => {
  if (!canReviewMembers.value) return
  const classId = String(route.params.classId || '')
  if (!classId) return
  membersLoading.value = true
  try {
    const res = await listExamClassMembers(classId)
    members.value = res.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '成员列表加载失败')
  } finally {
    membersLoading.value = false
  }
}

const loadResources = async () => {
  if (!classInfo.value?.space_id) return
  resourcesLoading.value = true
  try {
    const [resourceRes] = await Promise.all([
      listExamResources({ space_id: classInfo.value.space_id, resource_type: 'knowledge_base' }),
      knowledgeBases.value.length ? Promise.resolve(null) : loadKnowledgeBases(),
      domains.value.length ? Promise.resolve(null) : loadDomains(),
    ])
    resources.value = resourceRes.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '班级资料加载失败')
  } finally {
    resourcesLoading.value = false
  }
}

const approveMember = async (userId: string) => {
  const classId = String(route.params.classId || '')
  if (!classId || !userId) return
  reviewingUserId.value = userId
  try {
    await approveExamClassMember(classId, userId)
    MessagePlugin.success('已通过加入申请')
    await loadMembers()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '审核失败')
  } finally {
    reviewingUserId.value = ''
  }
}

const rejectMember = async (userId: string) => {
  const classId = String(route.params.classId || '')
  if (!classId || !userId) return
  reviewingUserId.value = userId
  try {
    await rejectExamClassMember(classId, userId)
    MessagePlugin.success('已拒绝加入申请')
    await loadMembers()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '审核失败')
  } finally {
    reviewingUserId.value = ''
  }
}

const openBindDialog = async () => {
  if (!classInfo.value?.space_id) return
  bindForm.value = {
    kb_id: '',
    domain_id: classInfo.value.domain_id || domains.value[0]?.id || '',
    subject_id: '',
    material_type: 'learning_material',
  }
  bindVisible.value = true
  await Promise.all([
    knowledgeBases.value.length ? Promise.resolve(null) : loadKnowledgeBases(),
    domains.value.length ? Promise.resolve(null) : loadDomains(),
  ])
  if (!bindForm.value.domain_id) {
    bindForm.value.domain_id = classInfo.value?.domain_id || domains.value[0]?.id || ''
  }
  if (bindForm.value.domain_id) {
    await loadSubjects(bindForm.value.domain_id)
  }
}

const handleBindDomainChange = async (value: string | number | boolean) => {
  const domainId = typeof value === 'string' ? value : ''
  bindForm.value.subject_id = ''
  await loadSubjects(domainId)
}

const submitBindResource = async () => {
  if (!classInfo.value?.space_id) return
  const result = await resourceFormRef.value?.validate()
  if (result !== true) return

  bindingResource.value = true
  try {
    await bindKnowledgeBaseResource(bindForm.value.kb_id, {
      space_id: classInfo.value.space_id,
      domain_id: bindForm.value.domain_id,
      subject_id: bindForm.value.subject_id || undefined,
      material_type: bindForm.value.material_type,
    })
    MessagePlugin.success('知识库已绑定到班级')
    bindVisible.value = false
    await loadResources()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '绑定知识库失败')
  } finally {
    bindingResource.value = false
  }
}

const startChat = (kbId: string) => {
  if (!kbId) return
  settingsStore.selectKnowledgeBases([kbId])
  settingsStore.clearFiles()
  settingsStore.clearTags()
  router.push('/platform/creatChat')
}

watch(activeTab, (tab) => {
  if (tab === 'members') {
    loadMembers()
  }
  if (tab === 'resources') {
    loadResources()
  }
})

onMounted(async () => {
  await loadData()
  await Promise.all([
    loadDomains(),
    loadKnowledgeBases(),
  ])
  if (activeTab.value === 'members') {
    await loadMembers()
  }
  if (activeTab.value === 'resources') {
    await loadResources()
  }
})
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
    margin: 8px 0 0;
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

.summary-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.summary-item {
  padding: 14px 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;

  span {
    display: block;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    margin-bottom: 8px;
  }

  strong {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 15px;
    font-weight: 600;
  }
}

.detail-tabs {
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  padding: 0 16px 16px;
}

.tab-panel {
  min-height: 260px;
  padding: 18px 0 0;

  h3 {
    margin: 0 0 6px;
    font-size: 16px;
    font-weight: 600;
  }

  p {
    margin: 0 0 16px;
    color: var(--td-text-color-secondary);
    font-size: 14px;
    line-height: 22px;
  }
}

.panel-title-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.muted-text {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.resource-name-cell {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;

  strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 13px;
    font-weight: 600;
  }

  span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--td-text-color-placeholder);
    font-size: 12px;
  }
}

.flow-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;

  div {
    padding: 16px;
    border-radius: 8px;
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-primary);
    font-weight: 600;
    text-align: center;
  }
}
</style>
