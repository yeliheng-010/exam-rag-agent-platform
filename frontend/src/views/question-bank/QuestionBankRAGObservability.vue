<template>
  <div class="rag-page">
    <header class="rag-header">
      <div>
        <t-button variant="text" size="small" @click="router.push(`/platform/question-banks/${bankId}`)">
          <template #icon><t-icon name="chevron-left" /></template>
          返回题库
        </t-button>
        <h2>{{ bank?.name || 'RAG 评测中心' }}</h2>
      </div>
      <t-button theme="primary" :loading="creating" @click="createRun">
        <template #icon><t-icon name="play-circle" /></template>
        运行评测
      </t-button>
    </header>

    <section class="parameter-band" aria-label="评测参数">
      <label>
        <span>MatchCount</span>
        <t-input-number v-model="parameters.match_count" :min="1" :max="50" />
      </label>
      <label class="threshold-control">
        <span>向量阈值 <strong>{{ parameters.vector_threshold.toFixed(2) }}</strong></span>
        <t-slider v-model="parameters.vector_threshold" :min="0" :max="1" :step="0.05" />
      </label>
      <label class="threshold-control">
        <span>关键词阈值 <strong>{{ parameters.keyword_threshold.toFixed(2) }}</strong></span>
        <t-slider v-model="parameters.keyword_threshold" :min="0" :max="1" :step="0.05" />
      </label>
    </section>

    <t-loading :loading="loading">
      <t-empty v-if="!runs.length && !loading" description="暂无评测运行">
        <t-button theme="primary" @click="createRun">运行首次评测</t-button>
      </t-empty>
      <template v-else-if="activeRun">
        <section class="metric-band" aria-label="评测指标">
          <div v-for="metric in activeMetrics" :key="metric.label">
            <span>{{ metric.label }}</span>
            <strong>{{ metric.value }}</strong>
          </div>
        </section>

        <section v-if="comparisonRuns.length === 2" class="comparison-band">
          <header><h3>运行对比</h3><span>{{ shortID(comparisonRuns[0].id) }} / {{ shortID(comparisonRuns[1].id) }}</span></header>
          <div class="comparison-table">
            <div v-for="row in comparisonRows" :key="row.key" :class="{ 'is-changed': row.changed }">
              <span>{{ row.label }}</span><strong>{{ row.first }}</strong><strong>{{ row.second }}</strong>
            </div>
          </div>
        </section>

        <div class="rag-workspace">
          <aside class="run-history">
            <header><h3>运行历史</h3><span>选择两次可对比</span></header>
            <button
              v-for="run in runs"
              :key="run.id"
              type="button"
              class="run-row"
              :class="{ 'is-active': run.id === activeRun.id }"
              @click="selectRun(run.id)"
            >
              <t-checkbox
                :checked="selectedComparisonRunIds.includes(run.id)"
                @click.stop
                @change="toggleComparison(run.id)"
              />
              <div class="run-row__summary">
                <span class="run-row__time">{{ formatDate(run.created_at) }}</span>
                <span>TopK {{ run.request_snapshot.match_count }}</span>
              </div>
              <span class="run-row__rate">{{ runPassRate(run) }}</span>
              <t-tag size="small" variant="light" :theme="statusTheme(run.status)">{{ runStatusLabel(run.status) }}</t-tag>
            </button>
          </aside>

          <main class="run-result">
            <header class="run-result__head">
              <div><h3>运行详情</h3><code>{{ activeRun.id }}</code></div>
              <span>{{ activeRun.progress.completed_cases }} / {{ activeRun.progress.total_cases }}</span>
            </header>
            <t-progress
              v-if="isWorking(activeRun)"
              :percentage="progressPercentage(activeRun)"
              :label="false"
              theme="line"
            />
            <t-alert v-if="activeRun.status === 'failed'" theme="error" :message="activeRun.error_message || '评测运行失败'" />
            <RAGEvaluationRunDetail :run="activeRun" />
          </main>
        </div>
      </template>
    </t-loading>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  createQuestionBankRAGEvaluationRun,
  getQuestionBank,
  getQuestionBankRAGEvaluationRun,
  listQuestionBankRAGEvaluationRuns,
} from '@/api/exam/question-bank'
import type { ExamRAGEvaluationRun, ExamRAGEvaluationRunStatus, QuestionBank } from '@/types/exam'
import RAGEvaluationRunDetail from './RAGEvaluationRunDetail.vue'
import { compareRunConfigurations, compareRunMetrics, formatRAGRate, getRunMetrics, runStatusLabel } from './ragEvaluationViewModel'

