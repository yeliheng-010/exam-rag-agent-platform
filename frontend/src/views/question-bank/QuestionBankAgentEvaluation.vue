<template>
  <div class="rag-page agent-page">
    <header class="rag-header">
      <div>
        <t-button variant="text" size="small" @click="router.push(`/platform/question-banks/${bankId}`)">
          <template #icon><t-icon name="chevron-left" /></template>
          返回题库
        </t-button>
        <h2>{{ bank?.name || 'Agent 行为评测' }}</h2>
      </div>
      <div class="rag-header__actions">
        <t-radio-group value="agent" variant="default-filled" @change="openEvaluationMode">
          <t-radio-button value="rag">RAG</t-radio-button>
          <t-radio-button value="agent">Agent</t-radio-button>
        </t-radio-group>
        <t-button theme="primary" :loading="creating" :disabled="!selectedAgentId" @click="createRun">
          <template #icon><t-icon name="play-circle" /></template>
          运行评测
        </t-button>
      </div>
    </header>

    <section class="agent-config-band" aria-label="Agent 配置">
      <label>
        <span>智能推理 Agent</span>
        <t-select v-model="selectedAgentId" :options="agentOptions" placeholder="选择 Agent" filterable />
      </label>
      <div class="agent-config-meta">
        <span><small>模型</small><strong :title="selectedModelName">{{ selectedModelName }}</strong></span>
        <span><small>知识库</small><strong>{{ selectedAgent?.config.knowledge_bases?.length || 0 }}</strong></span>
        <span><small>只读工具</small><strong>{{ selectedAgentTools.length }}</strong></span>
      </div>
    </section>

    <t-alert v-if="selectedAgent && !selectedAgentTools.length" theme="warning" message="该 Agent 没有可评测的只读工具" />
    <AgentEvaluationScenarioEditor v-model="scenarios" :tools="selectedAgentTools" />

    <t-divider />
    <t-loading :loading="loading">
      <t-empty v-if="!runs.length && !loading" description="暂无 Agent 评测运行" />
      <template v-else-if="activeRun">
        <section class="metric-band agent-metric-band" aria-label="Agent 评测指标">
          <div v-for="metric in activeMetrics" :key="metric.label"><span>{{ metric.label }}</span><strong>{{ metric.value }}</strong></div>
        </section>

        <div class="rag-workspace">
          <aside class="run-history">
            <header><h3>运行历史</h3><span>{{ runs.length }} 次</span></header>
            <button
              v-for="run in runs"
              :key="run.id"
              type="button"
              class="run-row agent-run-row"
              :class="{ 'is-active': run.id === activeRun.id }"
              @click="selectRun(run.id)"
            >
              <div class="run-row__summary">
                <span class="run-row__time">{{ formatDate(run.created_at) }}</span>
                <strong>{{ run.request_snapshot.agent.name || run.agent_id }}</strong>
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
            <t-progress v-if="isWorking(activeRun)" :percentage="progressPercentage(activeRun)" :label="false" theme="line" />
            <t-alert v-if="activeRun.status === 'failed'" theme="error" :message="activeRun.error_message || '评测运行失败'" />
            <AgentEvaluationRunDetail :run="activeRun" />
          </main>
        </div>
      </template>
    </t-loading>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import { listAgents, type CustomAgent } from '@/api/agent'
import { listModels, type ModelConfig } from '@/api/model'
import {
  createQuestionBankAgentEvaluationRun,
  getQuestionBank,
  getQuestionBankAgentEvaluationRun,
  listQuestionBankAgentEvaluationRuns,
} from '@/api/exam/question-bank'
import type { ExamAgentEvaluationRun, ExamRAGEvaluationRunStatus, QuestionBank } from '@/types/exam'
import AgentEvaluationRunDetail from './AgentEvaluationRunDetail.vue'
import AgentEvaluationScenarioEditor from './AgentEvaluationScenarioEditor.vue'
import {
  buildAgentEvaluationCases,
  createAgentEvaluationScenario,
  formatAgentRate,
  getAgentRunMetrics,
  getSafeAgentEvaluationTools,
  resolveAgentModelName,
  runStatusLabel,
} from './agentEvaluationViewModel'

