<script setup lang="ts">
import { Plus, RotateCcw, Search } from 'lucide-vue-next'

defineProps<{
  query: any
  families: any[]
  columns: any[]
  loading: boolean
  statusText: (value: string | boolean) => string
  formatDateTime: (value?: string) => string
}>()

const emit = defineEmits<{
  query: []
  resetQuery: []
  load: []
  create: []
  edit: [family: any]
  updateStatus: [family: any]
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
        <div class="filter-fields audit-filter">
          <n-input v-model:value="query.keyword" clearable placeholder="搜索家庭、联系人、邮箱或备注" @keydown.enter="emit('query')" />
          <n-select
            v-model:value="query.status"
            clearable
            :options="[
              { label: '启用', value: 'enabled' },
              { label: '禁用', value: 'disabled' }
            ]"
            placeholder="状态"
          />
        </div>
        <div class="filter-actions">
          <n-button type="primary" class="query-button" @click="emit('query')"><template #icon><Search :size="16" /></template>查询</n-button>
          <n-button secondary class="filter-action-button" @click="emit('resetQuery')"><template #icon><RotateCcw :size="16" /></template>重置</n-button>
        </div>
      </div>
    </n-card>

    <n-card title="家庭列表" size="small">
      <template #header-extra>
        <n-button type="primary" size="small" @click="emit('create')">
          <template #icon><Plus :size="16" /></template>
          新增家庭
        </n-button>
      </template>
      <n-data-table class="desktop-table" :data="families" :columns="columns" :loading="loading" />
      <div class="mobile-list">
        <article v-for="family in families" :key="family.id" class="mobile-item">
          <div class="item-main"><strong>{{ family.name }}</strong><n-tag size="small">{{ statusText(family.status) }}</n-tag></div>
          <p>{{ family.contact_name || '未填写联系人' }} · {{ family.contact_email || '未填写邮箱' }}</p>
          <small>{{ formatDateTime(family.created_at) }}</small>
          <div class="item-actions">
            <n-button size="small" secondary @click="emit('edit', family)">编辑</n-button>
            <n-button size="small" secondary @click="emit('updateStatus', family)">{{ family.status === 'enabled' ? '禁用' : '启用' }}</n-button>
          </div>
        </article>
      </div>
      <n-pagination
        v-model:page="query.page"
        v-model:page-size="query.page_size"
        :item-count="query.total"
        :page-sizes="[10, 20, 50]"
        show-size-picker
        class="pager"
        @update:page="emit('load')"
        @update:page-size="() => reloadFromFirstPage(query)"
      />
    </n-card>
  </section>
</template>
