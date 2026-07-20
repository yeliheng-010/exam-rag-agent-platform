import type {
  ExamEvaluationCenterItem,
  ExamEvaluationComparisonStatus,
  ExamEvaluationKind,
  ExamEvaluationSet,
  ExamRAGEvaluationRunStatus,
} from '../../types/exam.ts'

export interface EvaluationMetricDisplay {
  key: string
  label: string
  value: string
  delta?: string
  regressed?: boolean
}

export const formatEvaluationRate = (value?: number) => value == null ? '--' : `${Number((value * 100).toFixed(1))}%`
export const formatEvaluationDuration = (value?: number) => value == null ? '--' : `${Math.round(value)} ms`

export const formatEvaluationDelta = (value: number | undefined, type: 'rate' | 'duration') => {
  if (value == null) return '--'
  const scaled = type === 'rate' ? Number((value * 100).toFixed(1)) : Math.round(value)
  const suffix = type === 'rate' ? '%' : ' ms'
  return `${scaled > 0 ? '+' : ''}${scaled}${suffix}`
}

export const evaluationComparisonLabel = (status: ExamEvaluationComparisonStatus) => ({
  baseline: '当前基线', regressed: '发生回归', stable: '保持稳定', uncompared: '尚未比较',
}[status])

export const evaluationComparisonTheme = (status: ExamEvaluationComparisonStatus) => ({
  baseline: 'primary', regressed: 'danger', stable: 'success', uncompared: 'default',
}[status] as 'primary' | 'danger' | 'success' | 'default')

export const evaluationRunStatusLabel = (status: ExamRAGEvaluationRunStatus) => ({
  queued: '排队中', running: '运行中', completed: '已完成', failed: '失败',
}[status])

export const evaluationRunStatusTheme = (status: ExamRAGEvaluationRunStatus) => (
  status === 'completed' ? 'success' : status === 'failed' ? 'danger' : 'primary'
)

const metricConfig: Record<ExamEvaluationKind, Array<{ key: string; label: string; type: 'rate' | 'duration' }>> = {
  rag: [
    { key: 'pass_rate', label: '总体通过率', type: 'rate' },
    { key: 'retrieval_hit_rate', label: '召回通过率', type: 'rate' },
    { key: 'average_duration_ms', label: '平均耗时', type: 'duration' },
  ],
  agent: [
    { key: 'pass_rate', label: '总体通过率', type: 'rate' },
    { key: 'tool_sequence_rate', label: '工具顺序', type: 'rate' },
    { key: 'average_duration_ms', label: '平均耗时', type: 'duration' },
  ],
}

export function getEvaluationPrimaryMetrics(item: ExamEvaluationCenterItem): EvaluationMetricDisplay[] {
  const kind = item.run.evaluation_kind || 'rag'
  return metricConfig[kind].map(config => ({
    key: config.key,
    label: config.label,
    value: config.type === 'rate' ? formatEvaluationRate(item.metrics[config.key]) : formatEvaluationDuration(item.metrics[config.key]),
    delta: item.metric_deltas ? formatEvaluationDelta(item.metric_deltas[config.key], config.type) : undefined,
    regressed: item.regression_reasons.some(reason => reason.metric === config.key),
  }))
}

export function evaluationDrilldownPath(item: ExamEvaluationCenterItem) {
  const suffix = item.run.evaluation_kind === 'agent' ? 'agent-evaluation' : 'rag-observability'
  return `/platform/question-banks/${item.run.question_bank_id}/${suffix}?run_id=${item.run.id}`
}

export function compatibleEvaluationRuns(set: ExamEvaluationSet, items: ExamEvaluationCenterItem[]) {
  return items.filter(item => item.run.status === 'completed' &&
    item.run.question_bank_id === set.question_bank_id &&
    item.run.evaluation_kind === set.evaluation_kind &&
    (item.run.agent_id || '') === (set.agent_id || ''))
}

export const formatEvaluationDate = (value?: string) => value
  ? new Date(value).toLocaleString('zh-CN', { hour12: false })
  : '--'
