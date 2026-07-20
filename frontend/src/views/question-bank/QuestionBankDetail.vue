<template>
  <div class="exam-page">
    <div class="exam-header">
      <div>
        <t-button variant="text" size="small" @click="router.push('/platform/question-banks')">
          <template #icon><t-icon name="chevron-left" /></template>
          返回题库
        </t-button>
        <h2>{{ bank?.name || '题库详情' }}</h2>
        <p>{{ bank?.description || '结构化题目、答案、解析和知识点关联会沉淀在题库中。' }}</p>
      </div>
      <t-tag v-if="bank" :theme="reviewTag(bank.review_status)" variant="light">{{ reviewLabel(bank.review_status) }}</t-tag>
    </div>

    <t-loading :loading="loading">
      <div v-if="bank" class="summary-grid">
        <div class="summary-item">
          <span>空间 ID</span>
          <strong>{{ bank.space_id }}</strong>
        </div>
        <div class="summary-item">
          <span>考试域 ID</span>
          <strong>{{ bank.domain_id }}</strong>
        </div>
        <div class="summary-item">
          <span>来源</span>
          <strong>{{ bank.source_type }}</strong>
        </div>
      </div>

      <div class="content-grid">
        <section class="panel question-section">
          <div class="panel-title">
            <h3>题组</h3>
            <p>已确认的阅读篇章、数学题干和小题会以题组形态沉淀为正式题库资产。</p>
            <span>{{ questionGroups.length }} 组</span>
          </div>
          <div v-if="questionGroups.length" class="question-group-review">
            <div class="question-group-review__toolbar">
              <div>
                <strong>{{ activeGroupTitle }}</strong>
                <span>{{ groupPagerLabel }}</span>
              </div>
              <div class="question-group-review__actions">
                <t-button
                  size="small"
                  variant="outline"
                  :disabled="!canMoveGroup(-1)"
                  @click="moveGroup(-1)"
                >
                  <template #icon><t-icon name="chevron-left" /></template>
                  上一大题
                </t-button>
                <t-button
                  size="small"
                  theme="primary"
                  variant="outline"
                  :disabled="!canMoveGroup(1)"
                  @click="moveGroup(1)"
                >
                  <template #icon><t-icon name="chevron-right" /></template>
                  下一大题
                </t-button>
              </div>
            </div>

            <div v-if="questionGroups.length > 1" class="question-group-review__tabs" role="tablist">
              <button
                v-for="(group, index) in questionGroups"
                :key="group.group.id"
                type="button"
                class="question-group-review__tab"
                :class="{ 'is-active': currentGroupIndex === index }"
                :aria-selected="currentGroupIndex === index"
                @click="setActiveGroupIndex(index)"
              >
                {{ groupTabTitle(group, index) }}
              </button>
            </div>

            <div class="question-group-list">
              <article v-for="group in visibleQuestionGroups" :key="group.group.id" class="question-group-card">
              <header class="question-group-card__head">
                <div>
                  <h4>{{ group.group.title || group.group.group_type }}</h4>
                  <p>{{ group.group.group_type }} · {{ group.questions.length }} 题</p>
                </div>
                <t-tag :theme="reviewTag(group.group.review_status)" variant="light">
                  {{ reviewLabel(group.group.review_status) }}
                </t-tag>
              </header>

              <section v-if="group.group.material_text" class="question-group-material">
                <div class="question-group-material__head">
                  <div>
                    <h5>{{ materialTitle(group) }}</h5>
                    <span>{{ materialMeta(group) }}</span>
                  </div>
                  <t-button
                    v-if="canToggleMaterial(group)"
                    variant="text"
                    size="small"
                    @click="toggleMaterial(group)"
                  >
                    <template #icon>
                      <t-icon :name="isMaterialExpanded(group) ? 'chevron-up' : 'chevron-down'" />
                    </template>
                    {{ isMaterialExpanded(group) ? '收起原文' : '展开全文' }}
                  </t-button>
                </div>
                <div
                  class="question-group-material__body"
                  :class="{ 'is-expanded': isMaterialExpanded(group), 'is-collapsible': canToggleMaterial(group) }"
                >
                  <ExamRichText :content="group.group.material_text" />
                </div>
              </section>

              <div v-if="group.assets?.length" class="question-group-assets">
                <div v-for="asset in group.assets" :key="asset.id" class="question-group-asset">
                  {{ assetLabel(asset) }}
                </div>
              </div>

              <div class="question-review">
                <div class="question-review__toolbar">
                  <div>
                    <strong>{{ activeQuestionTitle(group) }}</strong>
                    <span>{{ questionPagerLabel(group) }}</span>
                  </div>
                  <div class="question-review__actions">
                    <t-button
                      size="small"
                      variant="outline"
                      :disabled="!canMoveQuestion(group, -1)"
                      @click="moveQuestion(group, -1)"
                    >
                      <template #icon><t-icon name="chevron-left" /></template>
                      上一题
                    </t-button>
                    <t-button
                      size="small"
                      theme="primary"
                      variant="outline"
                      :disabled="!canMoveQuestion(group, 1)"
                      @click="moveQuestion(group, 1)"
                    >
                      <template #icon><t-icon name="chevron-right" /></template>
                      下一题
                    </t-button>
                  </div>
                </div>

                <div v-if="group.questions.length > 1" class="question-review__tabs" role="tablist">
                  <button
                    v-for="(item, index) in group.questions"
                    :key="item.question.id"
                    type="button"
                    class="question-review__tab"
                    :class="{ 'is-active': activeQuestionIndex(group) === index }"
                    :aria-selected="activeQuestionIndex(group) === index"
                    @click="setActiveQuestionIndex(group, index)"
                  >
                    {{ questionNo(item) }}
                  </button>
                </div>

                <article v-for="item in visibleQuestions(group)" :key="item.question.id" class="question-item">
                  <div class="question-item__no">{{ questionNo(item) }}</div>
                  <div class="question-item__content">
                    <ExamRichText class="question-stem" :content="item.question.stem" inline />
                    <div v-if="item.options?.length" class="question-options">
                      <div
                        v-for="option in item.options"
                        :key="option.id || `${item.question.id}-${option.option_key}`"
                        class="question-option"
                        :class="{ 'is-correct': isCorrectOption(item, option.option_key) }"
                      >
                        <span class="question-option__key">{{ option.option_key }}</span>
                        <ExamRichText class="question-option__content" :content="option.content" inline />
                        <t-tag v-if="isCorrectOption(item, option.option_key)" size="small" theme="success" variant="light">
                          正确答案
                        </t-tag>
                      </div>
                    </div>
                    <div class="question-answer">
                      <div class="question-answer__item">
                        <span>答案：</span>
                        <ExamRichText :content="answerSummary(item)" inline />
                      </div>
                      <div v-if="explanationSummary(item)" class="question-answer__item">
                        <span>解析：</span>
                        <ExamRichText :content="explanationSummary(item)" inline />
                      </div>
                    </div>
                  </div>
                </article>
              </div>
              </article>
            </div>
          </div>
          <t-empty v-else description="暂无结构化题组" />
        </section>

        <aside class="side-stack">
          <section class="panel review-entry-panel">
            <div class="panel-title review-entry-title">
              <div>
                <h3>题组审核</h3>
                <p>最近一次结构化任务</p>
              </div>
              <t-tag v-if="latestReviewTask" :theme="reviewTaskTheme(latestReviewTask.status)" variant="light">
                {{ reviewTaskStatusLabel(latestReviewTask.status) }}
              </t-tag>
            </div>

            <t-loading :loading="reviewEntryLoading">
              <div v-if="latestReviewTask" class="review-entry">
                <div class="review-entry__stats">
                  <div>
                    <span>待审核</span>
                    <strong>{{ reviewDraftStats ? reviewDraftStats.pending_review : '-' }}</strong>
                  </div>
                  <div>
                    <span>已入库</span>
                    <strong>{{ reviewDraftStats ? reviewDraftStats.approved : '-' }}</strong>
                  </div>
                  <div>
                    <span>质量错误</span>
                    <strong>{{ reviewDraftStats ? reviewDraftErrorCount : '-' }}</strong>
                  </div>
                </div>
                <div class="review-entry__meta">
                  <span>{{ formatDateTime(latestReviewTask.updated_at) }}</span>
                  <span>{{ reviewDraftStats ? `${reviewDraftStats.total} 个草稿` : '统计暂不可用' }}</span>
                </div>
                <t-button theme="primary" block @click="openLatestReviewTask">
                  <template #icon><t-icon name="check-circle" /></template>
                  进入题组审核
                </t-button>
              </div>
              <t-empty v-else size="small" description="暂无结构化审核任务" />
            </t-loading>
          </section>

          <section class="panel">
          <div class="panel-title">
            <h3>处理链路</h3>
            <p>题库不是文档 chunk 的替代，而是引用 chunk 形成可练习、可讲解的结构化资产。</p>
          </div>
          <div class="pipeline">
            <div><span>1</span>资料入库</div>
            <div><span>2</span>题目边界识别</div>
            <div><span>3</span>答案解析关联</div>
            <div><span>4</span>写入题库并引用 chunk</div>
          </div>
          </section>

          <section class="panel rag-diagnostic-panel">
            <div class="panel-title rag-diagnostic-title">
              <div>
                <h3>RAG 诊断</h3>
                <p>用固定评测问题检查当前题库绑定知识库的召回、结构化上下文和答案覆盖。</p>
              </div>
              <t-button size="small" theme="primary" :loading="ragDiagnosticLoading" @click="runRagDiagnostic">
                <template #icon><t-icon name="search" /></template>
                运行
              </t-button>
            </div>

            <t-loading :loading="ragDiagnosticLoading">
              <t-empty
                v-if="!ragDiagnostic"
                size="small"
                description="暂无诊断结果"
              />
              <div v-else class="rag-diagnostic">
                <div class="rag-diagnostic__summary">
                  <div>
                    <span>总体</span>
                    <strong>{{ ragDiagnostic.summary.passed }} / {{ ragDiagnostic.summary.total }}</strong>
                  </div>
                  <div>
                    <span>答案覆盖</span>
                    <strong>{{ rateLabel(ragDiagnostic.summary.answer_hit_rate) }}</strong>
                  </div>
                  <div>
                    <span>召回覆盖</span>
                    <strong>{{ rateLabel(ragDiagnostic.summary.retrieval_hit_rate) }}</strong>
                  </div>
                </div>

                <div class="rag-diagnostic__meta">
                  <span>{{ ragDiagnostic.used_default_cases ? '默认评测集' : '自定义评测集' }}</span>
                  <span>{{ ragDiagnostic.knowledge_base_ids.length }} 个知识库</span>
                </div>

                <div class="rag-diagnostic__cases">
                  <article
                    v-for="item in ragDiagnostic.summary.results"
                    :key="item.name"
                    class="rag-diagnostic-case"
                  >
                    <header>
                      <div>
                        <strong>{{ item.name }}</strong>
                        <p>{{ item.query }}</p>
                      </div>
                      <t-tag size="small" :theme="diagnosticTagTheme(item)" variant="light">
                        {{ diagnosticStatusLabel(item) }}
                      </t-tag>
                    </header>
                    <div class="rag-diagnostic-case__rates">
                      <span>答案 {{ rateLabel(item.answer_score) }}</span>
                      <span>召回 {{ rateLabel(item.retrieval_score) }}</span>
                    </div>
                    <div v-if="item.context_label" class="rag-diagnostic-case__label">
                      {{ item.context_label }}
                    </div>
                    <div class="rag-diagnostic-case__phrases">
                      <span v-if="item.matched_phrases.length">命中：{{ compactList(item.matched_phrases) }}</span>
                      <span v-if="item.missing_phrases.length" class="is-missing">缺失：{{ compactList(item.missing_phrases) }}</span>
                    </div>
                    <div class="rag-diagnostic-case__chunks">
                      chunk：{{ compactList(item.retrieved_chunk_ids) || '-' }}
                    </div>
                    <p v-if="item.error" class="rag-diagnostic-case__error">{{ item.error }}</p>
                  </article>
                </div>
              </div>
            </t-loading>
            <t-button class="rag-observability-entry" variant="outline" block @click="openRAGEvaluationCenter">
              <template #icon><t-icon name="chart-bubble" /></template>
              进入评测中心
            </t-button>
            <t-button class="rag-observability-entry" variant="outline" block @click="openAgentEvaluationCenter">
              <template #icon><t-icon name="system-sum" /></template>
              Agent 行为评测
            </t-button>
          </section>
        </aside>
      </div>
    </t-loading>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import { getQuestionBank, listQuestionGroupDetails, runQuestionBankRAGDiagnostic } from '@/api/exam/question-bank'
