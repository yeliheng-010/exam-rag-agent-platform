import type {
  ExamQuestionGroupDraft,
  ExamQuestionGroupQualityReport,
  ExamStructuringTaskStatus,
} from '@/types/exam'

export function shouldPollQuestionGroupExtraction(status?: ExamStructuringTaskStatus) {
  return status === 'extracting'
}
export function normalizeExtractionPercent(percent?: number) {
  if (!Number.isFinite(percent)) return 0
  return Math.min(100, Math.max(0, Math.round(percent || 0)))
}

export function hasBlockingDraftQuality(
  report?: Pick<ExamQuestionGroupQualityReport, 'blocking'>,
) {
  return report?.blocking === true
}

export function draftHasBlockingQuality(draft?: ExamQuestionGroupDraft | null) {
  return hasBlockingDraftQuality(draft?.quality_report)
}

export function createQuestionGroupExtractionPoller(
  refresh: () => Promise<void>,
  intervalMs = 3000,
) {
  let timer: ReturnType<typeof setTimeout> | undefined

  const stop = () => {
    if (timer) clearTimeout(timer)
    timer = undefined
  }

  const sync = (status?: ExamStructuringTaskStatus) => {
    stop()
    if (!shouldPollQuestionGroupExtraction(status)) return
    timer = setTimeout(async () => {
      timer = undefined
      await refresh()
    }, intervalMs)
  }

  return { sync, stop }
}
