<template>
  <div class="exam-page">
    <div class="exam-header">
      <div>
        <h2>考试域配置</h2>
        <p>查看高考与雅思的基础科目模块，后续在此维护题型、知识点和解析策略。</p>
      </div>
      <t-button variant="outline" @click="loadData">
        <template #icon><t-icon name="refresh" /></template>
        刷新
      </t-button>
    </div>

    <t-loading :loading="loading">
      <div class="domain-grid">
        <section v-for="domain in domains" :key="domain.id" class="domain-card">
          <div class="domain-title">
            <div>
              <h3>{{ domain.name }}</h3>
              <p>{{ domain.description || domain.code }}</p>
            </div>
            <t-tag theme="success" variant="light">{{ domain.code }}</t-tag>
          </div>
          <div class="subject-list">
            <div v-for="subject in subjectsByDomain[domain.id] || []" :key="subject.id" class="subject-row">
              <span>{{ subject.name }}</span>
              <small>{{ subject.code }}</small>
            </div>
          </div>
        </section>
      </div>
    </t-loading>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { listExamDomains, listExamSubjects } from '@/api/exam/domain'
import type { ExamDomain, ExamSubject } from '@/types/exam'

const loading = ref(false)
const domains = ref<ExamDomain[]>([])
const subjectsByDomain = ref<Record<string, ExamSubject[]>>({})

const loadData = async () => {
  loading.value = true
  try {
    const domainRes = await listExamDomains()
    domains.value = domainRes.data || []
    const subjectPairs = await Promise.all(domains.value.map(async (domain) => {
      const res = await listExamSubjects(domain.id)
      return [domain.id, res.data || []] as const
    }))
    subjectsByDomain.value = Object.fromEntries(subjectPairs)
  } catch (error: any) {
    MessagePlugin.error(error?.message || '考试域配置加载失败')
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
    margin: 0;
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

.domain-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.domain-card {
  padding: 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
}

.domain-title {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;

  h3 {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
  }

  p {
    margin: 6px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 13px;
  }
}

.subject-list {
  display: grid;
  gap: 8px;
}

.subject-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);

  span {
    font-weight: 600;
  }

  small {
    color: var(--td-text-color-secondary);
  }
}
</style>
