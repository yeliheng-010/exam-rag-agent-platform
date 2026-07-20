import type { ExamClassStatus, UpdateExamClassPayload } from '../../types/exam.ts'

type ExamClassSettingsTarget = {
  owner_user_id: string
  status: ExamClassStatus
}

export type ExamClassSettingsCommand = 'archive' | 'restore'

export function canManageClassSettings(
  classInfo: ExamClassSettingsTarget | null | undefined,
  currentUserId: string,
): boolean {
  return Boolean(classInfo && currentUserId && classInfo.owner_user_id === currentUserId)
}

export function classSettingsCommand(
  classInfo: ExamClassSettingsTarget | null | undefined,
  currentUserId: string,
): ExamClassSettingsCommand | null {
  if (!canManageClassSettings(classInfo, currentUserId) || !classInfo) {
    return null
  }
  return classInfo.status === 'archived' ? 'restore' : 'archive'
}

export function isArchivedExamClass(classInfo: Pick<ExamClassSettingsTarget, 'status'> | null | undefined): boolean {
  return classInfo?.status === 'archived'
}

export function buildClassSettingsPayload(payload: UpdateExamClassPayload): UpdateExamClassPayload {
  return {
    name: payload.name.trim(),
    description: payload.description.trim(),
    member_limit: payload.member_limit,
  }
}

export function isValidClassSettingsPayload(payload: UpdateExamClassPayload): boolean {
  const nameLength = [...payload.name.trim()].length
  const descriptionLength = [...payload.description.trim()].length
  return nameLength >= 1 &&
    nameLength <= 255 &&
    descriptionLength <= 2000 &&
    Number.isInteger(payload.member_limit) &&
    payload.member_limit >= 0 &&
    payload.member_limit <= 1000
}
