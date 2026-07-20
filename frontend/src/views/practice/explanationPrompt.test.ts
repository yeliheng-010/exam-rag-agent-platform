import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildAuthoritativeReferenceAnswerPrompt,
  extractAuthoritativeReferenceAnswer,
} from './explanationPrompt.ts'

test('extracts only the current subquestion from a shared reference answer', () => {
  const answerText = [
    '（1）由题库递推式可得',
    '$$(n+1)a_{n+1}=na_n+1$$',
    '令 $b_n=na_n$，则 $b_{n+1}-b_n=1$，所以公差为 1。',
    '![当前小题最后一张公式](local://10000/exports/current.png)',
    '',
    '（2）![下一小题公式](local://10000/exports/next.png)',
    '这里开始是下一小题的证明，不应进入 16(1) 的讲解。',
  ].join('\n')

  const actual = extractAuthoritativeReferenceAnswer('16(1)', [
    { answer_text: answerText, is_correct: true },
  ])

  assert.match(actual, /\(n\+1\)a_\{n\+1\}=na_n\+1/)
  assert.match(actual, /b_\{n\+1\}-b_n=1/)
  assert.match(actual, /公差为 1/)
  assert.match(actual, /current\.png/)
  assert.doesNotMatch(actual, /next\.png/)
  assert.doesNotMatch(actual, /下一小题/)
})

test('does not mistake an inline equation number for the next subquestion', () => {
  const answerText = [
    '（1）先得到式 (2)，再继续推导。',
    '因此当前小题结论成立。',
    '（2）下一小题。',
  ].join('\n')

  const actual = extractAuthoritativeReferenceAnswer('16(1)', [
    { answer_text: answerText, is_correct: true },
  ])

  assert.match(actual, /式 \(2\)/)
  assert.doesNotMatch(actual, /下一小题/)
})

test('builds a prompt that treats the question-bank answer as authoritative', () => {
  const actual = buildAuthoritativeReferenceAnswerPrompt('16(1)', [
    { answer_text: '（1）公差为 1。', is_correct: true },
  ])

  assert.match(actual, /【题库参考答案（权威）】/)
  assert.match(actual, /只解释和展开参考答案/)
  assert.match(actual, /不要重新判断题目或答案是否成立/)
  assert.match(actual, /不得声称题目条件不足/)
  assert.match(actual, /直接沿用参考答案的推导顺序/)
  assert.match(actual, /所有数学公式使用 LaTeX/)
  assert.match(actual, /不得展示与参考答案冲突的试算、自我纠正或失败过程/)
  assert.match(actual, /不得把不同数列写成相等/)
  assert.match(actual, /不得把需要证明的目标数列偷换为原数列/)
  assert.match(actual, /完成目标数列结论和一条复盘建议后立即停止/)
  assert.match(actual, /不得继续讨论其他数列是否为等差数列/)
  assert.match(actual, /必须一步同乘所有分母的最小公倍式/)
  assert.match(actual, /禁止先乘部分因子/)
})
