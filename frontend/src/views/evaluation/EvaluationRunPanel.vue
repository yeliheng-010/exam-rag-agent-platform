<template>
  <t-loading :loading="loading">
    <t-empty v-if="!items.length && !loading" description="当前筛选条件下没有评测运行" />
    <div v-else class="evaluation-run-list">
      <div class="evaluation-table desktop-only">
        <div class="evaluation-table__head">
          <span>运行与范围</span><span>核心指标</span><span>基线比较</span><span>状态</span><span>操作</span>
        </div>
        <template v-for="item in items" :key="item.run.id">
          <div class="evaluation-table__row" :class="{ 'is-regressed': item.comparison_status === 'regressed' }">
            <div class="run-identity">
              <button type="button" class="row-toggle" :aria-expanded="expandedRunID === item.run.id" @click="toggleRun(item.run.id)">
                <t-icon :name="expandedRunID === item.run.id ? 'chevron-down' : 'chevron-right'" />
              </button>
              <div>
                <strong>{{ item.question_bank_name || item.run.question_bank_id }}</strong>
                <span>{{ item.agent_name || modeLabel(item.run.evaluation_kind) }}</span>
                <code>{{ item.run.id }}</code>
              </div>
            </div>
            <div class="metric-inline">
              <span v-for="metric in getEvaluationPrimaryMetrics(item)" :key="metric.key" :class="{ 'is-negative': metric.regressed }">
                <small>{{ metric.label }}</small><strong>{{ metric.value }}</strong><em v-if="metric.delta">{{ metric.delta }}</em>
              </span>
            </div>
            <div class="comparison-cell">
              <t-tag size="small" variant="light" :theme="evaluationComparisonTheme(item.comparison_status)">
                {{ evaluationComparisonLabel(item.comparison_status) }}
              </t-tag>
              <code v-if="item.baseline_run_id">{{ item.baseline_run_id }}</code>
            </div>
            <div class="status-cell">
              <t-tag size="small" variant="light" :theme="evaluationRunStatusTheme(item.run.status)">
                {{ evaluationRunStatusLabel(item.run.status) }}
              </t-tag>
              <span>{{ formatEvaluationDate(item.run.created_at) }}</span>
            </div>
            <div class="row-actions">
              <t-tooltip content="查看题库级 Trace">
                <t-button shape="square" variant="text" @click="$emit('drilldown', item)"><t-icon name="jump" /></t-button>
              </t-tooltip>
              <t-tooltip content="设为基线">
                <t-button shape="square" variant="text" :disabled="item.run.status !== 'completed' || item.run.is_baseline" @click="$emit('baseline', item)"><t-icon name="flag" /></t-button>
              </t-tooltip>
              <t-tooltip content="保存为评测集">
                <t-button shape="square" variant="text" :disabled="item.run.status !== 'completed'" @click="$emit('save-set', item)"><t-icon name="folder-add" /></t-button>
              </t-tooltip>
            </div>
          </div>
          <div v-if="expandedRunID === item.run.id" class="evaluation-table__detail">
            <div><span>评测集</span><strong>{{ item.evaluation_set_name || '未关联' }}<template v-if="item.run.evaluation_set_version"> · v{{ item.run.evaluation_set_version }}</template></strong></div>
            <div><span>执行进度</span><strong>{{ item.run.progress.completed_cases }} / {{ item.run.progress.total_cases }}</strong></div>
            <div class="detail-wide"><span>回归原因</span><strong>{{ regressionText(item) }}</strong></div>
            <div v-if="item.run.error_message" class="detail-wide is-error"><span>错误</span><strong>{{ item.run.error_message }}</strong></div>
            <div class="detail-actions">
              <t-button variant="outline" size="small" @click="$emit('drilldown', item)"><template #icon><t-icon name="jump" /></template>查看 Trace</t-button>
              <t-button variant="outline" size="small" :disabled="item.run.status !== 'completed' || item.run.is_baseline" @click="$emit('baseline', item)"><template #icon><t-icon name="flag" /></template>设为基线</t-button>
              <t-button variant="outline" size="small" :disabled="item.run.status !== 'completed'" @click="$emit('save-set', item)"><template #icon><t-icon name="folder-add" /></template>保存为评测集</t-button>
            </div>
          </div>
        </template>
      </div>

      <article v-for="item in items" :key="`mobile-${item.run.id}`" class="mobile-run-row mobile-only">
        <header>
          <div><strong>{{ item.question_bank_name || item.run.question_bank_id }}</strong><span>{{ item.agent_name || modeLabel(item.run.evaluation_kind) }}</span></div>
          <t-tag size="small" variant="light" :theme="evaluationComparisonTheme(item.comparison_status)">{{ evaluationComparisonLabel(item.comparison_status) }}</t-tag>
        </header>
        <code>{{ item.run.id }}</code>
        <div class="mobile-metrics">
          <span v-for="metric in getEvaluationPrimaryMetrics(item)" :key="metric.key" :class="{ 'is-negative': metric.regressed }"><small>{{ metric.label }}</small><strong>{{ metric.value }}</strong><em v-if="metric.delta">{{ metric.delta }}</em></span>
        </div>
        <div class="mobile-run-meta"><span>{{ evaluationRunStatusLabel(item.run.status) }}</span><span>{{ formatEvaluationDate(item.run.created_at) }}</span></div>
        <p v-if="item.regression_reasons.length" class="mobile-regression">{{ regressionText(item) }}</p>
        <footer>
          <t-button variant="text" size="small" @click="$emit('drilldown', item)"><template #icon><t-icon name="jump" /></template>Trace</t-button>
          <t-button variant="text" size="small" :disabled="item.run.status !== 'completed' || item.run.is_baseline" @click="$emit('baseline', item)"><template #icon><t-icon name="flag" /></template>设为基线</t-button>
          <t-button variant="text" size="small" :disabled="item.run.status !== 'completed'" @click="$emit('save-set', item)"><template #icon><t-icon name="folder-add" /></template>保存为评测集</t-button>
        </footer>
      </article>
    </div>
  </t-loading>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { ExamEvaluationCenterItem, ExamEvaluationKind } from '@/types/exam'
import {
  evaluationComparisonLabel,
  evaluationComparisonTheme,
  evaluationRunStatusLabel,
  evaluationRunStatusTheme,
  formatEvaluationDate,
  getEvaluationPrimaryMetrics,
} from './evaluationCenterViewModel'

defineProps<{ items: ExamEvaluationCenterItem[]; loading: boolean }>()
defineEmits<{
  baseline: [item: ExamEvaluationCenterItem]
  'save-set': [item: ExamEvaluationCenterItem]
  drilldown: [item: ExamEvaluationCenterItem]
}>()

const expandedRunID = ref('')
const toggleRun = (runID: string) => { expandedRunID.value = expandedRunID.value === runID ? '' : runID }
const modeLabel = (kind: ExamEvaluationKind) => kind === 'agent' ? 'Agent 行为评测' : 'RAG 检索评测'
const regressionText = (item: ExamEvaluationCenterItem) => item.regression_reasons.length
  ? item.regression_reasons.map(reason => `${reason.label} ${reason.delta > 0 ? '+' : ''}${reason.metric === 'average_duration_ms' ? `${Math.round(reason.delta)} ms` : `${Number((reason.delta * 100).toFixed(1))}%`}`).join('；')
  : '没有检测到确定性回归'
</script>
