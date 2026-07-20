<template>
  <div class="settings-panel">
    <div class="settings-heading">
      <div>
        <h3>班级设置</h3>
        <p>基本信息与班级状态</p>
      </div>
      <t-tag :theme="isArchived ? 'default' : 'success'" variant="light">
        {{ isArchived ? '已归档' : '运行中' }}
      </t-tag>
    </div>

    <t-alert
      v-if="!canManage"
      theme="info"
      message="仅班主任可以修改班级设置。"
      class="permission-alert"
    />

    <t-form
      v-if="canManage"
      ref="formRef"
      :data="form"
      :rules="rules"
      label-align="top"
      class="settings-form"
    >
      <div class="settings-grid">
        <t-form-item label="班级名称" name="name">
          <t-input v-model="form.name" :disabled="isArchived" :maxlength="255" />
        </t-form-item>
        <t-form-item label="成员上限（0 为不限）" name="member_limit">
          <t-input-number v-model="form.member_limit" :disabled="isArchived" :min="0" :max="1000" />
        </t-form-item>
      </div>
      <t-form-item label="班级说明" name="description">
        <t-textarea
          v-model="form.description"
          :disabled="isArchived"
          :maxlength="2000"
          :autosize="{ minRows: 4, maxRows: 8 }"
        />
      </t-form-item>
      <div class="form-actions">
        <t-button
          theme="primary"
          :loading="submitting"
          :disabled="isArchived || commandSubmitting || !hasChanges"
          @click="submitSettings"
        >
          <template #icon><t-icon name="save" /></template>
          保存
        </t-button>
        <t-button
          variant="outline"
          :disabled="isArchived || submitting || commandSubmitting || !hasChanges"
          @click="resetForm"
        >
          <template #icon><t-icon name="rollback" /></template>
          重置
        </t-button>
      </div>
    </t-form>

    <dl v-else class="settings-summary">
      <div>
        <dt>班级名称</dt>
        <dd>{{ classInfo.name }}</dd>
      </div>
      <div>
        <dt>成员上限</dt>
        <dd>{{ classInfo.member_limit || '不限' }}</dd>
      </div>
      <div class="summary-description">
        <dt>班级说明</dt>
        <dd>{{ classInfo.description || '暂无说明' }}</dd>
      </div>
    </dl>

    <template v-if="canManage">
      <t-divider />
      <div class="status-command">
        <div>
          <h4>{{ isArchived ? '恢复班级' : '归档班级' }}</h4>
          <p>{{ isArchived ? '恢复后可继续班级业务。' : '历史成员、任务和答题记录将保留。' }}</p>
        </div>
        <t-button
          v-if="isArchived"
          theme="primary"
          variant="outline"
          :loading="commandSubmitting"
          :disabled="submitting"
          @click="restoreClass"
        >
          <template #icon><t-icon name="rollback" /></template>
          恢复
        </t-button>
        <t-button
          v-else
          theme="danger"
          variant="outline"
          :loading="commandSubmitting"
          :disabled="submitting"
          @click="confirmArchive"
        >
          <template #icon><t-icon name="folder" /></template>
          归档
        </t-button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import type { FormInstanceFunctions, FormRule } from 'tdesign-vue-next'
import { archiveExamClass, restoreExamClass, updateExamClass } from '@/api/exam/class'
import type { ExamClass, UpdateExamClassPayload } from '@/types/exam'
import {
  buildClassSettingsPayload,
  canManageClassSettings,
  isArchivedExamClass,
} from './classSettings'

const props = defineProps<{
  classInfo: ExamClass
  currentUserId: string
}>()

const emit = defineEmits<{
  updated: [classInfo: ExamClass]
}>()

const formRef = ref<FormInstanceFunctions>()
const submitting = ref(false)
const commandSubmitting = ref(false)
const form = ref<UpdateExamClassPayload>({ name: '', description: '', member_limit: 0 })
const formSnapshot = ref('')
const canManage = computed(() => canManageClassSettings(props.classInfo, props.currentUserId))
const isArchived = computed(() => isArchivedExamClass(props.classInfo))
const normalizedPayload = computed(() => buildClassSettingsPayload(form.value))
const hasChanges = computed(() => JSON.stringify(normalizedPayload.value) !== formSnapshot.value)

const rules: Record<string, FormRule[]> = {
  name: [{ required: true, message: '请输入班级名称', type: 'error' }],
}

const syncForm = (classInfo: ExamClass) => {
  form.value = {
    name: classInfo.name,
    description: classInfo.description || '',
    member_limit: classInfo.member_limit,
  }
  formSnapshot.value = JSON.stringify(buildClassSettingsPayload(form.value))
}

const resetForm = () => syncForm(props.classInfo)

const publishUpdate = (classInfo?: ExamClass) => {
  if (!classInfo) return
  syncForm(classInfo)
  emit('updated', classInfo)
}

const submitSettings = async () => {
  if (submitting.value || commandSubmitting.value || !canManage.value || isArchived.value) return
  const result = await formRef.value?.validate()
  if (result !== true) return
  const payload = normalizedPayload.value
  if (!payload.name || [...payload.name].length > 255 || [...payload.description].length > 2000) {
    MessagePlugin.error('班级信息长度不符合要求')
    return
  }
  submitting.value = true
  try {
    const response = await updateExamClass(props.classInfo.id, payload)
    publishUpdate(response.data)
    MessagePlugin.success('班级设置已保存')
  } catch (error: any) {
    MessagePlugin.error(error?.message || '班级设置保存失败')
  } finally {
    submitting.value = false
  }
}

const restoreClass = async () => {
  if (commandSubmitting.value || submitting.value || !canManage.value || !isArchived.value) return
  commandSubmitting.value = true
  try {
    const response = await restoreExamClass(props.classInfo.id)
    publishUpdate(response.data)
    MessagePlugin.success('班级已恢复')
  } catch (error: any) {
    MessagePlugin.error(error?.message || '恢复班级失败')
  } finally {
    commandSubmitting.value = false
  }
}

const confirmArchive = () => {
  if (commandSubmitting.value || submitting.value || !canManage.value || isArchived.value) return
  const dialog = DialogPlugin.confirm({
    header: '归档班级',
    body: '归档后将停止新的加入、资料、练习任务与催交操作，历史数据保留。',
    confirmBtn: { content: '确认归档', theme: 'danger' },
    theme: 'warning',
    onConfirm: async () => {
      commandSubmitting.value = true
      try {
        const response = await archiveExamClass(props.classInfo.id)
        publishUpdate(response.data)
        MessagePlugin.success('班级已归档')
        dialog.destroy()
      } catch (error: any) {
        MessagePlugin.error(error?.message || '归档班级失败')
      } finally {
        commandSubmitting.value = false
      }
    },
  })
}

watch(() => props.classInfo, syncForm, { immediate: true, deep: true })
</script>

<style src="./classSettingsPanel.less" lang="less" scoped></style>
