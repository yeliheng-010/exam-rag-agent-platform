<template>
  <div class="exam-page">
    <div class="exam-header">
      <div>
        <h2>学习中心</h2>
        <p>统一查看个人空间、班级、题库和后续练习任务。</p>
      </div>
      <div class="header-actions">
        <t-button variant="outline" @click="router.push('/platform/question-banks')">
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
          <span class="metric-label">可用题库</span>
          <strong>{{ questionBanks.length }}</strong>
          <p>题库承载结构化试题、答案、解析和后续检索引用。</p>
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
            <t-button variant="text" @click="router.push('/platform/exam-config')">配置</t-button>
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

      <section class="panel">
        <div class="panel-title">
          <div>
            <h3>最近题库</h3>
            <p>第一阶段先建立题库资产入口，第二阶段接知识库资料，第三阶段接试卷结构化。</p>
          </div>
          <t-button variant="text" @click="router.push('/platform/question-banks')">进入题库</t-button>
        </div>
        <t-table
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
import type { ExamClass, ExamDomain, ExamSpace, QuestionBank, ReviewStatus } from '@/types/exam'

const router = useRouter()
const loading = ref(false)
const domains = ref<ExamDomain[]>([])
const personalSpace = ref<ExamSpace | null>(null)
const classes = ref<ExamClass[]>([])
const questionBanks = ref<QuestionBank[]>([])

const questionBankColumns = computed(() => [
  { colKey: 'name', title: '题库名称', ellipsis: true },
  { colKey: 'domain', title: '考试域', width: 120 },
  { colKey: 'review_status', title: '状态', width: 120 },
  { colKey: 'operation', title: '操作', width: 96 },
])

const domainName = (domainId?: string) => {
  if (!domainId) return '未绑定'
  return domains.value.find(item => item.id === domainId)?.name || '未知考试域'
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

const loadData = async () => {
  loading.value = true
  try {
    const [domainRes, spaceRes, classRes, bankRes] = await Promise.all([
      listExamDomains(),
      ensurePersonalExamSpace(),
      listExamClasses(),
      listQuestionBanks(),
    ])
    domains.value = domainRes.data || []
    personalSpace.value = spaceRes.data || null
    classes.value = classRes.data || []
    questionBanks.value = bankRes.data || []
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

@media (max-width: 1180px) {
  .metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .workbench-grid {
    grid-template-columns: 1fr;
  }
}
</style>
