<template>
  <div class="exam-page">
    <div class="exam-header">
      <div>
        <h2>班级中心</h2>
        <p>老师创建班级，学生通过班级关系访问对应资料、题库与练习任务。</p>
      </div>
      <div class="header-actions">
        <t-button theme="default" variant="outline" @click="openJoinDialog">
          <template #icon><t-icon name="login" /></template>
          加入班级
        </t-button>
        <t-button v-if="canCreateClass" theme="primary" @click="openCreateDialog">
          <template #icon><t-icon name="add" /></template>
          创建班级
        </t-button>
      </div>
    </div>

    <t-loading :loading="loading">
      <div v-if="classes.length" class="class-grid">
        <div v-for="item in classes" :key="item.id" class="class-card" @click="router.push(`/platform/classes/${item.id}`)">
          <div class="card-header">
            <div>
              <h3>{{ item.name }}</h3>
              <p>{{ item.description || '暂无班级说明' }}</p>
            </div>
            <t-tag :theme="item.status === 'active' ? 'success' : 'default'" variant="light">{{ statusLabel(item.status) }}</t-tag>
          </div>
          <div class="card-meta">
            <span><t-icon name="usergroup" />成员上限 {{ item.member_limit }}</span>
            <span><t-icon name="folder" />{{ domainName(item.domain_id) }}</span>
          </div>
          <div class="card-footer">
            <span>{{ formatDate(item.created_at) }}</span>
            <t-button variant="text" size="small">进入</t-button>
          </div>
        </div>
      </div>
      <t-empty v-else-if="!loading" description="暂无班级">
        <template #action>
          <t-space>
            <t-button theme="default" variant="outline" @click="openJoinDialog">输入邀请码加入</t-button>
            <t-button v-if="canCreateClass" theme="primary" @click="openCreateDialog">创建第一个班级</t-button>
          </t-space>
        </template>
      </t-empty>
    </t-loading>

    <t-dialog
      v-model:visible="createVisible"
      header="创建班级"
      :confirm-btn="{ content: '创建', loading: creating }"
      @confirm="submitCreate"
    >
      <t-form ref="formRef" :data="form" :rules="rules" label-align="top">
        <t-form-item label="班级名称" name="name">
          <t-input v-model="form.name" placeholder="例如：高三英语冲刺班" :maxlength="80" />
        </t-form-item>
        <t-form-item label="考试域" name="domain_id">
          <t-select v-model="form.domain_id" clearable placeholder="选择高考或雅思">
            <t-option v-for="item in domains" :key="item.id" :value="item.id" :label="item.name" />
          </t-select>
        </t-form-item>
        <t-form-item label="成员上限" name="member_limit">
          <t-input-number v-model="form.member_limit" :min="1" :max="500" />
        </t-form-item>
        <t-form-item label="说明" name="description">
          <t-textarea v-model="form.description" placeholder="记录班级用途、阶段目标或招生说明" :maxlength="500" />
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog
      v-model:visible="joinVisible"
      header="加入班级"
      :confirm-btn="{ content: '提交申请', loading: joining }"
      @confirm="submitJoin"
    >
      <t-form ref="joinFormRef" :data="joinForm" :rules="joinRules" label-align="top">
        <t-form-item label="班级邀请码" name="invite_code">
          <t-input v-model="joinForm.invite_code" placeholder="输入老师提供的邀请码" :maxlength="32" />
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import type { FormInstanceFunctions, FormRule } from 'tdesign-vue-next'
import { listExamDomains } from '@/api/exam/domain'
import { createExamClass, listExamClasses, requestJoinExamClass } from '@/api/exam/class'
import { useAuthStore } from '@/stores/auth'
import type { ExamClass, ExamDomain } from '@/types/exam'

const router = useRouter()
const authStore = useAuthStore()
const loading = ref(false)
const creating = ref(false)
const joining = ref(false)
const createVisible = ref(false)
const joinVisible = ref(false)
const classes = ref<ExamClass[]>([])
const domains = ref<ExamDomain[]>([])
const formRef = ref<FormInstanceFunctions>()
const joinFormRef = ref<FormInstanceFunctions>()
const canCreateClass = computed(() => authStore.hasRole('contributor'))

const form = ref({
  name: '',
  description: '',
  domain_id: undefined as string | undefined,
  member_limit: 50,
})

const joinForm = ref({
  invite_code: '',
})

const rules: Record<string, FormRule[]> = {
  name: [{ required: true, message: '请输入班级名称', type: 'error' }],
}

const joinRules: Record<string, FormRule[]> = {
  invite_code: [{ required: true, message: '请输入班级邀请码', type: 'error' }],
}

const domainName = (domainId?: string) => {
  if (!domainId) return '未绑定考试域'
  return domains.value.find(item => item.id === domainId)?.name || '未知考试域'
}

const statusLabel = (status: string) => status === 'active' ? '运行中' : '已归档'

const formatDate = (value?: string) => {
  if (!value) return ''
  return new Date(value).toLocaleDateString()
}

const loadData = async () => {
  loading.value = true
  try {
    const [classRes, domainRes] = await Promise.all([listExamClasses(), listExamDomains()])
    classes.value = classRes.data || []
    domains.value = domainRes.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '班级列表加载失败')
  } finally {
    loading.value = false
  }
}

const openCreateDialog = () => {
  if (!canCreateClass.value) return
  form.value = {
    name: '',
    description: '',
    domain_id: domains.value[0]?.id,
    member_limit: 50,
  }
  createVisible.value = true
}

const openJoinDialog = () => {
  joinForm.value = { invite_code: '' }
  joinVisible.value = true
}

const submitCreate = async () => {
  const result = await formRef.value?.validate()
  if (result !== true) return

  creating.value = true
  try {
    const payload = {
      ...form.value,
      domain_id: form.value.domain_id || undefined,
    }
    const res = await createExamClass(payload)
    MessagePlugin.success('班级已创建')
    createVisible.value = false
    await loadData()
    if (res.data?.id) {
      router.push(`/platform/classes/${res.data.id}`)
    }
  } catch (error: any) {
    MessagePlugin.error(error?.message || '创建班级失败')
  } finally {
    creating.value = false
  }
}

const submitJoin = async () => {
  const result = await joinFormRef.value?.validate()
  if (result !== true) return

  joining.value = true
  try {
    const member = await requestJoinExamClass(joinForm.value.invite_code.trim())
    joinVisible.value = false
    if (member.data?.status === 'active') {
      MessagePlugin.success('你已在该班级中')
    } else {
      MessagePlugin.success('加入申请已提交，等待老师审核')
    }
    await loadData()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '加入班级失败')
  } finally {
    joining.value = false
  }
}

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
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.class-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}

.class-card {
  display: flex;
  flex-direction: column;
  min-height: 172px;
  padding: 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  cursor: pointer;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;

  &:hover {
    border-color: var(--td-brand-color);
    box-shadow: 0 4px 14px rgba(7, 192, 95, 0.10);
  }
}

.card-header {
  display: flex;
  justify-content: space-between;
  gap: 12px;

  h3 {
    margin: 0;
    font-size: 16px;
    line-height: 24px;
    font-weight: 600;
  }

  p {
    display: -webkit-box;
    margin: 6px 0 0;
    overflow: hidden;
    color: var(--td-text-color-secondary);
    font-size: 13px;
    line-height: 20px;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }
}

.card-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 18px;

  span {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    height: 24px;
    padding: 0 8px;
    border-radius: 6px;
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-secondary);
    font-size: 12px;
  }
}

.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: auto;
  padding-top: 14px;
  border-top: 1px solid var(--td-component-stroke);
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}
</style>
