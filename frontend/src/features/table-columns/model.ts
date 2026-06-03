import { h } from 'vue'
import { NButton, NPopconfirm, type DataTableColumns } from 'naive-ui'
import { createAuditColumns } from '../../entities/audit/model'
import { boolText, directionText, formatDate, formatDateTime, money, reciprocityRoleText, statusText } from '../../shared/lib/formatters'

type ActionVariant = 'edit' | 'success' | 'warning' | 'danger' | 'neutral'

interface DictionaryHelpers {
  dictionaryLabel: (typeCode: string, value: string) => string
  canEditDictionaryItem: (row: any) => boolean
  dictionaryItemReadonlyText: (row: any) => string
}

interface TableColumnActions {
  openEditRecord: (row: any) => void
  openReciprocityRecord: (row: any) => void
  deleteRecord: (row: any) => void
  openEditFamily: (row: any) => void
  updateFamilyStatus: (row: any) => void
  openEditUser: (row: any) => void
  openResetPassword: (row: any) => void
  updateUserStatus: (row: any) => void
  openEditContact: (row: any) => void
  deleteContact: (row: any) => void
  openEditDictType: (row: any) => void
  openEditDict: (row: any) => void
  deleteDict: (row: any) => void
  executeImport: (id: number) => void
  showImportJobErrors: (row: any) => void
}

function actionButton(label: string, variant: ActionVariant, onClick?: () => void) {
  const className = `table-action-btn table-action-${variant}`
  return h(NButton, { class: className, size: 'small', secondary: true, onClick }, () => label)
}

function deleteAction(label: string, message: string, onPositiveClick: () => void) {
  return h(NPopconfirm, { onPositiveClick }, { trigger: () => actionButton(label, 'danger'), default: () => message })
}

function actionSpace(children: any[]) {
  return h('div', { class: 'table-actions' }, children)
}

function recordActionSpace(row: any, actions: TableColumnActions) {
  return h('div', { class: 'table-actions table-actions-record' }, [
    actionButton('编辑', 'edit', () => actions.openEditRecord(row)),
    row.reciprocity_status === 'pending'
      ? actionButton('回礼', 'success', () => actions.openReciprocityRecord(row))
      : h('span', { class: 'table-action-placeholder', 'aria-hidden': 'true' }),
    deleteAction('删除', '确认删除这条流水？', () => actions.deleteRecord(row))
  ])
}

export function createTableColumns(actions: TableColumnActions, helpers: DictionaryHelpers) {
  return {
    recordColumns: createRecordColumns(actions, helpers),
    dashboardRecordColumns: createDashboardRecordColumns(actions, helpers),
    familyColumns: createFamilyColumns(actions),
    userColumns: createUserColumns(actions),
    contactColumns: createContactColumns(actions, helpers),
    dictTypeColumns: createDictTypeColumns(actions),
    dictItemColumns: createDictItemColumns(actions, helpers),
    backupJobColumns: createBackupJobColumns(),
    importErrorColumns: createImportErrorColumns(),
    importJobColumns: createImportJobColumns(actions),
    auditColumns: createAuditColumns(statusText)
  }
}

function createDashboardRecordColumns(actions: TableColumnActions, helpers: DictionaryHelpers): DataTableColumns<any> {
  return [
    { title: '日期', key: 'occurred_on', width: 108, render: (row) => formatDate(row.occurred_on) },
    { title: '对象', key: 'contact.name' as any, width: 112, ellipsis: true, render: (row) => row.contact?.name || row.contact_id },
    { title: '事项', key: 'occasion_type', width: 96, render: (row) => helpers.dictionaryLabel('occasion_type', row.occasion_type) },
    { title: '方向', key: 'direction', width: 78, render: (row) => directionText(row.direction) },
    { title: '金额', key: 'amount_cents', width: 104, render: (row) => money(row.amount_cents) },
    { title: '回礼状态', key: 'reciprocity_status', width: 132, render: (row) => row.reciprocity_status === 'done' ? `${statusText(row.reciprocity_status)}（${reciprocityRoleText(row.direction)}）` : statusText(row.reciprocity_status) },
    {
      title: '操作',
      key: 'actions',
      width: 190,
      align: 'center',
      titleAlign: 'center',
      render: (row) => recordActionSpace(row, actions)
    }
  ]
}