import { listExamStructuringTasks } from '@/api/exam/material'
import { listQuestionGroupDrafts } from '@/api/exam/question-group-draft'
import type { ExamQuestionGroupDraftStats, ExamRAGDiagnosticResult, ExamRAGDiagnosticResultItem, ExamStructuringTask, ExamStructuringTaskStatus, QuestionBank, QuestionDetail, QuestionGroupAsset, QuestionGroupDetail, ReviewStatus } from '@/types/exam'
import ExamRichText from './ExamRichText.vue'
import { loadQuestionBankReviewEntry } from './questionBankReviewEntry'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const bank = ref<QuestionBank | null>(null)
const questionGroups = ref<QuestionGroupDetail[]>([])
const expandedMaterials = ref<Record<string, boolean>>({})
const activeQuestionIndexes = ref<Record<string, number>>({})
const activeGroupIndex = ref(0)
const ragDiagnosticLoading = ref(false)
const ragDiagnostic = ref<ExamRAGDiagnosticResult | null>(null)
const reviewEntryLoading = ref(false)
const latestReviewTask = ref<ExamStructuringTask | null>(null)
const reviewDraftStats = ref<ExamQuestionGroupDraftStats | null>(null)
const reviewDraftErrorCount = ref(0)

const materialToggleLength = 420
const materialToggleLines = 8

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

