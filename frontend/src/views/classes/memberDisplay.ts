export interface MemberDisplayFields {
  display_name?: string | null
  display_id?: string | null
  user_id?: string | null
}

export const memberDisplayName = (member?: MemberDisplayFields) => {
  return member?.display_name || '未命名学生'
}

export const memberDisplayId = (member?: MemberDisplayFields) => {
  const id = member?.display_id || member?.user_id || '未生成'
  return `编号：${id}`
}
