<template>
  <div class="billing-admin">
    <div class="toolbar">
      <t-space size="small">
        <t-button theme="primary" @click="planDialogVisible = true">
          <template #icon><t-icon name="add" /></template>
          新建套餐
        </t-button>
        <t-button variant="outline" :loading="loading" @click="loadAll">
          <template #icon><t-icon name="refresh" /></template>
          刷新
        </t-button>
      </t-space>
    </div>

    <t-tabs v-model="activeTab" size="medium">
      <t-tab-panel value="plans" label="套餐">
        <div class="data-table-shell">
          <t-table row-key="id" :data="plans" :columns="planColumns" size="small" hover :loading="loading">
            <template #price="{ row }">
              {{ formatMoney(row.amount_cents, row.currency) }} / {{ intervalLabel(row.interval) }}
            </template>
            <template #status="{ row }">
              <t-tag :theme="row.status === 'active' ? 'success' : 'default'" variant="light">
                {{ row.status === 'active' ? '启用' : '归档' }}
              </t-tag>
            </template>
            <template #entitlements="{ row }">
              <span class="mono">{{ compactEntitlements(row.entitlements) }}</span>
            </template>
          </t-table>
        </div>
      </t-tab-panel>

      <t-tab-panel value="subscriptions" label="订阅">
        <div class="data-table-shell">
          <t-table
            row-key="tenant_id"
            :data="subscriptions"
            :columns="subscriptionColumns"
            size="small"
            hover
            :loading="loading"
          >
            <template #plan="{ row }">{{ row.plan?.name || row.plan_id }}</template>
            <template #status="{ row }">
              <t-tag :theme="subscriptionTheme(row.status)" variant="light">
                {{ subscriptionLabel(row.status) }}
              </t-tag>
            </template>
            <template #period="{ row }">
              {{ formatDate(row.current_period_start) }} 至 {{ formatDate(row.current_period_end) }}
            </template>
          </t-table>
        </div>
      </t-tab-panel>

      <t-tab-panel value="orders" label="订单">
        <div class="data-table-shell">
          <t-table row-key="id" :data="orders" :columns="orderColumns" size="small" hover :loading="loading">
            <template #plan="{ row }">{{ row.plan?.name || row.plan_id }}</template>
            <template #amount="{ row }">{{ formatMoney(row.amount_cents, row.currency) }}</template>
            <template #status="{ row }">
              <t-tag :theme="orderTheme(row.status)" variant="light">
                {{ orderLabel(row.status) }}
              </t-tag>
            </template>
            <template #created_at="{ row }">{{ formatDateTime(row.created_at) }}</template>
          </t-table>
        </div>
      </t-tab-panel>
    </t-tabs>

    <t-dialog
      v-model:visible="planDialogVisible"
      header="新建套餐"
      width="560px"
      :confirm-loading="creatingPlan"
      @confirm="submitPlan"
    >
      <t-form :data="planForm" label-width="92px">
        <t-form-item label="套餐代码">
          <t-input v-model="planForm.code" placeholder="如 ielts-pro" />
        </t-form-item>
        <t-form-item label="套餐名称">
          <t-input v-model="planForm.name" placeholder="如 雅思专业版" />
        </t-form-item>
        <t-form-item label="价格">
          <t-input-number v-model="planForm.amountYuan" theme="normal" :min="0" :decimal-places="2" suffix="元" />
        </t-form-item>
        <t-form-item label="周期">
          <t-select v-model="planForm.interval">
            <t-option value="month" label="月" />
            <t-option value="year" label="年" />
            <t-option value="one_time" label="一次性" />
          </t-select>
        </t-form-item>
        <t-form-item label="描述">
          <t-textarea v-model="planForm.description" :autosize="{ minRows: 2, maxRows: 4 }" />
        </t-form-item>
        <t-form-item label="权益 JSON">
          <t-textarea v-model="planForm.entitlementsText" :autosize="{ minRows: 5, maxRows: 8 }" />
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  createBillingPlan,
  listBillingOrders,
  listBillingPlans,
  listTenantSubscriptions,
} from '@/api/billing'
import type {
  BillingInterval,
  BillingOrder,
  BillingOrderStatus,
  BillingPlan,
  SubscriptionStatus,
  TenantSubscription,
} from '@/api/billing'

const activeTab = ref('plans')
const loading = ref(false)
const creatingPlan = ref(false)
const planDialogVisible = ref(false)
const plans = ref<BillingPlan[]>([])
const subscriptions = ref<TenantSubscription[]>([])
const orders = ref<BillingOrder[]>([])

