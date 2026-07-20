<template>
  <div class="evaluation-page">
    <header class="evaluation-header">
      <div><h2>RAG / Agent 评测中心</h2><p>跨题库基线、回归运行与版本化评测资产</p></div>
      <t-button class="evaluation-refresh" variant="outline" :loading="loading" aria-label="刷新评测数据" @click="loadData">
        <template #icon><t-icon name="refresh" /></template>
        <span class="evaluation-refresh__label">刷新</span>
      </t-button>
    </header>

    <section class="evaluation-toolbar" aria-label="评测筛选">
      <t-radio-group v-model="kind" variant="default-filled" @change="changeKind">
        <t-radio-button value="rag">RAG</t-radio-button>
        <t-radio-button value="agent">Agent</t-radio-button>
      </t-radio-group>
      <label><span>题库</span><t-select v-model="bankID" :options="bankOptions" placeholder="全部可访问题库" clearable filterable @change="loadData" /></label>
      <label v-if="kind === 'agent'"><span>Agent</span><t-select v-model="agentID" :options="agentOptions" placeholder="全部 Agent" clearable filterable @change="loadData" /></label>
      <label v-if="activeView === 'runs'"><span>运行状态</span><t-select v-model="status" :options="statusOptions" placeholder="全部状态" clearable @change="loadData" /></label>
      <label v-if="activeView === 'runs'" class="regression-toggle"><span>回归筛选</span><t-switch v-model="onlyRegressions" @change="loadData" /><strong>仅看回归</strong></label>
    </section>

    <section class="evaluation-summary" aria-label="评测汇总">
      <div><span>当前运行</span><strong>{{ center.summary.total_runs }}</strong></div>
      <div><span>已完成</span><strong>{{ center.summary.completed_runs }}</strong></div>
      <div><span>基线</span><strong>{{ center.summary.baseline_runs }}</strong></div>
      <div :class="{ 'has-regression': center.summary.regression_runs > 0 }"><span>回归</span><strong>{{ center.summary.regression_runs }}</strong></div>
      <div><span>平均通过率</span><strong>{{ formatEvaluationRate(center.summary.average_pass_rate) }}</strong></div>
    </section>

    <t-tabs v-model="activeView" class="evaluation-tabs" @change="loadData">
      <t-tab-panel value="runs" label="回归运行" />
      <t-tab-panel value="sets" label="评测集" />
    </t-tabs>

    <EvaluationRunPanel
      v-if="activeView === 'runs'"
      :items="center.items"
      :loading="loading"
      @baseline="setBaseline"
      @save-set="openSaveSet"
      @drilldown="openDrilldown"
    />
    <EvaluationSetPanel
      v-else
      :sets="evaluationSets"
      :loading="loading"
      :running-set-i-d="runningSetID"
      @add-version="openAddVersion"
      @run-version="runSetVersion"
    />

    <t-dialog v-model:visible="saveSetVisible" header="保存为评测集" :confirm-btn="{ content: '保存', loading: savingSet }" @confirm="saveEvaluationSet">
      <t-form label-align="top">
        <t-form-item label="评测集名称"><t-input v-model="saveSetForm.name" :maxlength="160" /></t-form-item>
        <t-form-item label="说明"><t-textarea v-model="saveSetForm.description" :maxlength="500" :autosize="{ minRows: 3, maxRows: 6 }" /></t-form-item>
        <div class="dialog-source"><span>来源运行</span><code>{{ selectedRun?.run.id }}</code></div>
      </t-form>
    </t-dialog>

    <t-dialog v-model:visible="versionVisible" header="新增版本" :confirm-btn="{ content: '新增版本', loading: creatingVersion }" @confirm="createVersion">
      <t-form label-align="top">
        <t-form-item label="来源运行">
          <t-select v-model="versionSourceRunID" :loading="loadingCandidates" placeholder="选择同题库、同类型的已完成运行" filterable>
            <t-option v-for="item in versionCandidates" :key="item.run.id" :value="item.run.id" :label="`${formatEvaluationDate(item.run.created_at)} · ${item.run.id}`" />
          </t-select>
        </t-form-item>
        <div class="dialog-source"><span>目标评测集</span><strong>{{ selectedSet?.name }}</strong></div>
      </t-form>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  createExamEvaluationSet,
  createExamEvaluationSetVersion,
  getExamEvaluationCenter,
  listExamEvaluationSets,
  runExamEvaluationSet,
  setExamEvaluationBaseline,
} from '@/api/exam/evaluation-center'
import type {
  ExamEvaluationCenterItem,
  ExamEvaluationCenterResult,
  ExamEvaluationKind,
  ExamEvaluationSet,
  ExamRAGEvaluationRunStatus,
  QuestionBank,
} from '@/types/exam'
import EvaluationRunPanel from './EvaluationRunPanel.vue'
import EvaluationSetPanel from './EvaluationSetPanel.vue'
import {
  compatibleEvaluationRuns,
  evaluationDrilldownPath,
  formatEvaluationDate,
  formatEvaluationRate,
} from './evaluationCenterViewModel'

