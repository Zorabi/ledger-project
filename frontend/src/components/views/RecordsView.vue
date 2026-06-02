<script setup lang="ts">
import { Plus, Search } from 'lucide-vue-next'

defineProps<{
  query: any
  contacts: any[]
  records: any[]
  columns: any[]
  occasionTypeOptions: any[]
  dictionaryLabel: (typeCode: string, value: string) => string
  directionText: (value: string) => string
  statusText: (value: string | boolean) => string
  money: (cents?: number) => string
  formatDate: (value?: string) => string
  reciprocityRoleText: (direction: string) => string
}>()

const emit = defineEmits<{
  queryRecords: []
  loadRecords: []
  create: []
  edit: [record: any]
  delete: [record: any]
}>()

function reloadFromFirstPage(query: any) {
  query.page = 1
  emit('loadRecords')
}
</script>

<template>
  <section class="view-stack">
    <n-card title="礼尚往来流水" size="small">
      <template #header-extra>
        <n-button type="primary" size="small" @click="emit('create')">
          <template #icon><Plus :size="16" /></template>
          新增流水
        </n-button>
      </template>
      <div class="filter-grid">
        <n-select v-model:value="query.contact_id" clearable filterable :options="contacts.map((item) => ({ label: item.name, value: item.id }))" placeholder="筛选往来对象" />
        <n-select v-model:value="query.occasion_type" clearable :options="occasionTypeOptions" placeholder="筛选事项类型" />
        <n-select v-model:value="query.direction" clearable :options="[{ label: '我家收礼', value: 'in' }, { label: '我家送礼', value: 'out' }]" placeholder="筛选方向" />
        <n-select v-model:value="query.reciprocity_status" clearable :options="[{ label: '无需回礼', value: 'none' }, { label: '待回礼', value: 'pending' }, { label: '已回礼', value: 'done' }]" placeholder="筛选回礼状态" />
        <n-date-picker v-model:formatted-value="query.start_date" value-format="yyyy-MM-dd" type="date" placeholder="开始日期" />
        <n-date-picker v-model:formatted-value="query.end_date" value-format="yyyy-MM-dd" type="date" placeholder="结束日期" />
        <n-button type="primary" class="query-button" @click="emit('queryRecords')">
          <template #icon><Search :size="16" /></template>
          查询
        </n-button>
      </div>
      <n-data-table class="desktop-table" :columns="columns" :data="records" />
      <div class="mobile-list">
        <article v-for="record in records" :key="record.id" class="mobile-item">
          <div class="item-main"><strong>{{ record.contact?.name || record.contact_id }}</strong><n-tag size="small" :type="record.direction === 'in' ? 'success' : 'warning'">{{ directionText(record.direction) }}</n-tag></div>
          <p>{{ dictionaryLabel('occasion_type', record.occasion_type) }} · {{ money(record.amount_cents) }}</p>
          <small>{{ formatDate(record.occurred_on) }} · {{ statusText(record.reciprocity_status) }}<template v-if="record.reciprocity_status === 'done'">（{{ reciprocityRoleText(record.direction) }}）</template></small>
          <div class="item-actions">
            <n-button size="small" secondary @click="emit('edit', record)">编辑</n-button>
            <n-popconfirm @positive-click="emit('delete', record)">
              <template #trigger><n-button size="small" secondary type="error">删除</n-button></template>
              确认删除这条流水？
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
        @update:page="emit('loadRecords')"
        @update:page-size="() => reloadFromFirstPage(query)"
      />
    </n-card>
  </section>
</template>
