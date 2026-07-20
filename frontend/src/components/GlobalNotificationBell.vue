<template>
  <t-badge :count="badgeCount" :max-count="99" :offset="[6, 4]" class="global-notification-bell">
    <button
      type="button"
      class="global-notification-bell__button"
      title="通知"
      aria-label="通知"
      @click="dialogVisible = true"
    >
      <t-icon name="notification" size="18px" />
    </button>
  </t-badge>

  <ExamAssignmentNotificationDialog
    v-model:visible="dialogVisible"
    :data="notificationData"
    :loading="loading"
    :load-failed="loadFailed"
    :pending-invitation-count="pendingInvitationCount"
    @refresh="loadNotifications"
  />
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { listAssignmentNotifications } from '@/api/exam/assignmentNotification'
import { notificationBadgeCount } from '@/views/classes/assignmentNotification'
import type { ExamAssignmentNotificationList } from '@/types/exam'
import ExamAssignmentNotificationDialog from '@/components/ExamAssignmentNotificationDialog.vue'

const authStore = useAuthStore()
const dialogVisible = ref(false)
const loading = ref(false)
const loadFailed = ref(false)
const notificationData = ref<ExamAssignmentNotificationList | null>(null)
let pollTimer: ReturnType<typeof setInterval> | undefined

const pendingInvitationCount = computed(() => authStore.pendingInvitationCount)
const badgeCount = computed(() => notificationBadgeCount(
  notificationData.value?.unread_count || 0,
  pendingInvitationCount.value,
))

const loadNotifications = async () => {
  if (loading.value) return
  loading.value = true
  try {
    const response = await listAssignmentNotifications(50)
    notificationData.value = response.data
    loadFailed.value = false
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

const stopPolling = () => {
  if (pollTimer) clearInterval(pollTimer)
  pollTimer = undefined
}

const startPolling = () => {
  stopPolling()
  if (!document.hidden) pollTimer = setInterval(loadNotifications, 30_000)
}

const handleVisibilityChange = () => {
  if (document.hidden) {
    stopPolling()
    return
  }
  void loadNotifications()
  startPolling()
}

onMounted(() => {
  void loadNotifications()
  startPolling()
  document.addEventListener('visibilitychange', handleVisibilityChange)
})

onUnmounted(() => {
  stopPolling()
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})
</script>

<style lang="less" scoped>
.global-notification-bell {
  position: fixed;
  top: 12px;
  right: 16px;
  z-index: 100;
}

.global-notification-bell__button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.04);

  &:hover {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-brand-color);
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color-focus);
    outline-offset: 1px;
  }
}
</style>
