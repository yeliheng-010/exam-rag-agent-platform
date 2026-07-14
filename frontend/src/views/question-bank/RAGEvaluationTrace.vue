<template>
  <div class="trace-list">
    <t-empty v-if="!item.search_traces?.length" size="small" description="该案例没有检索 Trace" />
    <article v-for="(trace, traceIndex) in item.search_traces" :key="`${trace.knowledge_base_id}-${traceIndex}`" class="trace-block">
      <header class="trace-head">
        <div>
          <strong>知识库 {{ trace.knowledge_base_id }}</strong>
          <span>{{ trace.embedding_model_id || '未记录模型' }} · {{ trace.embedding_dimensions || 0 }} 维</span>
        </div>
        <div class="trace-parameters">
          <span>TopK {{ trace.parameters.match_count }}</span>
          <span>向量 {{ trace.parameters.vector_threshold }}</span>
          <span>关键词 {{ trace.parameters.keyword_threshold }}</span>
          <span>{{ trace.duration_ms }} ms</span>
        </div>
      </header>

      <section v-for="stage in getTraceStages(trace)" :key="stage.key" class="trace-stage">
        <div class="trace-stage__title">
          <strong>{{ stage.label }}</strong>
          <span>{{ stage.candidates.length }} 个候选</span>
        </div>
        <div v-if="stage.candidates.length" class="candidate-table">
          <div class="candidate-row candidate-row--head">
            <span>排名</span><span>Chunk ID</span><span>分数</span><span>来源排名</span><span></span>
          </div>
          <div
            v-for="candidate in visibleTraceCandidates(stage.candidates, isStageExpanded(traceIndex, stage.key))"
            :key="`${stage.key}-${candidate.chunk_id}-${candidate.rank}`"
            class="candidate-row"
          >
            <span class="candidate-rank">#{{ candidate.rank }}</span>
            <code>{{ candidate.chunk_id }}</code>
            <span>{{ formatScore(candidate.score) }}</span>
            <span>{{ sourceRanks(candidate) }}</span>
            <t-tooltip content="复制 Chunk ID">
              <t-button shape="square" size="small" variant="text" @click="copyChunkID(candidate.chunk_id)">
                <template #icon><t-icon name="file-copy" /></template>
              </t-button>
            </t-tooltip>
          </div>
        </div>
        <t-empty v-else size="small" description="无候选" />
        <t-button
          v-if="stage.candidates.length > 12"
          class="candidate-toggle"
          size="small"
          variant="text"
          @click="toggleStage(traceIndex, stage.key)"
        >
          <template #icon><t-icon :name="isStageExpanded(traceIndex, stage.key) ? 'chevron-up' : 'chevron-down'" /></template>
          {{ isStageExpanded(traceIndex, stage.key) ? '收起' : `展开全部 ${stage.candidates.length}` }}
        </t-button>
      </section>
    </article>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import type { ExamRAGDiagnosticResultItem, SearchTraceCandidate } from '@/types/exam'
import { getTraceStages, visibleTraceCandidates } from './ragEvaluationViewModel'

defineProps<{ item: ExamRAGDiagnosticResultItem }>()
const expandedStages = ref<Record<string, boolean>>({})

const expansionKey = (traceIndex: number, stageKey: string) => `${traceIndex}:${stageKey}`
const isStageExpanded = (traceIndex: number, stageKey: string) => Boolean(expandedStages.value[expansionKey(traceIndex, stageKey)])
const toggleStage = (traceIndex: number, stageKey: string) => {
  const key = expansionKey(traceIndex, stageKey)
  expandedStages.value = { ...expandedStages.value, [key]: !expandedStages.value[key] }
}

const formatScore = (score: number) => Number(score || 0).toFixed(4)

const sourceRanks = (candidate: SearchTraceCandidate) => {
  const ranks = []
  if (candidate.vector_rank) ranks.push(`V#${candidate.vector_rank}`)
  if (candidate.keyword_rank) ranks.push(`K#${candidate.keyword_rank}`)
  return ranks.join(' / ') || '-'
}

const copyChunkID = async (chunk_id: string) => {
  try {
    await navigator.clipboard.writeText(chunk_id)
    MessagePlugin.success('Chunk ID 已复制')
  } catch {
    MessagePlugin.error('复制失败')
  }
}
</script>

<style lang="less" scoped>
.trace-list,
.trace-block {
  display: grid;
  gap: 14px;
}

.trace-block {
  padding-top: 14px;
  border-top: 1px solid var(--td-component-stroke);
}

.trace-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;

  strong,
  span {
    display: block;
  }

  strong {
    margin-bottom: 4px;
    font-size: 14px;
  }

  span {
    color: var(--td-text-color-secondary);
    font-size: 12px;
  }
}

.trace-parameters {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px 12px;
}

.trace-stage {
  min-width: 0;
}

.trace-stage__title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 7px;

  strong {
    font-size: 13px;
  }

  span {
    color: var(--td-text-color-placeholder);
    font-size: 12px;
  }
}

.candidate-table {
  overflow: hidden;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
}

.candidate-row {
  display: grid;
  grid-template-columns: 54px minmax(180px, 1fr) 80px 100px 36px;
  align-items: center;
  min-height: 38px;
  padding: 0 8px;
  border-top: 1px solid var(--td-component-stroke);
  font-size: 12px;

  &:first-child {
    border-top: 0;
  }

  code {
    overflow-wrap: anywhere;
    color: var(--td-brand-color);
    font-size: 11px;
  }
}

.candidate-row--head {
  min-height: 32px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-weight: 600;
}

.candidate-rank {
  font-weight: 600;
}

.candidate-toggle {
  margin-top: 4px;
}

@media (max-width: 720px) {
  .trace-head {
    display: grid;
  }

  .trace-parameters {
    justify-content: flex-start;
  }

  .candidate-row {
    grid-template-columns: 42px minmax(120px, 1fr) 62px 36px;

    > span:nth-child(4) {
      display: none;
    }
  }
}
</style>
