<template>
  <div class="practice-page">
    <div class="practice-header">
      <div>
        <t-button variant="text" size="small" @click="router.push('/platform/learning')">
          <template #icon><t-icon name="chevron-left" /></template>
          返回学习中心
        </t-button>
        <h2>{{ groupTitle }}</h2>
        <p>{{ bankLine }}</p>
      </div>
      <div class="practice-header__actions">
        <t-tag v-if="attempt" variant="light">{{ attemptStatusLabel }}</t-tag>
        <t-button theme="primary" :disabled="!attempt" @click="completeAttempt">完成练习</t-button>
      </div>
    </div>

    <t-loading :loading="loading">
      <div v-if="group" class="practice-layout">
        <section class="panel material-panel">
          <div class="panel-title">
            <div>
              <h3>{{ materialTitle }}</h3>
              <p>{{ materialMeta }}</p>
            </div>
            <t-button v-if="canToggleMaterial" variant="text" size="small" @click="materialExpanded = !materialExpanded">
              <template #icon><t-icon :name="materialExpanded ? 'chevron-up' : 'chevron-down'" /></template>
              {{ materialExpanded ? '收起' : '展开' }}
            </t-button>
          </div>
          <div
            class="material-body"
            :class="{ 'is-collapsed': canToggleMaterial && !materialExpanded }"
          >
            {{ group.group.material_text || '当前题组没有单独材料。' }}
          </div>
          <div v-if="group.assets?.length" class="asset-list">
            <div v-for="asset in group.assets" :key="asset.id" class="asset-chip">
              {{ asset.asset_type }} · {{ asset.alt_text || asset.storage_uri }}
            </div>
          </div>
        </section>

        <section class="panel exercise-panel">
          <div class="question-toolbar">
            <div>
              <strong>{{ currentQuestionTitle }}</strong>
              <span>{{ questionPagerLabel }}</span>
            </div>
            <div class="question-toolbar__actions">
              <t-button size="small" variant="outline" :disabled="!canMoveQuestion(-1)" @click="moveQuestion(-1)">
                <template #icon><t-icon name="chevron-left" /></template>
                上一题
              </t-button>
              <t-button size="small" theme="primary" variant="outline" :disabled="!canMoveQuestion(1)" @click="moveQuestion(1)">
                <template #icon><t-icon name="chevron-right" /></template>
                下一题
              </t-button>
            </div>
          </div>

          <div v-if="group.questions.length > 1" class="question-tabs" role="tablist">
            <button
              v-for="(item, index) in group.questions"
              :key="item.question.id"
              type="button"
              class="question-tab"
              :class="{ 'is-active': currentQuestionIndex === index, 'is-done': Boolean(answerResults[item.question.id]) }"
              @click="setQuestionIndex(index)"
            >
              {{ questionNo(item) }}
            </button>
          </div>

          <article v-if="currentQuestion" class="question-card">
            <div class="question-stem">{{ currentQuestion.question.stem }}</div>
            <div v-if="currentQuestion.options?.length" class="option-list">
              <button
                v-for="option in currentQuestion.options"
                :key="option.id || `${currentQuestion.question.id}-${option.option_key}`"
                type="button"
                class="option-item"
                :class="optionClass(currentQuestion, option.option_key)"
                :disabled="hasResult(currentQuestion)"
                @click="selectOption(currentQuestion, option.option_key)"
              >
                <span>{{ option.option_key }}</span>
                <strong>{{ option.content }}</strong>
              </button>
            </div>
            <t-textarea
              v-else
              v-model="selectedAnswers[currentQuestion.question.id]"
              class="free-answer"
              :disabled="hasResult(currentQuestion)"
              placeholder="输入你的答案"
              autosize
            />

            <div class="question-actions">
              <t-button theme="primary" :disabled="!canSubmitCurrent" :loading="submitting" @click="submitCurrentAnswer">
                提交答案
              </t-button>
              <t-button variant="outline" @click="startExplanation">
                <template #icon><t-icon name="chat" /></template>
                AI 讲解
              </t-button>
            </div>

            <div v-if="currentResult" class="answer-panel" :class="{ 'is-correct': currentResult.answer.is_correct }">
              <div class="answer-panel__head">
                <strong>{{ currentResult.answer.is_correct ? '回答正确' : '需要订正' }}</strong>
                <span>你的答案：{{ currentResult.answer.answer_text || '-' }}</span>
              </div>
              <p>正确答案：{{ currentResult.correct_answers.join('，') || '-' }}</p>
              <p v-for="item in currentResult.explanations" :key="item.id || item.explanation_text">
                解析：{{ item.explanation_text }}
              </p>
              <div v-if="currentResult.chunk_refs?.length" class="evidence-list">
                <span v-for="ref in currentResult.chunk_refs" :key="`${ref.question_id}-${ref.chunk_id}`">
                  证据 chunk：{{ ref.chunk_id }}
                </span>
              </div>
            </div>
          </article>
        </section>
      </div>
      <t-empty v-else-if="!loading" description="题组不存在或暂无权限" />
    </t-loading>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import { createPracticeAttempt, submitPracticeAnswer, completePracticeAttempt } from '@/api/exam/practice'