const assetLabel = (asset: QuestionGroupAsset) => {
  return [asset.asset_type, asset.alt_text || asset.storage_uri].filter(Boolean).join(' · ') || '资源'
}

const groupCount = computed(() => questionGroups.value.length)

const clampGroupIndex = (index: number) => {
  const count = groupCount.value
  if (count <= 0) return 0
  return Math.min(Math.max(index, 0), count - 1)
}

const currentGroupIndex = computed(() => clampGroupIndex(activeGroupIndex.value))

const groupTypeLabel = (group: QuestionGroupDetail) => {
  const map: Record<string, string> = {
    reading_passage: '阅读',
    math_problem: '数学题组',
    single_question: '单题',
    cloze: '完型',
    essay: '作文',
  }
  return map[group.group.group_type] || group.group.group_type || '题组'
}

const groupDisplayTitle = (group: QuestionGroupDetail, index: number) => {
  return group.group.title || `第 ${index + 1} 组${groupTypeLabel(group)}`
}

const visibleQuestionGroups = computed(() => {
  const group = questionGroups.value[currentGroupIndex.value]
  return group ? [group] : []
})

const activeGroupTitle = computed(() => {
  const group = visibleQuestionGroups.value[0]
  return group ? groupDisplayTitle(group, currentGroupIndex.value) : '暂无题组'
})

