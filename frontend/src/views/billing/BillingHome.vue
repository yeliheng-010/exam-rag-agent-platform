<template>
  <div class="exam-page">
    <div class="exam-header">
      <div>
        <h2>支付与权益</h2>
        <p>统一承载解析额度、Agent 调用额度、班级人数和公共资源审核权益。</p>
      </div>
      <t-button theme="primary" disabled>
        <template #icon><t-icon name="wallet" /></template>
        支付接口待接入
      </t-button>
    </div>

    <div class="metric-grid">
      <div class="metric-card">
        <span>文档解析额度</span>
        <strong>0 / 0</strong>
        <p>第二阶段资料入库后开始统计。</p>
      </div>
      <div class="metric-card">
        <span>试卷结构化额度</span>
        <strong>0 / 0</strong>
        <p>第三阶段试卷解析任务接入。</p>
      </div>
      <div class="metric-card">
        <span>Agent 工具调用</span>
        <strong>0 / 0</strong>
        <p>第五阶段按工具调用进行权益校验。</p>
      </div>
      <div class="metric-card">
        <span>班级席位</span>
        <strong>基础版</strong>
        <p>按班级成员上限和付费套餐约束。</p>
      </div>
    </div>

    <div class="panel-grid">
      <section class="panel">
        <h3>支付接口规划</h3>
        <p>支付模块会与业务动作解耦：支付订单负责开通权益，资料解析、Agent 调用和练习生成只做权益检查与扣减。</p>
        <div class="step-list">
          <div><span>1</span>商品与套餐配置</div>
          <div><span>2</span>订单创建与支付回调</div>
          <div><span>3</span>权益账户开通</div>
          <div><span>4</span>业务动作幂等扣减</div>
        </div>
      </section>

      <section class="panel">
        <h3>当前阶段状态</h3>
        <t-list split>
          <t-list-item>已预留支付与权益入口</t-list-item>
          <t-list-item>后端将在第六阶段新增商品、订单、权益和流水表</t-list-item>
          <t-list-item>不在第一阶段硬编码支付密钥或真实支付渠道</t-list-item>
        </t-list>
      </section>
    </div>
  </div>
</template>

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

.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
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
    font-size: 24px;
    line-height: 32px;
    font-weight: 600;
  }

  p {
    margin: 8px 0 0;
    color: var(--td-text-color-placeholder);
    font-size: 12px;
    line-height: 18px;
  }
}

.panel-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.panel {
  padding: 16px;

  h3 {
    margin: 0 0 6px;
    font-size: 16px;
    font-weight: 600;
  }

  p {
    margin: 0 0 14px;
    color: var(--td-text-color-secondary);
    font-size: 14px;
    line-height: 22px;
  }
}

.step-list {
  display: grid;
  gap: 8px;

  div {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 12px;
    border-radius: 6px;
    background: var(--td-bg-color-secondarycontainer);
    font-weight: 600;
  }

  span {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    border-radius: 50%;
    background: var(--td-brand-color-light);
    color: var(--td-brand-color);
    font-size: 12px;
  }
}
</style>