const route = useRoute()
const router = useRouter()
const bankId = computed(() => String(route.params.bankId || ''))
const bank = ref<QuestionBank | null>(null)
const runs = ref<ExamRAGEvaluationRun[]>([])
const activeRun = ref<ExamRAGEvaluationRun | null>(null)
const selectedComparisonRunIds = ref<string[]>([])
const loading = ref(false)
const creating = ref(false)
const parameters = reactive({ match_count: 8, vector_threshold: 0.5, keyword_threshold: 0.3 })
let pollTimer: ReturnType<typeof setInterval> | undefined
let pollInFlight = false
let selectionRequestID = 0

const activeMetrics = computed(() => getRunMetrics(activeRun.value))
const comparisonRuns = computed(() => selectedComparisonRunIds.value
  .map(id => runs.value.find(run => run.id === id))
  .filter((run): run is ExamRAGEvaluationRun => Boolean(run)))
const comparisonRows = computed(() => comparisonRuns.value.length === 2
  ? [
      ...compareRunConfigurations(comparisonRuns.value[0], comparisonRuns.value[1]),
      ...compareRunMetrics(comparisonRuns.value[0], comparisonRuns.value[1]),
    ]
  : [])

const isWorking = (run: ExamRAGEvaluationRun) => ['queued', 'running'].includes(run.status)
const progressPercentage = (run: ExamRAGEvaluationRun) => run.progress.total_cases
  ? Math.round(run.progress.completed_cases * 100 / run.progress.total_cases)
  : 0
const shortID = (id: string) => id.slice(0, 8)
const formatDate = (value: string) => new Date(value).toLocaleString('zh-CN', { hour12: false })
const runPassRate = (run: ExamRAGEvaluationRun) => run.result_snapshot
  ? formatRAGRate(run.result_snapshot.summary.hit_rate)
  : '--'
const statusTheme = (status: ExamRAGEvaluationRunStatus) => status === 'completed'
  ? 'success'
  : status === 'failed' ? 'danger' : 'primary'

const fetchRun = async (runId: string) => {
  const response = await getQuestionBankRAGEvaluationRun(bankId.value, runId)
  return response.data
}

const replaceRun = (run: ExamRAGEvaluationRun) => {
  const index = runs.value.findIndex(item => item.id === run.id)
  if (index >= 0) runs.value.splice(index, 1, run)
}

const selectRun = async (runId: string) => {
  const requestID = ++selectionRequestID
  const run = await fetchRun(runId)
  replaceRun(run)
  if (requestID === selectionRequestID) activeRun.value = run
}

const loadRuns = async () => {
  const response = await listQuestionBankRAGEvaluationRuns(bankId.value)
  runs.value = response.data || []
  const selected = runs.value.find(run => run.id === activeRun.value?.id) || runs.value[0]
  if (selected) await selectRun(selected.id)
}

const createRun = async () => {
  creating.value = true
  try {
    const response = await createQuestionBankRAGEvaluationRun(bankId.value, parameters)
    runs.value.unshift(response.data)
    activeRun.value = response.data
    MessagePlugin.success('评测已进入队列')
  } catch (error: any) {
    MessagePlugin.error(error?.message || '创建评测失败')
  } finally {
    creating.value = false
  }
}

const toggleComparison = (runId: string) => {
  const selected = selectedComparisonRunIds.value
  if (selected.includes(runId)) {
    selectedComparisonRunIds.value = selected.filter(id => id !== runId)
  } else if (selected.length < 2) {
    selectedComparisonRunIds.value = [...selected, runId]
  } else {
    MessagePlugin.warning('最多选择两次运行')
  }
}

const pollActiveRun = async () => {
  if (pollInFlight || !activeRun.value || !isWorking(activeRun.value)) return
  const runId = activeRun.value.id
  pollInFlight = true
  try {
    const run = await fetchRun(runId)
    replaceRun(run)
    if (activeRun.value?.id !== runId) return
    activeRun.value = run
    if (!isWorking(run)) await loadRuns()
  } catch {
    // 轮询错误由下一次请求自动恢复。
  } finally {
    pollInFlight = false
  }
}

onMounted(async () => {
  loading.value = true
  try {
    const [bankResponse] = await Promise.all([getQuestionBank(bankId.value), loadRuns()])
    bank.value = bankResponse.data
  } catch (error: any) {
    MessagePlugin.error(error?.message || '评测中心加载失败')
  } finally {
    loading.value = false
  }
  pollTimer = setInterval(pollActiveRun, 1800)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style lang="less" scoped src="./QuestionBankRAGObservability.less"></style>
<style lang="less" scoped>
.run-row__summary {
  display: grid;
  gap: 3px;
  min-width: 0;

  span {
    color: var(--td-text-color-secondary);
    font-size: 11px;
  }
}

.run-row__time {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.run-row__rate {
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 600;
}
</style>
