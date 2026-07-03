import { get, post, put } from '@/utils/request'

export type BillingPlanStatus = 'active' | 'archived'
export type BillingInterval = 'month' | 'year' | 'one_time'
export type SubscriptionStatus = 'active' | 'past_due' | 'canceled' | 'expired'
export type BillingOrderStatus = 'pending' | 'paid' | 'canceled' | 'failed'

export interface BillingPlan {
  id: string
  code: string
  name: string
  description: string
  currency: string
  amount_cents: number
  interval: BillingInterval
  entitlements: Record<string, unknown>
  status: BillingPlanStatus
  created_by_user_id: string
  created_at: string
  updated_at: string
}

export interface TenantSubscription {
  id: string
  tenant_id: number
  plan_id: string
  status: SubscriptionStatus
  current_period_start?: string
  current_period_end?: string
  source_order_id?: string
  created_by_user_id: string
  created_at: string
  updated_at: string
  plan?: BillingPlan
}

export interface BillingOrder {
  id: string
  tenant_id: number
  plan_id: string
  provider: string
  provider_order_id: string
  currency: string
  amount_cents: number
  status: BillingOrderStatus
  entitlements: Record<string, unknown>
  created_by_user_id: string
  paid_at?: string
  created_at: string
  updated_at: string
  plan?: BillingPlan
}

export interface TenantBillingStatus {
  tenant_id: number
  plan: BillingPlan
  subscription?: TenantSubscription
  is_free: boolean
}

export interface ApiResponse<T> {
  success: boolean
  data?: T
  message?: string
}

export interface PageResponse<T> extends ApiResponse<T[]> {
  total: number
  page: number
  page_size: number
}

export interface CreateBillingPlanPayload {
  code: string
  name: string
  description?: string
  currency?: string
  amount_cents: number
  interval?: BillingInterval
  entitlements?: Record<string, unknown>
}

export interface UpsertTenantSubscriptionPayload {
  plan_id: string
  status?: SubscriptionStatus
  current_period_start?: string
  current_period_end?: string
  source_order_id?: string
}

export interface CreateBillingOrderPayload {
  tenant_id: number
  plan_id: string
  provider?: string
  provider_order_id?: string
  status?: BillingOrderStatus
  paid_at?: string
}

export function getMyBilling() {
  return get('/api/v1/billing/me') as unknown as Promise<ApiResponse<TenantBillingStatus>>
}

export function listBillingPlans(includeArchived = false) {
  const qs = includeArchived ? '?include_archived=true' : ''
  return get(`/api/v1/system/admin/billing/plans${qs}`) as unknown as Promise<ApiResponse<BillingPlan[]>>
}

export function createBillingPlan(payload: CreateBillingPlanPayload) {
  return post('/api/v1/system/admin/billing/plans', payload) as unknown as Promise<ApiResponse<BillingPlan>>
}

export function listTenantSubscriptions(params: { tenant_id?: number; status?: SubscriptionStatus; page?: number; page_size?: number } = {}) {
  const qs = new URLSearchParams()
  if (params.tenant_id) qs.set('tenant_id', String(params.tenant_id))
  if (params.status) qs.set('status', params.status)
  if (params.page) qs.set('page', String(params.page))
  if (params.page_size) qs.set('page_size', String(params.page_size))
  const suffix = qs.toString()
  return get(`/api/v1/system/admin/billing/subscriptions${suffix ? `?${suffix}` : ''}`) as unknown as Promise<PageResponse<TenantSubscription>>
}

export function upsertTenantSubscription(tenantId: number, payload: UpsertTenantSubscriptionPayload) {
  return put(`/api/v1/system/admin/billing/subscriptions/${tenantId}`, payload) as unknown as Promise<ApiResponse<TenantSubscription>>
}

export function listBillingOrders(params: { tenant_id?: number; status?: BillingOrderStatus; page?: number; page_size?: number } = {}) {
  const qs = new URLSearchParams()
  if (params.tenant_id) qs.set('tenant_id', String(params.tenant_id))
  if (params.status) qs.set('status', params.status)
  if (params.page) qs.set('page', String(params.page))
  if (params.page_size) qs.set('page_size', String(params.page_size))
  const suffix = qs.toString()
  return get(`/api/v1/system/admin/billing/orders${suffix ? `?${suffix}` : ''}`) as unknown as Promise<PageResponse<BillingOrder>>
}

export function createBillingOrder(payload: CreateBillingOrderPayload) {
  return post('/api/v1/system/admin/billing/orders', payload) as unknown as Promise<ApiResponse<BillingOrder>>
}
