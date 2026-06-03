<script setup lang="ts">
import { Plus, RotateCcw, Search } from 'lucide-vue-next'

defineProps<{
  query: any
  contacts: any[]
  columns: any[]
  loading: boolean
  contactTypeOptions: any[]
  relationTypeOptions: any[]
  ownerOptions: any[]
  dictionaryLabel: (typeCode: string, value: string) => string
}>()

const emit = defineEmits<{
  query: []
  resetQuery: []
  load: []
  create: []
  edit: [contact: any]
  delete: [contact: any]
}>()
</script>

<template>
  <section class="view-stack">
    <n-card title="筛选条件" size="small">
      <div class="filter-bar">
        <div class="filter-fields contact-filter">
          <n-input v-model:value="query.keyword" clearable placeholder="搜索姓名、手机号、地址或备注" @keydown.enter="emit('query')" />
          <n-select v-model:value="query.contact_type" clearable filterable :options="contactTypeOptions" placeholder="对象类型" />
          <n-select v-model:value="query.relation_type" clearable filterable :options="relationTypeOptions" placeholder="关系类型" />
          <n-select v-model:value="query.owner_user_id" clearable filterable :options="ownerOptions" placeholder="归属成员" />
          <n-select
            v-model:value="query.has_owner"
            clearable
            :options="[
              { label: '已设置归属', value: true },
              { label: '未设置归属', value: false }
            ]"
            placeholder="归属状态"
          />
          <n-select
            v-model:value="query.has_phone"
            clearable
            :options="[
              { label: '已填写手机号', value: true },
              { label: '未填写手机号', value: false }
            ]"
            placeholder="手机号状态"
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
        </div>
        <div class="filter-actions">
          <n-button type="primary" class="query-button" @click="emit('query')"><template #icon><Search :size="16" /></template>查询</n-button>
          <n-button secondary class="filter-action-button" @click="emit('resetQuery')"><template #icon><RotateCcw :size="16" /></template>重置</n-button>
        </div>
      </div>
    </n-card>

    <n-card title="对象列表" size="small">
      <template #header-extra>
        <n-button type="primary" size="small" @click="emit('create')">
          <template #icon><Plus :size="16" /></template>
          新增对象
        </n-button>
      </template>
      <n-data-table class="desktop-table" :data="contacts" :columns="columns" :loading="loading" />
      <div class="mobile-list">
        <article v-for="item in contacts" :key="item.id" class="mobile-item">
          <div class="item-main"><strong>{{ item.name }}</strong><n-tag size="small">{{ dictionaryLabel('relation_type', item.relation_type) }}</n-tag></div>
          <p>{{ dictionaryLabel('contact_type', item.contact_type) }} · {{ item.owner_user?.display_name || item.owner_user?.username || '未指定归属成员' }}</p>
          <p>{{ item.phone || '未填写电话' }}</p>
          <small>{{ item.address || '未填写地址' }}</small>
          <div class="item-actions">
            <n-button size="small" secondary @click="emit('edit', item)">编辑</n-button>
            <n-popconfirm @positive-click="emit('delete', item)">
              <template #trigger><n-button size="small" secondary type="error">删除</n-button></template>
              确认删除该对象？
            </n-popconfirm>
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
        @update:page-size="() => { query.page = 1; emit('load') }"
      />
    </n-card>
  </section>
</template>
