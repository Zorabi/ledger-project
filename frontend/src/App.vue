<script setup lang="ts">
import { computed, h, markRaw, nextTick, onMounted, reactive, ref, watch } from 'vue'
import {
  Activity,
  ArchiveRestore,
  BookOpen,
  Contact,
  DatabaseBackup,
  FileSpreadsheet,
  KeyRound,
  LogOut,
  Search,
  Settings,
  ShieldCheck,
  Upload,
  Users
} from 'lucide-vue-next'
import { createDiscreteApi, dateZhCN, NButton, NPopconfirm, NSpace, type DataTableColumns, type UploadFileInfo, zhCN } from 'naive-ui'
import AuditView from './components/views/AuditView.vue'
import BackupView from './components/views/BackupView.vue'
import ContactsView from './components/views/ContactsView.vue'
import DashboardView from './components/views/DashboardView.vue'
import DictionariesView from './components/views/DictionariesView.vue'
import FamiliesView from './components/views/FamiliesView.vue'
import ImportsView from './components/views/ImportsView.vue'
import RecordsView from './components/views/RecordsView.vue'
import StatsView from './components/views/StatsView.vue'
import UsersView from './components/views/UsersView.vue'

const API_BASE = import.meta.env.VITE_API_BASE || '/api/v1'
const FALLBACK_VERSION = import.meta.env.VITE_APP_VERSION || '1.3.0.0'
const FAMILY_CACHE_KEY = 'ledger_families_cache'
const SELECTED_FAMILY_KEY = 'selected_family_id'

type Role = 'super_admin' | 'family_admin' | 'member'
type ViewKey =
  | 'dashboard'
  | 'families'
  | 'users'
  | 'contacts'
  | 'records'
  | 'dictionaries'
  | 'stats'
  | 'backup'
  | 'imports'
  | 'audit'

interface SessionUser {
  id: number
  family_id?: number
  username: string
  display_name: string
  email: string
  role: Role
  must_change_password: boolean
  family?: { id: number; name: string }
}

interface PageResult<T> {
  total: number
  list: T[]
  page: number
  page_size: number
}

const { message } = createDiscreteApi(['message'])
const token = ref(localStorage.getItem('access_token') || '')
const refreshToken = ref(localStorage.getItem('refresh_token') || '')
const user = ref<SessionUser | null>(null)
const allowedViewKeys: ViewKey[] = ['dashboard', 'families', 'users', 'contacts', 'records', 'dictionaries', 'stats', 'backup', 'imports', 'audit']
const storedView = localStorage.getItem('active_view') as ViewKey | null
const activeView = ref<ViewKey>(storedView && allowedViewKeys.includes(storedView) ? storedView : 'dashboard')
const loading = ref(false)
const viewLoading = ref(false)
const viewError = ref('')
const authReady = ref(false)
const appVersion = ref(FALLBACK_VERSION)
const keyword = ref('')
const selectedFamilyId = ref<number | null>(readCachedSelectedFamilyId())
const loadedViews = reactive<Record<string, boolean>>({})
const showFamilyModal = ref(false)
const showUserModal = ref(false)
const showResetPasswordModal = ref(false)
const showContactModal = ref(false)
const showRecordModal = ref(false)
const showDictModal = ref(false)
const showDictTypeModal = ref(false)
const showBackupModal = ref(false)
const showImportModal = ref(false)
const importMode = ref<'manual' | 'backup'>('manual')
const editingFamilyId = ref<number | null>(null)
const editingUserId = ref<number | null>(null)
const resettingUser = ref<any | null>(null)
const editingContactId = ref<number | null>(null)
const editingRecordId = ref<number | null>(null)
const editingDictId = ref<number | null>(null)
const editingDictTypeId = ref<number | null>(null)

const loginForm = reactive({ username: '', password: '' })
const passwordForm = reactive({ old_password: '', new_password: '' })
const familyForm = reactive({ name: '', contact_name: '', contact_email: '', remark: '', admin_mode: 'existing', admin_user_id: null as number | null, admin_username: '', admin_display_name: '', admin_email: '', admin_password: '' })
const userForm = reactive({ username: '', display_name: '', email: '', password: '', role: 'member', family_id: null as number | null })
const resetPasswordForm = reactive({ password: '' })
const contactForm = reactive({ name: '', owner_user_id: null as number | null, contact_type: '', relation_type: '', phone: '', address: '', remark: '', status: 'enabled' })
const recordForm = reactive({ contact_id: null as number | null, occasion_type: '', direction: 'in', amount_yuan: 200, gift_description: '', occurred_on: new Date().toISOString().slice(0, 10), handled_by: '', reciprocity_status: 'pending', related_record_id: null as number | null, remark: '' })
const dictForm = reactive({ type_id: null as number | null, label: '', value: '', sort_order: 0, enabled: true })
const dictTypeForm = reactive({ code: '', name: '', description: '', allow_family_custom: true, sort_order: 0, enabled: true })
const backupForm = reactive({ enabled: false, frequency: 'weekly', recipient_emails_text: '', encrypt_enabled: false, zip_password: '', zip_password_hint: '' })
const importFile = ref<UploadFileInfo | null>(null)

const families = ref<any[]>(readCachedFamilies())
const users = ref<any[]>([])
const contacts = ref<any[]>([])
const records = ref<any[]>([])
const dictionaryTypes = ref<any[]>([])
const dictionaryItems = ref<any[]>([])
const auditLogs = ref<any[]>([])
const backupJobs = ref<any[]>([])
const importJobs = ref<any[]>([])
const importErrors = ref<any[]>([])
const stats = ref<any>(null)
const backupConfig = ref<any>(null)
const recordQuery = reactive({ page: 1, page_size: 10, total: 0, contact_id: null as number | null, direction: null as string | null, occasion_type: null as string | null, reciprocity_status: null as string | null, start_date: null as string | null, end_date: null as string | null })
const statsQuery = reactive({ range: 'all', start_date: null as string | null, end_date: null as string | null })
const auditQuery = reactive({ page: 1, page_size: 10, total: 0, keyword: '', action: '', success: null as boolean | null })
const mobileNavRef = ref<HTMLElement | null>(null)
const latestRequestSeq: Record<string, number> = {}
let viewLoadSeq = 0

function readCachedFamilies() {
  try {
    const raw = localStorage.getItem(FAMILY_CACHE_KEY)
    const parsed = raw ? JSON.parse(raw) : []
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function cacheFamilies(list: any[]) {
  localStorage.setItem(FAMILY_CACHE_KEY, JSON.stringify(list))
}

function readCachedSelectedFamilyId() {
  const raw = localStorage.getItem(SELECTED_FAMILY_KEY)
  if (!raw) return null
  const value = Number(raw)
  return Number.isFinite(value) ? value : null
}

function persistSelectedFamilyId(value: number | null) {
  if (value === null || value === undefined) {
    localStorage.removeItem(SELECTED_FAMILY_KEY)
    return
  }
  localStorage.setItem(SELECTED_FAMILY_KEY, String(value))
}

function debounce<T extends (...args: any[]) => void>(fn: T, wait = 300) {
  let timer: ReturnType<typeof setTimeout> | null = null
  return (...args: Parameters<T>) => {
    if (timer) window.clearTimeout(timer)
    timer = window.setTimeout(() => {
      timer = null
      fn(...args)
    }, wait)
  }
}

async function latestOnly<T>(key: string, loader: () => Promise<T>, apply: (data: T) => void) {
  const seq = (latestRequestSeq[key] || 0) + 1
  latestRequestSeq[key] = seq
  const data = await loader()
  if (latestRequestSeq[key] === seq) apply(data)
}

const isLoggedIn = computed(() => Boolean(token.value && user.value))
const mustChangePassword = computed(() => Boolean(user.value?.must_change_password))
const isSuperAdmin = computed(() => user.value?.role === 'super_admin')
const isAdmin = computed(() => user.value?.role === 'super_admin' || user.value?.role === 'family_admin')
const currentFamilyId = computed(() => selectedFamilyId.value || user.value?.family_id || null)
const viewCacheKey = (view: ViewKey) => `${view}:${currentFamilyId.value || 'system'}`
const activeViewCacheKey = computed(() => viewCacheKey(activeView.value))
const showFullViewLoading = computed(() => viewLoading.value && !loadedViews[activeViewCacheKey.value])
const dictionaryGroups = computed(() =>
  dictionaryTypes.value.map((type) => ({
    type,
    items: dictionaryItems.value.filter((item) => item.type_id === type.id || item.type?.id === type.id)
  }))
)
const writableDictionaryTypes = computed(() => dictionaryTypes.value.filter((item) => isSuperAdmin.value || item.allow_family_custom))
const dictionaryTypeOptions = computed(() =>
  (editingDictId.value ? dictionaryTypes.value : writableDictionaryTypes.value).map((item) => ({ label: item.name, value: item.id }))
)
const contactTypeOptions = computed(() => dictOptions('contact_type'))
const relationTypeOptions = computed(() => dictOptions('relation_type'))
const occasionTypeOptions = computed(() => dictOptions('occasion_type'))
const relatedRecordOptions = computed(() =>
  records.value
    .filter((item) => item.id !== editingRecordId.value && (!recordForm.contact_id || item.contact_id === recordForm.contact_id) && item.direction !== recordForm.direction)
    .map((item) => ({
      label: `${formatDate(item.occurred_on)} · ${reciprocityRelationText(item.direction)} · ${item.contact?.name || item.contact_id} · ${money(item.amount_cents)}`,
      value: item.id
    }))
)
const adminUserOptions = computed(() =>
  users.value
    .filter((item) => item.role !== 'super_admin' && (!editingFamilyId.value || !item.family_id || item.family_id === editingFamilyId.value))
    .map((item) => ({ label: `${item.display_name}（${item.username}）`, value: item.id }))
)
const handlerOptions = computed(() =>
  users.value
    .filter((item) => item.status !== 'disabled')
    .map((item) => ({ label: item.display_name || item.username, value: item.display_name || item.username }))
)
const familyMemberOptions = computed(() =>
  users.value
    .filter((item) => item.status !== 'disabled' && item.role !== 'super_admin' && (!isSuperAdmin.value || !currentFamilyId.value || item.family_id === currentFamilyId.value))
    .map((item) => ({ label: `${item.display_name || item.username}（${statusText(item.role)}）`, value: item.id }))
)

const viewIcons = {
  dashboard: markRaw(Activity),
  families: markRaw(Users),
  users: markRaw(Users),
  contacts: markRaw(Contact),
  records: markRaw(BookOpen),
  dictionaries: markRaw(Settings),
  stats: markRaw(FileSpreadsheet),
  backup: markRaw(DatabaseBackup),
  imports: markRaw(ArchiveRestore),
  audit: markRaw(ShieldCheck)
} satisfies Record<ViewKey, any>

const menuItems = computed(() => {
  const items = [
    { key: 'dashboard', label: '工作台' },
    { key: 'contacts', label: '往来对象' },
    { key: 'records', label: '往来流水' },
    { key: 'stats', label: '统计分析' }
  ]
  if (isAdmin.value) {
    items.push(
      { key: 'users', label: '用户管理' },
      { key: 'dictionaries', label: '数据字典' },
      { key: 'backup', label: '备份配置' },
      { key: 'imports', label: '导入恢复' },
      { key: 'audit', label: '审计日志' }
    )
  }
  if (isSuperAdmin.value) {
    items.splice(1, 0, { key: 'families', label: '家庭管理' })
  }
  return items
})

async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers)
  if (!(options.body instanceof FormData)) headers.set('Content-Type', 'application/json')
  if (token.value) headers.set('Authorization', `Bearer ${token.value}`)
  const response = await fetch(`${API_BASE}${path}`, { ...options, headers })
  const contentType = response.headers.get('Content-Type') || ''
  if (contentType.includes('application/vnd') || contentType.includes('application/zip')) return response.blob() as T
  const body = await response.json().catch(() => ({}))
  if (!response.ok || body.code) {
    const error = new Error(body.msg || `请求失败（${response.status}）`)
    ;(error as any).data = body.data
    throw error
  }
  return body.data as T
}

