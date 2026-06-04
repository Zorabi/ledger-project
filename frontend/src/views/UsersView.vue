<script setup lang="ts">
import { Plus, RotateCcw, Search } from 'lucide-vue-next'
import MobilePager from '../components/MobilePager.vue'

defineProps<{
  query: any
  users: any[]
  columns: any[]
  loading: boolean
  families: any[]
  isSuperAdmin: boolean
  statusText: (value: string | boolean) => string
  formatDateTime: (value?: string) => string
}>()

const emit = defineEmits<{
  query: []
  resetQuery: []
  load: []
  create: []
  edit: [user: any]
  resetPassword: [user: any]
  updateStatus: [user: any]
}>()

function reloadFromFirstPage(query: any) {
  query.page = 1
  emit('load')
}
</script>

<template>
  <section class="view-stack">
    <n-card title="筛选条件" size="small">
      <div class="filter-bar">
        <div class="filter-fields">
          <n-input v-model:value="query.keyword" clearable placeholder="搜索账号、姓名或邮箱" @keydown.enter="emit('query')" />
          <n-select
            v-model:value="query.role"
            clearable
            :options="[
              ...(isSuperAdmin ? [{ label: '系统管理员', value: 'super_admin' }] : []),
              { label: '家庭管理员', value: 'family_admin' },
              { label: '家庭成员', value: 'member' }
            ]"
            placeholder="角色"
          />
          <n-select
            v-model:value="query.status"
            clearable
            :options="[
              { label: '启用', value: 'enabled' },
              { label: '禁用', value: 'disabled' }
            ]"
            placeholder="状态"
          />
          <n-select
            v-if="isSuperAdmin"
            v-model:value="query.family_id"
            clearable
            filterable
            :options="families.map((item) => ({ label: item.name, value: item.id }))"
            placeholder="所属家庭"
          />
        </div>
        <div class="filter-actions">
          <n-button type="primary" class="query-button" @click="emit('query')"><template #icon><Search :size="16" /></template>查询</n-button>
          <n-button secondary class="filter-action-button" @click="emit('resetQuery')"><template #icon><RotateCcw :size="16" /></template>重置</n-button>
        </div>
      </div>
    </n-card>

    <n-card title="用户列表" size="small">
      <template #header-extra>
        <n-button type="primary" size="small" @click="emit('create')">
          <template #icon><Plus :size="16" /></template>
          新增用户
        </n-button>
      </template>
      <n-data-table class="desktop-table" :data="users" :columns="columns" :loading="loading" />
      <div class="mobile-list">
        <article v-for="item in users" :key="item.id" class="mobile-item">
          <div class="item-main"><strong>{{ item.display_name }}</strong><n-tag size="small">{{ statusText(item.role) }}</n-tag></div>
          <p>{{ item.username }} · {{ item.email || '未填写邮箱' }}</p>
          <small>{{ statusText(item.status) }} · 最后登录 {{ formatDateTime(item.last_login_at) }}</small>
          <div class="item-actions">
            <n-button size="small" secondary @click="emit('edit', item)">编辑</n-button>
            <n-button size="small" secondary type="primary" @click="emit('resetPassword', item)">重置密码</n-button>
            <n-button size="small" secondary @click="emit('updateStatus', item)">{{ item.status === 'enabled' ? '禁用' : '启用' }}</n-button>
          </div>
        </article>
      </div>
      <n-pagination
        v-model:page="query.page"
        v-model:page-size="query.page_size"
        :item-count="query.total"
        :page-sizes="[10, 20, 50]"
        show-size-picker
        class="pager desktop-pager"
        @update:page="emit('load')"
        @update:page-size="() => reloadFromFirstPage(query)"
      />
      <MobilePager :query="query" @load="emit('load')" />
    </n-card>
  </section>
</template>
