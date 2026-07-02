<template>
  <div class="admin-page">
    <div class="admin-header">
      <div class="admin-header-main">
        <h2>{{ $t('platformAdmin.title') }}</h2>
        <p>{{ $t('platformAdmin.subtitle') }}</p>
      </div>
      <div class="admin-context">
        <span class="admin-context-label">{{ authStore.currentTenantName || '-' }}</span>
        <t-tag :theme="roleTagTheme" variant="light">
          {{ roleLabel }}
        </t-tag>
      </div>
    </div>

    <t-tabs v-model="activeTab" class="admin-tabs">
      <t-tab-panel value="teacherApplications" label="班主任申请">
        <section class="admin-section">
          <TeacherApplications />
        </section>
      </t-tab-panel>
      <t-tab-panel value="members" :label="$t('platformAdmin.tabs.members')">
        <section class="admin-section">
          <TenantMembers />
        </section>
      </t-tab-panel>
    </t-tabs>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import TeacherApplications from '@/views/admin/TeacherApplications.vue'
import TenantMembers from '@/views/settings/TenantMembers.vue'

type TenantRole = 'owner' | 'admin' | 'contributor' | 'viewer'

const { t } = useI18n()
const authStore = useAuthStore()
const activeTab = ref('teacherApplications')

const currentRole = computed(() => (authStore.currentTenantRole || 'viewer') as TenantRole)
const roleLabel = computed(() => t(`tenantMember.role.${currentRole.value}`))
const roleTagTheme = computed(() => (currentRole.value === 'owner' ? 'success' : 'primary'))
</script>

<style lang="less" scoped>
.admin-page {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  padding: 28px 32px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
}

.admin-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;
}

.admin-header-main {
  min-width: 0;

  h2 {
    margin: 0;
    font-size: 22px;
    line-height: 30px;
    font-weight: 600;
  }

  p {
    max-width: 720px;
    margin: 6px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 14px;
    line-height: 22px;
  }
}

.admin-context {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
  max-width: min(340px, 40vw);
  padding: 6px 8px 6px 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
}

.admin-context-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--td-text-color-primary);
  font-size: 13px;
  font-weight: 600;
}

.admin-tabs {
  :deep(.t-tabs__nav-container) {
    margin-bottom: 14px;
  }

  :deep(.t-tabs__content) {
    padding-top: 0;
  }
}

.admin-section {
  padding: 18px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

@media (max-width: 760px) {
  .admin-page {
    padding: 20px 16px;
  }

  .admin-header {
    flex-direction: column;
  }

  .admin-context {
    max-width: 100%;
  }

  .admin-section {
    padding: 14px;
  }
}
</style>
