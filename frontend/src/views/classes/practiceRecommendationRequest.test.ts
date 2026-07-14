import assert from 'node:assert/strict'
import test from 'node:test'
import { createLatestRequestRunner } from './practiceRecommendationRequest.ts'

const deferred = <T>() => {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

test('marks an older response stale after a newer request starts', async () => {
  const runner = createLatestRequestRunner()
  const first = deferred<string>()
  const second = deferred<string>()
  const firstResult = runner.run(() => first.promise)
  const secondResult = runner.run(() => second.promise)

  second.resolve('new')
  assert.deepEqual(await secondResult, { status: 'success', data: 'new' })

  first.resolve('old')
  assert.deepEqual(await firstResult, { status: 'stale' })
})

test('reports only the latest request error', async () => {
  const runner = createLatestRequestRunner()
  const error = new Error('request failed')

  assert.deepEqual(await runner.run(() => Promise.reject(error)), { status: 'error', error })
})
