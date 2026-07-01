import { h } from 'vue'
import { NTooltip, type DataTableColumns } from 'naive-ui'
import { formatDateTime } from '../../shared/lib/formatters'

export const auditActionLabels: Record<string, string> = {
  login_success: '登录成功',
  login_failed: '登录失败',
  login_blocked: '登录封禁',
  logout: '退出登录',
  create_family: '创建家庭',
  create_user: '创建用户',
  change_password: '修改密码',
  reset_password: '重置密码',
  create_contact: '新增对象',
  update_contact: '编辑对象',
  delete_contact: '删除对象',
  create_gift_record: '新增流水',
  update_gift_record: '编辑流水',
  delete_gift_record: '删除流水',
  create_dictionary_item: '新增字典项',
  update_dictionary_item: '编辑字典项',
  delete_dictionary_item: '删除字典项',
  update_backup_config: '更新备份配置',
  manual_export: '手动导出',
  import_dry_run: '导入校验',
  execute_import: '执行导入',
  cleanup_audit_logs: '清理审计日志'
}

export const auditResourceLabels: Record<string, string> = {
  user: '用户',
  family: '家庭',
  contact: '往来对象',
  gift_record: '往来流水',
  dictionary_item: '字典项',
  dictionary_type: '字典类型',
  backup_config: '备份配置',
  backup_job: '备份任务',
  import_job: '导入批次',
  user_session: '用户会话',
  auth_throttle: '登录防护',
  audit_log: '审计日志'
}

export function actionText(value: string) {
  return auditActionLabels[value] || value || '-'
}

export function resourceText(value: string) {
  return auditResourceLabels[value] || value || '-'
}

export function appendAuditSearchParams(params: URLSearchParams, keywordValue: string) {
  const keyword = keywordValue.trim()
  if (!keyword) return

  params.set('keyword', keyword)
  const actions = matchedAuditCodes(auditActionLabels, keyword)
  const resources = matchedAuditCodes(auditResourceLabels, keyword)

  if (actions.length) params.set('keyword_actions', actions.join(','))
  if (resources.length) params.set('keyword_resources', resources.join(','))
  if (keyword.includes('成功') || keyword.toLowerCase() === 'true') params.set('keyword_success', 'true')
  if (keyword.includes('失败') || keyword.toLowerCase() === 'false') params.set('keyword_success', 'false')
}

export function createAuditColumns(statusText: (value: string | boolean) => string): DataTableColumns<any> {
  return [
    { title: '时间', key: 'created_at', render: (row) => formatDateTime(row.created_at) },
    { title: '用户', key: 'operator', render: (row) => row.operator?.display_name || row.operator?.username || '-' },
    { title: '动作', key: 'action', render: (row) => actionText(row.action) },
    { title: '资源', key: 'resource_type', render: (row) => resourceText(row.resource_type) },
    { title: '结果', key: 'success', render: (row) => statusText(row.success) },
    { title: 'IP', key: 'ip' },
    {
      title: '浏览器',
      key: 'user_agent',
      width: 260,
      ellipsis: true,
      render: (row) =>
        h(NTooltip, { showAfter: 80, hideAfter: 80 }, {
          trigger: () => h('span', { class: 'table-ellipsis' }, row.user_agent || '-'),
          default: () => row.user_agent || '-'
        })
    }
  ]
}

function matchedAuditCodes(labels: Record<string, string>, keyword: string) {
  const text = keyword.trim().toLowerCase()
  if (!text) return []

  return Object.entries(labels)
    .filter(([code, label]) => code.toLowerCase().includes(text) || label.toLowerCase().includes(text))
    .map(([code]) => code)
}
