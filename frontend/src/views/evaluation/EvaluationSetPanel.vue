<template>
  <t-loading :loading="loading">
    <t-empty v-if="!sets.length && !loading" description="当前范围还没有评测集，可从已完成运行保存" />
    <div v-else class="evaluation-set-list">
      <article v-for="set in sets" :key="set.id" class="evaluation-set-row">
        <header>
          <button type="button" class="row-toggle" :aria-expanded="expandedSetIDs.has(set.id)" @click="toggleSet(set.id)">
            <t-icon :name="expandedSetIDs.has(set.id) ? 'chevron-down' : 'chevron-right'" />
          </button>
          <div class="set-identity">
            <strong>{{ set.name }}</strong>
            <span>{{ set.question_bank_name || set.question_bank_id }} · {{ set.evaluation_kind === 'agent' ? set.agent_name || set.agent_id : 'RAG' }}</span>
            <p v-if="set.description">{{ set.description }}</p>
          </div>
          <div class="set-version"><small>当前版本</small><strong>v{{ set.current_version }}</strong></div>
          <div class="set-updated"><small>更新时间</small><span>{{ formatEvaluationDate(set.updated_at) }}</span></div>
          <div class="set-actions">
            <t-button variant="outline" size="small" @click="$emit('add-version', set)"><template #icon><t-icon name="git-branch" /></template>新增版本</t-button>
            <t-button theme="primary" size="small" :loading="runningSetID === set.id" @click="$emit('run-version', set, set.current_version)"><template #icon><t-icon name="play-circle" /></template>运行当前版本</t-button>
          </div>
        </header>
        <div v-if="expandedSetIDs.has(set.id)" class="version-history">
          <div v-for="version in set.versions" :key="version.id" class="version-row">
            <span class="version-number">v{{ version.version }}</span>
            <div><small>来源运行</small><code>{{ version.source_run_id }}</code></div>
            <div><small>创建时间</small><span>{{ formatEvaluationDate(version.created_at) }}</span></div>
            <t-button variant="text" size="small" :loading="runningSetID === set.id" @click="$emit('run-version', set, version.version)"><template #icon><t-icon name="play-circle" /></template>运行此版本</t-button>
          </div>
        </div>
      </article>
    </div>
  </t-loading>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { ExamEvaluationSet } from '@/types/exam'
import { formatEvaluationDate } from './evaluationCenterViewModel'

defineProps<{ sets: ExamEvaluationSet[]; loading: boolean; runningSetID: string }>()
defineEmits<{
  'add-version': [set: ExamEvaluationSet]
  'run-version': [set: ExamEvaluationSet, version: number]
}>()

const expandedSetIDs = ref(new Set<string>())
const toggleSet = (setID: string) => {
  const next = new Set(expandedSetIDs.value)
  if (next.has(setID)) next.delete(setID)
  else next.add(setID)
  expandedSetIDs.value = next
}
</script>