async function downloadBlob(path: string, options: RequestInit = {}, fallbackName = 'download.xlsx') {
  const headers = new Headers(options.headers)
  if (!(options.body instanceof FormData)) headers.set('Content-Type', 'application/json')
  if (token.value) headers.set('Authorization', `Bearer ${token.value}`)
  const response = await fetch(`${API_BASE}${path}`, { ...options, headers })
  if (!response.ok) {
    const body = await response.json().catch(() => ({}))
    throw new Error(body.msg || `下载失败（${response.status}）`)
  }
  const blob = await response.blob()
  const fileName = parseDownloadFileName(response.headers.get('Content-Disposition')) || fallbackName
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = fileName
  link.click()
  URL.revokeObjectURL(url)
}

function parseDownloadFileName(disposition: string | null) {
  if (!disposition) return ''
  const encoded = disposition.match(/filename\*=UTF-8''([^;]+)/i)?.[1]
  if (encoded) return decodeURIComponent(encoded)
  return disposition.match(/filename="?([^"]+)"?/i)?.[1] || ''
}

async function runAction(action: () => Promise<void>) {
  try {
    await action()
  } catch (error: any) {
    message.error(error?.message || '操作失败')
  }
}

function generatePassword(target: 'user' | 'family_admin' | 'reset_user') {
  const upper = 'ABCDEFGHJKLMNPQRSTUVWXYZ'
  const lower = 'abcdefghijkmnopqrstuvwxyz'
  const digits = '23456789'
  const special = '!@#$%^&*?'
  const all = upper + lower + digits + special
  const pick = (chars: string) => chars[secureRandom(chars.length)]
  const chars = [pick(upper), pick(lower), pick(digits), pick(special)]
  for (let i = chars.length; i < 14; i += 1) chars.push(pick(all))
  for (let i = chars.length - 1; i > 0; i -= 1) {
    const j = secureRandom(i + 1)
    ;[chars[i], chars[j]] = [chars[j], chars[i]]
  }
  const password = chars.join('')
  if (target === 'user') userForm.password = password
  else if (target === 'reset_user') resetPasswordForm.password = password
  else familyForm.admin_password = password
  message.success('已生成强密码')
}

async function copyGeneratedPassword(target: 'user' | 'family_admin' | 'reset_user') {
  const password = target === 'user' ? userForm.password : target === 'reset_user' ? resetPasswordForm.password : familyForm.admin_password
  if (!password) {
    message.warning('请先生成或填写密码')
    return
  }
  try {
    await navigator.clipboard.writeText(password)
    message.success('密码已复制')
  } catch {
    message.error('复制失败，请手动选择密码')
  }
}

function secureRandom(max: number) {
  const values = new Uint32Array(1)
  crypto.getRandomValues(values)
  return values[0] % max
}

async function login() {
  loading.value = true
  try {
    const data = await api<any>('/auth/login', { method: 'POST', body: JSON.stringify(loginForm) })
    token.value = data.access_token
    refreshToken.value = data.refresh_token
    user.value = data.user
    localStorage.setItem('access_token', token.value)
    localStorage.setItem('refresh_token', refreshToken.value)
    selectedFamilyId.value = user.value?.role === 'super_admin' ? readCachedSelectedFamilyId() : user.value?.family_id || null
    message.success('登录成功')
    if (!user.value?.must_change_password) await loadInitialData()
  } catch (error: any) {
    message.error(error.message)
  } finally {
    loading.value = false
  }
}

async function loadVersion() {
  try {
    const data = await api<any>('/version')
    appVersion.value = data.version || FALLBACK_VERSION
  } catch {
    appVersion.value = FALLBACK_VERSION
  }
}

async function loadMe() {
  if (!token.value) {
    authReady.value = true
    return
  }
  try {
    user.value = await api<SessionUser>('/me')
    selectedFamilyId.value = user.value.role === 'super_admin' ? readCachedSelectedFamilyId() : user.value.family_id || null
    if (!user.value.must_change_password) await loadInitialData()
  } catch {
    logout()
  } finally {
    authReady.value = true
  }
}

function logout() {
  token.value = ''
  refreshToken.value = ''
  user.value = null
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
}

async function changePassword() {
  await api('/auth/change-password', { method: 'POST', body: JSON.stringify(passwordForm) })
  message.success('密码已修改，请重新登录')
  logout()
}

function familyQuery() {
  return isSuperAdmin.value && currentFamilyId.value ? `?family_id=${currentFamilyId.value}` : ''
}

async function loadInitialData() {
  clearLoadedViews()
  if (isSuperAdmin.value) await loadFamilies()
  if (!menuItems.value.some((item) => item.key === activeView.value)) {
    activeView.value = 'dashboard'
    localStorage.setItem('active_view', activeView.value)
  }
  await loadView(activeView.value, true)
}

function clearLoadedViews() {
  Object.keys(loadedViews).forEach((key) => delete loadedViews[key])
}

async function loadView(view: ViewKey, force = false) {
  const seq = ++viewLoadSeq
  const cacheKey = viewCacheKey(view)
  viewError.value = ''
  if (isSuperAdmin.value && !currentFamilyId.value && ['backup', 'imports'].includes(view)) {
    viewError.value = '请先在顶部选择一个家庭，再使用该功能。'
    backupJobs.value = []
    importJobs.value = []
    importErrors.value = []
    return true
  }

  viewLoading.value = true
  try {
    if (view === 'dashboard') {
      await Promise.all([loadContacts(), loadDashboardRecords(), loadDictionaries(), loadUsers(), loadStats()])
    } else if (view === 'families') {
      await loadFamilies()
    } else if (view === 'users') {
      await loadUsers()
    } else if (view === 'contacts') {
      await Promise.all([loadContacts(), loadDictionaries(), loadUsers()])
    } else if (view === 'records') {
      await Promise.all([loadContacts(), loadRecords(), loadDictionaries(), loadUsers()])
    } else if (view === 'dictionaries') {
      await loadDictionaries()
    } else if (view === 'stats') {
      await loadStats()
    } else if (view === 'backup') {
      await loadBackup()
    } else if (view === 'imports') {
      await loadImports()
    } else if (view === 'audit') {
      await loadAudit()
    }
    loadedViews[cacheKey] = true
    return true
  } catch (error: any) {
    if (seq !== viewLoadSeq) return false
    viewError.value = error?.message || '页面加载失败'
    message.error(viewError.value)
    return false
  } finally {
    if (seq === viewLoadSeq) viewLoading.value = false
  }
}

async function switchView(view: ViewKey) {
  if (activeView.value === view && viewLoading.value) return
  if (activeView.value === view) {
    debouncedLoadActiveView(true)
    return
  }
  if (loadedViews[viewCacheKey(view)]) {
    activeView.value = view
    localStorage.setItem('active_view', view)
    window.scrollTo({ top: 0, left: 0 })
    debouncedLoadActiveView(true)
    return
  }
  const loaded = await loadView(view)
  if (!loaded) return
  activeView.value = view
  localStorage.setItem('active_view', view)
  window.scrollTo({ top: 0, left: 0 })
}

