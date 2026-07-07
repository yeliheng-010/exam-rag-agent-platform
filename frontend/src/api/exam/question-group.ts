import { get } from '@/utils/request'
import type { ApiResponse, QuestionGroupDetail } from '@/types/exam'

export function listQuestionGroupDetails(bankId: string) {
  return get(`/api/v1/exam/question-banks/${bankId}/question-groups`) as unknown as Promise<ApiResponse<QuestionGroupDetail[]>>
}