const groupPagerLabel = computed(() => {
  return groupCount.value ? `${currentGroupIndex.value + 1} / ${groupCount.value}` : '0 / 0'
})

const groupTabTitle = (group: QuestionGroupDetail, index: number) => {
  return `${index + 1}. ${group.group.title || groupTypeLabel(group)}`
}

const setActiveGroupIndex = (index: number) => {
  activeGroupIndex.value = clampGroupIndex(index)
}

const moveGroup = (delta: number) => {
  setActiveGroupIndex(currentGroupIndex.value + delta)
}

const canMoveGroup = (delta: number) => {
  const count = groupCount.value
  if (count <= 1) return false
  const nextIndex = currentGroupIndex.value + delta
  return nextIndex >= 0 && nextIndex < count
}

const questionNo = (item: QuestionDetail) => {
  return item.question.question_no || String(item.question.order_in_group || '-')
}

const questionCount = (group: QuestionGroupDetail) => {
  return group.questions?.length || 0
}

const clampQuestionIndex = (group: QuestionGroupDetail, index: number) => {
  const count = questionCount(group)
  if (count <= 0) return 0
  return Math.min(Math.max(index, 0), count - 1)
}

const activeQuestionIndex = (group: QuestionGroupDetail) => {
  return clampQuestionIndex(group, activeQuestionIndexes.value[group.group.id] ?? 0)
}