function scrollActiveMobileNavIntoView() {
  nextTick(() => {
    const nav = mobileNavRef.value
    if (!nav) return
    const activeButton = nav.querySelector<HTMLElement>(`button[data-menu-key="${activeView.value}"]`)
    if (!activeButton) return
    activeButton.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' })
  })
}

async function handleFamilyChange() {
  persistSelectedFamilyId(selectedFamilyId.value)
  clearLoadedViews()
  debouncedLoadActiveView(true)
}

const debouncedLoadActiveView = debounce((force = false) => {
  loadView(activeView.value, force)
}, 180)

async function loadFamilies() {
  await latestOnly('families', () => api<PageResult<any>>('/families?page_size=100'), (data) => {
    families.value = data.list
    cacheFamilies(data.list)
    if (selectedFamilyId.value && !data.list.some((item) => item.id === selectedFamilyId.value)) {
      selectedFamilyId.value = null
      persistSelectedFamilyId(null)
    }
  })
}

async function loadUsers() {
  await latestOnly('users', () => api<PageResult<any>>('/users?page_size=100'), (data) => {
    users.value = data.list
  })
}

async function loadContacts() {
  const requestKeyword = keyword.value
  const requestFamily = familyQuery().replace('?', '&')
  await latestOnly('contacts', () => api<PageResult<any>>(`/contacts?page_size=100&keyword=${encodeURIComponent(requestKeyword)}${requestFamily}`), (data) => {
    contacts.value = data.list
  })
}

async function loadRecords() {
  const params = new URLSearchParams({ page: String(recordQuery.page), page_size: String(recordQuery.page_size) })
  if (recordQuery.contact_id) params.set('contact_id', String(recordQuery.contact_id))
  if (recordQuery.direction) params.set('direction', recordQuery.direction)
  if (recordQuery.occasion_type) params.set('occasion_type', recordQuery.occasion_type)
  if (recordQuery.reciprocity_status) params.set('reciprocity_status', recordQuery.reciprocity_status)
  if (recordQuery.start_date) params.set('start_date', recordQuery.start_date)
  if (recordQuery.end_date) params.set('end_date', recordQuery.end_date)
  if (isSuperAdmin.value && currentFamilyId.value) params.set('family_id', String(currentFamilyId.value))
  const query = params.toString()
  await latestOnly('records', () => api<PageResult<any>>(`/gift-records?${query}`), (data) => {
    records.value = data.list
    recordQuery.total = data.total
  })
}

async function loadDashboardRecords() {
  const params = new URLSearchParams({ page: '1', page_size: '8' })
  appendStatsQueryParams(params)
  if (isSuperAdmin.value && currentFamilyId.value) params.set('family_id', String(currentFamilyId.value))
  const query = params.toString()
  await latestOnly('records', () => api<PageResult<any>>(`/gift-records?${query}`), (data) => {
    records.value = data.list
  })
}

function appendStatsQueryParams(params: URLSearchParams) {
  if (statsQuery.start_date) params.set('start_date', statsQuery.start_date)
  if (statsQuery.end_date) params.set('end_date', statsQuery.end_date)
}

async function applyStatsQuery() {
  await Promise.all([loadStats(), activeView.value === 'dashboard' ? loadDashboardRecords() : Promise.resolve()])
}

function setStatsRange(range: string) {
  statsQuery.range = range
  const today = new Date()
  const toDate = (date: Date) => {
    const pad = (n: number) => String(n).padStart(2, '0')
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
  }
  if (range === 'all') {
    statsQuery.start_date = null
    statsQuery.end_date = null
  } else if (range === 'month') {
    statsQuery.start_date = toDate(new Date(today.getFullYear(), today.getMonth(), 1))
    statsQuery.end_date = toDate(today)
  } else if (range === 'year') {
    statsQuery.start_date = toDate(new Date(today.getFullYear(), 0, 1))
    statsQuery.end_date = toDate(today)
  } else if (range === 'last30') {
    const start = new Date(today)
    start.setDate(today.getDate() - 29)
    statsQuery.start_date = toDate(start)
    statsQuery.end_date = toDate(today)
  }
}

async function loadDictionaries() {
  const requestFamily = familyQuery().replace('?', '&')
  await latestOnly('dictionaries', async () => {
    const [types, items] = await Promise.all([
      api<PageResult<any>>('/dictionary-types?page_size=100'),
      api<PageResult<any>>(`/dictionary-items?page_size=200${requestFamily}`)
    ])
    return { types, items }
  }, ({ types, items }) => {
    dictionaryTypes.value = types.list
    dictionaryItems.value = items.list
    if (!dictForm.type_id && types.list[0]) dictForm.type_id = types.list[0].id
  })
}

async function loadStats() {
  const params = new URLSearchParams()
  appendStatsQueryParams(params)
  if (isSuperAdmin.value && currentFamilyId.value) params.set('family_id', String(currentFamilyId.value))
  const query = params.toString()
  await latestOnly('stats', () => api<any>(`/stats/overview${query ? `?${query}` : ''}`), (data) => {
    stats.value = data
  })
}

async function loadAudit() {
  const params = new URLSearchParams({ page: String(auditQuery.page), page_size: String(auditQuery.page_size) })
  if (auditQuery.keyword) params.set('keyword', auditQuery.keyword)
  if (auditQuery.action) params.set('action', auditQuery.action)
  if (auditQuery.success !== null) params.set('success', String(auditQuery.success))
  const query = params.toString()
  await latestOnly('audit', () => api<PageResult<any>>(`/audit-logs?${query}`), (data) => {
    auditLogs.value = data.list
    auditQuery.total = data.total
  })
}

async function loadBackup() {
  if (!currentFamilyId.value) return
  const requestFamilyQuery = familyQuery()
  await latestOnly('backup', async () => {
    const [configData, jobs] = await Promise.all([
      api<any>(`/backup/config${requestFamilyQuery}`),
      api<PageResult<any>>(`/backup/jobs?page_size=50${requestFamilyQuery.replace('?', '&')}`)
    ])
    return { configData, jobs }
  }, ({ configData, jobs }) => {
    backupConfig.value = configData
    backupForm.enabled = configData.enabled
    backupForm.frequency = configData.frequency
    backupForm.recipient_emails_text = (configData.recipient_emails || []).join('\n')
    backupForm.encrypt_enabled = configData.encrypt_enabled
    backupForm.zip_password_hint = configData.zip_password_hint || ''
    backupJobs.value = jobs.list
  })
}

async function loadImports() {
  if (!currentFamilyId.value) return
  const requestFamily = familyQuery().replace('?', '&')
  await latestOnly('imports', () => api<PageResult<any>>(`/imports?page_size=50${requestFamily}`), (data) => {
    importJobs.value = data.list
  })
}

async function createFamily() {
  if (!familyForm.name) {
    message.warning('请填写家庭名称')
    return
  }
  const payload: any = { name: familyForm.name, contact_name: familyForm.contact_name, contact_email: familyForm.contact_email, remark: familyForm.remark }
  if (familyForm.admin_mode === 'existing' && familyForm.admin_user_id) {
    payload.admin_user_id = familyForm.admin_user_id
  }
  if (familyForm.admin_mode === 'new' && familyForm.admin_username) {
    payload.admin = { username: familyForm.admin_username, display_name: familyForm.admin_display_name, email: familyForm.admin_email, password: familyForm.admin_password }
  }
  if (editingFamilyId.value) {
    await api(`/families/${editingFamilyId.value}`, { method: 'PUT', body: JSON.stringify(payload) })
    message.success('家庭已更新')
  } else {
    await api('/families', { method: 'POST', body: JSON.stringify(payload) })
    message.success('家庭已创建')
  }
  Object.assign(familyForm, { name: '', contact_name: '', contact_email: '', remark: '', admin_mode: 'existing', admin_user_id: null, admin_username: '', admin_display_name: '', admin_email: '', admin_password: '' })
  editingFamilyId.value = null
  showFamilyModal.value = false
  await loadFamilies()
}

async function createUser() {
  if (editingUserId.value) {
    await api(`/users/${editingUserId.value}`, { method: 'PUT', body: JSON.stringify(userForm) })
    message.success('用户已更新')
  } else {
    await api('/users', { method: 'POST', body: JSON.stringify({ ...userForm, family_id: isSuperAdmin.value ? userForm.family_id || currentFamilyId.value : undefined }) })
    message.success('用户已创建')
  }
  Object.assign(userForm, { username: '', display_name: '', email: '', password: '', role: 'member', family_id: null })
  editingUserId.value = null
  showUserModal.value = false
  await loadUsers()
}

async function createContact() {
  const payload = { ...contactForm, family_id: currentFamilyId.value }
  if (editingContactId.value) {
    await api(`/contacts/${editingContactId.value}`, { method: 'PUT', body: JSON.stringify(payload) })
    message.success('对象已更新')
  } else {
    await api('/contacts', { method: 'POST', body: JSON.stringify(payload) })
    message.success('对象已保存')
  }
  Object.assign(contactForm, { name: '', owner_user_id: null, contact_type: '', relation_type: '', phone: '', address: '', remark: '', status: 'enabled' })
  editingContactId.value = null
  showContactModal.value = false
  await Promise.all([loadContacts(), loadRecords(), loadStats()])
}