function createRecordColumns(actions: TableColumnActions, helpers: DictionaryHelpers): DataTableColumns<any> {
  return [
    { title: '日期', key: 'occurred_on', width: 116, render: (row) => formatDate(row.occurred_on) },
    { title: '对象', key: 'contact.name' as any, width: 120, ellipsis: true, render: (row) => row.contact?.name || row.contact_id },
    { title: '事项', key: 'occasion_type', width: 100, render: (row) => helpers.dictionaryLabel('occasion_type', row.occasion_type) },
    { title: '方向', key: 'direction', width: 84, render: (row) => directionText(row.direction) },
    { title: '金额', key: 'amount_cents', width: 110, render: (row) => money(row.amount_cents) },
    { title: '回礼状态', key: 'reciprocity_status', width: 150, render: (row) => row.reciprocity_status === 'done' ? `${statusText(row.reciprocity_status)}（${reciprocityRoleText(row.direction)}）` : statusText(row.reciprocity_status) },
    { title: '备注', key: 'remark', width: 180, ellipsis: true, render: (row) => row.remark || row.gift_description || '-' },
    { title: '创建时间', key: 'created_at', width: 180, render: (row) => formatDateTime(row.created_at) },
    {
      title: '操作',
      key: 'actions',
      width: 190,
      align: 'center',
      titleAlign: 'center',
      render: (row) => recordActionSpace(row, actions)
    }
  ]
}

function createFamilyColumns(actions: TableColumnActions): DataTableColumns<any> {
  return [
    { title: 'ID', key: 'id' },
    { title: '名称', key: 'name' },
    { title: '联系人', key: 'contact_name' },
    { title: '邮箱', key: 'contact_email' },
    { title: '状态', key: 'status', render: (row) => statusText(row.status) },
    { title: '创建时间', key: 'created_at', render: (row) => formatDateTime(row.created_at) },
    {
      title: '操作',
      key: 'actions',
      align: 'center',
      titleAlign: 'center',
      render: (row) =>
        actionSpace([
          actionButton('编辑', 'edit', () => actions.openEditFamily(row)),
          actionButton(row.status === 'enabled' ? '禁用' : '启用', row.status === 'enabled' ? 'warning' : 'success', () => actions.updateFamilyStatus(row))
        ])
    }
  ]
}

function createUserColumns(actions: TableColumnActions): DataTableColumns<any> {
  return [
    { title: '账号', key: 'username' },
    { title: '姓名', key: 'display_name' },
    { title: '邮箱', key: 'email' },
    { title: '角色', key: 'role', render: (row) => statusText(row.role) },
    { title: '状态', key: 'status', render: (row) => statusText(row.status) },
    { title: '首次改密', key: 'must_change_password', render: (row) => boolText(row.must_change_password) },
    { title: '最后登录', key: 'last_login_at', render: (row) => formatDateTime(row.last_login_at) },
    {
      title: '操作',
      key: 'actions',
      align: 'center',
      titleAlign: 'center',
      render: (row) =>
        actionSpace([
          actionButton('编辑', 'edit', () => actions.openEditUser(row)),
          actionButton('重置密码', 'neutral', () => actions.openResetPassword(row)),
          actionButton(row.status === 'enabled' ? '禁用' : '启用', row.status === 'enabled' ? 'warning' : 'success', () => actions.updateUserStatus(row))
        ])
    }
  ]
}

function createContactColumns(actions: TableColumnActions, helpers: DictionaryHelpers): DataTableColumns<any> {
  return [
    { title: '名称', key: 'name' },
    { title: '归属成员', key: 'owner_user', render: (row) => row.owner_user?.display_name || row.owner_user?.username || '未指定' },
    { title: '对象类型', key: 'contact_type', render: (row) => helpers.dictionaryLabel('contact_type', row.contact_type) },
    { title: '关系类型', key: 'relation_type', render: (row) => helpers.dictionaryLabel('relation_type', row.relation_type) },
    { title: '手机号', key: 'phone' },
    { title: '地址', key: 'address' },
    { title: '状态', key: 'status', render: (row) => statusText(row.status) },
    { title: '创建时间', key: 'created_at', render: (row) => formatDateTime(row.created_at) },
    {
      title: '操作',
      key: 'actions',
      align: 'center',
      titleAlign: 'center',
      render: (row) =>
        actionSpace([
          actionButton('编辑', 'edit', () => actions.openEditContact(row)),
          deleteAction('删除', '确认删除该对象？', () => actions.deleteContact(row))
        ])
    }
  ]
}

