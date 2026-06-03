export const money = (cents?: number) => `¥${((cents || 0) / 100).toFixed(2)}`

export const moneyCompact = (cents?: number) => `¥${Number(((cents || 0) / 100).toFixed(0)).toLocaleString('zh-CN')}`

export const directionText = (value: string) => value === 'in' ? '收礼' : '送礼'

export const reciprocityRoleText = (direction: string) => direction === 'in' ? '我方回礼' : '对方回礼'

export const reciprocityRelationText = (direction: string) => direction === 'in' ? '对方回礼流水' : '我方回礼流水'

export const boolText = (value: boolean) => value ? '是' : '否'

export const statusText = (value: string | boolean) => {
  if (value === true) return '成功'
  if (value === false) return '失败'
  return ({
    enabled: '启用',
    disabled: '禁用',
    success: '成功',
    failed: '失败',
    running: '执行中',
    blocked: '已阻断',
    dry_run_ready: '校验通过',
    dry_run_failed: '校验失败',
    none: '无需回礼',
    pending: '待回礼',
    done: '已回礼',
    manual: '手动',
    scheduled: '定时',
    super_admin: '系统管理员',
    family_admin: '家庭管理员',
    member: '家庭成员'
  } as Record<string, string>)[value] || value || '-'
}

export function formatDateTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

export function formatDate(value?: string) {
  return formatDateTime(value).slice(0, 10)
}