async function createRecord() {
  if (!recordForm.contact_id || !recordForm.occasion_type || !recordForm.direction || !recordForm.occurred_on) {
    message.warning('请完整填写往来对象、事项类型、方向和日期')
    return
  }
  const payload = { ...recordForm, family_id: currentFamilyId.value, amount_cents: Math.round(Number(recordForm.amount_yuan || 0) * 100) }
  if (editingRecordId.value) {
    await api(`/gift-records/${editingRecordId.value}`, { method: 'PUT', body: JSON.stringify(payload) })
    message.success('流水已更新')
  } else {
    await api('/gift-records', { method: 'POST', body: JSON.stringify(payload) })
    message.success('流水已保存')
  }
  resetRecordForm()
  showRecordModal.value = false
  await Promise.all([loadRecords(), loadStats()])
}

async function createDictionaryItem() {
  if (!dictForm.value) {
    message.warning('请填写字典值，系统会按原样保存')
    return
  }
  if (editingDictId.value) {
    await api(`/dictionary-items/${editingDictId.value}`, { method: 'PUT', body: JSON.stringify(dictForm) })
    message.success('字典项已更新')
  } else {
    await api('/dictionary-items', { method: 'POST', body: JSON.stringify(dictForm) })
    message.success('字典项已保存')
  }
  Object.assign(dictForm, { type_id: writableDictionaryTypes.value[0]?.id || null, label: '', value: '', sort_order: 0, enabled: true })
  editingDictId.value = null
  showDictModal.value = false
  await loadDictionaries()
}

async function createDictionaryType() {
  if (editingDictTypeId.value) {
    await api(`/dictionary-types/${editingDictTypeId.value}`, { method: 'PUT', body: JSON.stringify(dictTypeForm) })
    message.success('字典类型已更新')
  } else {
    await api('/dictionary-types', { method: 'POST', body: JSON.stringify(dictTypeForm) })
    message.success('字典类型已创建')
  }
  Object.assign(dictTypeForm, { code: '', name: '', description: '', allow_family_custom: true, sort_order: 0, enabled: true })
  editingDictTypeId.value = null
  showDictTypeModal.value = false
  await loadDictionaries()
}

async function saveBackupConfig() {
  await api(`/backup/config${familyQuery()}`, {
    method: 'PUT',
    body: JSON.stringify({
      enabled: backupForm.enabled,
      frequency: backupForm.frequency,
      recipient_emails: backupForm.recipient_emails_text.split('\n').map((item) => item.trim()).filter(Boolean),
      encrypt_enabled: backupForm.encrypt_enabled,
      zip_password: backupForm.zip_password,
      zip_password_hint: backupForm.zip_password_hint
    })
  })
  message.success('备份配置已保存')
  backupForm.zip_password = ''
  await loadBackup()
}

async function saveBackupAndClose() {
  await saveBackupConfig()
  showBackupModal.value = false
}

async function downloadBackup(email = false) {
  if (email) {
    await api(`/backup/export${familyQuery()}${familyQuery() ? '&' : '?'}download=false&email=true`, { method: 'POST' })
  } else {
    await downloadBlob(`/backup/export${familyQuery()}${familyQuery() ? '&' : '?'}download=true&email=false`, { method: 'POST' }, '礼尚往来-完整备份.xlsx')
  }
  message.success(email ? '备份邮件已发送' : '备份已下载')
  await loadBackup()
}

async function downloadImportTemplate() {
  await downloadBlob(`/imports/manual/template${familyQuery()}`, {}, '礼尚往来-手工导入模板.xlsx')
  message.success('手工导入模板已下载')
}

async function dryRunImport() {
  if (!importFile.value?.file) {
    message.warning('请选择 Excel 文件')
    return
  }
  const form = new FormData()
  form.append('file', importFile.value.file)
  const endpoint = importMode.value === 'manual' ? '/imports/manual/dry-run' : '/imports/backup/dry-run'
  const data = await api<any>(`${endpoint}${familyQuery()}`, { method: 'POST', body: form })
  message.success(`导入校验完成：新增 ${data.job.created_count}，更新 ${data.job.updated_count}，错误 ${data.job.error_count}`)
  importErrors.value = data.errors || []
  showImportModal.value = false
  await loadImports()
}

function openImportModal(mode: 'manual' | 'backup') {
  importMode.value = mode
  importFile.value = null
  showImportModal.value = true
}

async function executeImport(id: number) {
  try {
    await api(`/imports/${id}/execute`, { method: 'POST' })
    message.success('导入已执行')
    importErrors.value = []
    await Promise.all([loadImports(), loadContacts(), loadRecords(), loadStats(), loadDashboardRecords()])
  } catch (error: any) {
    message.error(error.message || '导入执行失败')
    if (Array.isArray(error.data?.errors)) {
      importErrors.value = error.data.errors
    }
    await Promise.all([loadImports(), loadImportErrors(id)])
  }
}

async function loadImportErrors(id: number) {
  const data = await api<PageResult<any>>(`/imports/${id}/errors?page_size=100`)
  importErrors.value = data.list || []
}

async function showImportJobErrors(row: any) {
  await loadImportErrors(row.id)
  if (importErrors.value.length) {
    message.error(`发现 ${importErrors.value.length} 条导入错误，请查看错误明细`)
    return
  }
  message.warning(row.error_message || '该批次没有可展示的行级错误')
}

const debouncedContactsSearch = debounce(() => {
  if (activeView.value === 'contacts') loadContacts()
}, 350)

const debouncedRecordsQuery = debounce(() => {
  if (activeView.value === 'records') loadRecords()
}, 250)

const debouncedAuditQuery = debounce(() => {
  if (activeView.value === 'audit') loadAudit()
}, 250)

const debouncedStatsQuery = debounce(() => {
  if (activeView.value === 'dashboard' || activeView.value === 'stats') applyStatsQuery()
}, 250)

function queryRecords() {
  recordQuery.page = 1
  debouncedRecordsQuery()
}

function queryAudit() {
  auditQuery.page = 1
  debouncedAuditQuery()
}

function refreshAudit() {
  debouncedAuditQuery()
}

function queryStats() {
  debouncedStatsQuery()
}

function openCreateFamily() {
  editingFamilyId.value = null
  Object.assign(familyForm, { name: '', contact_name: '', contact_email: '', remark: '', admin_mode: 'existing', admin_user_id: null, admin_username: '', admin_display_name: '', admin_email: '', admin_password: '' })
  loadUsers()
  showFamilyModal.value = true
}

function openEditFamily(row: any) {
  editingFamilyId.value = row.id
  Object.assign(familyForm, { name: row.name, contact_name: row.contact_name, contact_email: row.contact_email, remark: row.remark || '', admin_mode: row.admin_user_id ? 'existing' : 'new', admin_user_id: row.admin_user_id || null, admin_username: row.admin_user?.username || '', admin_display_name: row.admin_user?.display_name || '', admin_email: row.admin_user?.email || '', admin_password: '' })
  loadUsers()
  showFamilyModal.value = true
}

function openCreateUser() {
  editingUserId.value = null
  Object.assign(userForm, { username: '', display_name: '', email: '', password: '', role: 'member', family_id: null })
  showUserModal.value = true
}

function openEditUser(row: any) {
  editingUserId.value = row.id
  Object.assign(userForm, { username: row.username, display_name: row.display_name, email: row.email, password: '', role: row.role, family_id: row.family_id || null })
  showUserModal.value = true
}

async function updateFamilyStatus(row: any) {
  await api(`/families/${row.id}/status`, { method: 'PATCH', body: JSON.stringify({ status: row.status === 'enabled' ? 'disabled' : 'enabled' }) })
  message.success('家庭状态已更新')
  await loadFamilies()
}

function openCreateContact() {
  editingContactId.value = null
  Object.assign(contactForm, { name: '', owner_user_id: null, contact_type: contactTypeOptions.value[0]?.value || '', relation_type: relationTypeOptions.value[0]?.value || '', phone: '', address: '', remark: '', status: 'enabled' })
  showContactModal.value = true
}

function openEditContact(row: any) {
  editingContactId.value = row.id
  Object.assign(contactForm, { name: row.name, owner_user_id: row.owner_user_id || null, contact_type: row.contact_type, relation_type: row.relation_type, phone: row.phone, address: row.address, remark: row.remark || '', status: row.status || 'enabled' })
  showContactModal.value = true
}

async function deleteContact(row: any) {
  await api(`/contacts/${row.id}`, { method: 'DELETE' })
  message.success('对象已删除')
  await Promise.all([loadContacts(), loadRecords(), loadStats()])
}

function openCreateRecord() {
  editingRecordId.value = null
  resetRecordForm()
  showRecordModal.value = true
}

function openEditRecord(row: any) {
  editingRecordId.value = row.id
  Object.assign(recordForm, {
    contact_id: row.contact_id,
    occasion_type: row.occasion_type,
    direction: row.direction,
    amount_yuan: Number(row.amount_cents || 0) / 100,
    gift_description: row.gift_description || '',
    occurred_on: formatDate(row.occurred_on),
    handled_by: row.handled_by || '',
    reciprocity_status: row.reciprocity_status || 'none',
    related_record_id: row.related_record_id || null,
    remark: row.remark || ''
  })
  showRecordModal.value = true
}

async function deleteRecord(row: any) {
  await api(`/gift-records/${row.id}`, { method: 'DELETE' })
  message.success('流水已删除')
  await Promise.all([loadRecords(), loadStats()])
}