const router = useRouter()
const emptyCenter = (): ExamEvaluationCenterResult => ({
  summary: { total_runs: 0, completed_runs: 0, baseline_runs: 0, regression_runs: 0, average_pass_rate: 0 },
  items: [], question_banks: [], agents: [],
})
const center = ref<ExamEvaluationCenterResult>(emptyCenter())
const evaluationSets = ref<ExamEvaluationSet[]>([])
const knownBanks = ref<QuestionBank[]>([])
const kind = ref<ExamEvaluationKind>('rag')
const activeView = ref<'runs' | 'sets'>('runs')
const bankID = ref('')
const agentID = ref('')
const status = ref<ExamRAGEvaluationRunStatus | ''>('')
const onlyRegressions = ref(false)
const loading = ref(false)
const pollInFlight = ref(false)
const runningSetID = ref('')
let pollTimer: ReturnType<typeof setInterval> | undefined

const selectedRun = ref<ExamEvaluationCenterItem | null>(null)
const saveSetVisible = ref(false)
const savingSet = ref(false)
const saveSetForm = reactive({ name: '', description: '' })
const selectedSet = ref<ExamEvaluationSet | null>(null)
const versionVisible = ref(false)
const creatingVersion = ref(false)
const loadingCandidates = ref(false)
const versionCandidates = ref<ExamEvaluationCenterItem[]>([])
const versionSourceRunID = ref('')

const bankOptions = computed(() => knownBanks.value.map(bank => ({ label: bank.name, value: bank.id })))
const agentOptions = computed(() => center.value.agents.map(agent => ({ label: agent.name, value: agent.id })))
const statusOptions = [
  { label: '排队中', value: 'queued' }, { label: '运行中', value: 'running' },
  { label: '已完成', value: 'completed' }, { label: '失败', value: 'failed' },
]
const query = () => ({
  kind: kind.value, bank_id: bankID.value || undefined, agent_id: agentID.value || undefined,
  status: status.value, regression: onlyRegressions.value ? 'regressed' as const : 'all' as const, limit: 100,
})

const mergeKnownBanks = (banks: QuestionBank[]) => {
  const merged = new Map(knownBanks.value.map(bank => [bank.id, bank]))
  banks.forEach(bank => merged.set(bank.id, bank))
  knownBanks.value = [...merged.values()]
}

const loadData = async () => {
  if (loading.value) return
  loading.value = true
  try {
    const [centerResponse, setsResponse] = await Promise.all([
      getExamEvaluationCenter(query()), listExamEvaluationSets(query()),
    ])
    center.value = centerResponse.data || emptyCenter()
    evaluationSets.value = setsResponse.data || []
    mergeKnownBanks(center.value.question_banks || [])
  } catch (error: any) {
    MessagePlugin.error(error?.message || '评测中心加载失败')
  } finally {
    loading.value = false
  }
}

