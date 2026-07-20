export interface ReferenceAnswerInput {
  answer_text: string
  is_correct: boolean
}

const findNextSubquestionStart = (questionNo: string, answerText: string) => {
  const subquestion = questionNo.match(/[（(]\s*(\d+)\s*[）)]\s*$/)
  if (!subquestion) return -1

  const nextNumber = Number(subquestion[1]) + 1
  const blockBoundary = '(?:^|\\r?\\n|<br\\s*\\/?>|<\\/p>)'
  const optionalBlockStart = '\\s*(?:<p[^>]*>\\s*)?(?:#{1,6}\\s*)?'
  const patterns = [
    new RegExp(`${blockBoundary}${optionalBlockStart}[（(]\\s*${nextNumber}\\s*[）)]`, 'im'),
    new RegExp(`${blockBoundary}${optionalBlockStart}第\\s*${nextNumber}\\s*(?:问|小题)`, 'im'),
  ]
  const starts = patterns
    .map(pattern => answerText.search(pattern))
    .filter(index => index > 0)
  return starts.length ? Math.min(...starts) : -1
}

const scopeReferenceAnswer = (questionNo: string, answerText: string) => {
  const trimmed = answerText.trim()
  const nextSubquestionStart = findNextSubquestionStart(questionNo, trimmed)
  return (nextSubquestionStart > 0 ? trimmed.slice(0, nextSubquestionStart) : trimmed).trim()
}

export const extractAuthoritativeReferenceAnswer = (
  questionNo: string,
  answers: ReferenceAnswerInput[],
) => {
  const usableAnswers = answers.filter(answer => answer.answer_text?.trim())
  const correctAnswers = usableAnswers.filter(answer => answer.is_correct)
  const preferredAnswers = correctAnswers.length ? correctAnswers : usableAnswers
  const scopedAnswers = preferredAnswers
    .map(answer => scopeReferenceAnswer(questionNo, answer.answer_text))
    .filter(Boolean)
  return [...new Set(scopedAnswers)].join('\n\n')
}

export const buildAuthoritativeReferenceAnswerPrompt = (
  questionNo: string,
  answers: ReferenceAnswerInput[],
) => {
  const referenceAnswer = extractAuthoritativeReferenceAnswer(questionNo, answers)
  if (!referenceAnswer) return ''

  return [
    '【题库参考答案（权威）】',
    referenceAnswer,
    '以上题库参考答案是本题判断与推导的唯一权威依据。只解释和展开参考答案，不要重新判断题目或答案是否成立，不得声称题目条件不足；若题面识别与参考答案存在冲突，以参考答案为准。',
    '讲解必须直接沿用参考答案的推导顺序，所有数学公式使用 LaTeX，并先完整复述原始条件再逐行解释等价变形。不得展示与参考答案冲突的试算、自我纠正或失败过程；输出前检查全文，删除任何与参考答案矛盾的步骤。',
    '相邻参考公式之间只允许一步等价变形；含分式时必须一步同乘所有分母的最小公倍式，直接得到参考答案中的下一式，禁止先乘部分因子或插入额外中间式。',
    '若引入辅助数列，必须先用明确等式定义并始终保持该定义；不得把不同数列写成相等，不得把需要证明的目标数列偷换为原数列，最终结论中的数列必须与题目和参考答案一致。',
    '输出只按“原始条件、等价变形、定义目标数列、相邻差与公差、复盘建议”生成一次；完成目标数列结论和一条复盘建议后立即停止，不得重新证明，不得继续讨论其他数列是否为等差数列。',
  ].join('\n')
}