function resetRecordForm() {
  Object.assign(recordForm, { contact_id: null, occasion_type: occasionTypeOptions.value[0]?.value || '', direction: 'in', amount_yuan: 200, gift_description: '', occurred_on: new Date().toISOString().slice(0, 10), handled_by: user.value?.display_name || '', reciprocity_status: 'pending', related_record_id: null, remark: '' })
}

function setRecordDirection(direction: string) {
  if (recordForm.direction !== direction) {
    recordForm.related_record_id = null
  }
  recordForm.direction = direction
}

function setRecordReciprocityStatus(status: string) {
  if (status !== 'done') {
    recordForm.related_record_id = null
  }
  recordForm.reciprocity_status = status
}

function openCreateDict() {
  if (!writableDictionaryTypes.value.length) {
    message.warning('当前没有允许维护的字典类型')
    return
  }
  editingDictId.value = null
  Object.assign(dictForm, { type_id: writableDictionaryTypes.value[0]?.id || null, label: '', value: '', sort_order: 0, enabled: true })
  showDictModal.value = true
}

function openCreateDictType() {
  editingDictTypeId.value = null
  Object.assign(dictTypeForm, { code: '', name: '', description: '', allow_family_custom: true, sort_order: 0, enabled: true })
  showDictTypeModal.value = true
}

function openEditDictType(row: any) {
  editingDictTypeId.value = row.id
  Object.assign(dictTypeForm, { code: row.code, name: row.name, description: row.description || '', allow_family_custom: row.allow_family_custom, sort_order: row.sort_order, enabled: row.enabled })
  showDictTypeModal.value = true
}

function openEditDict(row: any) {
  if (!canEditDictionaryItem(row)) {
    message.warning('该字典类型已关闭家庭自定义，家庭字典项不可维护')
    return
  }
  editingDictId.value = row.id
  Object.assign(dictForm, { type_id: row.type_id, label: row.label, value: row.value, sort_order: row.sort_order, enabled: row.enabled })
  showDictModal.value = true
}

async function deleteDict(row: any) {
  if (!canEditDictionaryItem(row)) {
    message.warning('该字典类型已关闭家庭自定义，家庭字典项不可维护')
    return
  }
  await api(`/dictionary-items/${row.id}`, { method: 'DELETE' })
  message.success('字典项已删除')
  await loadDictionaries()
}

function canEditDictionaryItem(row: any) {
  if (row.is_system) return isSuperAdmin.value
  if (isSuperAdmin.value) return true
  const type = dictionaryTypes.value.find((item) => item.id === row.type_id || item.id === row.type?.id)
  return Boolean(type?.allow_family_custom)
}

function dictionaryItemReadonlyText(row: any) {
  if (row.is_system) return '系统项'
  return '家庭项只读'
}

async function updateUserStatus(row: any) {
  await api(`/users/${row.id}/status`, { method: 'PATCH', body: JSON.stringify({ status: row.status === 'enabled' ? 'disabled' : 'enabled' }) })
  message.success('用户状态已更新')
  await loadUsers()
}

function openResetPassword(row: any) {
  resettingUser.value = row
  resetPasswordForm.password = ''
  showResetPasswordModal.value = true
}

async function resetUserPassword() {
  if (!resettingUser.value?.id) return
  if (!resetPasswordForm.password) {
    message.warning('请填写或生成新密码')
    return
  }
  await api(`/users/${resettingUser.value.id}/reset-password`, { method: 'POST', body: JSON.stringify({ password: resetPasswordForm.password }) })
  message.success('密码已重置，用户下次登录需修改密码')
  showResetPasswordModal.value = false
  resettingUser.value = null
  resetPasswordForm.password = ''
  await loadUsers()
}

const money = (cents?: number) => `¥${((cents || 0) / 100).toFixed(2)}`
const moneyCompact = (cents?: number) => `¥${Number(((cents || 0) / 100).toFixed(0)).toLocaleString('zh-CN')}`
const directionText = (value: string) => value === 'in' ? '收礼' : '送礼'
const reciprocityRoleText = (direction: string) => direction === 'in' ? '我方回礼' : '对方回礼'
const reciprocityRelationText = (direction: string) => direction === 'in' ? '对方回礼流水' : '我方回礼流水'
function dictOptions(typeCode: string) {
  return dictionaryItems.value
    .filter((item) => item.type?.code === typeCode && item.enabled)
    .map((item) => ({ label: item.label, value: item.value }))
}
function dictionaryLabel(typeCode: string, value: string) {
  return dictionaryItems.value.find((item) => item.type?.code === typeCode && item.value === value)?.label || value || '-'
}
const statusText = (value: string | boolean) => {
  if (value === true) return '成功'
  if (value === false) return '失败'
  return ({ enabled: '启用', disabled: '禁用', success: '成功', failed: '失败', running: '执行中', blocked: '已阻断', dry_run_ready: '校验通过', dry_run_failed: '校验失败', none: '无需回礼', pending: '待回礼', done: '已回礼', manual: '手动', scheduled: '定时', super_admin: '系统管理员', family_admin: '家庭管理员', member: '家庭成员' } as Record<string, string>)[value] || value || '-'
}
const actionText = (value: string) => ({
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
  execute_import: '执行导入'
} as Record<string, string>)[value] || value || '-'
const resourceText = (value: string) => ({ user: '用户', family: '家庭', contact: '往来对象', gift_record: '往来流水', dictionary_item: '字典项', dictionary_type: '字典类型', backup_config: '备份配置', backup_job: '备份任务', import_job: '导入批次', user_session: '用户会话', auth_throttle: '登录防护' } as Record<string, string>)[value] || value || '-'
const boolText = (value: boolean) => value ? '是' : '否'
function formatDateTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}
function formatDate(value?: string) {
  return formatDateTime(value).slice(0, 10)
}

const recordColumns: DataTableColumns<any> = [
  { title: '日期', key: 'occurred_on', render: (row) => formatDate(row.occurred_on) },
  { title: '对象', key: 'contact.name' as any, render: (row) => row.contact?.name || row.contact_id },
  { title: '事项', key: 'occasion_type', render: (row) => dictionaryLabel('occasion_type', row.occasion_type) },
  { title: '方向', key: 'direction', render: (row) => directionText(row.direction) },
  { title: '金额', key: 'amount_cents', render: (row) => money(row.amount_cents) },
  { title: '回礼状态', key: 'reciprocity_status', render: (row) => row.reciprocity_status === 'done' ? `${statusText(row.reciprocity_status)}（${reciprocityRoleText(row.direction)}）` : statusText(row.reciprocity_status) },
  { title: '创建时间', key: 'created_at', render: (row) => formatDateTime(row.created_at) },
  {
    title: '操作',
    key: 'actions',
    width: 140,
    render: (row) =>
      h(NSpace, { size: 6 }, () => [
        h(NButton, { size: 'small', quaternary: true, type: 'primary', onClick: () => openEditRecord(row) }, () => '编辑'),
        h(NPopconfirm, { onPositiveClick: () => deleteRecord(row) }, { trigger: () => h(NButton, { size: 'small', quaternary: true, type: 'error' }, () => '删除'), default: () => '确认删除这条流水？' })
      ])
  }
]

const familyColumns: DataTableColumns<any> = [
  { title: 'ID', key: 'id' },
  { title: '名称', key: 'name' },
  { title: '联系人', key: 'contact_name' },
  { title: '邮箱', key: 'contact_email' },
  { title: '状态', key: 'status', render: (row) => statusText(row.status) },
  { title: '创建时间', key: 'created_at', render: (row) => formatDateTime(row.created_at) },
  {
    title: '操作',
    key: 'actions',
    render: (row) =>
      h(NSpace, { size: 6 }, () => [
        h(NButton, { size: 'small', quaternary: true, type: 'primary', onClick: () => openEditFamily(row) }, () => '编辑'),
        h(NButton, { size: 'small', quaternary: true, type: row.status === 'enabled' ? 'warning' : 'success', onClick: () => updateFamilyStatus(row) }, () => (row.status === 'enabled' ? '禁用' : '启用'))
      ])
  }
]

const userColumns: DataTableColumns<any> = [
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
    render: (row) =>
      h(NSpace, { size: 6 }, () => [
        h(NButton, { size: 'small', quaternary: true, type: 'primary', onClick: () => openEditUser(row) }, () => '编辑'),
        h(NButton, { size: 'small', quaternary: true, type: 'primary', onClick: () => openResetPassword(row) }, () => '重置密码'),
        h(NButton, { size: 'small', quaternary: true, type: row.status === 'enabled' ? 'warning' : 'success', onClick: () => updateUserStatus(row) }, () => (row.status === 'enabled' ? '禁用' : '启用'))
      ])
  }
]

const contactColumns: DataTableColumns<any> = [
  { title: '名称', key: 'name' },
  { title: '归属成员', key: 'owner_user', render: (row) => row.owner_user?.display_name || row.owner_user?.username || '未指定' },
  { title: '对象类型', key: 'contact_type', render: (row) => dictionaryLabel('contact_type', row.contact_type) },
  { title: '关系类型', key: 'relation_type', render: (row) => dictionaryLabel('relation_type', row.relation_type) },
  { title: '手机号', key: 'phone' },
  { title: '地址', key: 'address' },
  { title: '状态', key: 'status', render: (row) => statusText(row.status) },
  { title: '创建时间', key: 'created_at', render: (row) => formatDateTime(row.created_at) },
  {
    title: '操作',
    key: 'actions',
    render: (row) =>
      h(NSpace, { size: 6 }, () => [
        h(NButton, { size: 'small', quaternary: true, type: 'primary', onClick: () => openEditContact(row) }, () => '编辑'),
        h(NPopconfirm, { onPositiveClick: () => deleteContact(row) }, { trigger: () => h(NButton, { size: 'small', quaternary: true, type: 'error' }, () => '删除'), default: () => '确认删除该对象？' })
      ])
  }
]

