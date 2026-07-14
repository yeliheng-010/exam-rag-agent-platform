<template>
  <div class="run-detail">
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
              <span>召回 {{ formatRAGRate(item.retrieval_score) }}</span>
              <span>答案 {{ formatRAGRate(item.answer_score) }}</span>
            </div>
            <t-tag size="small" :theme="caseTheme(item)" variant="light">{{ caseStatus(item) }}</t-tag>
          </summary>
          <div class="case-body">
            <dl>
              <div><dt>上下文来源</dt><dd>{{ item.context_source || 'none' }}</dd></div>
              <div><dt>题组 ID</dt><dd>{{ item.group_id || '-' }}</dd></div>
              <div><dt>耗时</dt><dd>{{ formatRAGDuration(item.duration_ms) }}</dd></div>
              <div><dt>MRR</dt><dd>{{ item.reciprocal_rank?.toFixed(3) || '0.000' }}</dd></div>
            </dl>
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
import { formatRAGDuration, formatRAGRate } from './ragEvaluationViewModel'

const props = defineProps<{ run: ExamRAGEvaluationRun }>()
const result = computed(() => props.run.result_snapshot)
const emptyDescription = computed(() => props.run.status === 'failed' ? '本次运行失败' : '评测结果生成中')

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

<style lang="less" scoped>
.run-detail,
.case-list {
  display: grid;
  gap: 12px;
}

.result-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 18px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.case-row {
  border-top: 1px solid var(--td-component-stroke);

  summary {
    display: grid;
    grid-template-columns: 34px minmax(180px, 1fr) auto auto;
    align-items: center;
    gap: 12px;
    min-height: 74px;
    cursor: pointer;
    list-style: none;

    &::-webkit-details-marker {
      display: none;
    }
  }
}

.case-index {
  color: var(--td-text-color-placeholder);
  font-variant-numeric: tabular-nums;
  font-weight: 600;
}

.case-copy {
  min-width: 0;

  strong {
    display: block;
    overflow-wrap: anywhere;
    font-size: 14px;
  }

  p {
    margin: 4px 0 0;
    overflow: hidden;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.case-rates {
  display: flex;
  gap: 12px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.case-body {
  display: grid;
  gap: 14px;
  padding: 0 0 18px 46px;

  dl {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 8px;
    margin: 0;
  }

  dl div {
    min-width: 0;
    padding: 9px 10px;
    background: var(--td-bg-color-secondarycontainer);
    border-radius: 6px;
  }

  dt {
    margin-bottom: 4px;
    color: var(--td-text-color-placeholder);
    font-size: 11px;
  }

  dd {
    margin: 0;
    overflow-wrap: anywhere;
    font-size: 12px;
    font-weight: 600;
  }
}

.missing-line,
.error-line {
  margin: 0;
  font-size: 12px;
  line-height: 20px;
}

.missing-line {
  color: var(--td-warning-color);
}

.error-line {
  color: var(--td-error-color);
}

@media (max-width: 720px) {
  .case-row summary {
    grid-template-columns: 28px minmax(0, 1fr) auto;
  }

  .case-rates {
    display: none;
  }

  .case-body {
    padding-left: 0;

    dl {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }
}
</style>
