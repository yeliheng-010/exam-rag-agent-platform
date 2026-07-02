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
        <section class="panel">
          <div class="panel-title">
            <h3>题目列表</h3>
            <p>第三阶段接入试卷解析后，题目会按题型、难度、年份、知识点进入这里。</p>
          </div>
          <t-empty description="暂无结构化题目">
            <template #action>
              <t-button variant="outline" @click="router.push('/platform/knowledge-bases')">
                先上传学习资料
              </t-button>
            </template>
          </t-empty>
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
import { getQuestionBank } from '@/api/exam/question-bank'
import type { QuestionBank, ReviewStatus } from '@/types/exam'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const bank = ref<QuestionBank | null>(null)

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

const loadData = async () => {
  const bankId = String(route.params.bankId || '')
  if (!bankId) return
  loading.value = true
  try {
    const res = await getQuestionBank(bankId)
    bank.value = res.data
  } catch (error: any) {
    MessagePlugin.error(error?.message || '题库详情加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
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

.summary-grid,
.content-grid {
  display: grid;
  gap: 12px;
}

.summary-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin-bottom: 12px;
}

.content-grid {
  grid-template-columns: minmax(0, 1.4fr) minmax(320px, 0.6fr);
}

.summary-item,
.panel {
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.summary-item {
  min-width: 0;
  padding: 14px 16px;

  span {
    display: block;
    margin-bottom: 8px;
    color: var(--td-text-color-secondary);
    font-size: 12px;
  }

  strong {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 14px;
    font-weight: 600;
  }
}

.panel {
  min-height: 360px;
  padding: 16px;
}

.panel-title {
  margin-bottom: 14px;

  h3 {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
  }

  p {
    margin: 6px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 13px;
    line-height: 20px;
  }
}

.pipeline {
  display: flex;
  flex-direction: column;
  gap: 10px;

  div {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px;
    border-radius: 8px;
    background: var(--td-bg-color-secondarycontainer);
    font-size: 14px;
    font-weight: 600;
  }

  span {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    border-radius: 50%;
    background: var(--td-brand-color-light);
    color: var(--td-brand-color);
    font-size: 12px;
  }
}
</style>