const setActiveQuestionIndex = (group: QuestionGroupDetail, index: number) => {
  activeQuestionIndexes.value = {
    ...activeQuestionIndexes.value,
    [group.group.id]: clampQuestionIndex(group, index),
  }
}

const moveQuestion = (group: QuestionGroupDetail, delta: number) => {
  setActiveQuestionIndex(group, activeQuestionIndex(group) + delta)
}

const canMoveQuestion = (group: QuestionGroupDetail, delta: number) => {
  const count = questionCount(group)
  if (count <= 1) return false
  const nextIndex = activeQuestionIndex(group) + delta
  return nextIndex >= 0 && nextIndex < count
}

const visibleQuestions = (group: QuestionGroupDetail) => {
  const item = group.questions?.[activeQuestionIndex(group)]
  return item ? [item] : []
}

const activeQuestionTitle = (group: QuestionGroupDetail) => {
  const item = visibleQuestions(group)[0]
  return item ? `当前题目 ${questionNo(item)}` : '暂无题目'
}

const questionPagerLabel = (group: QuestionGroupDetail) => {
  const count = questionCount(group)
  return count ? `${activeQuestionIndex(group) + 1} / ${count}` : '0 / 0'
}

const answerValues = (item: QuestionDetail) => {
  return item.answers?.map(answer => answer.answer_text).filter(Boolean) || []
}

const answerSummary = (item: QuestionDetail) => {
  return answerValues(item).join('，') || '-'
}

const answerKeys = (item: QuestionDetail) => {
  return answerValues(item)
    .flatMap(value => value.toUpperCase().split(/[\s,，、;；/]+/))
    .map(value => value.trim())
    .filter(Boolean)
}

const isCorrectOption = (item: QuestionDetail, optionKey: string) => {
  return answerKeys(item).includes(optionKey.toUpperCase())
}

const explanationSummary = (item: QuestionDetail) => {
  return item.explanations?.map(explanation => explanation.explanation_text).filter(Boolean).join('；') || ''
}

const canToggleMaterial = (group: QuestionGroupDetail) => {
  const text = group.group.material_text || ''
  return text.length > materialToggleLength || text.split(/\r?\n/).length > materialToggleLines
}

const isMaterialExpanded = (group: QuestionGroupDetail) => {
  return Boolean(expandedMaterials.value[group.group.id])
}

const toggleMaterial = (group: QuestionGroupDetail) => {
  expandedMaterials.value = {
    ...expandedMaterials.value,
    [group.group.id]: !expandedMaterials.value[group.group.id],
  }
}

const materialTitle = (group: QuestionGroupDetail) => {
  if (group.group.group_type === 'reading_passage') return '阅读原文'
  if (group.group.group_type === 'math_problem') return '题干材料'
  return '共享材料'
}

