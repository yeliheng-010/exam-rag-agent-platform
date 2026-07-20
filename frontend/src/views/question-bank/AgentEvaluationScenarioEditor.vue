<template>
  <section class="scenario-editor" aria-label="Agent 评测场景">
    <header class="scenario-editor__head">
      <div><h3>评测场景</h3><span>{{ modelValue.length }} / 20</span></div>
      <t-button size="small" variant="outline" :disabled="modelValue.length >= 20" @click="addScenario">
        <template #icon><t-icon name="add" /></template>
        新增场景
      </t-button>
    </header>

    <article v-for="(scenario, scenarioIndex) in modelValue" :key="scenarioIndex" class="scenario-item">
      <header>
        <strong>场景 {{ scenarioIndex + 1 }}</strong>
        <t-button
          shape="square"
          variant="text"
          size="small"
          title="删除场景"
          :disabled="modelValue.length <= 1"
          @click="removeScenario(scenarioIndex)"
        >
          <template #icon><t-icon name="delete" /></template>
        </t-button>
      </header>

      <div class="scenario-fields scenario-fields--primary">
        <label><span>场景名称</span><t-input :value="scenario.name" @change="updateScenario(scenarioIndex, 'name', $event)" /></label>
        <label><span>用户输入</span><t-textarea :value="scenario.input" :autosize="{ minRows: 2, maxRows: 5 }" @change="updateScenario(scenarioIndex, 'input', $event)" /></label>
      </div>

      <div class="tool-expectations">
        <header><span>预期工具</span><t-button size="small" variant="text" @click="addToolCall(scenarioIndex)">添加调用</t-button></header>
        <div v-if="!scenario.expectedToolCalls.length" class="empty-tool-row">不要求工具调用</div>
        <div v-for="(call, callIndex) in scenario.expectedToolCalls" :key="callIndex" class="tool-row">
          <label>
            <span>工具</span>
            <t-select :value="call.name" :options="toolOptions" @change="updateToolCall(scenarioIndex, callIndex, 'name', $event)" />
          </label>
          <label>
            <span>参数 JSON</span>
            <t-textarea :value="call.argumentsJSON" :autosize="{ minRows: 1, maxRows: 4 }" @change="updateToolCall(scenarioIndex, callIndex, 'argumentsJSON', $event)" />
          </label>
          <t-button shape="square" variant="text" size="small" title="删除调用" @click="removeToolCall(scenarioIndex, callIndex)">
            <template #icon><t-icon name="close" /></template>
          </t-button>
        </div>
      </div>

      <div class="scenario-fields scenario-fields--assertions">
        <label><span>证据短语</span><t-textarea :value="scenario.evidencePhrases" placeholder="每行一个" @change="updateScenario(scenarioIndex, 'evidencePhrases', $event)" /></label>
        <label><span>引用标识</span><t-textarea :value="scenario.citations" placeholder="每行一个" @change="updateScenario(scenarioIndex, 'citations', $event)" /></label>
        <label><span>答案短语</span><t-textarea :value="scenario.answerPhrases" placeholder="每行一个" @change="updateScenario(scenarioIndex, 'answerPhrases', $event)" /></label>
        <label><span>事实短语</span><t-textarea :value="scenario.groundedPhrases" placeholder="每行一个" @change="updateScenario(scenarioIndex, 'groundedPhrases', $event)" /></label>
      </div>
    </article>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AgentEvaluationScenarioDraft } from './agentEvaluationViewModel'
import { createAgentEvaluationScenario } from './agentEvaluationViewModel'

const props = defineProps<{ modelValue: AgentEvaluationScenarioDraft[]; tools: string[] }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: AgentEvaluationScenarioDraft[]): void }>()

const toolOptions = computed(() => props.tools.map(tool => ({ label: tool, value: tool })))
const copy = () => props.modelValue.map(item => ({
  ...item,
  expectedToolCalls: item.expectedToolCalls.map(call => ({ ...call })),
}))

const updateScenario = (index: number, key: keyof AgentEvaluationScenarioDraft, value: unknown) => {
  const next = copy()
  next[index] = { ...next[index], [key]: String(value ?? '') }
  emit('update:modelValue', next)
}

const addScenario = () => {
  if (props.modelValue.length >= 20) return
  emit('update:modelValue', [...copy(), createAgentEvaluationScenario(props.modelValue.length + 1)])
}

const removeScenario = (index: number) => {
  if (props.modelValue.length <= 1) return
  emit('update:modelValue', copy().filter((_, itemIndex) => itemIndex !== index))
}

const addToolCall = (scenarioIndex: number) => {
  const next = copy()
  next[scenarioIndex].expectedToolCalls.push({ name: props.tools[0] || '', argumentsJSON: '{}' })
  emit('update:modelValue', next)
}

const removeToolCall = (scenarioIndex: number, callIndex: number) => {
  const next = copy()
  next[scenarioIndex].expectedToolCalls.splice(callIndex, 1)
  emit('update:modelValue', next)
}

const updateToolCall = (scenarioIndex: number, callIndex: number, key: 'name' | 'argumentsJSON', value: unknown) => {
  const next = copy()
  next[scenarioIndex].expectedToolCalls[callIndex][key] = String(value ?? '')
  emit('update:modelValue', next)
}
</script>

<style lang="less" scoped src="./AgentEvaluationScenarioEditor.less"></style>
