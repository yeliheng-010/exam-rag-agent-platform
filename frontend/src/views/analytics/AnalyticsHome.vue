<template>
  <div class="exam-page">
    <div class="exam-header">
      <div>
        <h2>学习分析</h2>
        <p>聚合班级、题库、练习和 Agent 使用数据，服务老师教学决策。</p>
      </div>
      <t-button variant="outline" disabled>
        <template #icon><t-icon name="chart-bar" /></template>
        指标待生成
      </t-button>
    </div>

    <div class="metric-grid">
      <div v-for="item in metrics" :key="item.label" class="metric-card">
        <span>{{ item.label }}</span>
        <strong>{{ item.value }}</strong>
        <p>{{ item.desc }}</p>
      </div>
    </div>

    <section class="panel">
      <h3>指标来源</h3>
      <div class="source-grid">
        <div>答题记录</div>
        <div>错题本</div>
        <div>题库检索</div>
        <div>班级作业</div>
        <div>Agent 工具调用</div>
        <div>资料阅读</div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
const metrics = [
  { label: '练习次数', value: '-', desc: '第四阶段接入 practice_records 后统计。' },
  { label: '错题数量', value: '-', desc: '自动错题本形成后按知识点聚合。' },
  { label: '班级完成率', value: '-', desc: '作业系统接入后计算。' },
  { label: '资料命中率', value: '-', desc: '结合 RAG 检索引用与题目来源分析。' },
]
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

.metric-grid,
.source-grid {
  display: grid;
  gap: 12px;
}

.metric-grid {
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin-bottom: 12px;
}

.metric-card,
.panel {
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.metric-card {
  padding: 16px;

  span {
    display: block;
    margin-bottom: 10px;
    color: var(--td-text-color-secondary);
    font-size: 13px;
  }

  strong {
    display: block;
    font-size: 28px;
    font-weight: 600;
  }

  p {
    color: var(--td-text-color-placeholder);
    font-size: 12px;
    line-height: 18px;
  }
}

.panel {
  padding: 16px;

  h3 {
    margin: 0 0 12px;
    font-size: 16px;
    font-weight: 600;
  }
}

.source-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));

  div {
    padding: 14px;
    border-radius: 8px;
    background: var(--td-bg-color-secondarycontainer);
    font-weight: 600;
  }
}
</style>