import { listExamResources } from '@/api/exam/resource'
import { useMenuStore } from '@/stores/menu'
import { useSettingsStore } from '@/stores/settings'
import type { ExamPracticeAttempt, PracticeAnswerResult, QuestionDetail, QuestionGroupDetail } from '@/types/exam'

const route = useRoute()
const router = useRouter()
const menuStore = useMenuStore()
const settingsStore = useSettingsStore()

const loading = ref(false)
const submitting = ref(false)
const group = ref<QuestionGroupDetail | null>(null)
const attempt = ref<ExamPracticeAttempt | null>(null)
const currentQuestionIndex = ref(0)
const selectedAnswers = ref<Record<string, string>>({})
const answerResults = ref<Record<string, PracticeAnswerResult>>({})
const materialExpanded = ref(false)

const groupTitle = computed(() => group.value?.group.title || group.value?.group.group_type || '题组练习')
const bankLine = computed(() => {
  if (!group.value) return '加载正式题组中'
  return `${group.value.group.group_type} · ${group.value.questions.length} 题`
})
const materialTitle = computed(() => {
  if (group.value?.group.group_type === 'reading_passage') return '阅读原文'
  if (group.value?.group.group_type === 'math_problem') return '题干材料'
  return '题组材料'
})
const materialMeta = computed(() => {
  const chunks = group.value?.group.source_chunk_ids?.length || 0
  return chunks ? `引用 ${chunks} 个 chunk` : '正式题库材料'
})
const materialTextLength = computed(() => group.value?.group.material_text?.length || 0)
const canToggleMaterial = computed(() => materialTextLength.value > 520)
const currentQuestion = computed(() => group.value?.questions[currentQuestionIndex.value] || null)
const currentResult = computed(() => {
  const id = currentQuestion.value?.question.id
  return id ? answerResults.value[id] : null
})
const questionPagerLabel = computed(() => {
  const count = group.value?.questions.length || 0
  return count ? `${currentQuestionIndex.value + 1} / ${count}` : '0 / 0'
})
const currentQuestionTitle = computed(() => {
  return currentQuestion.value ? `当前题目 ${questionNo(currentQuestion.value)}` : '暂无题目'
})
const attemptStatusLabel = computed(() => {
  if (!attempt.value) return ''
  if (attempt.value.status === 'completed') {
    return `已完成 ${attempt.value.correct_count}/${attempt.value.question_count}`
  }
  return `进行中 ${attempt.value.answered_count}/${attempt.value.question_count}`
})
const canSubmitCurrent = computed(() => {
  if (!attempt.value || !currentQuestion.value || hasResult(currentQuestion.value)) return false
  return Boolean((selectedAnswers.value[currentQuestion.value.question.id] || '').trim())
})

const questionNo = (item: QuestionDetail) => {
  return item.question.question_no || String(item.question.order_in_group || '-')
}

const hasResult = (item: QuestionDetail) => {
  return Boolean(answerResults.value[item.question.id])
}

const selectOption = (item: QuestionDetail, optionKey: string) => {
  selectedAnswers.value = {
    ...selectedAnswers.value,
    [item.question.id]: optionKey,
  }
}

