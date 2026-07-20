<template>
  <t-empty v-if="!run.result_snapshot" size="small" description="等待场景结果" />
  <section v-else class="agent-run-detail">
    <details v-for="(result, index) in run.result_snapshot.results" :key="`${result.name}-${index}`" class="agent-case" :open="index === 0">
      <summary>
        <div><strong>{{ result.name }}</strong><span>{{ result.input }}</span></div>
        <t-tag size="small" variant="light" :theme="result.passed ? 'success' : 'danger'">{{ result.passed ? '通过' : '未通过' }}</t-tag>
      </summary>

      <div class="assertion-grid">
        <div v-for="score in scores(result)" :key="score.label"><span>{{ score.label }}</span><strong>{{ formatAgentRate(score.value) }}</strong></div>
      </div>

      <t-alert v-if="result.error" theme="error" :message="result.error" />
      <div v-if="result.assertion_failures?.length" class="assertion-failures">
        <strong>未通过断言</strong>
        <ul><li v-for="failure in result.assertion_failures" :key="failure">{{ failure }}</li></ul>
      </div>

      <section class="tool-timeline">
        <h4>实际工具调用</h4>
        <t-empty v-if="!result.actual_tool_calls.length" size="small" description="未调用工具" />
        <article v-for="(call, callIndex) in result.actual_tool_calls" :key="`${call.name}-${callIndex}`" class="tool-call">
          <header>
            <code>{{ callIndex + 1 }}. {{ call.name }}</code>
            <div><span>{{ call.duration_ms }} ms</span><t-tag size="small" variant="light" :theme="call.success ? 'success' : 'danger'">{{ call.success ? '成功' : '失败' }}</t-tag></div>
          </header>
          <div><span>arguments</span><pre>{{ prettyJSON(call.arguments) }}</pre></div>
          <div v-if="call.output_excerpt"><span>output_excerpt</span><pre>{{ call.output_excerpt }}</pre></div>
          <p v-if="call.error" class="tool-call__error">{{ call.error }}</p>
        </article>
      </section>

      <section class="final-answer">
        <h4>final_answer</h4>
        <p>{{ result.final_answer || '无最终回答' }}</p>
      </section>
    </details>
  </section>
</template>

<script setup lang="ts">
import type { ExamAgentEvaluationCaseResult, ExamAgentEvaluationRun } from '@/types/exam'
import { formatAgentRate } from './agentEvaluationViewModel'

defineProps<{ run: ExamAgentEvaluationRun }>()

const scores = (result: ExamAgentEvaluationCaseResult) => [
  { label: '工具序列', value: result.tool_sequence_score },
  { label: '参数命中', value: result.tool_arguments_score },
  { label: '证据覆盖', value: result.evidence_score },
  { label: '引用覆盖', value: result.citation_score },
  { label: '答案覆盖', value: result.answer_score },
  { label: '事实支撑', value: result.groundedness_score },
]

const prettyJSON = (value?: Record<string, unknown>) => JSON.stringify(value || {}, null, 2)
</script>

<style lang="less" scoped src="./AgentEvaluationRunDetail.less"></style>
