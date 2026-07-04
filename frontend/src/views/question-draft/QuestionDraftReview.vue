<template>
  <div class="exam-page">
    <div class="exam-header">
      <div>
        <t-button variant="text" size="small" @click="router.back()">
          <template #icon><t-icon name="chevron-left" /></template>
          返回
        </t-button>
        <h2>题目校对</h2>
        <p>{{ headerText }}</p>
      </div>
      <t-space v-if="data" size="small">
        <t-tag variant="light">{{ data.stats.approved }} / {{ data.stats.total }} 已确认</t-tag>
        <t-button variant="outline" :loading="loading" @click="loadDrafts">
          <template #icon><t-icon name="refresh" /></template>
          刷新
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
            <span class="draft-no">{{ draft.question_no || '未编号' }}</span>
            <span class="draft-stem">{{ draft.stem }}</span>
            <t-tag size="small" variant="light" :theme="draftStatusTheme(draft.status)">
              {{ draftStatusLabel(draft.status) }}
            </t-tag>
          </button>
        </aside>

        <section v-if="selectedDraft" class="editor-panel">
          <div class="editor-title-row">
            <div>
              <h3>{{ form.question_no || '未编号题目' }}</h3>
              <span>{{ selectedDraft.id }}</span>
            </div>
            <t-tag variant="light" :theme="draftStatusTheme(selectedDraft.status)">
              {{ draftStatusLabel(selectedDraft.status) }}
            </t-tag>
          </div>

          <t-form :data="form" label-align="top">
            <div class="form-grid">
              <t-form-item label="题号">
                <t-input v-model="form.question_no" />
              </t-form-item>
              <t-form-item label="题型">
                <t-input v-model="form.question_type_code" />
              </t-form-item>
              <t-form-item label="难度">
                <t-select v-model="form.difficulty">
                  <t-option value="unknown" label="未知" />
                  <t-option value="easy" label="简单" />
                  <t-option value="medium" label="中等" />
                  <t-option value="hard" label="困难" />
                </t-select>
              </t-form-item>
            </div>

            <t-form-item label="题干">
              <t-textarea v-model="form.stem" :autosize="{ minRows: 5, maxRows: 10 }" />
            </t-form-item>

            <t-form-item label="选项">
              <div class="options-editor">
                <div v-for="(option, index) in form.options_json" :key="index" class="option-row">
                  <t-input v-model="option.key" class="option-key" placeholder="A" />
                  <t-input v-model="option.content" placeholder="选项内容" />
                  <t-button variant="text" shape="circle" @click="removeOption(index)">
                    <template #icon><t-icon name="close" /></template>
                  </t-button>
                </div>
                <t-button variant="outline" size="small" @click="addOption">
                  <template #icon><t-icon name="add" /></template>
                  添加选项
                </t-button>
              </div>
            </t-form-item>

            <div class="form-grid two">
              <t-form-item label="答案 JSON">
                <t-textarea v-model="answerText" :autosize="{ minRows: 4, maxRows: 8 }" />
              </t-form-item>
              <t-form-item label="来源 chunk">
                <t-textarea v-model="sourceChunkText" :autosize="{ minRows: 4, maxRows: 8 }" />
              </t-form-item>
            </div>

            <t-form-item label="解析">
              <t-textarea v-model="form.explanation" :autosize="{ minRows: 4, maxRows: 8 }" />
            </t-form-item>
          </t-form>

          <div class="action-row">
            <t-space>
              <t-button theme="primary" :loading="saving" :disabled="!canEditSelected" @click="saveDraft">
                保存
              </t-button>
              <t-button theme="success" :loading="approving" :disabled="!canEditSelected" @click="approveDraft">
                确认入库
              </t-button>
              <t-button theme="danger" variant="outline" :loading="rejecting" :disabled="!canEditSelected" @click="rejectDraft">
                驳回
              </t-button>
            </t-space>
          </div>
        </section>
      </div>

      <t-empty v-else-if="!loading" description="暂无待校对草稿" />
    </t-loading>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { draftStatusLabel, draftStatusTheme, useQuestionDraftReview } from './useQuestionDraftReview'

const {
  router,
  loading,
  saving,
  approving,
  rejecting,
  data,
  drafts,
  selectedDraft,
  answerText,
  sourceChunkText,
  form,
  headerText,
  canEditSelected,
  loadDrafts,
  selectDraft,
  addOption,
  removeOption,
  saveDraft,
  approveDraft,
  rejectDraft,
} = useQuestionDraftReview()

onMounted(loadDrafts)
</script>

<style lang="less" scoped>
.exam-page {
  flex: 1;
  overflow-y: auto;
  padding: 28px 32px;
  background: var(--td-bg-color-container);
}

.exam-header {
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
    line-height: 22px;
  }
}

.review-layout {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 16px;
  min-height: 620px;
}

.draft-list,
.editor-panel {
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.draft-list {
  overflow-y: auto;
  max-height: calc(100vh - 170px);
  padding: 8px;
}

.draft-item {
  display: grid;
  width: 100%;
  grid-template-columns: 48px minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 10px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;
  text-align: left;
}

.draft-item:hover,
.draft-item.active {
  background: var(--td-bg-color-secondarycontainer);
}

.draft-no {
  font-weight: 600;
}

.draft-stem {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}

.editor-panel {
  min-width: 0;
  padding: 18px;
}

.editor-title-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;

  h3 {
    margin: 0 0 4px;
    font-size: 16px;
    font-weight: 600;
  }

  span {
    color: var(--td-text-color-placeholder);
    font-size: 12px;
  }
}

.form-grid {
  display: grid;
  grid-template-columns: 160px 220px 160px;
  gap: 14px;
}

.form-grid.two {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.option-row {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr) 36px;
  gap: 8px;
  margin-bottom: 8px;
}

.action-row {
  display: flex;
  justify-content: flex-end;
  padding-top: 12px;
  border-top: 1px solid var(--td-component-stroke);
}

@media (max-width: 900px) {
  .exam-page {
    padding: 20px 16px;
  }

  .review-layout,
  .form-grid,
  .form-grid.two {
    grid-template-columns: 1fr;
  }

  .draft-list {
    max-height: 320px;
  }
}
</style>