const planForm = ref({
  code: '',
  name: '',
  description: '',
  amountYuan: 0,
  interval: 'month' as BillingInterval,
  entitlementsText: JSON.stringify({
    class_member_limit: 100,
    document_parse_monthly: 500,
    exam_structuring_monthly: 100,
    agent_calls_monthly: 5000,
  }, null, 2),
})

const planColumns = [
  { colKey: 'code', title: '代码', width: 130 },
  { colKey: 'name', title: '名称', minWidth: 160 },
  { colKey: 'price', title: '价格', cell: 'price', width: 150 },
  { colKey: 'status', title: '状态', cell: 'status', width: 90 },
  { colKey: 'entitlements', title: '权益', cell: 'entitlements', minWidth: 260 },
]

const subscriptionColumns = [
  { colKey: 'tenant_id', title: '租户', width: 100 },
  { colKey: 'plan', title: '套餐', cell: 'plan', minWidth: 160 },
  { colKey: 'status', title: '状态', cell: 'status', width: 100 },
  { colKey: 'period', title: '周期', cell: 'period', minWidth: 220 },
]

const orderColumns = [
  { colKey: 'id', title: '订单', minWidth: 220 },
  { colKey: 'tenant_id', title: '租户', width: 90 },
  { colKey: 'plan', title: '套餐', cell: 'plan', minWidth: 150 },
  { colKey: 'amount', title: '金额', cell: 'amount', width: 120 },
  { colKey: 'provider', title: '渠道', width: 90 },
  { colKey: 'status', title: '状态', cell: 'status', width: 90 },
  { colKey: 'created_at', title: '创建时间', cell: 'created_at', width: 170 },
]

const loadAll = async () => {
  loading.value = true
  try {
    const [planRes, subRes, orderRes] = await Promise.all([
      listBillingPlans(true),
      listTenantSubscriptions({ page_size: 50 }),
      listBillingOrders({ page_size: 50 }),
    ])
    plans.value = planRes.data || []
    subscriptions.value = subRes.data || []
    orders.value = orderRes.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '支付与权益数据加载失败')
  } finally {
    loading.value = false
  }
}

const submitPlan = async () => {
  creatingPlan.value = true
  try {
    const entitlements = JSON.parse(planForm.value.entitlementsText || '{}')
    await createBillingPlan({
      code: planForm.value.code,
      name: planForm.value.name,
      description: planForm.value.description,
      amount_cents: Math.round(Number(planForm.value.amountYuan || 0) * 100),
      currency: 'CNY',
      interval: planForm.value.interval,
      entitlements,
    })
    MessagePlugin.success('套餐已创建')
    planDialogVisible.value = false
    await loadAll()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '套餐创建失败，请检查权益 JSON')
  } finally {
    creatingPlan.value = false
  }
}

const formatMoney = (amount?: number, currency = 'CNY') => `${currency} ${((amount || 0) / 100).toLocaleString(undefined, {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})}`

const intervalLabel = (interval?: BillingInterval) => {
  if (interval === 'year') return '年'
  if (interval === 'one_time') return '一次性'
  return '月'
}

const compactEntitlements = (value: Record<string, unknown>) => {
  const entries = Object.entries(value || {})
  if (!entries.length) return '{}'
  return entries.map(([key, val]) => `${key}:${val}`).join(' / ')
}

const subscriptionLabel = (status: SubscriptionStatus) => {
  if (status === 'active') return '生效'
  if (status === 'past_due') return '待补缴'
  if (status === 'canceled') return '取消'
  return '过期'
}

const subscriptionTheme = (status: SubscriptionStatus) => {
  if (status === 'active') return 'success'
  if (status === 'past_due') return 'warning'
  return 'default'
}

const orderLabel = (status: BillingOrderStatus) => {
  if (status === 'paid') return '已支付'
  if (status === 'failed') return '失败'
  if (status === 'canceled') return '取消'
  return '待支付'
}

const orderTheme = (status: BillingOrderStatus) => {
  if (status === 'paid') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'canceled') return 'default'
  return 'warning'
}

const formatDate = (value?: string) => (value ? new Date(value).toLocaleDateString() : '-')
const formatDateTime = (value?: string) => (value ? new Date(value).toLocaleString() : '-')

onMounted(loadAll)
</script>

<style lang="less" scoped>
.billing-admin {
  min-width: 0;
}

.toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 14px;
}

.data-table-shell {
  overflow-x: auto;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);

  :deep(.t-table td),
  :deep(.t-table th) {
    padding-top: 12px;
    padding-bottom: 12px;
  }
}

.mono {
  font-family: var(--td-font-family-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  font-size: 12px;
  color: var(--td-text-color-secondary);
}
</style>
