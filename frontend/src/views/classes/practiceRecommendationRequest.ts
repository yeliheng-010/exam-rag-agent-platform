export type LatestRequestOutcome<T> =
  | { status: 'success'; data: T }
  | { status: 'error'; error: unknown }
  | { status: 'stale' }

export const createLatestRequestRunner = () => {
  let latestRequestID = 0

  return {
    async run<T>(request: () => Promise<T>): Promise<LatestRequestOutcome<T>> {
      const requestID = ++latestRequestID
      try {
        const data = await request()
        return requestID === latestRequestID ? { status: 'success', data } : { status: 'stale' }
      } catch (error) {
        return requestID === latestRequestID ? { status: 'error', error } : { status: 'stale' }
      }
    },
  }
}