const changeKind = () => { agentID.value = ''; status.value = ''; onlyRegressions.value = false; loadData() }
const setBaseline = async (item: ExamEvaluationCenterItem) => {
  try { await setExamEvaluationBaseline(item.run.id); MessagePlugin.success('基线已更新'); await loadData() }
  catch (error: any) { MessagePlugin.error(error?.message || '设置基线失败') }
}
const openSaveSet = (item: ExamEvaluationCenterItem) => {
  selectedRun.value = item
  saveSetForm.name = `${item.question_bank_name || '题库'} ${item.run.evaluation_kind.toUpperCase()} 回归集`
  saveSetForm.description = ''
  saveSetVisible.value = true
}
const saveEvaluationSet = async () => {
  if (!selectedRun.value || !saveSetForm.name.trim()) return MessagePlugin.warning('请输入评测集名称')
  savingSet.value = true
  try {
    await createExamEvaluationSet({ run_id: selectedRun.value.run.id, name: saveSetForm.name.trim(), description: saveSetForm.description.trim() })
    saveSetVisible.value = false; MessagePlugin.success('评测集已保存'); await loadData()
  } catch (error: any) { MessagePlugin.error(error?.message || '保存评测集失败') }
  finally { savingSet.value = false }
}

const openAddVersion = async (set: ExamEvaluationSet) => {
  selectedSet.value = set; versionSourceRunID.value = ''; versionCandidates.value = []; versionVisible.value = true; loadingCandidates.value = true
  try {
    const response = await getExamEvaluationCenter({ kind: set.evaluation_kind, bank_id: set.question_bank_id, agent_id: set.agent_id, status: 'completed', regression: 'all', limit: 200 })
    versionCandidates.value = compatibleEvaluationRuns(set, response.data.items || [])
    versionSourceRunID.value = versionCandidates.value[0]?.run.id || ''
  } catch (error: any) { MessagePlugin.error(error?.message || '可用运行加载失败') }
  finally { loadingCandidates.value = false }
}
const createVersion = async () => {
  if (!selectedSet.value || !versionSourceRunID.value) return MessagePlugin.warning('请选择来源运行')
  creatingVersion.value = true
  try {
    await createExamEvaluationSetVersion(selectedSet.value.id, versionSourceRunID.value)
    versionVisible.value = false; MessagePlugin.success('新版本已创建'); await loadData()
  } catch (error: any) { MessagePlugin.error(error?.message || '新增版本失败') }
  finally { creatingVersion.value = false }
}

const runSetVersion = async (set: ExamEvaluationSet, version: number) => {
  runningSetID.value = set.id
  try {
    await runExamEvaluationSet(set.id, version)
    kind.value = set.evaluation_kind; bankID.value = set.question_bank_id; agentID.value = set.agent_id || ''
    status.value = ''; onlyRegressions.value = false; activeView.value = 'runs'
    MessagePlugin.success(`v${version} 已进入评测队列`); await loadData()
  } catch (error: any) { MessagePlugin.error(error?.message || '运行评测集失败') }
  finally { runningSetID.value = '' }
}
const openDrilldown = (item: ExamEvaluationCenterItem) => router.push(evaluationDrilldownPath(item))

const pollRuns = async () => {
  if (pollInFlight.value || !center.value.items.some(item => ['queued', 'running'].includes(item.run.status))) return
  pollInFlight.value = true
  try { await loadData() } finally { pollInFlight.value = false }
}
onMounted(async () => { await loadData(); pollTimer = setInterval(pollRuns, 2500) })
onUnmounted(() => { if (pollTimer) clearInterval(pollTimer) })
</script>

<style lang="less" src="./EvaluationCenter.less"></style>