const dictTypeColumns: DataTableColumns<any> = [
  { title: '类型编码', key: 'code' },
  { title: '类型名称', key: 'name' },
  { title: '允许家庭自定义', key: 'allow_family_custom', render: (row) => boolText(row.allow_family_custom) },
  { title: '启用', key: 'enabled', render: (row) => statusText(row.enabled) },
  { title: '排序', key: 'sort_order' },
  { title: '操作', key: 'actions', render: (row) => h(NButton, { size: 'small', quaternary: true, type: 'primary', onClick: () => openEditDictType(row) }, () => '编辑') }
]

const dictItemColumns: DataTableColumns<any> = [
  { title: '标签', key: 'label' },
  { title: '值', key: 'value' },
  { title: '来源', key: 'is_system', render: (row) => (row.is_system ? '系统预置' : '家庭自定义') },
  { title: '启用', key: 'enabled', render: (row) => statusText(row.enabled) },
  { title: '排序', key: 'sort_order' },
  { title: '更新时间', key: 'updated_at', render: (row) => formatDateTime(row.updated_at) },
  {
    title: '操作',
    key: 'actions',
    render: (row) =>
      !canEditDictionaryItem(row)
        ? dictionaryItemReadonlyText(row)
        : h(NSpace, { size: 6 }, () => [
            h(NButton, { size: 'small', quaternary: true, type: 'primary', onClick: () => openEditDict(row) }, () => '编辑'),
            h(NPopconfirm, { onPositiveClick: () => deleteDict(row) }, { trigger: () => h(NButton, { size: 'small', quaternary: true, type: 'error' }, () => '删除'), default: () => '确认删除该字典项？' })
          ])
  }
]

const backupJobColumns: DataTableColumns<any> = [
  { title: '文件', key: 'file_name' },
  { title: '触发方式', key: 'trigger_type', render: (row) => statusText(row.trigger_type) },
  { title: '状态', key: 'status', render: (row) => statusText(row.status) },
  { title: '开始时间', key: 'started_at', render: (row) => formatDateTime(row.started_at) },
  { title: '完成时间', key: 'finished_at', render: (row) => formatDateTime(row.finished_at) },
  { title: '错误信息', key: 'error_message' }
]

const importErrorColumns: DataTableColumns<any> = [
  { title: '工作表', key: 'sheet_name' },
  { title: '行', key: 'row_number' },
  { title: '字段', key: 'field_name' },
  { title: '错误', key: 'error_message' }
]

const importJobColumns: DataTableColumns<any> = [
  { title: '文件', key: 'source_file_name' },
  { title: '状态', key: 'status', render: (row) => statusText(row.status) },
  { title: '新增', key: 'created_count' },
  { title: '更新', key: 'updated_count' },
  { title: '错误', key: 'error_count' },
  { title: '开始时间', key: 'started_at', render: (row) => formatDateTime(row.started_at) },
  {
    title: '操作',
    key: 'actions',
    render: (row) => renderImportJobAction(row)
  }
]

function renderImportJobAction(row: any) {
  if (row.status === 'dry_run_ready') {
    return h(NButton, { size: 'small', type: 'primary', secondary: true, onClick: () => executeImport(row.id) }, () => '执行导入')
  }
  if (isImportJobFailed(row)) {
    return h(NButton, { size: 'small', type: 'error', secondary: true, onClick: () => showImportJobErrors(row) }, () => '查看详情')
  }
  return importJobActionText(row)
}

function isImportJobFailed(row: any) {
  return row.status === 'dry_run_failed' || row.status === 'failed' || row.status === 'blocked' || row.error_count > 0
}

function importJobActionText(row: any) {
  if (row.status === 'success') return '已完成'
  return '-'
}

const auditColumns: DataTableColumns<any> = [
  { title: '时间', key: 'created_at', render: (row) => formatDateTime(row.created_at) },
  { title: '用户', key: 'operator', render: (row) => row.operator?.display_name || row.operator?.username || '-' },
  { title: '动作', key: 'action', render: (row) => actionText(row.action) },
  { title: '资源', key: 'resource_type', render: (row) => resourceText(row.resource_type) },
  { title: '结果', key: 'success', render: (row) => statusText(row.success) },
  { title: 'IP', key: 'ip' },
  { title: '浏览器', key: 'user_agent' }
]

watch(activeView, () => {
  scrollActiveMobileNavIntoView()
})
watch(() => menuItems.value.map((item) => item.key).join(','), scrollActiveMobileNavIntoView)
watch(keyword, () => debouncedContactsSearch())
async function bootApp() {
  await loadVersion()
  await loadMe()
}
onMounted(async () => {
  await bootApp()
  scrollActiveMobileNavIntoView()
})
</script>

