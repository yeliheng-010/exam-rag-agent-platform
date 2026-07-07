<template>
  <div class="exam-page">
    <div class="exam-header">
      <div>
        <t-button variant="text" size="small" @click="router.push('/platform/question-banks')">
          <template #icon><t-icon name="chevron-left" /></template>
          返回题库
        </t-button>
        <h2>{{ bank?.name || '题库详情' }}</h2>
        <p>{{ bank?.description || '结构化题目、答案、解析和知识点关联会沉淀在题库中。' }}</p>
      </div>
      <t-tag v-if="bank" :theme="reviewTag(bank.review_status)" variant="light">{{ reviewLabel(bank.review_status) }}</t-tag>
    </div>

    <t-loading :loading="loading">
      <div v-if="bank" class="summary-grid">
        <div class="summary-item">
          <span>空间 ID</span>
          <strong>{{ bank.space_id }}</strong>
        </div>
        <div class="summary-item">
          <span>考试域 ID</span>
          <strong>{{ bank.domain_id }}</strong>
        </div>
        <div class="summary-item">
          <span>来源</span>
          <strong>{{ bank.source_type }}</strong>
        </div>
      </div>

      <div class="content-grid">
        <section class="panel question-section">
          <div class="panel-title">
            <h3>题组</h3>
            <p>已确认的阅读篇章、数学题干和小题会以题组形态沉淀为正式题库资产。</p>
            <span>{{ questionGroups.length }} 组</span>
          </div>
          <div v-if="questionGroups.length" class="question-group-list">
            <article v-for="group in questionGroups" :key="group.group.id" class="question-group-card">
              <header class="question-group-card__head">
                <div>
                  <h4>{{ group.group.title || group.group.group_type }}</h4>
                  <p>{{ group.group.group_type }} · {{ group.questions.length }} 题</p>
                </div>
                <t-tag :theme="reviewTag(group.group.review_status)" variant="light">
                  {{ reviewLabel(group.group.review_status) }}
                </t-tag>
              </header>

              <section v-if="group.group.material_text" class="question-group-material">
                <div class="question-group-material__head">
                  <div>
                    <h5>{{ materialTitle(group) }}</h5>
                    <span>{{ materialMeta(group) }}</span>
                  </div>
                  <t-button
                    v-if="canToggleMaterial(group)"
                    variant="text"
                    size="small"
                    @click="toggleMaterial(group)"
                  >
                    <template #icon>
                      <t-icon :name="isMaterialExpanded(group) ? 'chevron-up' : 'chevron-down'" />
                    </template>
                    {{ isMaterialExpanded(group) ? '收起原文' : '展开全文' }}
                  </t-button>
                </div>
                <div
                  class="question-group-material__body"
                  :class="{ 'is-expanded': isMaterialExpanded(group), 'is-collapsible': canToggleMaterial(group) }"
                >
                  {{ group.group.material_text }}
                </div>
              </section>

              <div v-if="group.assets?.length" class="question-group-assets">
                <div v-for="asset in group.assets" :key="asset.id" class="question-group-asset">
                  {{ assetLabel(asset) }}
                </div>
              </div>

              <div class="question-items">
                <article v-for="item in group.questions" :key="item.question.id" class="question-item">
                  <div class="question-item__no">{{ questionNo(item) }}</div>
                  <div class="question-item__content">
                    <p class="question-stem">{{ item.question.stem }}</p>
                    <div v-if="item.options?.length" class="question-options">
                      <div
                        v-for="option in item.options"
                        :key="option.id || `${item.question.id}-${option.option_key}`"
                        class="question-option"
                        :class="{ 'is-correct': isCorrectOption(item, option.option_key) }"
                      >
                        <span class="question-option__key">{{ option.option_key }}</span>
                        <span class="question-option__content">{{ option.content }}</span>
                        <t-tag v-if="isCorrectOption(item, option.option_key)" size="small" theme="success" variant="light">
                          正确答案
                        </t-tag>
                      </div>
                    </div>
                    <div class="question-answer">
                      <span>答案：{{ answerSummary(item) }}</span>
                      <span v-if="explanationSummary(item)">解析：{{ explanationSummary(item) }}</span>
                    </div>
                  </div>
                </article>
              </div>
            </article>
          </div>
          <t-empty v-else description="暂无结构化题组" />
        </section>

        <section class="panel">
          <div class="panel-title">
            <h3>处理链路</h3>
            <p>题库不是文档 chunk 的替代，而是引用 chunk 形成可练习、可讲解的结构化资产。</p>
          </div>
          <div class="pipeline">
            <div><span>1</span>资料入库</div>
            <div><span>2</span>题目边界识别</div>
            <div><span>3</span>答案解析关联</div>
            <div><span>4</span>写入题库并引用 chunk</div>
          </div>
        </section>
      </div>
    </t-loading>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import { getQuestionBank, listQuestionGroupDetails } from '@/api/exam/question-bank'
