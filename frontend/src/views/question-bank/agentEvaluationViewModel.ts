import type {
  ExamAgentEvaluationCase,
  ExamAgentEvaluationRun,
  ExamRAGEvaluationRunStatus,
} from '../../types/exam.ts'

export interface AgentEvaluationToolCallDraft {
  name: string
  argumentsJSON: string
}

export interface AgentEvaluationScenarioDraft {
  name: string
  input: string
  expectedToolCalls: AgentEvaluationToolCallDraft[]
  evidencePhrases: string
  citations: string
  answerPhrases: string
  groundedPhrases: string
}

export interface AgentEvaluationMetric {
  label: string
  value: string
}

export interface AgentEvaluationModelOption {
  id?: string
  name: string
  display_name?: string
}

export const AGENT_EVALUATION_READ_ONLY_TOOLS = new Set([
  'thinking',
  'todo_write',
  'grep_chunks',
  'knowledge_search',
  'exam_question_context',
  'exam_learning_diagnosis',
  'exam_class_diagnosis',
  'exam_practice_recommendation',
  'list_knowledge_chunks',
  'query_knowledge_graph',
  'get_document_info',
  'database_query',
  'wiki_read_page',
  'wiki_search',
  'wiki_read_issue',
])

const DEFAULT_AGENT_EVALUATION_TOOLS = [
  'thinking',
  'todo_write',
  'knowledge_search',
  'exam_question_context',
  'exam_learning_diagnosis',
  'exam_class_diagnosis',
  'exam_practice_recommendation',
  'grep_chunks',
  'list_knowledge_chunks',
  'query_knowledge_graph',
  'get_document_info',
  'database_query',
]

export const createAgentEvaluationScenario = (index: number): AgentEvaluationScenarioDraft => ({
  name: `场景 ${index}`,
  input: '请根据题库证据回答问题。',
  expectedToolCalls: [],
  evidencePhrases: '',
  citations: '',
  answerPhrases: '',
  groundedPhrases: '',
})

export const getSafeAgentEvaluationTools = (tools: string[] = []) => {
  const seen = new Set<string>()
  const source = tools.length ? tools : DEFAULT_AGENT_EVALUATION_TOOLS
  return source.filter((tool) => {
    if (!AGENT_EVALUATION_READ_ONLY_TOOLS.has(tool) || seen.has(tool)) return false
    seen.add(tool)
    return true
  })
}

const phraseLines = (value: string) => value
  .split(/\r?\n/)
  .map(item => item.trim())
  .filter(Boolean)

const parseArguments = (value: string, scenarioIndex: number) => {
  const source = value.trim() || '{}'
  try {
    const parsed = JSON.parse(source)
    if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') throw new Error()
    return parsed as Record<string, unknown>
  } catch {
    throw new Error(`场景 ${scenarioIndex + 1} 的工具参数必须是 JSON 对象`)
  }
}

export const buildAgentEvaluationCases = (
  drafts: AgentEvaluationScenarioDraft[],
): ExamAgentEvaluationCase[] => drafts.map((draft, scenarioIndex) => {
  const name = draft.name.trim()
  const input = draft.input.trim()
  const expectedToolCalls = draft.expectedToolCalls.map(call => ({
    name: call.name.trim(),
    arguments: parseArguments(call.argumentsJSON, scenarioIndex),
  }))
  if (!name || !input) throw new Error(`场景 ${scenarioIndex + 1} 缺少名称或用户输入`)
  if (expectedToolCalls.some(call => !call.name)) throw new Error(`场景 ${scenarioIndex + 1} 缺少预期工具`)
  return {
    name,
    input,
    expected_tool_calls: expectedToolCalls,
    expected_evidence_phrases: phraseLines(draft.evidencePhrases),
    expected_citations: phraseLines(draft.citations),
    expected_answer_phrases: phraseLines(draft.answerPhrases),
    grounded_phrases: phraseLines(draft.groundedPhrases),
  }
})

export const formatAgentRate = (value = 0) => `${Math.round(value * 1000) / 10}%`

export const formatAgentDuration = (value = 0) => `${Math.round(value)} ms`

export const resolveAgentModelName = (
  modelID: string | undefined,
  models: AgentEvaluationModelOption[],
) => {
  const id = modelID?.trim() || ''
  if (!id) return '-'
  const model = models.find(item => item.id === id)
  return model?.display_name?.trim() || model?.name?.trim() || id
}

export const runStatusLabel = (status: ExamRAGEvaluationRunStatus) => ({
  queued: '排队中',
  running: '运行中',
  completed: '已完成',
  failed: '失败',
}[status])

export const getAgentRunMetrics = (run: ExamAgentEvaluationRun | null): AgentEvaluationMetric[] => {
  const summary = run?.result_snapshot?.summary
  return [
    { label: '通过率', value: formatAgentRate(summary?.pass_rate) },
    { label: '工具序列', value: formatAgentRate(summary?.tool_sequence_rate) },
    { label: '参数命中', value: formatAgentRate(summary?.tool_arguments_rate) },
    { label: '证据覆盖', value: formatAgentRate(summary?.evidence_rate) },
    { label: '事实支撑', value: formatAgentRate(summary?.groundedness_rate) },
    { label: '平均耗时', value: formatAgentDuration(summary?.average_duration_ms) },
  ]
}