const optionClass = (item: QuestionDetail, optionKey: string) => {
  const result = answerResults.value[item.question.id]
  return {
    'is-selected': selectedAnswers.value[item.question.id] === optionKey,
    'is-correct': result?.correct_answers?.map(answer => answer.toUpperCase()).includes(optionKey.toUpperCase()),
    'is-wrong': result && selectedAnswers.value[item.question.id] === optionKey && !result.answer.is_correct,
  }
}

const setQuestionIndex = (index: number) => {
  const count = group.value?.questions.length || 0
  if (!count) return
  currentQuestionIndex.value = Math.min(Math.max(index, 0), count - 1)
}

const moveQuestion = (delta: number) => {
  setQuestionIndex(currentQuestionIndex.value + delta)
}

const canMoveQuestion = (delta: number) => {
  const count = group.value?.questions.length || 0
  const nextIndex = currentQuestionIndex.value + delta
  return count > 1 && nextIndex >= 0 && nextIndex < count
}

const submitCurrentAnswer = async () => {
  if (!attempt.value || !currentQuestion.value) return
  const question = currentQuestion.value
  const answerText = (selectedAnswers.value[question.question.id] || '').trim()
  if (!answerText) {
    MessagePlugin.warning('请先填写答案')
    return
  }
  submitting.value = true
  try {
    const res = await submitPracticeAnswer(attempt.value.id, {
      question_id: question.question.id,
      answer_text: answerText,
    })
    attempt.value = res.data.attempt
    answerResults.value = {
      ...answerResults.value,
      [question.question.id]: res.data,
    }
  } catch (error: any) {
    MessagePlugin.error(error?.message || '答案提交失败')
  } finally {
    submitting.value = false
  }
}

const completeAttempt = async () => {
  if (!attempt.value) return
  try {
    const res = await completePracticeAttempt(attempt.value.id)
    attempt.value = res.data
    MessagePlugin.success('练习已完成')
  } catch (error: any) {
    MessagePlugin.error(error?.message || '完成练习失败')
  }
}

const startExplanation = async () => {
  if (!group.value || !currentQuestion.value) return
  const kbId = await resolvePracticeKnowledgeBase()
  if (kbId) {
    settingsStore.selectKnowledgeBases([kbId])
  }
  settingsStore.clearFiles()
  settingsStore.clearTags()
  menuStore.setPrefillQuery(buildExplanationPrompt(currentQuestion.value, currentResult.value))
  router.push('/platform/creatChat')
}

const resolvePracticeKnowledgeBase = async () => {
  const spaceId = group.value?.group.space_id
  if (!spaceId) return ''
  try {
    const res = await listExamResources({ space_id: spaceId, resource_type: 'knowledge_base' })
    return res.data?.[0]?.resource_id || ''
  } catch {
    return ''
  }
}

const buildExplanationPrompt = (question: QuestionDetail, result: PracticeAnswerResult | null) => {
  const options = question.options?.map(option => `${option.option_key}. ${option.content}`).join('\n') || '无选项'
  const material = clipText(group.value?.group.material_text || '', 3600)
  const parts = [
    '请作为考试老师，基于下面题组材料讲解这道题。',
    `【题组材料】\n${material || '无单独材料'}`,
    `【题目】\n${question.question.stem}`,
    `【选项】\n${options}`,
    `【我的答案】\n${selectedAnswers.value[question.question.id] || '未填写'}`,
  ]
  if (result) {
    parts.push(`【正确答案】\n${result.correct_answers.join('，') || '-'}`)
    const explanation = result.explanations?.map(item => item.explanation_text).filter(Boolean).join('\n')
    if (explanation) parts.push(`【已有解析】\n${explanation}`)
  }
  parts.push('请说明解题思路、原文依据或关键步骤，并指出我应该如何复盘。')
  return parts.join('\n\n')
}

const clipText = (text: string, limit: number) => {
  if (text.length <= limit) return text
  return `${text.slice(0, limit)}\n...`
}