const route = useRoute()
const router = useRouter()
const bankId = computed(() => String(route.params.bankId || ''))
const bank = ref<QuestionBank | null>(null)
const agents = ref<CustomAgent[]>([])
const models = ref<ModelConfig[]>([])
const selectedAgentId = ref('')
const scenarios = ref([createAgentEvaluationScenario(1)])
const runs = ref<ExamAgentEvaluationRun[]>([])
const activeRun = ref<ExamAgentEvaluationRun | null>(null)
const loading = ref(false)
const creating = ref(false)
let pollTimer: ReturnType<typeof setInterval> | undefined
let pollInFlight = false
let selectionRequestID = 0

const selectedAgent = computed(() => agents.value.find(agent => agent.id === selectedAgentId.value))
const selectedModelName = computed(() => resolveAgentModelName(selectedAgent.value?.config.model_id, models.value))
const selectedAgentTools = computed(() => getSafeAgentEvaluationTools(selectedAgent.value?.config.allowed_tools || []))
const agentOptions = computed(() => agents.value.map(agent => ({ label: agent.name, value: agent.id })))
const activeMetrics = computed(() => getAgentRunMetrics(activeRun.value))
const isWorking = (run: ExamAgentEvaluationRun) => ['queued', 'running'].includes(run.status)
const progressPercentage = (run: ExamAgentEvaluationRun) => run.progress.total_cases
  ? Math.round(run.progress.completed_cases * 100 / run.progress.total_cases)
  : 0
const formatDate = (value: string) => new Date(value).toLocaleString('zh-CN', { hour12: false })
const runPassRate = (run: ExamAgentEvaluationRun) => run.result_snapshot ? formatAgentRate(run.result_snapshot.summary.pass_rate) : '--'
const statusTheme = (status: ExamRAGEvaluationRunStatus) => status === 'completed' ? 'success' : status === 'failed' ? 'danger' : 'primary'

const openEvaluationMode = (mode: string | number) => {
  if (mode === 'rag') router.push(`/platform/question-banks/${bankId.value}/rag-observability`)
}

const fetchRun = async (runId: string) => (await getQuestionBankAgentEvaluationRun(bankId.value, runId)).data
const replaceRun = (run: ExamAgentEvaluationRun) => {
  const index = runs.value.findIndex(item => item.id === run.id)
  if (index >= 0) runs.value.splice(index, 1, run)
  else runs.value.unshift(run)
}

const selectRun = async (runId: string) => {
  const requestID = ++selectionRequestID
  const run = await fetchRun(runId)
  replaceRun(run)
  if (requestID === selectionRequestID) activeRun.value = run
}

const loadRuns = async () => {
  runs.value = (await listQuestionBankAgentEvaluationRuns(bankId.value)).data || []
  const requestedRunID = String(route.query.run_id || '')
  const selectedRunID = requestedRunID || activeRun.value?.id || runs.value[0]?.id
  if (!selectedRunID) return
  try {
    await selectRun(selectedRunID)
  } catch (error) {
    if (!requestedRunID || !runs.value[0]) throw error
    await selectRun(runs.value[0].id)
  }
}

const createRun = async () => {
  if (!selectedAgentId.value) return
  creating.value = true
  try {
    const cases = buildAgentEvaluationCases(scenarios.value)
    const run = (await createQuestionBankAgentEvaluationRun(bankId.value, { agent_id: selectedAgentId.value, cases })).data
    replaceRun(run)
    activeRun.value = run
    MessagePlugin.success('Agent 评测已进入队列')
  } catch (error: any) {
    MessagePlugin.error(error?.message || '创建 Agent 评测失败')
  } finally {
    creating.value = false
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
    // The next polling interval retries transient failures.
  } finally {
    pollInFlight = false
  }
}

onMounted(async () => {
  loading.value = true
  try {
    const [bankResponse, agentResponse, modelResponse] = await Promise.all([
      getQuestionBank(bankId.value),
      listAgents(),
      listModels('KnowledgeQA').catch(() => []),
      loadRuns(),
    ])
    bank.value = bankResponse.data
    models.value = modelResponse
    const availableAgents = ((agentResponse as unknown as { data?: CustomAgent[] }).data || [])
    agents.value = availableAgents.filter((agent: CustomAgent) => agent.config.agent_mode === 'smart-reasoning')
    selectedAgentId.value = activeRun.value?.agent_id || agents.value[0]?.id || ''
  } catch (error: any) {
    MessagePlugin.error(error?.message || 'Agent 评测中心加载失败')
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
<style lang="less" scoped src="./QuestionBankAgentEvaluation.less"></style>
