<script setup lang="ts">
import { RotateCcw, Search } from 'lucide-vue-next'
import MobilePager from '../components/MobilePager.vue'

defineProps<{
  query: any
  logs: any[]
  columns: any[]
  loading: boolean
  actionText: (value: string) => string
  statusText: (value: string | boolean) => string
  resourceText: (value: string) => string
  formatDateTime: (value?: string) => string
}>()

const emit = defineEmits<{
  queryAudit: []
  resetAudit: []
  loadAudit: []
}>()

function reloadFromFirstPage(query: any) {
  query.page = 1
  emit('loadAudit')
}
</script>

<template>
  <section class="view-stack">
    <n-card title="审计日志" size="small">
      <div class="filter-bar">
        <div class="filter-fields audit-filter">
          <n-input v-model:value="query.keyword" clearable placeholder="搜索用户、动作、资源或 IP" />
          <n-select v-model:value="query.success" clearable :options="[{ label: '成功', value: true }, { label: '失败', value: false }]" placeholder="筛选结果" />
        </div>
        <div class="filter-actions">
          <n-button type="primary" class="query-button" @click="emit('queryAudit')"><template #icon><Search :size="16" /></template>查询</n-button>
          <n-button secondary class="filter-action-button" @click="emit('resetAudit')"><template #icon><RotateCcw :size="16" /></template>重置</n-button>
        </div>
      </div>
      <n-data-table class="desktop-table" :data="logs" :columns="columns" :loading="loading" />
      <div class="mobile-list">
        <article v-for="log in logs" :key="log.id" class="mobile-item">
          <div class="item-main"><strong>{{ actionText(log.action) }}</strong><n-tag size="small">{{ statusText(log.success) }}</n-tag></div>
          <p>{{ log.operator?.display_name || log.operator?.username || '未知用户' }} · {{ resourceText(log.resource_type) }} · {{ log.ip || '未知 IP' }}</p>
          <small>{{ formatDateTime(log.created_at) }}</small>
        </article>
      </div>
      <n-pagination
        v-model:page="query.page"
        v-model:page-size="query.page_size"
        :item-count="query.total"
        :page-sizes="[10, 20, 50]"
        show-size-picker
        class="pager desktop-pager"
        @update:page="emit('loadAudit')"
        @update:page-size="() => reloadFromFirstPage(query)"
      />
      <MobilePager :query="query" @load="emit('loadAudit')" />
    </n-card>
  </section>
</template>