function createDictTypeColumns(actions: TableColumnActions): DataTableColumns<any> {
  return [
    { title: '类型编码', key: 'code' },
    { title: '类型名称', key: 'name' },
    { title: '允许家庭自定义', key: 'allow_family_custom', render: (row) => boolText(row.allow_family_custom) },
    { title: '启用', key: 'enabled', render: (row) => statusText(row.enabled) },
    { title: '排序', key: 'sort_order' },
    { title: '操作', key: 'actions', align: 'center', titleAlign: 'center', render: (row) => actionSpace([actionButton('编辑', 'edit', () => actions.openEditDictType(row))]) }
  ]
}

function createDictItemColumns(actions: TableColumnActions, helpers: DictionaryHelpers): DataTableColumns<any> {
  return [
    { title: '标签', key: 'label' },
    { title: '值', key: 'value' },
    { title: '来源', key: 'is_system', render: (row) => (row.is_system ? '系统预置' : '家庭自定义') },
    { title: '启用', key: 'enabled', render: (row) => statusText(row.enabled) },
    { title: '排序', key: 'sort_order' },
    { title: '更新时间', key: 'updated_at', render: (row) => formatDateTime(row.updated_at) },
    {
      title: '操作',
      key: 'actions',
      align: 'center',
      titleAlign: 'center',
      render: (row) =>
        !helpers.canEditDictionaryItem(row)
          ? helpers.dictionaryItemReadonlyText(row)
          : actionSpace([
              actionButton('编辑', 'edit', () => actions.openEditDict(row)),
              deleteAction('删除', '确认删除该字典项？', () => actions.deleteDict(row))
            ])
    }
  ]
}

function createBackupJobColumns(): DataTableColumns<any> {
  return [
    { title: '文件', key: 'file_name' },
    { title: '触发方式', key: 'trigger_type', render: (row) => statusText(row.trigger_type) },
    { title: '状态', key: 'status', render: (row) => statusText(row.status) },
    { title: '开始时间', key: 'started_at', render: (row) => formatDateTime(row.started_at) },
    { title: '完成时间', key: 'finished_at', render: (row) => formatDateTime(row.finished_at) },
    { title: '错误信息', key: 'error_message' }
  ]
}

function createImportErrorColumns(): DataTableColumns<any> {
  return [
    { title: '工作表', key: 'sheet_name' },
    { title: '行', key: 'row_number' },
    { title: '字段', key: 'field_name' },
    { title: '错误', key: 'error_message' }
  ]
}

function createImportJobColumns(actions: TableColumnActions): DataTableColumns<any> {
  return [
    { title: '文件', key: 'source_file_name' },
    { title: '状态', key: 'status', render: (row) => statusText(row.status) },
    { title: '新增', key: 'created_count' },
    { title: '更新', key: 'updated_count' },
    { title: '错误', key: 'error_count' },
    { title: '开始时间', key: 'started_at', render: (row) => formatDateTime(row.started_at) },
    {
      title: '操作',
      key: 'actions',
      align: 'center',
      titleAlign: 'center',
      render: (row) => renderImportJobAction(row, actions)
    }
  ]
}

function renderImportJobAction(row: any, actions: TableColumnActions) {
  if (row.status === 'dry_run_ready') {
    return actionSpace([actionButton('执行导入', 'success', () => actions.executeImport(row.id))])
  }
  if (isImportJobFailed(row)) {
    return actionSpace([actionButton('查看详情', 'danger', () => actions.showImportJobErrors(row))])
  }
  return row.status === 'success' ? '已完成' : '-'
}

export function isImportJobFailed(row: any) {
  return row.status === 'dry_run_failed' || row.status === 'failed' || row.status === 'blocked' || row.error_count > 0
}
