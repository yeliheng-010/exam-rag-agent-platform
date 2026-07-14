<template>
  <div class="recommendation-command">
    <t-button variant="outline" :loading="loading" @click="openRecommendations">
      <template #icon><t-icon name="chart-bubble" /></template>
      推荐练习
    </t-button>

    <t-dialog v-model:visible="visible" header="智能练习推荐" width="min(920px, calc(100vw - 24px))" :footer="false">
      <t-loading :loading="loading">
        <div class="recommendation-dialog">
          <div class="recommendation-options">
            <span>包含已发布题组</span>
            <t-switch v-model="includeAssigned" aria-label="包含已发布题组" :disabled="loading" @change="loadRecommendations" />
          </div>

          <t-alert
            v-if="result?.warnings.includes('no_active_wrong_questions')"
            theme="info"
            message="当前没有有效错题信号，以下内容为可用的补充练习，不代表班级薄弱点。"
          />

          <div v-if="result?.diagnosis.length" class="diagnosis-strip">
            <span>诊断依据</span>
            <strong>{{ result.diagnosis.length }} 道高频错题</strong>
            <span>推荐只使用正式题组，发布前仍需老师确认。</span>
          </div>

          <t-table
            v-if="recommendations.length"
            class="recommendation-table"
            row-key="recommendation_key"
            :data="recommendations"
            :columns="columns"
            :pagination="undefined"
            size="small"
          >
            <template #group="{ row }">
              <div class="group-cell">
                <div class="group-title">
                  <strong>{{ recommendationTitle(row) }}</strong>
                  <t-tag v-if="row.assigned" size="small" variant="light">已发布</t-tag>
                </div>
                <span class="group-meta">{{ row.group.bank_name || '题库' }} · {{ groupTypeLabel(row.group.group.group_type) }} · {{ row.group.question_count }} 题</span>
              </div>
            </template>
            <template #score="{ row }">
              <t-tag variant="light" theme="primary">{{ row.score }}</t-tag>
            </template>
            <template #reasons="{ row }">
              <div class="reason-list">
                <t-tag v-for="reason in row.reasons" :key="reason.code" size="small" variant="light">
                  {{ reasonLabel(reason.code) }}
                </t-tag>
              </div>
            </template>
            <template #operation="{ row }">
              <t-button size="small" theme="primary" @click="useForAssignment(row)">用于发布</t-button>
            </template>
          </t-table>

          <div v-if="recommendations.length" class="recommendation-mobile-list">
            <div v-for="item in recommendations" :key="item.recommendation_key" class="recommendation-mobile-item">
              <div class="recommendation-mobile-heading">
                <strong>{{ recommendationTitle(item) }}</strong>
                <t-tag v-if="item.assigned" size="small" variant="light">已发布</t-tag>
              </div>
              <span class="recommendation-mobile-meta">
                {{ item.group.bank_name || '题库' }} · {{ groupTypeLabel(item.group.group.group_type) }} · {{ item.group.question_count }} 题
              </span>
              <div class="recommendation-mobile-footer">
                <div class="reason-list">
                  <t-tag variant="light" theme="primary">{{ item.score }} 分</t-tag>
                  <t-tag v-for="reason in item.reasons" :key="reason.code" size="small" variant="light">
                    {{ reasonLabel(reason.code) }}
                  </t-tag>
                </div>
                <t-button size="small" theme="primary" @click="useForAssignment(item)">用于发布</t-button>
              </div>
            </div>
          </div>

          <t-empty v-else-if="!loading" size="small" description="暂无可推荐的正式题组" />
        </div>
      </t-loading>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { getClassPracticeRecommendations } from '@/api/exam/intervention'
import { createLatestRequestRunner } from './practiceRecommendationRequest'
import type {
  ExamPracticeRecommendation,
  ExamPracticeRecommendationReasonCode,
  ExamPracticeRecommendationResult,
} from '@/types/exam'

const props = defineProps<{ classId: string }>()
const emit = defineEmits<{ publish: [recommendation: ExamPracticeRecommendation] }>()

