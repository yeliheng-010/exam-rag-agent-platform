<template>
  <div class="exam-page">
    <div class="exam-header">
      <div>
        <t-button variant="text" size="small" @click="router.back()">
          <template #icon><t-icon name="chevron-left" /></template>
          返回
        </t-button>
        <h2>题组校对</h2>
        <p>{{ headerText }}</p>
      </div>
      <t-space size="small">
        <t-tag v-if="data" variant="light">{{ data.stats.approved }} / {{ data.stats.total }} 已确认</t-tag>
        <t-button variant="outline" :loading="loading" @click="loadDrafts">
          <template #icon><t-icon name="refresh" /></template>
          刷新
        </t-button>
        <t-button theme="primary" :loading="extracting" @click="extractDrafts(false)">
          <template #icon><t-icon name="scan" /></template>
          开始抽取
        </t-button>
        <t-button variant="outline" :loading="extracting" @click="extractDrafts(true)">
          重新抽取
        </t-button>
      </t-space>
    </div>

    <t-loading :loading="loading">
      <div v-if="drafts.length" class="review-layout">
        <aside class="draft-list">
          <button
            v-for="draft in drafts"
            :key="draft.id"
            class="draft-item"
            :class="{ active: draft.id === selectedDraft?.id }"
            @click="selectDraft(draft)"
          >
            <span>{{ draft.title || draft.group_type }}</span>
            <small>{{ draft.questions_json?.length || 0 }} 题</small>
            <t-tag size="small" variant="light" :theme="groupDraftStatusTheme(draft.status)">
              {{ groupDraftStatusLabel(draft.status) }}
            </t-tag>
          </button>
        </aside>

        <section v-if="selectedDraft" class="editor-panel">
          <div class="editor-title-row">
            <div>
              <h3>{{ form.title || form.group_type }}</h3>
              <span>{{ selectedDraft.id }}</span>
            </div>
            <t-tag variant="light" :theme="groupDraftStatusTheme(selectedDraft.status)">
              {{ groupDraftStatusLabel(selectedDraft.status) }}
            </t-tag>
          </div>

          <t-form :data="form" label-align="top">
            <div class="form-grid">
              <t-form-item label="题组类型">
                <t-input v-model="form.group_type" :disabled="!canEditSelected" />
              </t-form-item>
              <t-form-item label="标题">
                <t-input v-model="form.title" :disabled="!canEditSelected" />
              </t-form-item>
              <t-form-item label="材料格式">
                <t-input v-model="form.material_format" :disabled="!canEditSelected" />
              </t-form-item>
            </div>

            <t-form-item label="阅读材料 / 题干背景">
              <t-textarea v-model="form.material_text" :disabled="!canEditSelected" :autosize="{ minRows: 8, maxRows: 14 }" />
            </t-form-item>

            <div class="form-grid two">
              <t-form-item label="题组来源 chunk">
                <t-textarea v-model="groupChunkText" :disabled="!canEditSelected" :autosize="{ minRows: 4, maxRows: 8 }" />
              </t-form-item>
              <t-form-item label="资源 JSON">
                <t-textarea v-model="assetText" :disabled="!canEditSelected" :autosize="{ minRows: 4, maxRows: 8 }" />
              </t-form-item>
            </div>

            <div class="question-tabs">
              <button
                v-for="(question, index) in form.questions"
                :key="`${question.question_no}-${index}`"
                type="button"
                :class="{ active: selectedQuestionIndex === index }"
                @click="selectQuestion(index)"
              >
                {{ question.question_no || index + 1 }}
              </button>
              <t-button size="small" variant="outline" :disabled="!canEditSelected" @click="addQuestion">
                <template #icon><t-icon name="add" /></template>
                小题
              </t-button>
            </div>

            <div v-if="selectedQuestion" class="question-editor">
              <div class="form-grid compact">
                <t-form-item label="题号">
                  <t-input v-model="selectedQuestion.question_no" :disabled="!canEditSelected" />
                </t-form-item>
                <t-form-item label="题型">
                  <t-input v-model="selectedQuestion.question_type_code" :disabled="!canEditSelected" />
                </t-form-item>
                <t-form-item label="顺序">
                  <t-input-number v-model="selectedQuestion.order_in_group" :disabled="!canEditSelected" :min="1" />
                </t-form-item>
                <t-form-item label="难度">
                  <t-select v-model="selectedQuestion.difficulty" :disabled="!canEditSelected">
                    <t-option value="unknown" label="未知" />
                    <t-option value="easy" label="简单" />
                    <t-option value="medium" label="中等" />
                    <t-option value="hard" label="困难" />
                  </t-select>
                </t-form-item>
              </div>
              <t-form-item label="题干">
                <t-textarea v-model="selectedQuestion.stem" :disabled="!canEditSelected" :autosize="{ minRows: 4, maxRows: 8 }" />
              </t-form-item>
              <t-form-item label="选项">
                <div class="options-editor">
                  <div v-for="(option, index) in selectedQuestion.options" :key="index" class="option-row">
                    <t-input v-model="option.key" class="option-key" :disabled="!canEditSelected" />
                    <t-input v-model="option.content" :disabled="!canEditSelected" />
                    <t-button variant="text" shape="circle" :disabled="!canEditSelected" @click="removeOption(index)">
                      <template #icon><t-icon name="close" /></template>
                    </t-button>
                  </div>
                  <t-button size="small" variant="outline" :disabled="!canEditSelected" @click="addOption">
                    <template #icon><t-icon name="add" /></template>
                    选项
                  </t-button>
                </div>
              </t-form-item>
              <div class="form-grid two">
                <t-form-item label="答案 JSON">
                  <t-textarea v-model="answerText" :disabled="!canEditSelected" :autosize="{ minRows: 4, maxRows: 8 }" />
                </t-form-item>
                <t-form-item label="小题来源 chunk">
                  <t-textarea v-model="questionChunkText" :disabled="!canEditSelected" :autosize="{ minRows: 4, maxRows: 8 }" />
                </t-form-item>
              </div>
              <t-form-item label="解析">
                <t-textarea v-model="selectedQuestion.explanation" :disabled="!canEditSelected" :autosize="{ minRows: 4, maxRows: 8 }" />
              </t-form-item>
              <t-button size="small" theme="danger" variant="outline" :disabled="!canEditSelected" @click="removeQuestion(selectedQuestionIndex)">
                删除小题
              </t-button>
            </div>
          </t-form>

          <div class="action-row">
            <t-space>
              <t-button theme="primary" :loading="saving" :disabled="!canEditSelected" @click="saveDraft">保存</t-button>
              <t-button theme="success" :loading="approving" :disabled="!canEditSelected" @click="approveDraft">确认入库</t-button>
              <t-button theme="danger" variant="outline" :loading="rejecting" :disabled="!canEditSelected" @click="rejectDraft">驳回</t-button>
            </t-space>
          </div>
        </section>
      </div>

      <t-empty v-else-if="!loading" description="暂无题组草稿" />
    </t-loading>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { groupDraftStatusLabel, groupDraftStatusTheme, useQuestionGroupDraftReview } from './useQuestionGroupDraftReview'

const {
  router,
  loading,
  extracting,
  saving,
  approving,
  rejecting,
  data,
  drafts,
  selectedDraft,
  selectedQuestionIndex,
  selectedQuestion,
  form,
  answerText,
  questionChunkText,
  groupChunkText,
  assetText,
  headerText,
  canEditSelected,
  loadDrafts,
  extractDrafts,
  selectDraft,
  selectQuestion,
  addQuestion,
  removeQuestion,
  addOption,
  removeOption,
  saveDraft,
  approveDraft,
  rejectDraft,
} = useQuestionGroupDraftReview()

onMounted(loadDrafts)
</script>

<style lang="less" scoped src="./QuestionGroupDraftReview.less"></style>
