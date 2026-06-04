import { markRaw } from 'vue'
import { Activity, ArchiveRestore, BookOpen, Contact, DatabaseBackup, FileSpreadsheet, HandCoins, Send, Settings, ShieldCheck, Users } from 'lucide-vue-next'
import type { ViewKey } from '../shared/types'

export const allowedViewKeys: ViewKey[] = ['dashboard', 'families', 'users', 'contacts', 'records', 'dictionaries', 'stats', 'receivables', 'payables', 'backup', 'imports', 'audit']

export const viewIcons = {
  dashboard: markRaw(Activity),
  families: markRaw(Users),
  users: markRaw(Users),
  contacts: markRaw(Contact),
  records: markRaw(BookOpen),
  dictionaries: markRaw(Settings),
  stats: markRaw(FileSpreadsheet),
  receivables: markRaw(HandCoins),
  payables: markRaw(Send),
  backup: markRaw(DatabaseBackup),
  imports: markRaw(ArchiveRestore),
  audit: markRaw(ShieldCheck)
} satisfies Record<ViewKey, any>

export function buildMenuItems(isAdmin: boolean, isSuperAdmin: boolean) {
  const items = [
    { key: 'dashboard' as ViewKey, label: '工作台' },
    { key: 'contacts' as ViewKey, label: '往来对象' },
    { key: 'records' as ViewKey, label: '往来流水' },
    { key: 'stats' as ViewKey, label: '收送概览' },
    { key: 'receivables' as ViewKey, label: '礼金回收' },
    { key: 'payables' as ViewKey, label: '礼金送出' }
  ]
  if (isAdmin) {
    items.push(
      { key: 'users', label: '用户管理' },
      { key: 'dictionaries', label: '数据字典' },
      { key: 'backup', label: '备份配置' },
      { key: 'imports', label: '导入恢复' },
      { key: 'audit', label: '审计日志' }
    )
  }
  if (isSuperAdmin) {
    items.splice(1, 0, { key: 'families', label: '家庭管理' })
  }
  return items
}