<template>
  <n-config-provider :locale="zhCN" :date-locale="dateZhCN">
    <n-message-provider>
      <main v-if="!authReady" class="login-page">
        <section class="login-panel loading-panel">
          <n-spin size="medium" />
          <strong>正在恢复登录状态</strong>
        </section>
      </main>

      <main v-else-if="!isLoggedIn" class="login-page">
        <section class="login-panel">
          <div>
            <p class="eyebrow">礼尚往来</p>
            <h1>家庭往来账本</h1>
          </div>
          <n-form label-placement="top">
            <n-form-item label="账号">
              <n-input v-model:value="loginForm.username" placeholder="请输入账号" />
            </n-form-item>
            <n-form-item label="密码">
              <n-input v-model:value="loginForm.password" type="password" show-password-on="click" placeholder="请输入密码" @keydown.enter="login" />
            </n-form-item>
            <n-button type="primary" block :loading="loading" @click="login">
              <template #icon><KeyRound :size="18" /></template>
              登录
            </n-button>
          </n-form>
          <p class="version-text">版本 {{ appVersion }}</p>
        </section>
      </main>

      <main v-else-if="mustChangePassword" class="login-page">
        <section class="login-panel">
          <div>
            <p class="eyebrow">首次登录</p>
            <h1>修改初始密码</h1>
          </div>
          <n-form label-placement="top">
            <n-form-item label="原密码">
              <n-input v-model:value="passwordForm.old_password" type="password" show-password-on="click" placeholder="请输入当前初始密码" />
            </n-form-item>
            <n-form-item label="新密码">
              <n-input v-model:value="passwordForm.new_password" type="password" show-password-on="click" placeholder="至少 10 位，包含大小写字母、数字和特殊字符" @keydown.enter="changePassword" />
            </n-form-item>
            <n-button type="primary" block @click="changePassword">完成修改并重新登录</n-button>
            <n-button quaternary block @click="logout">返回登录</n-button>
          </n-form>
        </section>
      </main>

      <main v-else class="app-layout">
        <aside class="sidebar">
          <div class="brand">
            <strong>礼尚往来</strong>
            <span>{{ user?.display_name }} · {{ user?.role }} · v{{ appVersion }}</span>
          </div>
          <n-select
            v-if="isSuperAdmin"
            class="family-picker"
            v-model:value="selectedFamilyId"
            :options="[{ label: '全部家庭', value: 0 }, ...families.map((item) => ({ label: item.name, value: item.id }))]"
            placeholder="选择家庭"
            @update:value="handleFamilyChange"
          />
          <div v-else class="family-chip">{{ user?.family?.name || user?.display_name }}</div>
          <nav>
            <button v-for="item in menuItems" :key="item.key" :class="{ active: activeView === item.key }" @click="switchView(item.key as ViewKey)">
              <component :is="viewIcons[item.key as ViewKey]" :size="18" />
              <span>{{ item.label }}</span>
            </button>
          </nav>
          <n-button class="logout-action" secondary @click="logout">
            <template #icon><LogOut :size="17" /></template>
            退出
          </n-button>
        </aside>

        <nav ref="mobileNavRef" class="mobile-nav">
          <button v-for="item in menuItems" :key="item.key" :data-menu-key="item.key" :class="{ active: activeView === item.key }" @click="switchView(item.key as ViewKey)">
            <component :is="viewIcons[item.key as ViewKey]" :size="18" />
            <span>{{ item.label }}</span>
          </button>
        </nav>

        <section class="content">
          <header class="topbar" :class="{ 'topbar-compact': activeView !== 'contacts' }">
            <div>
              <p class="eyebrow">{{ user?.family?.name || '系统管理' }}</p>
              <h1>{{ menuItems.find((item) => item.key === activeView)?.label }}</h1>
            </div>
            <n-input v-if="activeView === 'contacts'" v-model:value="keyword" clearable placeholder="搜索对象、手机号" @keydown.enter="loadContacts">
              <template #prefix><Search :size="18" /></template>
            </n-input>
          </header>
          <div class="refresh-line" :class="{ active: viewLoading }" aria-hidden="true"></div>
          <n-card v-if="viewError" :key="`${activeView}-error`" class="view-state" size="small">
            <n-alert type="warning" :show-icon="false">{{ viewError }}</n-alert>
          </n-card>
          <section v-if="showFullViewLoading" :key="`${activeView}-loading`" class="view-stack view-loading">
            <n-card size="small" class="skeleton-card">
              <n-skeleton text style="width: 34%" />
              <n-skeleton text :repeat="2" />
            </n-card>
            <div class="skeleton-grid">
              <n-card v-for="item in 4" :key="item" size="small" class="skeleton-card">
                <n-skeleton text style="width: 52%" />
                <n-skeleton text style="width: 34%" />
              </n-card>
            </div>
            <n-card size="small" class="skeleton-card skeleton-main">
              <n-skeleton text style="width: 28%" />
              <n-skeleton text :repeat="5" />
            </n-card>
          </section>

          <DashboardView
            v-show="activeView === 'dashboard'"
            key="dashboard-view"
            :stats="stats"
            :query="statsQuery"
            :record-form="recordForm"
            :contacts="contacts"
            :records="records"
            :occasion-type-options="occasionTypeOptions"
            :handler-options="handlerOptions"
            :record-columns="recordColumns"
            :money="money"
            :dictionary-label="dictionaryLabel"
            :direction-text="directionText"
            :status-text="statusText"
            :format-date="formatDate"
            :reciprocity-role-text="reciprocityRoleText"
            @query="queryStats"
            @set-range="setStatsRange"
            @set-record-direction="setRecordDirection"
            @save-record="runAction(createRecord)"
            @edit-record="openEditRecord"
            @delete-record="deleteRecord"
          />

          <FamiliesView
            v-show="activeView === 'families'"
            key="families-view"
            :families="families"
            :columns="familyColumns"
            :status-text="statusText"
            :format-date-time="formatDateTime"
            @create="openCreateFamily"
            @edit="openEditFamily"
            @update-status="(family) => runAction(() => updateFamilyStatus(family))"
          />

          <UsersView
            v-show="activeView === 'users'"
            key="users-view"
            :users="users"
            :columns="userColumns"
            :status-text="statusText"
            :format-date-time="formatDateTime"
            @create="openCreateUser"
            @edit="openEditUser"
            @reset-password="openResetPassword"
            @update-status="(item) => runAction(() => updateUserStatus(item))"
          />

          <ContactsView
            v-show="activeView === 'contacts'"
            key="contacts-view"
            :contacts="contacts"
            :columns="contactColumns"
            :dictionary-label="dictionaryLabel"
            @create="openCreateContact"
            @edit="openEditContact"
            @delete="deleteContact"
          />

          <RecordsView
            v-show="activeView === 'records'"
            key="records-view"
            :query="recordQuery"
            :contacts="contacts"
            :records="records"
            :columns="recordColumns"
            :occasion-type-options="occasionTypeOptions"
            :dictionary-label="dictionaryLabel"
            :direction-text="directionText"
            :status-text="statusText"
            :money="money"
            :format-date="formatDate"
            :reciprocity-role-text="reciprocityRoleText"
            @query-records="queryRecords"
            @load-records="loadRecords"
            @create="openCreateRecord"
            @edit="openEditRecord"
            @delete="deleteRecord"
          />

          <DictionariesView
            v-show="activeView === 'dictionaries'"
            key="dictionaries-view"
            :is-super-admin="isSuperAdmin"
            :dictionary-types="dictionaryTypes"
            :dictionary-groups="dictionaryGroups"
            :type-columns="dictTypeColumns"
            :item-columns="dictItemColumns"
            :writable-dictionary-types="writableDictionaryTypes"
            :status-text="statusText"
            :bool-text="boolText"
            :format-date-time="formatDateTime"
            :can-edit-dictionary-item="canEditDictionaryItem"
            @create-type="openCreateDictType"
            @create-item="openCreateDict"
            @edit-type="openEditDictType"
            @edit-item="openEditDict"
            @delete-item="deleteDict"
          />

          <StatsView
            v-show="activeView === 'stats'"
            key="stats-view"
            :active="activeView === 'stats'"
            :stats="stats"
            :query="statsQuery"
            :money="money"
            :money-compact="moneyCompact"
            :dictionary-label="dictionaryLabel"
            :status-text="statusText"
            @query="queryStats"
            @set-range="setStatsRange"
          />

          <BackupView
            v-show="activeView === 'backup'"
            key="backup-view"
            :form="backupForm"
            :jobs="backupJobs"
            :columns="backupJobColumns"
            :status-text="statusText"
            :format-date-time="formatDateTime"
            @edit="showBackupModal = true"
            @download="runAction(() => downloadBackup(false))"
            @send-email="runAction(() => downloadBackup(true))"
          />

          <ImportsView
            v-show="activeView === 'imports'"
            key="imports-view"
            :errors="importErrors"
            :jobs="importJobs"
            :error-columns="importErrorColumns"
            :job-columns="importJobColumns"
            :status-text="statusText"
            :format-date-time="formatDateTime"
            :is-job-failed="isImportJobFailed"
            @download-template="runAction(downloadImportTemplate)"
            @open-manual="openImportModal('manual')"
            @open-backup="openImportModal('backup')"
            @execute-import="(jobId) => runAction(() => executeImport(jobId))"
            @show-errors="(job) => runAction(() => showImportJobErrors(job))"
          />

          <AuditView
            v-show="activeView === 'audit'"
            key="audit-view"
            :query="auditQuery"
            :logs="auditLogs"
            :columns="auditColumns"
            :action-text="actionText"
            :status-text="statusText"
            :resource-text="resourceText"
            :format-date-time="formatDateTime"
            @query-audit="queryAudit"
            @refresh="refreshAudit"
            @load-audit="loadAudit"
          />

          <n-modal v-model:show="showFamilyModal" preset="card" :title="editingFamilyId ? '编辑家庭' : '新增家庭'" class="form-modal">
            <n-form label-placement="top">
              <n-form-item label="家庭名称"><n-input v-model:value="familyForm.name" placeholder="请输入家庭名称" /></n-form-item>
              <n-form-item label="联系人"><n-input v-model:value="familyForm.contact_name" placeholder="请输入联系人姓名" /></n-form-item>
              <n-form-item label="联系邮箱"><n-input v-model:value="familyForm.contact_email" placeholder="请输入联系邮箱" /></n-form-item>
              <n-form-item label="备注"><n-input v-model:value="familyForm.remark" type="textarea" placeholder="请输入家庭备注" /></n-form-item>
              <n-divider>家庭管理员</n-divider>
              <n-form-item label="管理员来源"><n-radio-group v-model:value="familyForm.admin_mode"><n-radio-button value="existing">选择现有用户</n-radio-button><n-radio-button value="new">新建管理员</n-radio-button></n-radio-group></n-form-item>
              <n-form-item v-if="familyForm.admin_mode === 'existing'" label="现有用户"><n-select v-model:value="familyForm.admin_user_id" clearable filterable :options="adminUserOptions" placeholder="请选择现有用户" /></n-form-item>
              <template v-else>
                <n-form-item label="管理员账号"><n-input v-model:value="familyForm.admin_username" placeholder="请输入管理员登录账号" /></n-form-item>
                <n-form-item label="管理员姓名"><n-input v-model:value="familyForm.admin_display_name" placeholder="请输入管理员姓名" /></n-form-item>
                <n-form-item label="管理员邮箱"><n-input v-model:value="familyForm.admin_email" placeholder="请输入管理员邮箱" /></n-form-item>
                <n-form-item label="初始密码">
                  <n-input-group>
                    <n-input v-model:value="familyForm.admin_password" type="password" show-password-on="click" placeholder="至少 10 位，包含大小写字母、数字和特殊字符" />
                    <n-button @click="generatePassword('family_admin')">生成</n-button>
                    <n-button @click="copyGeneratedPassword('family_admin')">复制</n-button>
                  </n-input-group>
                </n-form-item>
              </template>
              <n-button type="primary" block @click="runAction(createFamily)">保存家庭</n-button>
            </n-form>
          </n-modal>

          <n-modal v-model:show="showUserModal" preset="card" :title="editingUserId ? '编辑用户' : '新增用户'" class="form-modal">
            <n-form label-placement="top">
              <n-form-item label="账号"><n-input v-model:value="userForm.username" placeholder="请输入登录账号" /></n-form-item>
              <n-form-item label="姓名"><n-input v-model:value="userForm.display_name" placeholder="请输入用户姓名" /></n-form-item>
              <n-form-item label="邮箱"><n-input v-model:value="userForm.email" placeholder="请输入邮箱" /></n-form-item>
              <n-form-item label="角色"><n-select v-model:value="userForm.role" :options="[{ label: '家庭管理员', value: 'family_admin' }, { label: '成员', value: 'member' }]" placeholder="请选择角色" /></n-form-item>
              <n-form-item v-if="!editingUserId" label="初始密码">
                <n-input-group>
                  <n-input v-model:value="userForm.password" type="password" show-password-on="click" placeholder="至少 10 位，包含大小写字母、数字和特殊字符" />
                  <n-button @click="generatePassword('user')">生成</n-button>
                  <n-button @click="copyGeneratedPassword('user')">复制</n-button>
                </n-input-group>
              </n-form-item>
              <n-button type="primary" block @click="runAction(createUser)">保存用户</n-button>
            </n-form>
          </n-modal>

          <n-modal v-model:show="showResetPasswordModal" preset="card" :title="`重置密码${resettingUser ? ` · ${resettingUser.display_name}` : ''}`" class="form-modal">
            <n-alert type="warning" :show-icon="false" class="modal-tip">
              重置后用户下次登录必须修改密码，当前有效会话会被注销。
            </n-alert>
            <n-form label-placement="top">
              <n-form-item label="新临时密码">
                <n-input-group>
                  <n-input v-model:value="resetPasswordForm.password" type="password" show-password-on="click" placeholder="至少 10 位，包含大小写字母、数字和特殊字符" />
                  <n-button @click="generatePassword('reset_user')">生成</n-button>
                  <n-button @click="copyGeneratedPassword('reset_user')">复制</n-button>
                </n-input-group>
              </n-form-item>
              <n-button type="primary" block @click="runAction(resetUserPassword)">确认重置</n-button>
            </n-form>
          </n-modal>

          <n-modal v-model:show="showContactModal" preset="card" :title="editingContactId ? '编辑往来对象' : '新增往来对象'" class="form-modal">
            <n-form label-placement="top">
              <n-form-item label="名称"><n-input v-model:value="contactForm.name" placeholder="请输入姓名、家庭或单位名称" /></n-form-item>
              <n-form-item label="归属成员"><n-select v-model:value="contactForm.owner_user_id" clearable filterable :options="familyMemberOptions" placeholder="请选择该对象属于哪位家庭成员" /></n-form-item>
              <n-form-item label="对象类型"><n-select v-model:value="contactForm.contact_type" filterable :options="contactTypeOptions" placeholder="请选择对象类型" /></n-form-item>
              <n-form-item label="关系类型"><n-select v-model:value="contactForm.relation_type" filterable :options="relationTypeOptions" placeholder="请选择关系类型" /></n-form-item>
              <n-form-item label="手机号"><n-input v-model:value="contactForm.phone" placeholder="请输入手机号" /></n-form-item>
              <n-form-item label="地址"><n-input v-model:value="contactForm.address" placeholder="请输入地址" /></n-form-item>
              <n-form-item label="状态"><n-select v-model:value="contactForm.status" :options="[{ label: '启用', value: 'enabled' }, { label: '禁用', value: 'disabled' }]" placeholder="请选择状态" /></n-form-item>
              <n-form-item label="备注"><n-input v-model:value="contactForm.remark" type="textarea" placeholder="请输入备注" /></n-form-item>
              <n-button type="primary" block @click="runAction(createContact)">保存对象</n-button>
            </n-form>
          </n-modal>

          <n-modal v-model:show="showRecordModal" preset="card" :title="editingRecordId ? '编辑往来流水' : '新增往来流水'" class="form-modal">
            <n-form label-placement="top">
              <n-form-item label="往来对象"><n-select v-model:value="recordForm.contact_id" filterable :options="contacts.map((item) => ({ label: item.name, value: item.id }))" placeholder="请选择往来对象" /></n-form-item>
              <n-form-item label="事项类型"><n-select v-model:value="recordForm.occasion_type" filterable :options="occasionTypeOptions" placeholder="请选择事项类型" /></n-form-item>
              <n-form-item label="往来方向">
                <div class="direction-choice">
                  <button type="button" :class="{ active: recordForm.direction === 'in' }" @click="setRecordDirection('in')">
                    <strong>我家收礼</strong>
                    <span>收到对方礼金或礼品</span>
                  </button>
                  <button type="button" :class="{ active: recordForm.direction === 'out' }" @click="setRecordDirection('out')">
                    <strong>我家送礼</strong>
                    <span>送出礼金或礼品</span>
                  </button>
                </div>
              </n-form-item>
              <div class="form-row">
                <n-form-item label="金额"><n-input-number v-model:value="recordForm.amount_yuan" :min="0" :step="50" placeholder="请输入金额（元）" /></n-form-item>
                <n-form-item label="发生日期"><n-date-picker v-model:formatted-value="recordForm.occurred_on" value-format="yyyy-MM-dd" type="date" :clearable="false" /></n-form-item>
              </div>
              <n-form-item label="经办人"><n-select v-model:value="recordForm.handled_by" clearable filterable tag :options="handlerOptions" placeholder="请选择或输入经办人" /></n-form-item>
              <n-form-item label="回礼状态"><n-select :value="recordForm.reciprocity_status" :options="[{ label: '无需回礼', value: 'none' }, { label: '待回礼', value: 'pending' }, { label: '已回礼', value: 'done' }]" placeholder="请选择回礼状态" @update:value="setRecordReciprocityStatus" /></n-form-item>
              <n-form-item v-if="recordForm.reciprocity_status === 'done'" :label="`关联${reciprocityRoleText(recordForm.direction)}流水`">
                <n-select v-model:value="recordForm.related_record_id" clearable filterable :options="relatedRecordOptions" :placeholder="`请选择${reciprocityRoleText(recordForm.direction)}对应的反向流水`" />
              </n-form-item>
              <n-form-item label="礼品说明"><n-input v-model:value="recordForm.gift_description" type="textarea" placeholder="请输入礼品名称、数量或其他说明" /></n-form-item>
              <n-form-item label="备注"><n-input v-model:value="recordForm.remark" type="textarea" placeholder="请输入备注" /></n-form-item>
              <n-button type="primary" block @click="runAction(createRecord)">保存流水</n-button>
            </n-form>
          </n-modal>

          <n-modal v-model:show="showDictModal" preset="card" :title="editingDictId ? '编辑字典项' : '新增字典项'" class="form-modal">
            <n-form label-placement="top">
              <n-form-item label="字典类型"><n-select v-model:value="dictForm.type_id" :disabled="Boolean(editingDictId)" :options="dictionaryTypeOptions" placeholder="请选择字典类型" /></n-form-item>
              <n-form-item label="标签"><n-input v-model:value="dictForm.label" placeholder="请输入显示名称" /></n-form-item>
              <n-form-item label="值"><n-input v-model:value="dictForm.value" placeholder="请输入实际保存值，系统会按原样保存" /></n-form-item>
              <n-form-item label="排序"><n-input-number v-model:value="dictForm.sort_order" placeholder="请输入排序值" /></n-form-item>
              <n-button type="primary" block @click="runAction(createDictionaryItem)">保存字典项</n-button>
            </n-form>
          </n-modal>

          <n-modal v-model:show="showBackupModal" preset="card" title="编辑备份配置" class="form-modal">
            <n-form label-placement="top">
              <n-form-item label="启用定时备份"><n-switch v-model:value="backupForm.enabled" /></n-form-item>
              <n-form-item label="备份频率"><n-select v-model:value="backupForm.frequency" :options="[{ label: '每日', value: 'daily' }, { label: '每周', value: 'weekly' }, { label: '每月', value: 'monthly' }]" placeholder="请选择备份频率" /></n-form-item>
              <n-form-item label="收件邮箱"><n-input v-model:value="backupForm.recipient_emails_text" type="textarea" placeholder="每行一个邮箱" /></n-form-item>
              <n-form-item label="ZIP 加密"><n-switch v-model:value="backupForm.encrypt_enabled" /></n-form-item>
              <n-form-item label="ZIP 密码"><n-input v-model:value="backupForm.zip_password" type="password" placeholder="启用加密时请输入或重设密码" /></n-form-item>
              <n-form-item label="密码提示"><n-input v-model:value="backupForm.zip_password_hint" placeholder="请输入给家人的密码提示，不要写明文密码" /></n-form-item>
              <n-button type="primary" block @click="runAction(saveBackupAndClose)">保存配置</n-button>
            </n-form>
          </n-modal>

          <n-modal v-model:show="showImportModal" preset="card" :title="importMode === 'manual' ? '手工导入' : '备份恢复'" class="form-modal">
            <p v-if="importMode === 'manual'" class="modal-hint">适合普通录入，只需要填写往来流水和可选的往来对象信息。系统会先校验，不会立即写入数据。</p>
            <p v-else class="modal-hint">仅用于恢复系统导出的完整备份文件，会识别编号、字典和家庭成员等高级字段。系统会先校验，不会立即写入数据。</p>
            <n-upload :max="1" accept=".xlsx" @update:file-list="(files) => importFile = files[0] || null">
              <n-button><template #icon><Upload :size="17" /></template>选择 Excel 文件</n-button>
            </n-upload>
            <div class="modal-actions">
              <n-button type="primary" @click="runAction(dryRunImport)">导入校验</n-button>
            </div>
          </n-modal>

          <n-modal v-model:show="showDictTypeModal" preset="card" :title="editingDictTypeId ? '编辑字典类型' : '新增字典类型'" class="form-modal">
            <n-form label-placement="top">
              <n-form-item label="类型编码"><n-input v-model:value="dictTypeForm.code" placeholder="请输入类型编码，例如 relation_type" /></n-form-item>
              <n-form-item label="类型名称"><n-input v-model:value="dictTypeForm.name" placeholder="请输入类型名称，例如 关系类型" /></n-form-item>
              <n-form-item label="说明"><n-input v-model:value="dictTypeForm.description" type="textarea" placeholder="请输入类型说明" /></n-form-item>
              <n-form-item label="允许家庭自定义"><n-switch v-model:value="dictTypeForm.allow_family_custom" /></n-form-item>
              <n-form-item label="排序"><n-input-number v-model:value="dictTypeForm.sort_order" placeholder="请输入排序值" /></n-form-item>
              <n-form-item label="启用"><n-switch v-model:value="dictTypeForm.enabled" /></n-form-item>
              <n-button type="primary" block @click="runAction(createDictionaryType)">保存字典类型</n-button>
            </n-form>
          </n-modal>
        </section>
      </main>
    </n-message-provider>
  </n-config-provider>
</template>