import type { QuestionBank, QuestionDetail, QuestionGroupAsset, QuestionGroupDetail, ReviewStatus } from '@/types/exam'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const bank = ref<QuestionBank | null>(null)
const questionGroups = ref<QuestionGroupDetail[]>([])
const expandedMaterials = ref<Record<string, boolean>>({})

const materialToggleLength = 420
const materialToggleLines = 8

const reviewLabel = (status: ReviewStatus) => {
  const map: Record<ReviewStatus, string> = {
    private: '私有',
    pending: '待审核',
    approved: '已公开',
    rejected: '已驳回',
  }
  return map[status] || status
}

const reviewTag = (status: ReviewStatus) => {
  if (status === 'approved') return 'success'
  if (status === 'pending') return 'warning'
  if (status === 'rejected') return 'danger'
  return 'default'
}

const assetLabel = (asset: QuestionGroupAsset) => {
  return [asset.asset_type, asset.alt_text || asset.storage_uri].filter(Boolean).join(' · ') || '资源'
}

const questionNo = (item: QuestionDetail) => {
  return item.question.question_no || String(item.question.order_in_group || '-')
}

const answerValues = (item: QuestionDetail) => {
  return item.answers?.map(answer => answer.answer_text).filter(Boolean) || []
}

const answerSummary = (item: QuestionDetail) => {
  return answerValues(item).join('，') || '-'
}

const answerKeys = (item: QuestionDetail) => {
  return answerValues(item)
    .flatMap(value => value.toUpperCase().split(/[\s,，、;；/]+/))
    .map(value => value.trim())
    .filter(Boolean)
}

const isCorrectOption = (item: QuestionDetail, optionKey: string) => {
  return answerKeys(item).includes(optionKey.toUpperCase())
}

const explanationSummary = (item: QuestionDetail) => {
  return item.explanations?.map(explanation => explanation.explanation_text).filter(Boolean).join('；') || ''
}

const canToggleMaterial = (group: QuestionGroupDetail) => {
  const text = group.group.material_text || ''
  return text.length > materialToggleLength || text.split(/\r?\n/).length > materialToggleLines
}

const isMaterialExpanded = (group: QuestionGroupDetail) => {
  return Boolean(expandedMaterials.value[group.group.id])
}

const toggleMaterial = (group: QuestionGroupDetail) => {
  expandedMaterials.value = {
    ...expandedMaterials.value,
    [group.group.id]: !expandedMaterials.value[group.group.id],
  }
}

const materialTitle = (group: QuestionGroupDetail) => {
  if (group.group.group_type === 'reading_passage') return '阅读原文'
  if (group.group.group_type === 'math_problem') return '题干材料'
  return '共享材料'
}

const materialMeta = (group: QuestionGroupDetail) => {
  return group.group.source_chunk_ids?.length ? `引用 ${group.group.source_chunk_ids.length} 个 chunk` : '已沉淀为题组材料'
}

const loadData = async () => {
  const bankId = String(route.params.bankId || '')
  if (!bankId) return
  loading.value = true
  try {
    const res = await getQuestionBank(bankId)
    bank.value = res.data
    const groupRes = await listQuestionGroupDetails(bankId)
    questionGroups.value = groupRes.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '题库详情加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
</script>

<style lang="less" scoped src="./QuestionBankDetail.less"></style>
