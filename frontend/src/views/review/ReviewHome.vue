<template>
  <div class="exam-page">
    <div class="exam-header">
      <div>
        <h2>审核工作台</h2>
        <p>公共题库、公共知识库和低置信度结构化结果都会进入审核队列。</p>
      </div>
      <t-button variant="outline" disabled>
        <template #icon><t-icon name="refresh" /></template>
        队列待接入
      </t-button>
    </div>

    <div class="review-grid">
      <section v-for="item in reviewItems" :key="item.title" class="review-card">
        <div class="card-title">
          <t-icon :name="item.icon" />
          <h3>{{ item.title }}</h3>
        </div>
        <p>{{ item.desc }}</p>
        <t-tag variant="light">{{ item.status }}</t-tag>
      </section>
    </div>

    <section class="panel">
      <h3>审核原则</h3>
      <t-list split>
        <t-list-item>普通用户不能直接写入公共空间。</t-list-item>
        <t-list-item>试卷解析置信度不足时进入人工校正，不直接污染题库。</t-list-item>
        <t-list-item>驳回不会删除用户原始资料，只改变公共可见状态。</t-list-item>
      </t-list>
    </section>
  </div>
</template>

<script setup lang="ts">
const reviewItems = [
  { title: '公共题库提交', icon: 'folder', desc: '题库从私有或班级空间提交为公共资源时进入审核。', status: '第七阶段接入' },
  { title: '公共知识库提交', icon: 'file', desc: '优质学习资料可提交公共库，审核通过后开放检索。', status: '第七阶段接入' },
  { title: '结构化校正', icon: 'edit-1', desc: '低置信度题目边界、答案关联和解析需要人工确认。', status: '第三阶段接入' },
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

.review-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 12px;
}

.review-card,
.panel {
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  padding: 16px;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;

  h3 {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
  }
}

.review-card p,
.panel p {
  color: var(--td-text-color-secondary);
  font-size: 14px;
  line-height: 22px;
}

.panel h3 {
  margin: 0 0 10px;
  font-size: 16px;
  font-weight: 600;
}
</style>
