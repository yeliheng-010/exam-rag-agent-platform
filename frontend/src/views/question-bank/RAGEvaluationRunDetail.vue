<template>
  <div class="run-detail">
    <section class="chunking-snapshot" aria-label="切块快照">
      <header>
        <h4>切块快照</h4>
        <span>{{ chunkingRows.length ? `${chunkingRows.length} 个知识库` : '未记录' }}</span>
      </header>
      <div v-if="chunkingRows.length" class="chunking-table-wrap">
        <table>
          <thead>
            <tr>
              <th>知识库</th><th>策略 / 实际 tier</th><th>块数量</th><th>P50 / P90</th>
              <th>Tiny</th><th>Oversize</th><th>父块覆盖</th><th>未知 tier</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in chunkingRows" :key="row.knowledgeBaseID">
              <td data-label="知识库"><code>{{ row.knowledgeBaseID }}</code></td>
              <td data-label="策略 / 实际 tier"><strong>{{ row.requestedStrategy }}</strong><span>{{ row.actualTiers }}</span></td>
              <td data-label="块数量">{{ row.chunks }}</td>
              <td data-label="P50 / P90">{{ row.sizes }}</td>
              <td data-label="Tiny">{{ row.tinyRate }}</td>
              <td data-label="Oversize">{{ row.oversizeRate }}</td>
              <td data-label="父块覆盖">{{ row.parentCoverage }}</td>
              <td data-label="未知 tier">{{ row.unknownTiers }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
    <t-empty v-if="!result" size="small" :description="emptyDescription" />
    <template v-else>
      <div class="result-meta">
        <span>{{ result.used_default_cases ? '默认评测集' : '自定义评测集' }}</span>
        <span>{{ result.knowledge_base_ids.length }} 个知识库</span>
        <span>{{ result.summary.failed_case_count }} 个异常案例</span>
      </div>
      <div class="case-list">
        <details v-for="(item, index) in result.summary.results" :key="item.name" class="case-row" :open="index === 0">
          <summary>
            <div class="case-index">{{ String(index + 1).padStart(2, '0') }}</div>
            <div class="case-copy">
              <strong>{{ item.name }}</strong>
              <p>{{ item.query }}</p>
            </div>
            <div class="case-rates">
              <span>Top-K {{ caseRetrievalRate(item) }}</span>
              <span>候选 {{ caseCandidateRate(item) }}</span>
              <span>答案 {{ formatRAGRate(item.answer_score) }}</span>
            </div>
            <t-tag size="small" :theme="caseTheme(item)" variant="light">{{ caseStatus(item) }}</t-tag>
          </summary>
          <div class="case-body">
            <dl>
              <div><dt>上下文来源</dt><dd>{{ formatContextSource(item.context_source) }}</dd></div>
              <div><dt>关联置信度</dt><dd>{{ formatAssociationConfidence(item.context_source, item.association_confidence) }}</dd></div>
              <div><dt>题组 ID</dt><dd>{{ item.group_id || '-' }}</dd></div>
              <div><dt>耗时</dt><dd>{{ formatRAGDuration(item.duration_ms) }}</dd></div>
              <div><dt>首个相关排名</dt><dd>{{ formatFirstRelevantRank(item) }}</dd></div>
              <div><dt>候选全集召回</dt><dd>{{ caseCandidateRate(item) }}</dd></div>
            </dl>
            <p v-if="item.missing_retrieval_phrases?.length" class="missing-line">缺失检索金标：{{ item.missing_retrieval_phrases.join('、') }}</p>
            <p v-if="item.missing_phrases?.length" class="missing-line">缺失证据：{{ item.missing_phrases.join('、') }}</p>
            <p v-if="item.error" class="error-line">{{ item.error }}</p>
            <RAGEvaluationTrace :item="item" />
          </div>
        </details>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ExamRAGDiagnosticResultItem, ExamRAGEvaluationRun } from '@/types/exam'
import RAGEvaluationTrace from './RAGEvaluationTrace.vue'
import {
  caseHasRetrievalGold,
  formatAssociationConfidence,
  formatContextSource,
  formatFirstRelevantRank,
  formatRAGDuration,
  formatRAGRate,
  formatRankedRate,
  formatOptionalRankedRate,
  getChunkingSnapshotRows,
} from './ragEvaluationViewModel'

const props = defineProps<{ run: ExamRAGEvaluationRun }>()
const result = computed(() => props.run.result_snapshot)
const chunkingRows = computed(() => getChunkingSnapshotRows(props.run))
const emptyDescription = computed(() => props.run.status === 'failed' ? '本次运行失败' : '评测结果生成中')

const caseRetrievalRate = (item: ExamRAGDiagnosticResultItem) => formatRankedRate(
  item.retrieval_score,
  caseHasRetrievalGold(item) ? 1 : 0,
)

const caseCandidateRate = (item: ExamRAGDiagnosticResultItem) => formatOptionalRankedRate(
  item.candidate_retrieval_score,
  caseHasRetrievalGold(item) ? 1 : 0,
)

const caseTheme = (item: ExamRAGDiagnosticResultItem) => {
  if (item.error) return 'danger'
  if (item.passed) return 'success'
  return 'warning'
}

const caseStatus = (item: ExamRAGDiagnosticResultItem) => {
  if (item.error) return '异常'
  return item.passed ? '通过' : '未通过'
}
</script>

<style lang="less" scoped src="./RAGEvaluationRunDetail.less"></style>
