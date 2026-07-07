<template>
  <div class="review-page">
    <div class="review-header">
      <div>
        <t-button variant="text" size="small" @click="router.push('/platform/learning')">
          <template #icon><t-icon name="chevron-left" /></template>
          返回学习中心
        </t-button>
        <h2>练习复盘</h2>
        <p>回看自己的练习记录，集中处理错题。</p>
      </div>
      <t-button variant="outline" :loading="loading" @click="loadData">
        <template #icon><t-icon name="refresh" /></template>
        刷新
      </t-button>
    </div>

    <t-loading :loading="loading">
      <t-tabs v-model="activeTab" class="review-tabs">
        <t-tab-panel value="attempts" :label="`练习记录 ${attempts.length}`">
          <div v-if="attempts.length" class="review-list">
            <article v-for="item in attempts" :key="item.attempt.id" class="review-card">
              <div class="review-card__main">
                <div>
                  <strong>{{ groupTitle(item.group) }}</strong>
                  <span>{{ item.bank_name || '题库' }} · {{ formatDate(item.attempt.started_at) }}</span>
                </div>
                <t-tag :theme="attemptTheme(item.attempt.status)" variant="light">{{ attemptStatus(item.attempt) }}</t-tag>
              </div>
              <div class="stat-row">
                <span>已答 {{ item.attempt.answered_count }}/{{ item.attempt.question_count }}</span>
                <span>正确 {{ item.attempt.correct_count }}/{{ item.attempt.question_count }}</span>
              </div>
              <div class="review-card__footer">
                <span>{{ item.group?.material_text || item.group?.title || '正式题组练习' }}</span>
                <t-button size="small" theme="primary" variant="outline" @click="goPractice(item.attempt.group_id)">
                  再练一次
                </t-button>
              </div>
            </article>
          </div>
          <t-empty v-else description="暂无练习记录" />
        </t-tab-panel>

        <t-tab-panel value="wrong" :label="`错题本 ${wrongQuestions.length}`">
          <div v-if="wrongQuestions.length" class="review-list">
            <article v-for="item in wrongQuestions" :key="item.answer.id" class="wrong-card">
              <div class="review-card__main">
                <div>
                  <strong>{{ questionTitle(item) }}</strong>
                  <span>{{ groupTitle(item.group) }} · {{ formatDate(item.answer.answered_at) }}</span>
                </div>
                <t-tag theme="danger" variant="light">错题</t-tag>
              </div>
              <p class="question-stem">{{ item.answer.question_snapshot?.stem || '题干快照缺失' }}</p>
              <div class="answer-grid">
                <div>
                  <span>我的答案</span>
                  <strong>{{ item.answer.answer_text || '-' }}</strong>
                </div>
                <div>
                  <span>正确答案</span>
                  <strong>{{ item.answer.correct_answer || '-' }}</strong>
                </div>
              </div>
              <p v-if="explanationText(item)" class="explanation-text">{{ explanationText(item) }}</p>
              <div class="review-card__footer">
                <span>{{ item.bank_name || '题库' }}</span>
                <t-button size="small" theme="primary" variant="outline" @click="goPractice(item.attempt.group_id)">
                  回到题组
                </t-button>
              </div>
            </article>
          </div>
          <t-empty v-else description="暂无错题" />
        </t-tab-panel>
      </t-tabs>
    </t-loading>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import { listPracticeAttempts, listWrongQuestions } from '@/api/exam/practice'
import type { ExamPracticeAttempt, PracticeAttemptSummary, QuestionGroup, WrongQuestionItem } from '@/types/exam'

const router = useRouter()
const loading = ref(false)
const activeTab = ref<'attempts' | 'wrong'>('attempts')
const attempts = ref<PracticeAttemptSummary[]>([])
const wrongQuestions = ref<WrongQuestionItem[]>([])

const loadData = async () => {
  loading.value = true
  try {
    const [attemptRes, wrongRes] = await Promise.all([
      listPracticeAttempts({ limit: 30 }),
      listWrongQuestions({ limit: 30 }),
    ])
    attempts.value = attemptRes.data || []
    wrongQuestions.value = wrongRes.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '练习复盘加载失败')
  } finally {
    loading.value = false
  }
}

const groupTitle = (group?: QuestionGroup) => {
  return group?.title || group?.group_type || '题组练习'
}

const attemptTheme = (status: ExamPracticeAttempt['status']) => {
  return status === 'completed' ? 'success' : 'warning'
}

const attemptStatus = (attempt: ExamPracticeAttempt) => {
  if (attempt.status === 'completed') return '已完成'
  return '进行中'
}

const questionTitle = (item: WrongQuestionItem) => {
  return item.answer.question_no ? `第 ${item.answer.question_no} 题` : '错题'
}

const explanationText = (item: WrongQuestionItem) => {
  const explanations = item.answer.explanation_snapshot || []
  return explanations
    .map((explanation: any) => explanation.explanation_text)
    .filter(Boolean)
    .join('；')
}

const formatDate = (value?: string) => {
  return value ? new Date(value).toLocaleString() : '-'
}

const goPractice = (groupId: string) => {
  router.push(`/platform/practice/question-groups/${groupId}`)
}

onMounted(loadData)
</script>

<style lang="less" scoped>
.review-page {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  padding: 28px 32px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
}

.review-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;

  h2 {
    margin: 8px 0 0;
    font-size: 22px;
    line-height: 30px;
    font-weight: 600;
  }

  p {
    margin: 6px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 14px;
  }
}

.review-tabs {
  :deep(.t-tabs__content) {
    padding-top: 14px;
  }
}

.review-list {
  display: grid;
  gap: 12px;
}

.review-card,
.wrong-card {
  display: grid;
  gap: 12px;
  padding: 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.review-card__main,
.review-card__footer,
.stat-row,
.answer-grid {
  display: flex;
  gap: 12px;
}

.review-card__main,
.review-card__footer {
  align-items: flex-start;
  justify-content: space-between;

  strong {
    display: block;
    margin-bottom: 4px;
    font-size: 15px;
    line-height: 22px;
  }

  span {
    color: var(--td-text-color-secondary);
    font-size: 13px;
    line-height: 20px;
  }
}

.review-card__footer {
  align-items: center;

  span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.stat-row {
  flex-wrap: wrap;

  span {
    padding: 6px 10px;
    border-radius: 6px;
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-secondary);
    font-size: 13px;
  }
}

.question-stem,
.explanation-text {
  margin: 0;
  color: var(--td-text-color-primary);
  font-size: 14px;
  line-height: 22px;
  white-space: pre-wrap;
}

.answer-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));

  div {
    min-width: 0;
    padding: 10px;
    border-radius: 8px;
    background: var(--td-bg-color-secondarycontainer);
  }

  span {
    display: block;
    margin-bottom: 4px;
    color: var(--td-text-color-secondary);
    font-size: 12px;
  }

  strong {
    font-size: 16px;
    overflow-wrap: anywhere;
  }
}

@media (max-width: 720px) {
  .review-page {
    padding: 18px;
  }

  .review-header,
  .review-card__main,
  .review-card__footer {
    align-items: stretch;
    flex-direction: column;
  }

  .answer-grid {
    grid-template-columns: 1fr;
  }
}
</style>
