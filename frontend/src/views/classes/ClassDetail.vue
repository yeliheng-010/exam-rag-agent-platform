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
        <t-tab-panel v-for="item in futureTabs" :key="item.value" :value="item.value" :label="item.label">
          <div class="tab-panel">
            <h3>{{ item.label }}</h3>
            <p>{{ item.desc }}</p>
            <t-empty size="small" :description="item.empty" />
          </div>
        </t-tab-panel>
      </t-tabs>
    </t-loading>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import { approveExamClassMember, getExamClass, listExamClassMembers, rejectExamClassMember } from '@/api/exam/class'
import { useAuthStore } from '@/stores/auth'
import type { ExamClass, ExamClassMember, ExamClassRole } from '@/types/exam'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const loading = ref(false)
const membersLoading = ref(false)
const reviewingUserId = ref('')
const activeTab = ref('overview')
const classInfo = ref<ExamClass | null>(null)
const members = ref<ExamClassMember[]>([])
const canReviewMembers = computed(() => authStore.hasRole('contributor'))

const futureTabs = [
  { value: 'resources', label: '资料', desc: '班级资料将与 WeKnora 知识库关联，按班级空间隔离可见范围。', empty: '资料入口将在知识库空间化后启用' },
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

const formatDate = (value?: string) => {
  if (!value) return ''
  return new Date(value).toLocaleString()
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

watch(activeTab, (tab) => {
  if (tab === 'members') {
    loadMembers()
  }
})

onMounted(async () => {
  await loadData()
  if (activeTab.value === 'members') {
    await loadMembers()
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
