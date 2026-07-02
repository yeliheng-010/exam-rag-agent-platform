<template>
  <div class="teacher-applications">
    <div class="toolbar">
      <t-radio-group v-model="statusFilter" variant="default-filled" @change="loadApplications">
        <t-radio-button value="">全部</t-radio-button>
        <t-radio-button value="pending">待审核</t-radio-button>
        <t-radio-button value="approved">已通过</t-radio-button>
        <t-radio-button value="rejected">已拒绝</t-radio-button>
      </t-radio-group>
      <t-button variant="outline" :loading="loading" @click="loadApplications">
        <template #icon><t-icon name="refresh" /></template>
        刷新
      </t-button>
    </div>

    <t-loading :loading="loading">
      <t-table
        row-key="id"
        :data="applications"
        :columns="columns"
        :pagination="{ pageSize: 10, total: applications.length }"
        size="small"
      >
        <template #user="{ row }">
          <div class="user-cell">
            <strong>{{ row.username || row.user_id }}</strong>
            <span>{{ row.user_email || row.user_id }}</span>
          </div>
        </template>
        <template #status="{ row }">
          <t-tag variant="light" :theme="statusTheme(row.status)">
            {{ statusLabel(row.status) }}
          </t-tag>
        </template>
        <template #reason="{ row }">
          <span class="line-clamp">{{ row.reason || '-' }}</span>
        </template>
        <template #created_at="{ row }">
          {{ formatDate(row.created_at) }}
        </template>
        <template #actions="{ row }">
          <t-space v-if="row.status === 'pending'" size="small">
            <t-button size="small" theme="primary" :loading="reviewingId === row.id" @click="review(row.id, true)">通过</t-button>
            <t-button size="small" theme="danger" variant="outline" :loading="reviewingId === row.id" @click="review(row.id, false)">拒绝</t-button>
          </t-space>
          <span v-else class="muted">{{ row.reviewer_name || row.reviewer_id || '-' }}</span>
        </template>
      </t-table>
    </t-loading>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { approveTeacherApplication, listTeacherApplications, rejectTeacherApplication } from '@/api/exam/teacher'
import type { ExamTeacherApplication, ExamTeacherApplicationStatus } from '@/types/exam'

const loading = ref(false)
const reviewingId = ref('')
const statusFilter = ref('')
const applications = ref<ExamTeacherApplication[]>([])

const columns = [
  { colKey: 'user', title: '申请人', cell: 'user', minWidth: 180 },
  { colKey: 'status', title: '状态', cell: 'status', width: 110 },
  { colKey: 'reason', title: '申请理由', cell: 'reason', minWidth: 220 },
  { colKey: 'created_at', title: '申请时间', cell: 'created_at', width: 170 },
  { colKey: 'actions', title: '操作/审核人', cell: 'actions', width: 170 },
]

const statusLabel = (status: ExamTeacherApplicationStatus) => {
  if (status === 'approved') return '已通过'
  if (status === 'rejected') return '已拒绝'
  return '待审核'
}

const statusTheme = (status: ExamTeacherApplicationStatus) => {
  if (status === 'approved') return 'success'
  if (status === 'rejected') return 'danger'
  return 'warning'
}

const formatDate = (value?: string) => {
  if (!value) return ''
  return new Date(value).toLocaleString()
}

const loadApplications = async () => {
  loading.value = true
  try {
    const res = await listTeacherApplications(statusFilter.value || undefined)
    applications.value = res.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '班主任申请加载失败')
  } finally {
    loading.value = false
  }
}

const review = async (id: string, approve: boolean) => {
  reviewingId.value = id
  try {
    if (approve) {
      await approveTeacherApplication(id, { review_note: '通过班主任申请' })
      MessagePlugin.success('已通过班主任申请')
    } else {
      await rejectTeacherApplication(id, { review_note: '暂不通过班主任申请' })
      MessagePlugin.success('已拒绝班主任申请')
    }
    await loadApplications()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '审核失败')
  } finally {
    reviewingId.value = ''
  }
}

onMounted(loadApplications)
</script>

<style lang="less" scoped>
.teacher-applications {
  min-width: 0;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
}

.user-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;

  strong {
    font-size: 13px;
    font-weight: 600;
  }

  span {
    color: var(--td-text-color-secondary);
    font-size: 12px;
  }
}

.line-clamp {
  display: -webkit-box;
  overflow: hidden;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  color: var(--td-text-color-secondary);
  line-height: 20px;
}

.muted {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

@media (max-width: 760px) {
  .toolbar {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