const visible = ref(false)
const loading = ref(false)
const includeAssigned = ref(false)
const result = ref<ExamPracticeRecommendationResult | null>(null)
const recommendationRequestRunner = createLatestRequestRunner()

const recommendations = computed(() => (result.value?.recommendations || []).map(item => ({
  ...item,
  recommendation_key: item.group.group.id,
})))

const columns = [
  { colKey: 'group', title: '推荐题组', cell: 'group', ellipsis: true },
  { colKey: 'score', title: '匹配分', cell: 'score', width: 90 },
  { colKey: 'reasons', title: '推荐依据', cell: 'reasons', width: 260 },
  { colKey: 'operation', title: '操作', cell: 'operation', width: 110 },
]

const openRecommendations = async () => {
  if (!props.classId) return
  visible.value = true
  includeAssigned.value = false
  await loadRecommendations()
}

const loadRecommendations = async () => {
  if (!props.classId) return
  loading.value = true
  result.value = null
  const outcome = await recommendationRequestRunner.run(async () => {
    const response = await getClassPracticeRecommendations(props.classId, {
      limit: 8,
      include_assigned: includeAssigned.value,
    })
    return response.data
  })
  if (outcome.status === 'stale') return

  loading.value = false
  if (outcome.status === 'error') {
    const error = outcome.error as { message?: string } | null
    MessagePlugin.error(error?.message || '练习推荐加载失败')
    return
  }
  result.value = outcome.data
}

const useForAssignment = (recommendation: ExamPracticeRecommendation) => {
  emit('publish', recommendation)
  visible.value = false
}

const recommendationTitle = (item: ExamPracticeRecommendation) => {
  return item.group.group.title || item.group.group.material_text || '未命名题组'
}

const groupTypeLabel = (type: string) => {
  const labels: Record<string, string> = {
    reading_passage: '阅读',
    math_problem: '数学题',
    single_question: '单题',
    cloze: '完型',
    essay: '作文',
  }
  return labels[type] || type || '题组'
}

const reasonLabel = (code: ExamPracticeRecommendationReasonCode) => {
  const labels: Record<ExamPracticeRecommendationReasonCode, string> = {
    targeted_review: '错题复练',
    same_subject: '同学科',
    same_group_type: '同题型',
    same_domain: '同考试域',
    low_accuracy_retry: '低正确率复练',
    supplemental_practice: '补充练习',
  }
  return labels[code]
}
</script>

<style lang="less" scoped>
.recommendation-command {
  display: inline-flex;
}

.recommendation-dialog {
  display: flex;
  min-height: 240px;
  flex-direction: column;
  gap: 14px;
}

.recommendation-options {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  min-height: 32px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
}

.recommendation-mobile-list {
  display: none;
}

.diagnosis-strip {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 40px;
  padding: 8px 12px;
  border-left: 3px solid var(--td-brand-color);
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: 13px;

  strong {
    color: var(--td-text-color-primary);
  }
}

.group-cell {
  min-width: 0;

  > .group-meta {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  > .group-meta {
    margin-top: 3px;
    color: var(--td-text-color-secondary);
    font-size: 12px;
  }
}

.group-title {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;

  strong {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.reason-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

@media (max-width: 720px) {
  .recommendation-dialog {
    min-height: 0;
    max-height: calc(100vh - 300px);
    overflow-y: auto;
  }

  .recommendation-table {
    display: none;
  }

  .recommendation-mobile-list {
    display: block;
  }

  .recommendation-mobile-item {
    padding: 12px 0;
    border-bottom: 1px solid var(--td-component-border);
  }

  .recommendation-mobile-item:first-child {
    padding-top: 0;
  }

  .recommendation-mobile-heading,
  .recommendation-mobile-footer {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 10px;
  }

  .recommendation-mobile-heading strong {
    min-width: 0;
    overflow-wrap: anywhere;
  }

  .recommendation-mobile-meta {
    display: block;
    margin: 5px 0 10px;
    color: var(--td-text-color-secondary);
    font-size: 12px;
  }

  .recommendation-mobile-footer {
    align-items: center;
  }

  .recommendation-mobile-footer .reason-list {
    min-width: 0;
  }

  .diagnosis-strip {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
