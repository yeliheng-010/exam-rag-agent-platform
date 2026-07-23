import assert from 'node:assert/strict'
import test from 'node:test'

import { baseCompile } from '@intlify/message-compiler'

import enUS from './locales/en-US.ts'
import koKR from './locales/ko-KR.ts'
import ruRU from './locales/ru-RU.ts'
import zhCN from './locales/zh-CN.ts'

const contextualGuides = {
  'en-US': enUS.contextualGuide,
  'ko-KR': koKR.contextualGuide,
  'ru-RU': ruRU.contextualGuide,
  'zh-CN': zhCN.contextualGuide,
}

function flattenMessages(value: unknown, path = ''): Array<[string, string]> {
  if (typeof value === 'string') {
    return [[path, value]]
  }
  if (!value || typeof value !== 'object') {
    return []
  }

  return Object.entries(value).flatMap(([key, child]) =>
    flattenMessages(child, path ? `${path}.${key}` : key),
  )
}

test('all contextual guide translations compile as Vue I18n messages', () => {
  const errors: string[] = []

  for (const [locale, guide] of Object.entries(contextualGuides)) {
    for (const [key, message] of flattenMessages(guide)) {
      baseCompile(message, {
        onError(error) {
          errors.push(`${locale}.${key}: ${error.message}`)
        },
      })
    }
  }

  assert.deepEqual(errors, [])
})
