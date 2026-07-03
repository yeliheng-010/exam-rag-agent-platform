<template>
  <div class="exam-page">
    <div class="exam-header">
      <div>
        <h2>支付与权益</h2>
        <p>当前工作区可用的解析、班级和智能体额度。</p>
      </div>
      <t-button theme="primary" variant="outline" :loading="loading" @click="loadBilling">
        <template #icon><t-icon name="refresh" /></template>
        刷新
      </t-button>
    </div>

    <t-alert v-if="error" class="page-alert" theme="error" :message="error">
      <template #operation>
        <t-button size="small" @click="loadBilling">重试</t-button>
      </template>
    </t-alert>

    <t-loading :loading="loading">
      <div v-if="billing" class="plan-summary">
        <div class="plan-main">
          <span class="eyebrow">当前套餐</span>
          <strong>{{ billing.plan?.name || '-' }}</strong>
          <p>{{ billing.plan?.description || '平台默认权益' }}</p>
        </div>
        <div class="plan-side">
          <t-tag :theme="billing.is_free ? 'default' : 'success'" variant="light">
            {{ billing.is_free ? '免费权益' : '订阅权益' }}
          </t-tag>
          <span>{{ formatMoney(billing.plan?.amount_cents, billing.plan?.currency) }} / {{ intervalLabel(billing.plan?.interval) }}</span>
        </div>
      </div>

      <div class="metric-grid">
        <div v-for="item in entitlementCards" :key="item.key" class="metric-card">
          <span>{{ item.label }}</span>
          <strong>{{ item.value }}</strong>
          <p>{{ item.desc }}</p>
        </div>
      </div>

      <div class="panel-grid">
        <section class="panel">
          <h3>订阅状态</h3>
          <t-descriptions :column="1" size="small" bordered>
            <t-descriptions-item label="租户 ID">{{ billing?.tenant_id || '-' }}</t-descriptions-item>
            <t-descriptions-item label="订阅状态">{{ subscriptionLabel }}</t-descriptions-item>
            <t-descriptions-item label="当前周期">{{ currentPeriodText }}</t-descriptions-item>
          </t-descriptions>
        </section>

        <section class="panel">
          <h3>权益明细</h3>
          <t-list split>
            <t-list-item v-for="item in entitlementRows" :key="item.key">
              <span class="entitlement-row">
                <span>{{ item.key }}</span>
                <strong>{{ item.value }}</strong>
              </span>
            </t-list-item>
          </t-list>
        </section>
      </div>
    </t-loading>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { getMyBilling } from '@/api/billing'
import type { TenantBillingStatus, BillingInterval, SubscriptionStatus } from '@/api/billing'

const loading = ref(false)
const error = ref('')
const billing = ref<TenantBillingStatus | null>(null)

const entitlementLabels: Record<string, { label: string; desc: string }> = {
  document_parse_monthly: { label: '文档解析额度', desc: '学习资料、试卷 PDF 等入库解析的月度额度。' },
  exam_structuring_monthly: { label: '试卷结构化额度', desc: '试卷题目、答案和解析结构化的月度额度。' },
  agent_calls_monthly: { label: 'Agent 工具调用', desc: '智能体检索、分析和工具执行的月度调用额度。' },
  class_member_limit: { label: '班级席位', desc: '单个班级可容纳的学生人数上限。' },
}

const entitlementCards = computed(() => {
  const entitlements = billing.value?.plan?.entitlements || {}
  return Object.entries(entitlementLabels).map(([key, meta]) => ({
    key,
    ...meta,
    value: formatEntitlement(entitlements[key]),
  }))
})

const entitlementRows = computed(() => {
  const entitlements = billing.value?.plan?.entitlements || {}
  return Object.entries(entitlements).map(([key, value]) => ({
    key,
    value: formatEntitlement(value),
  }))
})

const subscriptionLabel = computed(() => {
  const status = billing.value?.subscription?.status
  if (!status) return billing.value?.is_free ? '无订阅，使用免费权益' : '-'
  return subscriptionStatusLabel(status)
})

const currentPeriodText = computed(() => {
  const sub = billing.value?.subscription
  if (!sub?.current_period_start && !sub?.current_period_end) return '-'
  return `${formatDate(sub.current_period_start)} 至 ${formatDate(sub.current_period_end)}`
})

const loadBilling = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await getMyBilling()
    if (res.success && res.data) {
      billing.value = res.data
    } else {
      throw new Error(res.message || '权益状态加载失败')
    }
  } catch (err: any) {
    error.value = err?.message || '权益状态加载失败'
    MessagePlugin.error(error.value)
  } finally {
    loading.value = false
  }
}

const formatEntitlement = (value: unknown) => {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'number') return value.toLocaleString()
  if (typeof value === 'boolean') return value ? '已开启' : '未开启'
  return String(value)
}

const formatMoney = (amount?: number, currency = 'CNY') => {
  if (!amount) return `${currency} 0`
  return `${currency} ${(amount / 100).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

const intervalLabel = (interval?: BillingInterval) => {
  if (interval === 'year') return '年'
  if (interval === 'one_time') return '一次性'
  return '月'
}

const subscriptionStatusLabel = (status: SubscriptionStatus) => {
  if (status === 'active') return '生效中'
  if (status === 'past_due') return '待补缴'
  if (status === 'canceled') return '已取消'
  return '已过期'
}

const formatDate = (value?: string) => {
  if (!value) return '-'
  return new Date(value).toLocaleDateString()
}

onMounted(loadBilling)
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

.page-alert {
  margin-bottom: 12px;
}

.plan-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 18px;
  margin-bottom: 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
}

.plan-main {
  min-width: 0;

  .eyebrow {
    display: block;
    margin-bottom: 6px;
    color: var(--td-text-color-secondary);
    font-size: 12px;
  }

  strong {
    display: block;
    font-size: 24px;
    line-height: 32px;
    font-weight: 600;
  }

  p {
    margin: 6px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 13px;
    line-height: 20px;
  }
}

.plan-side {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  flex: 0 0 auto;
  color: var(--td-text-color-secondary);
  font-size: 13px;
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

.entitlement-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
  font-size: 13px;

  span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--td-text-color-secondary);
  }

  strong {
    flex: 0 0 auto;
    font-weight: 600;
  }
}

.metric-card {
  padding: 16px;

  span {
    display: block;
    margin-bottom: 10px;
    color: var(--td-text-color-secondary);
    font-size: 13px;
  }

  strong {
    display: block;
    font-size: 24px;
    line-height: 32px;
    font-weight: 600;
  }

  p {
    margin: 8px 0 0;
    color: var(--td-text-color-placeholder);
    font-size: 12px;
    line-height: 18px;
  }
}

.panel-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.panel {
  padding: 16px;

  h3 {
    margin: 0 0 6px;
    font-size: 16px;
    font-weight: 600;
  }

  p {
    margin: 0 0 14px;
    color: var(--td-text-color-secondary);
    font-size: 14px;
    line-height: 22px;
  }
}

@media (max-width: 980px) {
  .metric-grid,
  .panel-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 680px) {
  .exam-page {
    padding: 20px 16px;
  }

  .exam-header,
  .plan-summary {
    flex-direction: column;
    align-items: stretch;
  }

  .metric-grid,
  .panel-grid {
    grid-template-columns: 1fr;
  }
}
</style>