const loadData = async () => {
  const groupId = String(route.params.groupId || '')
  if (!groupId) return
  loading.value = true
  try {
    const res = await createPracticeAttempt(groupId)
    group.value = res.data.group
    attempt.value = res.data.attempt
    currentQuestionIndex.value = 0
  } catch (error: any) {
    MessagePlugin.error(error?.message || '题组练习加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
</script>

<style lang="less" scoped>
.practice-page {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  padding: 28px 32px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
}

.practice-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;

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

.practice-header__actions {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 10px;
}

.practice-layout {
  display: grid;
  grid-template-columns: minmax(320px, 0.82fr) minmax(0, 1.18fr);
  gap: 12px;
}

.panel {
  min-width: 0;
  padding: 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.panel-title,
.question-toolbar,
.question-toolbar__actions,
.answer-panel__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.panel-title {
  margin-bottom: 12px;

  h3 {
    margin: 0;
    font-size: 16px;
    line-height: 24px;
  }

  p {
    margin: 4px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 13px;
  }
}

.material-body {
  color: var(--td-text-color-primary);
  font-size: 14px;
  line-height: 24px;
  white-space: pre-wrap;

  &.is-collapsed {
    display: -webkit-box;
    overflow: hidden;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 14;
  }
}

.asset-list,
.question-tabs,
.evidence-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.asset-list {
  margin-top: 12px;
}

.asset-chip,
.evidence-list span {
  max-width: 100%;
  overflow: hidden;
  padding: 5px 8px;
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}

.exercise-panel {
  display: grid;
  align-content: start;
  gap: 12px;
}

.question-toolbar {
  align-items: center;
  padding: 12px;
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);

  strong {
    display: block;
    margin-bottom: 2px;
    font-size: 15px;
  }

  span {
    color: var(--td-text-color-secondary);
    font-size: 12px;
  }
}

.question-toolbar__actions {
  flex-shrink: 0;
  align-items: center;
}

.question-tab {
  min-width: 34px;
  height: 30px;
  padding: 0 10px;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 6px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;

  &.is-active {
    border-color: var(--td-brand-color);
    background: var(--td-brand-color-light);
    color: var(--td-brand-color);
  }

  &.is-done {
    box-shadow: inset 0 -2px 0 var(--td-success-color);
  }
}

.question-card {
  display: grid;
  gap: 14px;
  padding: 16px;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 8px;
}

.question-stem {
  font-size: 15px;
  line-height: 24px;
  font-weight: 600;
  white-space: pre-wrap;
}

.option-list {
  display: grid;
  gap: 10px;
}

.option-item {
  display: grid;
  grid-template-columns: 30px minmax(0, 1fr);
  align-items: flex-start;
  gap: 10px;
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  text-align: left;
  cursor: pointer;

  span {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 26px;
    height: 26px;
    border-radius: 50%;
    background: var(--td-bg-color-secondarycontainer);
    font-weight: 600;
  }

  strong {
    min-width: 0;
    font-size: 14px;
    line-height: 22px;
    font-weight: 500;
    overflow-wrap: anywhere;
  }

  &.is-selected {
    border-color: var(--td-brand-color);
    background: var(--td-brand-color-light);
  }

  &.is-correct {
    border-color: var(--td-success-color-4);
    background: var(--td-success-color-1);
  }

  &.is-wrong {
    border-color: var(--td-error-color-4);
    background: var(--td-error-color-1);
  }

  &:disabled {
    cursor: default;
  }
}

.free-answer {
  width: 100%;
}

.question-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.answer-panel {
  display: grid;
  gap: 8px;
  padding: 12px;
  border: 1px solid var(--td-error-color-3);
  border-radius: 8px;
  background: var(--td-error-color-1);

  &.is-correct {
    border-color: var(--td-success-color-3);
    background: var(--td-success-color-1);
  }

  p {
    margin: 0;
    color: var(--td-text-color-primary);
    font-size: 13px;
    line-height: 20px;
  }
}

.answer-panel__head {
  align-items: center;

  span {
    color: var(--td-text-color-secondary);
    font-size: 12px;
  }
}

@media (max-width: 1080px) {
  .practice-layout {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .practice-page {
    padding: 18px;
  }

  .practice-header,
  .question-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .practice-header__actions,
  .question-toolbar__actions {
    width: 100%;

    :deep(.t-button) {
      flex: 1;
      min-width: 0;
    }
  }
}
</style>
