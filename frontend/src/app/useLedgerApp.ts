import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { createDiscreteApi, type UploadFileInfo } from 'naive-ui'
import { allowedViewKeys, buildMenuItems, viewIcons } from './navigation'
import { actionText, appendAuditSearchParams, resourceText } from '../entities/audit/model'
import { createTableColumns, isImportJobFailed } from '../features/table-columns/model'
import { boolText, directionText, formatDate, formatDateTime, money, moneyCompact, reciprocityRelationText, reciprocityRoleText, statusText } from '../shared/lib/formatters'
import { createApiClient, createLatestOnly, debounce, persistNumberCache, readArrayCache, readNumberCache, writeJsonCache } from '../shared/lib/runtime'
import type { PageResult, SessionUser, ViewKey } from '../shared/types'

export function useLedgerApp() {
  const API_BASE = import.meta.env.VITE_API_BASE || '/api/v1'
  const FALLBACK_VERSION = import.meta.env.VITE_APP_VERSION || '1.6.0.3'
  const FAMILY_CACHE_KEY = 'ledger_families_cache'
  const SELECTED_FAMILY_KEY = 'selected_family_id'

  const { message, dialog } = createDiscreteApi(['message', 'dialog'])
  const token = ref(localStorage.getItem('access_token') || '')
  const refreshToken = ref(localStorage.getItem('refresh_token') || '')
  const user = ref<SessionUser | null>(null)
  const storedView = localStorage.getItem('active_view') as ViewKey | null
  const activeView = ref<ViewKey>(storedView && allowedViewKeys.includes(storedView) ? storedView : 'dashboard')
  const loading = ref(false)
  const viewLoading = ref(false)
  const viewError = ref('')
  const tableLoading = reactive<Record<string, boolean>>({})
  const tableLoadingCounts: Record<string, number> = {}
  const authReady = ref(false)
  const appVersion = ref(FALLBACK_VERSION)
  const selectedFamilyId = ref<number | null>(readNumberCache(SELECTED_FAMILY_KEY))
  const loadedViews = reactive<Record<string, boolean>>({})
  const showFamilyModal = ref(false)
  const showUserModal = ref(false)
  const showResetPasswordModal = ref(false)
  const showContactModal = ref(false)
  const showContactDetail = ref(false)
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
  const actionSubmitting = ref(false)
  let lastActionAt = 0
  let hydratingRecordForm = false
  const contactSaving = ref(false)
  const contactDetailLoading = ref(false)
  const recordSaving = ref(false)
  const quickRecordKeepDraft = ref(true)

  const loginForm = reactive({ username: '', password: '' })
  const passwordForm = reactive({ old_password: '', new_password: '' })
  const familyForm = reactive({ name: '', contact_name: '', contact_email: '', remark: '', admin_mode: 'existing', admin_user_id: null as number | null, admin_username: '', admin_display_name: '', admin_email: '', admin_password: '' })
  const userForm = reactive({ username: '', display_name: '', email: '', password: '', role: 'member', family_id: null as number | null })
  const resetPasswordForm = reactive({ password: '' })
  const contactForm = reactive({ name: '', owner_user_id: null as number | null, contact_type: '', relation_type: '', phone: '', address: '', remark: '', status: 'enabled' })
  const recordForm = reactive({ contact_id: null as number | null, occasion_type: '', direction: 'in', amount_yuan: 200, gift_description: '', occurred_on: new Date().toISOString().slice(0, 10), handled_by: '', reciprocity_status: 'pending', related_record_id: null as number | null, remark: '' })
  const recordDialogForm = reactive({ contact_id: null as number | null, occasion_type: '', direction: 'in', amount_yuan: 200, gift_description: '', occurred_on: new Date().toISOString().slice(0, 10), handled_by: '', reciprocity_status: 'pending', related_record_id: null as number | null, remark: '' })
  const dictForm = reactive({ type_id: null as number | null, label: '', value: '', sort_order: 0, enabled: true })
  const dictTypeForm = reactive({ code: '', name: '', description: '', allow_family_custom: true, sort_order: 0, enabled: true })
  const backupForm = reactive({ enabled: false, frequency: 'weekly', recipient_emails_text: '', encrypt_enabled: false, zip_password: '', zip_password_hint: '' })
  const importFile = ref<UploadFileInfo | null>(null)

  const families = ref<any[]>(readArrayCache(FAMILY_CACHE_KEY))
  const familyRows = ref<any[]>([])
  const users = ref<any[]>([])
  const userOptionsSource = ref<any[]>([])
  const contacts = ref<any[]>([])
  const contactDetail = ref<any>(null)
  const contactOptions = ref<any[]>([])
  const records = ref<any[]>([])
  const dashboardRecords = ref<any[]>([])
  const relatedRecords = ref<any[]>([])
  const dictionaryTypes = ref<any[]>([])
  const dictionaryItems = ref<any[]>([])
  const auditLogs = ref<any[]>([])
  const auditRetention = ref<any>(null)
  const backupJobs = ref<any[]>([])
  const importJobs = ref<any[]>([])
  const importErrors = ref<any[]>([])
  const stats = ref<any>(null)
  const todos = ref<any>(null)
  const receivables = ref<any>(null)
  const payables = ref<any>(null)
  const nonReciprocal = ref<any>(null)
  const backupConfig = ref<any>(null)
  const familyListQuery = reactive({ page: 1, page_size: 10, total: 0, keyword: '', status: null as string | null })
  const userQuery = reactive({ page: 1, page_size: 10, total: 0, keyword: '', role: null as string | null, status: null as string | null, family_id: null as number | null })
  const contactQuery = reactive({ page: 1, page_size: 10, total: 0, keyword: '', contact_type: null as string | null, relation_type: null as string | null, owner_user_id: null as number | null, status: null as string | null, has_owner: null as boolean | null, has_phone: null as boolean | null })
  const recordQuery = reactive({ page: 1, page_size: 10, total: 0, contact_id: null as number | null, owner_user_id: null as number | null, contact_type: null as string | null, relation_type: null as string | null, direction: null as string | null, occasion_type: null as string | null, reciprocity_status: null as string | null, reciprocity_unlinked: null as boolean | null, remark: '', start_date: null as string | null, end_date: null as string | null })
  const statsQuery = reactive({ range: 'all', start_date: null as string | null, end_date: null as string | null })
  const todoQuery = reactive({ page: 1, page_size: 10, total: 0, range: 'all', start_date: null as string | null, end_date: null as string | null })
  const receivableQuery = reactive({ page: 1, page_size: 10, total: 0, range: 'all', start_date: null as string | null, end_date: null as string | null })
  const payableQuery = reactive({ page: 1, page_size: 10, total: 0, range: 'all', start_date: null as string | null, end_date: null as string | null })
  const nonReciprocalQuery = reactive({ page: 1, page_size: 10, total: 0, range: 'all', start_date: null as string | null, end_date: null as string | null })
  const backupQuery = reactive({ page: 1, page_size: 10, total: 0 })
  const importQuery = reactive({ page: 1, page_size: 10, total: 0 })
  const auditQuery = reactive({ page: 1, page_size: 10, total: 0, keyword: '', action: '', success: null as boolean | null })
  const mobileNavRef = ref<HTMLElement | null>(null)
  const latestOnly = createLatestOnly()
  let sessionExpiredPrompting = false
  const { api, downloadBlob } = createApiClient(API_BASE, () => token.value, handleSessionExpired)
  let viewLoadSeq = 0

  function defaultRecordFormValues() {
    return {
      contact_id: null as number | null,
      occasion_type: occasionTypeOptions.value?.[0]?.value || '',
      direction: 'in',
      amount_yuan: 200,
      gift_description: '',
      occurred_on: new Date().toISOString().slice(0, 10),
      handled_by: user.value?.display_name || '',
      reciprocity_status: 'pending',
      related_record_id: null as number | null,
      remark: ''
    }
  }

  function activeRecordForm() {
    return showRecordModal.value ? recordDialogForm : recordForm
  }

  async function withTableLoading<T>(key: string, task: () => Promise<T>) {
    tableLoadingCounts[key] = (tableLoadingCounts[key] || 0) + 1
    tableLoading[key] = true
    try {
      return await task()
    } finally {
      tableLoadingCounts[key] -= 1
      if (tableLoadingCounts[key] <= 0) {
        tableLoadingCounts[key] = 0
        tableLoading[key] = false
      }
    }
  }

  function cacheFamilies(list: any[]) {
    writeJsonCache(FAMILY_CACHE_KEY, list)
  }

  function persistSelectedFamilyId(value: number | null) {
    persistNumberCache(SELECTED_FAMILY_KEY, value)
  }

  function handleSessionExpired() {
    if (!token.value || sessionExpiredPrompting) return
    sessionExpiredPrompting = true
    dialog.warning({
      title: '登录已过期',
      content: '登录状态已过期，请重新登录。',
      positiveText: '确定',
      closable: false,
      maskClosable: false,
      onPositiveClick: () => {
        sessionExpiredPrompting = false
        logout()
      }
    })
  }

  const isLoggedIn = computed(() => Boolean(token.value))
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
    relatedRecords.value
      .filter((item) => {
        const form = activeRecordForm()
        return item.id !== editingRecordId.value && (!form.contact_id || item.contact_id === form.contact_id) && item.direction !== form.direction
      })
      .map((item) => ({
        label: `${formatDate(item.occurred_on)} · ${reciprocityRelationText(item.direction)} · ${item.contact?.name || item.contact_id} · ${money(item.amount_cents)}`,
        value: item.id
      }))
  )
  const adminUserOptions = computed(() =>
    userOptionsSource.value
      .filter((item) => item.role !== 'super_admin' && (!editingFamilyId.value || !item.family_id || item.family_id === editingFamilyId.value))
      .map((item) => ({ label: `${item.display_name}（${item.username}）`, value: item.id }))
  )
  const handlerOptions = computed(() =>
    userOptionsSource.value
      .filter((item) => item.status !== 'disabled')
      .map((item) => ({ label: item.display_name || item.username, value: item.display_name || item.username }))
  )
  const familyMemberOptions = computed(() =>
    userOptionsSource.value
      .filter((item) => item.status !== 'disabled' && item.role !== 'super_admin' && (!isSuperAdmin.value || !currentFamilyId.value || item.family_id === currentFamilyId.value))
      .map((item) => ({ label: `${item.display_name || item.username}（${statusText(item.role)}）`, value: item.id }))
  )

  const menuItems = computed(() => buildMenuItems(isAdmin.value, isSuperAdmin.value))

  async function runAction(action: () => Promise<void>) {
    const now = Date.now()
    if (actionSubmitting.value || now - lastActionAt < 600) return
    actionSubmitting.value = true
    lastActionAt = now
    try {
      await action()
    } catch (error: any) {
      message.error(error?.message || '操作失败')
    } finally {
      actionSubmitting.value = false
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
      await copyText(password)
      message.success('密码已复制')
    } catch {
      message.error('复制失败，请手动选择密码')
    }
  }

  async function copyText(text: string) {
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(text)
        return
      }
    } catch {
      // Fall back to textarea copy below.
    }
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.setAttribute('readonly', 'true')
    textarea.style.position = 'fixed'
    textarea.style.left = '-9999px'
    textarea.style.top = '0'
    document.body.appendChild(textarea)
    textarea.focus()
    textarea.select()
    const copied = document.execCommand('copy')
    document.body.removeChild(textarea)
    if (!copied) throw new Error('copy failed')
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
      selectedFamilyId.value = user.value?.role === 'super_admin' ? readNumberCache(SELECTED_FAMILY_KEY) : user.value?.family_id || null
      message.success('登录成功')
    } catch (error: any) {
      message.error(error.message)
      return
    } finally {
      loading.value = false
    }
    if (!user.value?.must_change_password) {
      void loadInitialData()
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
      selectedFamilyId.value = user.value.role === 'super_admin' ? readNumberCache(SELECTED_FAMILY_KEY) : user.value.family_id || null
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
    clearLoadedViews()
  }

  async function changePassword() {
    loading.value = true
    try {
      await api('/auth/change-password', { method: 'POST', body: JSON.stringify(passwordForm) })
      message.success('密码已修改，请重新登录')
      logout()
    } catch (error: any) {
      message.error(error?.message || '密码修改失败')
    } finally {
      loading.value = false
    }
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
      message.warning(viewError.value)
      backupJobs.value = []
      importJobs.value = []
      importErrors.value = []
      return true
    }

    viewLoading.value = true
    try {
      if (view === 'dashboard') {
        await Promise.all([loadContactOptions(), loadDashboardRecords(), loadDictionaries(), loadUsers(), loadStats()])
      } else if (view === 'todos') {
        await loadTodos()
      } else if (view === 'families') {
        await loadFamilies()
      } else if (view === 'users') {
        await loadUsers()
      } else if (view === 'contacts') {
        await Promise.all([loadContacts(), loadContactOptions(), loadDictionaries(), loadUsers()])
      } else if (view === 'records') {
        await Promise.all([loadContactOptions(), loadRecords(), loadDictionaries(), loadUsers()])
      } else if (view === 'dictionaries') {
        await loadDictionaries()
      } else if (view === 'stats') {
        await loadStats()
      } else if (view === 'receivables') {
        await loadReceivables()
      } else if (view === 'payables') {
        await loadPayables()
      } else if (view === 'nonreciprocal') {
        await loadNonReciprocal()
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
      if (error?.unauthorized) return false
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
    await withTableLoading('families', async () => {
      const params = new URLSearchParams({ page: String(familyListQuery.page), page_size: String(familyListQuery.page_size) })
      if (familyListQuery.keyword) params.set('keyword', familyListQuery.keyword)
      if (familyListQuery.status) params.set('status', familyListQuery.status)
      await latestOnly('families', () => api<PageResult<any>>(`/families?${params}`), (data) => {
        familyRows.value = data.list
        familyListQuery.total = data.total
      })
      await loadFamilyOptions()
    })
  }

  async function loadFamilyOptions() {
    if (!isSuperAdmin.value) return
    const data = await api<PageResult<any>>('/families?page_size=100')
    families.value = data.list
    cacheFamilies(data.list)
    if (selectedFamilyId.value && !data.list.some((item) => item.id === selectedFamilyId.value)) {
      selectedFamilyId.value = null
      persistSelectedFamilyId(null)
    }
  }

  async function loadUsers() {
    await withTableLoading('users', async () => {
      const params = new URLSearchParams({ page: String(userQuery.page), page_size: String(userQuery.page_size) })
      if (userQuery.keyword) params.set('keyword', userQuery.keyword)
      if (userQuery.role) params.set('role', userQuery.role)
      if (userQuery.status) params.set('status', userQuery.status)
      if (isSuperAdmin.value && userQuery.family_id) params.set('family_id', String(userQuery.family_id))
      await latestOnly('users', () => api<PageResult<any>>(`/users?${params}`), (data) => {
        users.value = data.list
        userQuery.total = data.total
      })
      await loadUserOptions()
    })
  }

  async function loadUserOptions() {
    const family = isSuperAdmin.value && currentFamilyId.value ? `&family_id=${currentFamilyId.value}` : ''
    const data = await api<PageResult<any>>(`/users?page_size=100${family}`)
    userOptionsSource.value = data.list
  }

  async function loadContacts() {
    await withTableLoading('contacts', async () => {
      const params = new URLSearchParams({ page: String(contactQuery.page), page_size: String(contactQuery.page_size) })
      if (contactQuery.keyword) params.set('keyword', contactQuery.keyword)
      if (contactQuery.contact_type) params.set('contact_type', contactQuery.contact_type)
      if (contactQuery.relation_type) params.set('relation_type', contactQuery.relation_type)
      if (contactQuery.owner_user_id) params.set('owner_user_id', String(contactQuery.owner_user_id))
      if (contactQuery.status) params.set('status', contactQuery.status)
      if (contactQuery.has_owner !== null) params.set('has_owner', String(contactQuery.has_owner))
      if (contactQuery.has_phone !== null) params.set('has_phone', String(contactQuery.has_phone))
      if (isSuperAdmin.value && currentFamilyId.value) params.set('family_id', String(currentFamilyId.value))
      const query = params.toString()
      await latestOnly('contacts', () => api<PageResult<any>>(`/contacts?${query}`), (data) => {
        contacts.value = data.list
        contactQuery.total = data.total
      })
    })
  }

  async function loadContactOptions() {
    const params = new URLSearchParams({ page: '1', page_size: '1000' })
    if (isSuperAdmin.value && currentFamilyId.value) params.set('family_id', String(currentFamilyId.value))
    const query = params.toString()
    await latestOnly('contact-options', () => api<PageResult<any>>(`/contacts?${query}`), (data) => {
      contactOptions.value = [...data.list].sort((a, b) => String(a.name || '').localeCompare(String(b.name || ''), 'zh-Hans-CN', { numeric: true }))
    })
  }

  async function loadRecords() {
    await withTableLoading('records', async () => {
      const params = new URLSearchParams({ page: String(recordQuery.page), page_size: String(recordQuery.page_size) })
      if (recordQuery.contact_id) params.set('contact_id', String(recordQuery.contact_id))
      if (recordQuery.owner_user_id) params.set('owner_user_id', String(recordQuery.owner_user_id))
      if (recordQuery.contact_type) params.set('contact_type', recordQuery.contact_type)
      if (recordQuery.relation_type) params.set('relation_type', recordQuery.relation_type)
      if (recordQuery.direction) params.set('direction', recordQuery.direction)
      if (recordQuery.occasion_type) params.set('occasion_type', recordQuery.occasion_type)
      if (recordQuery.reciprocity_status) params.set('reciprocity_status', recordQuery.reciprocity_status)
      if (recordQuery.reciprocity_unlinked) params.set('reciprocity_unlinked', 'true')
      if (recordQuery.remark) params.set('remark', recordQuery.remark)
      if (recordQuery.start_date) params.set('start_date', recordQuery.start_date)
      if (recordQuery.end_date) params.set('end_date', recordQuery.end_date)
      if (isSuperAdmin.value && currentFamilyId.value) params.set('family_id', String(currentFamilyId.value))
      const query = params.toString()
      await latestOnly('records', () => api<PageResult<any>>(`/gift-records?${query}`), (data) => {
        records.value = data.list
        recordQuery.total = data.total
      })
    })
  }

  async function loadDashboardRecords() {
    await withTableLoading('dashboardRecords', async () => {
      const params = new URLSearchParams({ page: '1', page_size: '8' })
      appendStatsQueryParams(params)
      if (isSuperAdmin.value && currentFamilyId.value) params.set('family_id', String(currentFamilyId.value))
      const query = params.toString()
      await latestOnly('dashboard-records', () => api<PageResult<any>>(`/gift-records?${query}`), (data) => {
        dashboardRecords.value = data.list
      })
    })
  }

  async function loadRelatedRecords(form = activeRecordForm()) {
    if (form.reciprocity_status !== 'done') {
      relatedRecords.value = []
      return
    }
    const params = new URLSearchParams({ page: '1', page_size: '1000' })
    if (form.contact_id) params.set('contact_id', String(form.contact_id))
    if (form.direction) params.set('direction', form.direction === 'in' ? 'out' : 'in')
    if (isSuperAdmin.value && currentFamilyId.value) params.set('family_id', String(currentFamilyId.value))
    const query = params.toString()
    await latestOnly('related-records', () => api<PageResult<any>>(`/gift-records?${query}`), (data) => {
      relatedRecords.value = data.list
    })
  }

  async function refreshCurrentRecordView() {
    if (activeView.value === 'dashboard') {
      await Promise.all([loadDashboardRecords(), loadStats()])
      return
    }
    if (activeView.value === 'records') {
      await loadRecords()
    }
  }

  function patchRecordRows(items: any[]) {
    const patches = items.filter((item) => item?.id)
    if (!patches.length) return
    const apply = (list: any[]) => list.map((row) => patches.find((item) => item.id === row.id) || row)
    records.value = apply(records.value)
    dashboardRecords.value = apply(dashboardRecords.value)
    relatedRecords.value = apply(relatedRecords.value)
  }

  async function syncSavedRecordRows(savedRecord: any) {
    const patches = [savedRecord]
    if (savedRecord?.related_record_id) {
      try {
        patches.push(await api(`/gift-records/${savedRecord.related_record_id}`))
      } catch {
        // The list refresh below remains the source of truth if the related row is unavailable.
      }
    }
    patchRecordRows(patches)
  }

  function appendStatsQueryParams(params: URLSearchParams) {
    if (statsQuery.start_date) params.set('start_date', statsQuery.start_date)
    if (statsQuery.end_date) params.set('end_date', statsQuery.end_date)
  }

  function appendTodoQueryParams(params: URLSearchParams) {
    params.set('page', String(todoQuery.page))
    params.set('page_size', String(todoQuery.page_size))
    if (todoQuery.start_date) params.set('start_date', todoQuery.start_date)
    if (todoQuery.end_date) params.set('end_date', todoQuery.end_date)
  }

  function appendReceivableQueryParams(params: URLSearchParams) {
    params.set('page', String(receivableQuery.page))
    params.set('page_size', String(receivableQuery.page_size))
    if (receivableQuery.start_date) params.set('start_date', receivableQuery.start_date)
    if (receivableQuery.end_date) params.set('end_date', receivableQuery.end_date)
  }

  function appendPayableQueryParams(params: URLSearchParams) {
    params.set('page', String(payableQuery.page))
    params.set('page_size', String(payableQuery.page_size))
    if (payableQuery.start_date) params.set('start_date', payableQuery.start_date)
    if (payableQuery.end_date) params.set('end_date', payableQuery.end_date)
  }

  function appendNonReciprocalQueryParams(params: URLSearchParams) {
    params.set('page', String(nonReciprocalQuery.page))
    params.set('page_size', String(nonReciprocalQuery.page_size))
    if (nonReciprocalQuery.start_date) params.set('start_date', nonReciprocalQuery.start_date)
    if (nonReciprocalQuery.end_date) params.set('end_date', nonReciprocalQuery.end_date)
  }

  async function applyStatsQuery() {
    await Promise.all([loadStats(), activeView.value === 'dashboard' ? loadDashboardRecords() : Promise.resolve()])
  }

  async function applyTodoQuery() {
    await loadTodos()
  }

  async function applyReceivableQuery() {
    await loadReceivables()
  }

  async function applyPayableQuery() {
    await loadPayables()
  }

  async function applyNonReciprocalQuery() {
    await loadNonReciprocal()
  }

  function applyDateRange(query: { range: string; start_date: string | null; end_date: string | null }, range: string) {
    query.range = range
    const today = new Date()
    const toDate = (date: Date) => {
      const pad = (n: number) => String(n).padStart(2, '0')
      return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
    }
    if (range === 'all') {
      query.start_date = null
      query.end_date = null
    } else if (range === 'month') {
      query.start_date = toDate(new Date(today.getFullYear(), today.getMonth(), 1))
      query.end_date = toDate(today)
    } else if (range === 'year') {
      query.start_date = toDate(new Date(today.getFullYear(), 0, 1))
      query.end_date = toDate(today)
    } else if (range === 'last30') {
      const start = new Date(today)
      start.setDate(today.getDate() - 29)
      query.start_date = toDate(start)
      query.end_date = toDate(today)
    }
  }

  function setStatsRange(range: string) {
    applyDateRange(statsQuery, range)
  }

  function setTodoRange(range: string) {
    applyDateRange(todoQuery, range)
  }

  function setReceivableRange(range: string) {
    applyDateRange(receivableQuery, range)
  }

  function setPayableRange(range: string) {
    applyDateRange(payableQuery, range)
  }

  function setNonReciprocalRange(range: string) {
    applyDateRange(nonReciprocalQuery, range)
  }

  async function loadDictionaries() {
    await withTableLoading('dictionaries', async () => {
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

  async function loadTodos() {
    await withTableLoading('todos', async () => {
      const params = new URLSearchParams()
      appendTodoQueryParams(params)
      if (isSuperAdmin.value && currentFamilyId.value) params.set('family_id', String(currentFamilyId.value))
      const query = params.toString()
      await latestOnly('todos', () => api<any>(`/stats/todos${query ? `?${query}` : ''}`), (data) => {
        todos.value = data
        todoQuery.total = data.records?.total || 0
        todoQuery.page = data.records?.page || todoQuery.page
        todoQuery.page_size = data.records?.page_size || todoQuery.page_size
      })
    })
  }

  async function loadReceivables() {
    await withTableLoading('receivables', async () => {
      const params = new URLSearchParams()
      appendReceivableQueryParams(params)
      if (isSuperAdmin.value && currentFamilyId.value) params.set('family_id', String(currentFamilyId.value))
      const query = params.toString()
      await latestOnly('receivables', () => api<any>(`/stats/receivables${query ? `?${query}` : ''}`), (data) => {
        receivables.value = data
        receivableQuery.total = data.records?.total || 0
        receivableQuery.page = data.records?.page || receivableQuery.page
        receivableQuery.page_size = data.records?.page_size || receivableQuery.page_size
      })
    })
  }

  async function loadPayables() {
    await withTableLoading('payables', async () => {
      const params = new URLSearchParams()
      appendPayableQueryParams(params)
      if (isSuperAdmin.value && currentFamilyId.value) params.set('family_id', String(currentFamilyId.value))
      const query = params.toString()
      await latestOnly('payables', () => api<any>(`/stats/payables${query ? `?${query}` : ''}`), (data) => {
        payables.value = data
        payableQuery.total = data.records?.total || 0
        payableQuery.page = data.records?.page || payableQuery.page
        payableQuery.page_size = data.records?.page_size || payableQuery.page_size
      })
    })
  }

  async function loadNonReciprocal() {
    await withTableLoading('nonreciprocal', async () => {
      const params = new URLSearchParams()
      appendNonReciprocalQueryParams(params)
      if (isSuperAdmin.value && currentFamilyId.value) params.set('family_id', String(currentFamilyId.value))
      const query = params.toString()
      await latestOnly('nonreciprocal', () => api<any>(`/stats/non-reciprocal${query ? `?${query}` : ''}`), (data) => {
        nonReciprocal.value = data
        nonReciprocalQuery.total = data.records?.total || 0
        nonReciprocalQuery.page = data.records?.page || nonReciprocalQuery.page
        nonReciprocalQuery.page_size = data.records?.page_size || nonReciprocalQuery.page_size
      })
    })
  }

  async function loadAudit() {
    await withTableLoading('audit', async () => {
      const params = new URLSearchParams({ page: String(auditQuery.page), page_size: String(auditQuery.page_size) })
      appendAuditSearchParams(params, auditQuery.keyword)
      if (auditQuery.action) params.set('action', auditQuery.action)
      if (auditQuery.success !== null) params.set('success', String(auditQuery.success))
      const query = params.toString()
      await latestOnly('audit', async () => {
        const [logs, retention] = await Promise.all([
          api<PageResult<any>>(`/audit-logs?${query}`),
          api<any>('/audit-retention')
        ])
        return { logs, retention }
      }, ({ logs, retention }) => {
        auditLogs.value = logs.list
        auditQuery.total = logs.total
        auditRetention.value = retention
      })
    })
  }

  async function loadBackup() {
    if (!currentFamilyId.value) return
    await withTableLoading('backup', async () => {
      const requestFamilyQuery = familyQuery()
      await latestOnly('backup', async () => {
        const jobsParams = new URLSearchParams({ page: String(backupQuery.page), page_size: String(backupQuery.page_size) })
        if (requestFamilyQuery) jobsParams.set('family_id', String(currentFamilyId.value))
        const [configData, jobs] = await Promise.all([
          api<any>(`/backup/config${requestFamilyQuery}`),
          api<PageResult<any>>(`/backup/jobs?${jobsParams}`)
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
        backupQuery.total = jobs.total
      })
    })
  }

  async function loadImports() {
    if (!currentFamilyId.value) {
      tableLoading.imports = false
      tableLoadingCounts.imports = 0
      importJobs.value = []
      return
    }
    await withTableLoading('imports', async () => {
      const params = new URLSearchParams({ page: String(importQuery.page), page_size: String(importQuery.page_size) })
      if (isSuperAdmin.value && currentFamilyId.value) params.set('family_id', String(currentFamilyId.value))
      const data = await api<PageResult<any>>(`/imports?${params}`)
      importJobs.value = data.list
      importQuery.total = data.total
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
    if (contactSaving.value) return
    contactSaving.value = true
    try {
      const payload = { ...contactForm, family_id: currentFamilyId.value }
      if (editingContactId.value) {
        await api(`/contacts/${editingContactId.value}`, { method: 'PUT', body: JSON.stringify(payload) })
        message.success('对象已更新')
      } else {
        await api('/contacts', { method: 'POST', body: JSON.stringify(payload) })
        message.success('对象已保存')
      }
      closeContactModal()
      await Promise.all([loadContacts(), loadContactOptions(), loadRecords(), loadStats()])
    } finally {
      contactSaving.value = false
    }
  }

  async function createRecord(options: { keepDraft?: boolean; form?: typeof recordForm } = {}) {
    const form = options.form || recordForm
    if (recordSaving.value) return
    if (!form.contact_id || !form.occasion_type || !form.direction || !form.occurred_on) {
      message.warning('请完整填写往来对象、事项类型、方向和日期')
      return
    }
    recordSaving.value = true
    try {
      const payload = { ...form, family_id: currentFamilyId.value, amount_cents: Math.round(Number(form.amount_yuan || 0) * 100) }
      let savedRecord: any
      if (editingRecordId.value) {
        savedRecord = await api(`/gift-records/${editingRecordId.value}`, { method: 'PUT', body: JSON.stringify(payload) })
        message.success('流水已更新')
      } else {
        savedRecord = await api('/gift-records', { method: 'POST', body: JSON.stringify(payload) })
        message.success('流水已保存')
      }
      await syncSavedRecordRows(savedRecord)
      await refreshCurrentRecordView()
      if (showContactDetail.value && savedRecord?.contact_id === contactDetail.value?.contact?.id) {
        await loadContactDetail(savedRecord.contact_id)
      }
      editingRecordId.value = null
      if (options.keepDraft) {
        clearQuickRecordTarget()
      } else if (form === recordDialogForm) {
        resetRecordDialogForm()
      } else {
        resetRecordForm()
      }
      showRecordModal.value = false
    } finally {
      recordSaving.value = false
    }
  }

  async function createQuickRecord() {
    editingRecordId.value = null
    await createRecord({ keepDraft: quickRecordKeepDraft.value, form: recordForm })
  }

  async function createDialogRecord() {
    await createRecord({ form: recordDialogForm })
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

  const debouncedContactsQuery = debounce(() => {
    if (activeView.value === 'contacts') loadContacts()
  }, 350)

  const debouncedFamiliesQuery = debounce(() => {
    if (activeView.value === 'families') loadFamilies()
  }, 250)

  const debouncedUsersQuery = debounce(() => {
    if (activeView.value === 'users') loadUsers()
  }, 250)

  const debouncedRecordsQuery = debounce(() => {
    if (activeView.value === 'records') loadRecords()
  }, 250)

  const debouncedAuditQuery = debounce(() => {
    if (activeView.value === 'audit') loadAudit()
  }, 250)

  const debouncedStatsQuery = debounce(() => {
    if (activeView.value === 'dashboard' || activeView.value === 'stats') applyStatsQuery()
  }, 250)

  const debouncedTodoQuery = debounce(() => {
    if (activeView.value === 'todos') applyTodoQuery()
  }, 250)

  const debouncedReceivableQuery = debounce(() => {
    if (activeView.value === 'receivables') applyReceivableQuery()
  }, 250)

  const debouncedPayableQuery = debounce(() => {
    if (activeView.value === 'payables') applyPayableQuery()
  }, 250)

  const debouncedNonReciprocalQuery = debounce(() => {
    if (activeView.value === 'nonreciprocal') applyNonReciprocalQuery()
  }, 250)

  function queryRecords() {
    recordQuery.page = 1
    debouncedRecordsQuery()
  }

  function queryContacts() {
    contactQuery.page = 1
    debouncedContactsQuery()
  }

  function queryFamilies() {
    familyListQuery.page = 1
    debouncedFamiliesQuery()
  }

  function resetFamilyQuery() {
    Object.assign(familyListQuery, { page: 1, page_size: 10, total: 0, keyword: '', status: null })
    loadFamilies()
  }

  function queryUsers() {
    userQuery.page = 1
    debouncedUsersQuery()
  }

  function resetUserQuery() {
    Object.assign(userQuery, { page: 1, page_size: 10, total: 0, keyword: '', role: null, status: null, family_id: null })
    loadUsers()
  }

  function resetContactQuery() {
    Object.assign(contactQuery, { page: 1, page_size: 10, total: 0, keyword: '', contact_type: null, relation_type: null, owner_user_id: null, status: null, has_owner: null, has_phone: null })
    loadContacts()
  }

  function resetRecordQuery() {
    Object.assign(recordQuery, { page: 1, page_size: 10, total: 0, contact_id: null, owner_user_id: null, contact_type: null, relation_type: null, direction: null, occasion_type: null, reciprocity_status: null, reciprocity_unlinked: null, remark: '', start_date: null, end_date: null })
    loadRecords()
  }

  function queryAudit() {
    auditQuery.page = 1
    debouncedAuditQuery()
  }

  function resetAuditQuery() {
    Object.assign(auditQuery, { page: 1, page_size: 10, total: 0, keyword: '', action: '', success: null })
    loadAudit()
  }

  function queryStats() {
    debouncedStatsQuery()
  }

  function resetStatsQuery() {
    Object.assign(statsQuery, { range: 'all', start_date: null, end_date: null })
    applyStatsQuery()
  }

  function queryTodos() {
    todoQuery.page = 1
    debouncedTodoQuery()
  }

  function resetTodoQuery() {
    Object.assign(todoQuery, { page: 1, page_size: 10, total: 0, range: 'all', start_date: null, end_date: null })
    applyTodoQuery()
  }

  function queryReceivables() {
    receivableQuery.page = 1
    debouncedReceivableQuery()
  }

  function resetReceivableQuery() {
    Object.assign(receivableQuery, { page: 1, page_size: 10, total: 0, range: 'all', start_date: null, end_date: null })
    applyReceivableQuery()
  }

  function queryPayables() {
    payableQuery.page = 1
    debouncedPayableQuery()
  }

  function resetPayableQuery() {
    Object.assign(payableQuery, { page: 1, page_size: 10, total: 0, range: 'all', start_date: null, end_date: null })
    applyPayableQuery()
  }

  function queryNonReciprocal() {
    nonReciprocalQuery.page = 1
    debouncedNonReciprocalQuery()
  }

  function resetNonReciprocalQuery() {
    Object.assign(nonReciprocalQuery, { page: 1, page_size: 10, total: 0, range: 'all', start_date: null, end_date: null })
    applyNonReciprocalQuery()
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

  async function loadContactDetail(id: number) {
    contactDetailLoading.value = true
    try {
      contactDetail.value = await api(`/contacts/${id}`)
    } finally {
      contactDetailLoading.value = false
    }
  }

  async function openContactDetail(row: any) {
    showContactDetail.value = true
    await loadContactDetail(row.id)
  }

  function closeContactDetail() {
    showContactDetail.value = false
    contactDetail.value = null
  }

  function openCreateRecordForContact(contact: any) {
    editingRecordId.value = null
    resetRecordDialogForm()
    recordDialogForm.contact_id = contact.id
    void loadRelatedRecords(recordDialogForm)
    showRecordModal.value = true
  }

  function openEditContact(row: any) {
    editingContactId.value = row.id
    Object.assign(contactForm, { name: row.name, owner_user_id: row.owner_user_id || null, contact_type: row.contact_type, relation_type: row.relation_type, phone: row.phone, address: row.address, remark: row.remark || '', status: row.status || 'enabled' })
    showContactModal.value = true
  }

  function closeContactModal() {
    showContactModal.value = false
    editingContactId.value = null
    Object.assign(contactForm, { name: '', owner_user_id: null, contact_type: '', relation_type: '', phone: '', address: '', remark: '', status: 'enabled' })
  }

  async function deleteContact(row: any) {
    await api(`/contacts/${row.id}`, { method: 'DELETE' })
    message.success('对象已删除')
    await Promise.all([loadContacts(), loadContactOptions(), loadRecords(), loadStats()])
  }

  function openCreateRecord() {
    editingRecordId.value = null
    resetRecordDialogForm()
    void loadRelatedRecords(recordDialogForm)
    showRecordModal.value = true
  }

  function openEditRecord(row: any) {
    editingRecordId.value = row.id
    hydratingRecordForm = true
    Object.assign(recordDialogForm, {
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
    nextTick(() => {
      hydratingRecordForm = false
    })
    void loadRelatedRecords(recordDialogForm)
    showRecordModal.value = true
  }

  function openReciprocityRecord(row: any) {
    editingRecordId.value = null
    const nextDirection = row.direction === 'in' ? 'out' : 'in'
    hydratingRecordForm = true
    Object.assign(recordDialogForm, {
      contact_id: row.contact_id,
      occasion_type: row.occasion_type,
      direction: nextDirection,
      amount_yuan: Number(row.amount_cents || 0) / 100,
      gift_description: '',
      occurred_on: new Date().toISOString().slice(0, 10),
      handled_by: user.value?.display_name || '',
      reciprocity_status: 'done',
      related_record_id: row.id,
      remark: ''
    })
    nextTick(() => {
      hydratingRecordForm = false
    })
    relatedRecords.value = [row]
    void loadRelatedRecords(recordDialogForm)
    showRecordModal.value = true
  }

  function closeRecordModal() {
    showRecordModal.value = false
    editingRecordId.value = null
    resetRecordDialogForm()
  }

  async function deleteRecord(row: any) {
    await api(`/gift-records/${row.id}`, { method: 'DELETE' })
    message.success('流水已删除')
    await refreshCurrentRecordView()
  }

  function resetRecordForm() {
    Object.assign(recordForm, defaultRecordFormValues())
  }

  function resetRecordDialogForm() {
    Object.assign(recordDialogForm, defaultRecordFormValues())
    relatedRecords.value = []
  }

  function clearQuickRecordTarget() {
    recordForm.contact_id = null
    recordForm.related_record_id = null
    recordForm.gift_description = ''
  }

  function setRecordDirection(direction: string) {
    const form = activeRecordForm()
    if (form.direction !== direction) {
      form.related_record_id = null
    }
    form.direction = direction
    void loadRelatedRecords(form)
  }

  function setRecordReciprocityStatus(status: string) {
    const form = activeRecordForm()
    if (status !== 'done') {
      form.related_record_id = null
    }
    form.reciprocity_status = status
    void loadRelatedRecords(form)
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

  function dictOptions(typeCode: string) {
    return dictionaryItems.value
      .filter((item) => item.type?.code === typeCode && item.enabled)
      .map((item) => ({ label: item.label, value: item.value }))
  }
  function dictionaryLabel(typeCode: string, value: string) {
    return dictionaryItems.value.find((item) => item.type?.code === typeCode && item.value === value)?.label || value || '-'
  }
  const {
    recordColumns,
    dashboardRecordColumns,
    familyColumns,
    userColumns,
    contactColumns,
    dictTypeColumns,
    dictItemColumns,
    backupJobColumns,
    importErrorColumns,
    importJobColumns,
    auditColumns
  } = createTableColumns(
    {
      openEditRecord,
      openReciprocityRecord,
      deleteRecord: (row) => runAction(() => deleteRecord(row)),
      openEditFamily,
      updateFamilyStatus: (row) => runAction(() => updateFamilyStatus(row)),
      openEditUser,
      openResetPassword,
      updateUserStatus: (row) => runAction(() => updateUserStatus(row)),
      openContactDetail: (row) => runAction(() => openContactDetail(row)),
      openEditContact,
      deleteContact: (row) => runAction(() => deleteContact(row)),
      openEditDictType,
      openEditDict,
      deleteDict: (row) => runAction(() => deleteDict(row)),
      executeImport: (id) => runAction(() => executeImport(id)),
      showImportJobErrors: (row) => runAction(() => showImportJobErrors(row))
    },
    {
      dictionaryLabel,
      canEditDictionaryItem,
      dictionaryItemReadonlyText
    }
  )

  watch(activeView, () => {
    scrollActiveMobileNavIntoView()
  })
  watch(() => menuItems.value.map((item) => item.key).join(','), scrollActiveMobileNavIntoView)
  watch(() => recordForm.contact_id, () => {
    if (hydratingRecordForm || showRecordModal.value) return
    recordForm.related_record_id = null
    void loadRelatedRecords(recordForm)
  })
  watch(() => recordDialogForm.contact_id, () => {
    if (hydratingRecordForm || !showRecordModal.value) return
    recordDialogForm.related_record_id = null
    void loadRelatedRecords(recordDialogForm)
  })
  async function bootApp() {
    await loadVersion()
    await loadMe()
  }
  onMounted(async () => {
    await bootApp()
    scrollActiveMobileNavIntoView()
  })

  return {
    activeView,
    actionSubmitting,
    activeViewCacheKey,
    adminUserOptions,
    appendStatsQueryParams,
    applyStatsQuery,
    appVersion,
    auditColumns,
    auditLogs,
    auditQuery,
    auditRetention,
    authReady,
    backupConfig,
    backupForm,
    backupJobColumns,
    backupJobs,
    backupQuery,
    bootApp,
    cacheFamilies,
    canEditDictionaryItem,
    changePassword,
    clearLoadedViews,
    closeContactModal,
    closeContactDetail,
    closeRecordModal,
    contactColumns,
    contactDetail,
    contactDetailLoading,
    contactForm,
    contactOptions,
    contactQuery,
    contacts,
    contactSaving,
    contactTypeOptions,
    copyGeneratedPassword,
    copyText,
    createContact,
    createDialogRecord,
    createDictionaryItem,
    createDictionaryType,
    createFamily,
    createQuickRecord,
    createRecord,
    createUser,
    currentFamilyId,
    debouncedAuditQuery,
    debouncedContactsQuery,
    debouncedLoadActiveView,
    debouncedNonReciprocalQuery,
    debouncedRecordsQuery,
    debouncedPayableQuery,
    debouncedReceivableQuery,
    debouncedStatsQuery,
    debouncedTodoQuery,
    deleteContact,
    deleteDict,
    deleteRecord,
    dictForm,
    dictionaryGroups,
    dictionaryItemReadonlyText,
    dictionaryItems,
    dictionaryLabel,
    dictionaryTypeOptions,
    dictionaryTypes,
    dictItemColumns,
    dictOptions,
    dictTypeColumns,
    dictTypeForm,
    dashboardRecords,
    downloadBackup,
    downloadImportTemplate,
    dryRunImport,
    editingContactId,
    editingDictId,
    editingDictTypeId,
    editingFamilyId,
    editingRecordId,
    editingUserId,
    executeImport,
    families,
    familyColumns,
    familyForm,
    familyMemberOptions,
    familyListQuery,
    familyRows,
    familyQuery,
    generatePassword,
    handleFamilyChange,
    handlerOptions,
    importErrorColumns,
    importErrors,
    importFile,
    importJobColumns,
    importJobs,
    importMode,
    importQuery,
    isImportJobFailed,
    isAdmin,
    isLoggedIn,
    isSuperAdmin,
    loadAudit,
    loadBackup,
    loadContactOptions,
    loadContacts,
    loadDashboardRecords,
    loadDictionaries,
    loadedViews,
    loadFamilies,
    loadImportErrors,
    loadImports,
    loading,
    loadInitialData,
    loadMe,
    loadNonReciprocal,
    loadRecords,
    loadPayables,
    loadReceivables,
    loadStats,
    loadTodos,
    loadUsers,
    loadVersion,
    loadView,
    login,
    loginForm,
    logout,
    menuItems,
    mobileNavRef,
    mustChangePassword,
    nonReciprocal,
    nonReciprocalQuery,
    occasionTypeOptions,
    openCreateContact,
    openCreateRecordForContact,
    openCreateDict,
    openCreateDictType,
    openCreateFamily,
    openCreateRecord,
    openCreateUser,
    openEditContact,
    openContactDetail,
    openEditDict,
    openEditDictType,
    openEditFamily,
    openEditRecord,
    openReciprocityRecord,
    openEditUser,
    openImportModal,
    openResetPassword,
    passwordForm,
    payables,
    payableQuery,
    persistSelectedFamilyId,
    queryAudit,
    queryContacts,
    queryFamilies,
    queryRecords,
    queryNonReciprocal,
    queryPayables,
    queryReceivables,
    queryStats,
    queryTodos,
    queryUsers,
    dashboardRecordColumns,
    quickRecordKeepDraft,
    recordColumns,
    recordDialogForm,
    recordForm,
    recordQuery,
    records,
    recordSaving,
    receivables,
    receivableQuery,
    refreshToken,
    relatedRecordOptions,
    relationTypeOptions,
    resetContactQuery,
    resetFamilyQuery,
    resetNonReciprocalQuery,
    resetAuditQuery,
    resetPasswordForm,
    resetPayableQuery,
    resetRecordQuery,
    resetReceivableQuery,
    resetRecordForm,
    resetStatsQuery,
    resetTodoQuery,
    resetUserQuery,
    resettingUser,
    resetUserPassword,
    runAction,
    saveBackupAndClose,
    saveBackupConfig,
    scrollActiveMobileNavIntoView,
    secureRandom,
    selectedFamilyId,
    setRecordDirection,
    setRecordReciprocityStatus,
    setPayableRange,
    setNonReciprocalRange,
    setReceivableRange,
    setStatsRange,
    setTodoRange,
    showBackupModal,
    showContactModal,
    showContactDetail,
    showDictModal,
    showDictTypeModal,
    showFamilyModal,
    showFullViewLoading,
    showImportJobErrors,
    showImportModal,
    showRecordModal,
    showResetPasswordModal,
    showUserModal,
    stats,
    statsQuery,
    todos,
    todoQuery,
    switchView,
    tableLoading,
    token,
    updateFamilyStatus,
    updateUserStatus,
    user,
    userColumns,
    userForm,
    userQuery,
    users,
    viewCacheKey,
    viewError,
    viewIcons,
    viewLoading,
    writableDictionaryTypes
  }
}