const materialMeta = (group: QuestionGroupDetail) => {
  return group.group.source_chunk_ids?.length ? `引用 ${group.group.source_chunk_ids.length} 个 chunk` : '已沉淀为题组材料'
}

const rateLabel = (value: number) => {
  return `${Math.round((value || 0) * 100)}%`
}

const compactList = (items: string[]) => {
  return (items || []).filter(Boolean).slice(0, 6).join('、')
}

const diagnosticTagTheme = (item: ExamRAGDiagnosticResultItem) => {
  if (item.passed) return 'success'
  if (!item.answer_passed) return 'danger'
  return 'warning'
}

const diagnosticStatusLabel = (item: ExamRAGDiagnosticResultItem) => {
  if (item.passed) return '通过'
  if (!item.answer_passed) return '答案缺失'
  return '召回待查'
}

const reviewTaskStatusLabel = (status: ExamStructuringTaskStatus) => {
  const labels: Record<ExamStructuringTaskStatus, string> = {
    pending: '待处理',
    ready_for_review: '待抽取',
    blocked: '已阻塞',
    extracting: '抽取中',
    reviewing: '待审核',
    completed: '已完成',
    failed: '失败',
  }
  return labels[status] || status
}

const reviewTaskTheme = (status: ExamStructuringTaskStatus) => {
  if (status === 'completed') return 'success'
  if (status === 'failed' || status === 'blocked') return 'danger'
  if (status === 'reviewing' || status === 'ready_for_review') return 'warning'
  return 'primary'
}

const formatDateTime = (value?: string) => {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-CN', { hour12: false })
}

const openLatestReviewTask = () => {
  if (!latestReviewTask.value) return
  router.push(`/platform/question-group-drafts/${latestReviewTask.value.id}`)
}

const openRAGEvaluationCenter = () => {
  router.push(`/platform/question-banks/${String(route.params.bankId || '')}/rag-observability`)
}

const openAgentEvaluationCenter = () => {
  router.push(`/platform/question-banks/${String(route.params.bankId || '')}/agent-evaluation`)
}

const loadReviewEntry = async (bankId: string, spaceId: string) => {
  reviewEntryLoading.value = true
  try {
    const result = await loadQuestionBankReviewEntry(bankId, spaceId, {
      listTasks: listExamStructuringTasks,
      listDrafts: listQuestionGroupDrafts,
    })
    latestReviewTask.value = result.task
    reviewDraftStats.value = result.stats
    reviewDraftErrorCount.value = result.pendingErrorCount
  } catch {
    latestReviewTask.value = null
    reviewDraftStats.value = null
    reviewDraftErrorCount.value = 0
  } finally {
    reviewEntryLoading.value = false
  }
}

const runRagDiagnostic = async () => {
  const bankId = String(route.params.bankId || '')
  if (!bankId) return
  ragDiagnosticLoading.value = true
  try {
    const res = await runQuestionBankRAGDiagnostic(bankId)
    ragDiagnostic.value = res.data
    MessagePlugin.success('RAG 诊断完成')
  } catch (error: any) {
    MessagePlugin.error(error?.message || 'RAG 诊断失败')
  } finally {
    ragDiagnosticLoading.value = false
  }
}

const loadData = async () => {
  const bankId = String(route.params.bankId || '')
  if (!bankId) return
  loading.value = true
  try {
    const res = await getQuestionBank(bankId)
    bank.value = res.data
    void loadReviewEntry(bankId, res.data.space_id)
    const groupRes = await listQuestionGroupDetails(bankId)
    questionGroups.value = groupRes.data || []
    activeGroupIndex.value = clampGroupIndex(activeGroupIndex.value)
  } catch (error: any) {
    MessagePlugin.error(error?.message || '题库详情加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
</script>

<style lang="less" scoped src="./QuestionBankDetail.less"></style>
