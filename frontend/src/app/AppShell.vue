<script setup lang="ts">
import { KeyRound, LogOut, Upload } from 'lucide-vue-next'
import { dateZhCN, zhCN } from 'naive-ui'
import { actionText, resourceText } from '../entities/audit/model'
import { boolText, directionText, formatDate, formatDateTime, money, moneyCompact, reciprocityRoleText, statusText } from '../shared/lib/formatters'
import { useLedgerApp } from './useLedgerApp'
import type { ViewKey } from '../shared/types'
import AuditView from '../views/AuditView.vue'
import BackupView from '../views/BackupView.vue'
import ContactsView from '../views/ContactsView.vue'
import DashboardView from '../views/DashboardView.vue'
import DictionariesView from '../views/DictionariesView.vue'
import FamiliesView from '../views/FamiliesView.vue'
import ImportsView from '../views/ImportsView.vue'
import RecordsView from '../views/RecordsView.vue'
import ReceivablesView from '../views/ReceivablesView.vue'
import StatsView from '../views/StatsView.vue'
import UsersView from '../views/UsersView.vue'

const {
  activeView,
  activeViewCacheKey,
  actionSubmitting,
  adminUserOptions,
  appendStatsQueryParams,
  applyStatsQuery,
  appVersion,
  auditColumns,
  auditLogs,
  auditQuery,
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
  contactColumns,
  contactForm,
  contactOptions,
  contactQuery,
  contacts,
  contactSaving,
  contactTypeOptions,
  copyGeneratedPassword,
  copyText,
  createContact,
  createDictionaryItem,
  createDictionaryType,
  createFamily,
  createQuickRecord,
  createRecord,
  createUser,
  currentFamilyId,
  dashboardRecords,
  debouncedAuditQuery,
  debouncedContactsQuery,
  debouncedLoadActiveView,
  debouncedRecordsQuery,
  debouncedPayableQuery,
  debouncedReceivableQuery,
  debouncedStatsQuery,
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
  familyListQuery,
  familyMemberOptions,
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
  loadRecords,
  loadPayables,
  loadReceivables,
  loadStats,
  loadUsers,
  loadVersion,
  loadView,
  login,
  loginForm,
  logout,
  menuItems,
  mobileNavRef,
  mustChangePassword,
  occasionTypeOptions,
  openCreateContact,
  openCreateDict,
  openCreateDictType,
  openCreateFamily,
  openCreateRecord,
  openCreateUser,
  openEditContact,
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
  queryPayables,
  queryReceivables,
  queryStats,
  queryUsers,
  dashboardRecordColumns,
  quickRecordKeepDraft,
  recordColumns,
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
  resetAuditQuery,
  resetPasswordForm,
  resetPayableQuery,
  resetRecordQuery,
  resetReceivableQuery,
  resetRecordForm,
  resetStatsQuery,
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
  setReceivableRange,
  setStatsRange,
  showBackupModal,
  showContactModal,
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
} = useLedgerApp()
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
            <n-button type="primary" block :loading="loading" @click="changePassword">完成修改并重新登录</n-button>
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
          <header class="topbar topbar-compact">
            <div>
              <p class="eyebrow">{{ user?.family?.name || '系统管理' }}</p>
              <h1>{{ menuItems.find((item) => item.key === activeView)?.label }}</h1>
            </div>
          </header>
          <div class="refresh-line" :class="{ active: viewLoading }" aria-hidden="true"></div>
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
            :contacts="contactOptions"
            :records="dashboardRecords"
            :occasion-type-options="occasionTypeOptions"
            :handler-options="handlerOptions"
            :related-record-options="relatedRecordOptions"
            :record-columns="dashboardRecordColumns"
            :records-loading="Boolean(tableLoading.dashboardRecords || tableLoading.records)"
            :saving="recordSaving || actionSubmitting"
            :quick-record-keep-draft="quickRecordKeepDraft"
            :money="money"
            :dictionary-label="dictionaryLabel"
            :direction-text="directionText"
            :status-text="statusText"
            :format-date="formatDate"
            :reciprocity-role-text="reciprocityRoleText"
            @query="queryStats"
            @reset="resetStatsQuery"
            @set-range="setStatsRange"
            @set-record-direction="setRecordDirection"
            @set-record-reciprocity-status="setRecordReciprocityStatus"
            @update-quick-record-keep-draft="quickRecordKeepDraft = $event"
            @save-record="runAction(createQuickRecord)"
            @edit-record="openEditRecord"
            @reciprocity-record="openReciprocityRecord"
            @delete-record="(record) => runAction(() => deleteRecord(record))"
          />

          <FamiliesView
            v-show="activeView === 'families'"
            key="families-view"
            :query="familyListQuery"
            :families="familyRows"
            :columns="familyColumns"
            :loading="Boolean(tableLoading.families)"
            :status-text="statusText"
            :format-date-time="formatDateTime"
            @query="queryFamilies"
            @reset-query="resetFamilyQuery"
            @load="loadFamilies"
            @create="openCreateFamily"
            @edit="openEditFamily"
            @update-status="(family) => runAction(() => updateFamilyStatus(family))"
          />

          <UsersView
            v-show="activeView === 'users'"
            key="users-view"
            :query="userQuery"
            :users="users"
            :columns="userColumns"
            :loading="Boolean(tableLoading.users)"
            :families="families"
            :is-super-admin="isSuperAdmin"
            :status-text="statusText"
            :format-date-time="formatDateTime"
            @query="queryUsers"
            @reset-query="resetUserQuery"
            @load="loadUsers"
            @create="openCreateUser"
            @edit="openEditUser"
            @reset-password="openResetPassword"
            @update-status="(item) => runAction(() => updateUserStatus(item))"
          />

          <ContactsView
            v-show="activeView === 'contacts'"
            key="contacts-view"
            :query="contactQuery"
            :contacts="contacts"
            :columns="contactColumns"
            :loading="Boolean(tableLoading.contacts)"
            :contact-type-options="contactTypeOptions"
            :relation-type-options="relationTypeOptions"
            :owner-options="familyMemberOptions"
            :dictionary-label="dictionaryLabel"
            @query="queryContacts"
            @reset-query="resetContactQuery"
            @load="loadContacts"
            @create="openCreateContact"
            @edit="openEditContact"
            @delete="(contact) => runAction(() => deleteContact(contact))"
          />

          <RecordsView
            v-show="activeView === 'records'"
            key="records-view"
            :query="recordQuery"
            :contacts="contactOptions"
            :records="records"
            :columns="recordColumns"
            :loading="Boolean(tableLoading.records)"
            :family-member-options="familyMemberOptions"
            :contact-type-options="contactTypeOptions"
            :relation-type-options="relationTypeOptions"
            :occasion-type-options="occasionTypeOptions"
            :dictionary-label="dictionaryLabel"
            :direction-text="directionText"
            :status-text="statusText"
            :money="money"
            :format-date="formatDate"
            :reciprocity-role-text="reciprocityRoleText"
            @query-records="queryRecords"
            @reset-records="resetRecordQuery"
            @load-records="loadRecords"
            @create="openCreateRecord"
            @edit="openEditRecord"
            @reciprocity="openReciprocityRecord"
            @delete="(record) => runAction(() => deleteRecord(record))"
          />

          <DictionariesView
            v-show="activeView === 'dictionaries'"
            key="dictionaries-view"
            :is-super-admin="isSuperAdmin"
            :dictionary-types="dictionaryTypes"
            :dictionary-groups="dictionaryGroups"
            :type-columns="dictTypeColumns"
            :item-columns="dictItemColumns"
            :loading="Boolean(tableLoading.dictionaries)"
            :writable-dictionary-types="writableDictionaryTypes"
            :status-text="statusText"
            :bool-text="boolText"
            :format-date-time="formatDateTime"
            :can-edit-dictionary-item="canEditDictionaryItem"
            @create-type="openCreateDictType"
            @create-item="openCreateDict"
            @edit-type="openEditDictType"
            @edit-item="openEditDict"
            @delete-item="(item) => runAction(() => deleteDict(item))"
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
            @reset="resetStatsQuery"
            @set-range="setStatsRange"
          />

          <ReceivablesView
            v-show="activeView === 'receivables'"
            key="receivables-view"
            :receivables="receivables"
            :query="receivableQuery"
            :loading="Boolean(tableLoading.receivables)"
            amount-label="待回收金额"
            contact-label="待回收对象"
            record-label="待回收笔数"
            top-title="待回收对象 Top 5"
            top-empty-text="暂无待回收对象"
            owner-title="归属成员待回收"
            relation-title="关系类型待回收"
            detail-title="待回收明细"
            detail-empty-text="暂无待回收明细"
            :money="money"
            :dictionary-label="dictionaryLabel"
            :format-date="formatDate"
            @query="queryReceivables"
            @reset="resetReceivableQuery"
            @load="loadReceivables"
            @set-range="setReceivableRange"
          />

          <ReceivablesView
            v-show="activeView === 'payables'"
            key="payables-view"
            :receivables="payables"
            :query="payableQuery"
            :loading="Boolean(tableLoading.payables)"
            amount-label="待送出金额"
            contact-label="待送出对象"
            record-label="待送出笔数"
            top-title="待送出对象 Top 5"
            top-empty-text="暂无待送出对象"
            owner-title="归属成员待送出"
            relation-title="关系类型待送出"
            detail-title="待送出明细"
            detail-empty-text="暂无待送出明细"
            :money="money"
            :dictionary-label="dictionaryLabel"
            :format-date="formatDate"
            @query="queryPayables"
            @reset="resetPayableQuery"
            @load="loadPayables"
            @set-range="setPayableRange"
          />

          <BackupView
            v-show="activeView === 'backup'"
            key="backup-view"
            :form="backupForm"
            :query="backupQuery"
            :jobs="backupJobs"
            :columns="backupJobColumns"
            :loading="Boolean(tableLoading.backup)"
            :status-text="statusText"
            :format-date-time="formatDateTime"
            @load="loadBackup"
            @edit="showBackupModal = true"
            @download="runAction(() => downloadBackup(false))"
            @send-email="runAction(() => downloadBackup(true))"
          />

          <ImportsView
            v-show="activeView === 'imports'"
            key="imports-view"
            :query="importQuery"
            :errors="importErrors"
            :jobs="importJobs"
            :error-columns="importErrorColumns"
            :job-columns="importJobColumns"
            :loading="Boolean(tableLoading.imports)"
            :status-text="statusText"
            :format-date-time="formatDateTime"
            :is-job-failed="isImportJobFailed"
            @download-template="runAction(downloadImportTemplate)"
            @open-manual="openImportModal('manual')"
            @open-backup="openImportModal('backup')"
            @load="loadImports"
            @execute-import="(jobId) => runAction(() => executeImport(jobId))"
            @show-errors="(job) => runAction(() => showImportJobErrors(job))"
          />

          <AuditView
            v-show="activeView === 'audit'"
            key="audit-view"
            :query="auditQuery"
            :logs="auditLogs"
            :columns="auditColumns"
            :loading="Boolean(tableLoading.audit)"
            :action-text="actionText"
            :status-text="statusText"
            :resource-text="resourceText"
            :format-date-time="formatDateTime"
            @query-audit="queryAudit"
            @reset-audit="resetAuditQuery"
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
              <n-button type="primary" block :loading="actionSubmitting" :disabled="actionSubmitting" @click="runAction(createFamily)">保存家庭</n-button>
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
              <n-button type="primary" block :loading="actionSubmitting" :disabled="actionSubmitting" @click="runAction(createUser)">保存用户</n-button>
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
              <n-button type="primary" block :loading="actionSubmitting" :disabled="actionSubmitting" @click="runAction(resetUserPassword)">确认重置</n-button>
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
              <n-button type="primary" block :loading="contactSaving || actionSubmitting" :disabled="contactSaving || actionSubmitting" @click="runAction(createContact)">保存对象</n-button>
            </n-form>
          </n-modal>

          <n-modal v-model:show="showRecordModal" preset="card" :title="editingRecordId ? '编辑往来流水' : '新增往来流水'" class="form-modal">
            <n-form label-placement="top">
              <n-form-item label="往来对象"><n-select v-model:value="recordForm.contact_id" filterable :options="contactOptions.map((item) => ({ label: item.name, value: item.id }))" placeholder="请选择往来对象" /></n-form-item>
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
                <n-form-item label="金额"><n-input-number v-model:value="recordForm.amount_yuan" :min="0" :step="100" placeholder="请输入金额（元）" /></n-form-item>
                <n-form-item label="发生日期"><n-date-picker v-model:formatted-value="recordForm.occurred_on" value-format="yyyy-MM-dd" type="date" :clearable="false" /></n-form-item>
              </div>
              <n-form-item label="经办人"><n-select v-model:value="recordForm.handled_by" clearable filterable tag :options="handlerOptions" placeholder="请选择或输入经办人" /></n-form-item>
              <n-form-item label="回礼状态"><n-select :value="recordForm.reciprocity_status" :options="[{ label: '无需回礼', value: 'none' }, { label: '待回礼', value: 'pending' }, { label: '已回礼', value: 'done' }]" placeholder="请选择回礼状态" @update:value="setRecordReciprocityStatus" /></n-form-item>
              <n-form-item v-if="recordForm.reciprocity_status === 'done'" :label="`关联${reciprocityRoleText(recordForm.direction)}流水`">
                <n-select v-model:value="recordForm.related_record_id" clearable filterable :options="relatedRecordOptions" :placeholder="`请选择${reciprocityRoleText(recordForm.direction)}对应的反向流水`" />
              </n-form-item>
              <n-form-item label="礼品说明"><n-input v-model:value="recordForm.gift_description" type="textarea" placeholder="请输入礼品名称、数量或其他说明" /></n-form-item>
              <n-form-item label="备注"><n-input v-model:value="recordForm.remark" type="textarea" placeholder="请输入备注" /></n-form-item>
              <n-button type="primary" block :loading="recordSaving || actionSubmitting" :disabled="recordSaving || actionSubmitting" @click="runAction(createRecord)">保存流水</n-button>
            </n-form>
          </n-modal>

          <n-modal v-model:show="showDictModal" preset="card" :title="editingDictId ? '编辑字典项' : '新增字典项'" class="form-modal">
            <n-form label-placement="top">
              <n-form-item label="字典类型"><n-select v-model:value="dictForm.type_id" :disabled="Boolean(editingDictId)" :options="dictionaryTypeOptions" placeholder="请选择字典类型" /></n-form-item>
              <n-form-item label="标签"><n-input v-model:value="dictForm.label" placeholder="请输入显示名称" /></n-form-item>
              <n-form-item label="值"><n-input v-model:value="dictForm.value" placeholder="请输入实际保存值，系统会按原样保存" /></n-form-item>
              <n-form-item label="排序"><n-input-number v-model:value="dictForm.sort_order" placeholder="请输入排序值" /></n-form-item>
              <n-button type="primary" block :loading="actionSubmitting" :disabled="actionSubmitting" @click="runAction(createDictionaryItem)">保存字典项</n-button>
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
              <n-button type="primary" block :loading="actionSubmitting" :disabled="actionSubmitting" @click="runAction(saveBackupAndClose)">保存配置</n-button>
            </n-form>
          </n-modal>

          <n-modal v-model:show="showImportModal" preset="card" :title="importMode === 'manual' ? '手工导入' : '备份恢复'" class="form-modal">
            <p v-if="importMode === 'manual'" class="modal-hint">适合普通录入，只需要填写往来流水和可选的往来对象信息。系统会先校验，不会立即写入数据。</p>
            <p v-else class="modal-hint">仅用于恢复系统导出的完整备份文件，会识别编号、字典和家庭成员等高级字段。系统会先校验，不会立即写入数据。</p>
            <n-upload :max="1" accept=".xlsx" @update:file-list="(files) => importFile = files[0] || null">
              <n-button><template #icon><Upload :size="17" /></template>选择 Excel 文件</n-button>
            </n-upload>
            <div class="modal-actions">
              <n-button type="primary" :loading="actionSubmitting" :disabled="actionSubmitting" @click="runAction(dryRunImport)">导入校验</n-button>
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
              <n-button type="primary" block :loading="actionSubmitting" :disabled="actionSubmitting" @click="runAction(createDictionaryType)">保存字典类型</n-button>
            </n-form>
          </n-modal>
        </section>
      </main>
    </n-message-provider>
  </n-config-provider>
</template>
