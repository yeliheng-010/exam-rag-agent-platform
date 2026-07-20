<template>
  <t-dialog
    :visible="visible"
    header="通知"
    width="560px"
    :footer="false"
    class="exam-notification-dialog"
    @update:visible="emit('update:visible', $event)"
  >
    <div class="notification-toolbar">
      <button
        v-if="pendingInvitationCount > 0"
        type="button"
        class="notification-toolbar__command"
        @click="invitationVisible = true"
      >
        <t-icon name="usergroup-add" />
        待处理邀请
        <span>{{ pendingInvitationCount }}</span>
      </button>
      <button
        type="button"
        class="notification-toolbar__command notification-toolbar__command--end"
        :disabled="!data?.unread_count || markAllLoading"
        @click="markAllRead"
      >
        <t-icon name="check" />
        全部已读
      </button>
    </div>

    <t-loading :loading="loading">
      <div v-if="loadFailed" class="notification-state">
        <span>通知加载失败</span>
        <t-button variant="text" size="small" @click="emit('refresh')">重试</t-button>
      </div>
      <div v-else-if="!data?.items?.length" class="notification-state">暂无通知</div>
      <div v-else class="notification-list">
        <button
          v-for="item in data.items"
          :key="item.notification.id"
          type="button"
          class="notification-row"
          :class="{ 'notification-row--unread': !item.notification.read_at }"
          :disabled="readingId === item.notification.id"
          @click="openNotification(item)"
        >
          <span class="notification-row__icon" :title="notificationKindLabel(item.notification.kind)">
            <t-icon :name="notificationKindIcon(item.notification.kind)" />
          </span>
          <span class="notification-row__body">
            <span class="notification-row__heading">
              <strong>{{ item.notification.title }}</strong>
              <i v-if="!item.notification.read_at" aria-label="未读" />
            </span>
            <span class="notification-row__content">{{ item.notification.content }}</span>
            <time :datetime="item.notification.created_at">{{ formatNotificationTime(item.notification.created_at) }}</time>
          </span>
          <t-icon name="chevron-right" class="notification-row__arrow" />
        </button>
      </div>
    </t-loading>
  </t-dialog>

  <MyInvitationsDialog v-model:visible="invitationVisible" />
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import MyInvitationsDialog from '@/components/MyInvitationsDialog.vue'
import {
  markAllAssignmentNotificationsRead,
  markAssignmentNotificationRead,
} from '@/api/exam/assignmentNotification'
import { assignmentNotificationTarget } from '@/views/classes/assignmentNotification'
import type {
  ExamAssignmentNotificationItem,
  ExamAssignmentNotificationKind,
  ExamAssignmentNotificationList,
} from '@/types/exam'

defineProps<{
  visible: boolean
  data: ExamAssignmentNotificationList | null
  loading: boolean
  loadFailed: boolean
  pendingInvitationCount: number
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  refresh: []
}>()

const router = useRouter()
const invitationVisible = ref(false)
const readingId = ref('')
const markAllLoading = ref(false)

const markAllRead = async () => {
  markAllLoading.value = true
  try {
    await markAllAssignmentNotificationsRead()
    emit('refresh')
  } catch (error: any) {
    MessagePlugin.error(error?.message || '标记已读失败')
  } finally {
    markAllLoading.value = false
  }
}

const openNotification = async (item: ExamAssignmentNotificationItem) => {
  readingId.value = item.notification.id
  try {
    if (!item.notification.read_at) {
      await markAssignmentNotificationRead(item.notification.id)
      emit('refresh')
    }
    const target = assignmentNotificationTarget(item)
    if (target.kind === 'attempt') {
      emit('update:visible', false)
      await router.push(`/platform/practice/question-groups/${target.groupId}?attempt_id=${target.attemptId}`)
      return
    }
    if (target.kind === 'assignment') {
      emit('update:visible', false)
      await router.push({ path: '/platform/learning', query: { assignment_id: target.assignmentId } })
      return
    }
    MessagePlugin.warning('该任务当前不可开始，已有答题记录仍可从练习记录查看')
  } catch (error: any) {
    MessagePlugin.error(error?.message || '打开通知失败')
  } finally {
    readingId.value = ''
  }
}

const notificationKindLabel = (kind: ExamAssignmentNotificationKind) => ({
  published: '新作业',
  republished: '重新发布',
  withdrawn: '已撤回',
  reminder: '催交提醒',
})[kind]

const notificationKindIcon = (kind: ExamAssignmentNotificationKind) => ({
  published: 'task',
  republished: 'refresh',
  withdrawn: 'rollback',
  reminder: 'send',
})[kind]

const formatNotificationTime = (value: string) => {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '-' : date.toLocaleString()
}
</script>

<style lang="less" scoped>
.notification-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 36px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.notification-toolbar__command {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 32px;
  padding: 0 8px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;

  span {
    color: var(--td-brand-color);
    font-weight: 600;
  }

  &:hover:not(:disabled) {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-primary);
  }

  &:disabled {
    cursor: not-allowed;
    opacity: 0.45;
  }
}

.notification-toolbar__command--end {
  margin-left: auto;
}

.notification-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 180px;
  color: var(--td-text-color-placeholder);
}

.notification-list {
  max-height: min(60vh, 560px);
  overflow-y: auto;
}

.notification-row {
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr) 20px;
  align-items: start;
  width: 100%;
  padding: 14px 4px;
  border: 0;
  border-bottom: 1px solid var(--td-component-stroke);
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: pointer;

  &:hover:not(:disabled) {
    background: var(--td-bg-color-secondarycontainer);
  }

  &:disabled {
    cursor: wait;
    opacity: 0.65;
  }
}

.notification-row--unread {
  background: var(--td-brand-color-light);
}

.notification-row__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  color: var(--td-brand-color);
}

.notification-row__body,
.notification-row__heading {
  min-width: 0;
}

.notification-row__body {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.notification-row__heading {
  display: flex;
  align-items: center;
  gap: 7px;

  strong {
    overflow-wrap: anywhere;
    font-size: 14px;
    line-height: 1.35;
  }

  i {
    flex: 0 0 auto;
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--td-brand-color);
  }
}

.notification-row__content {
  overflow-wrap: anywhere;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.notification-row time {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.notification-row__arrow {
  align-self: center;
  color: var(--td-text-color-placeholder);
}

:global(.exam-notification-dialog .t-dialog) {
  max-width: calc(100vw - 24px);
}

@media (max-width: 600px) {
  .notification-toolbar {
    flex-wrap: wrap;
  }

  .notification-row {
    grid-template-columns: 30px minmax(0, 1fr) 18px;
    padding: 12px 2px;
  }
}
</style>
