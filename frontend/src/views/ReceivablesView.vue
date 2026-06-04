<script setup lang="ts">
import { RotateCcw, Search } from 'lucide-vue-next'

const props = defineProps<{
  receivables: any
  query: any
  loading: boolean
  amountLabel: string
  contactLabel: string
  recordLabel: string
  topTitle: string
  topEmptyText: string
  ownerTitle: string
  relationTitle: string
  detailTitle: string
  detailEmptyText: string
  money: (cents?: number) => string
  dictionaryLabel: (typeCode: string, value: string) => string
  formatDate: (value?: string) => string
}>()

const emit = defineEmits<{
  query: []
  reset: []
  load: []
  setRange: [range: string]
}>()

const agingOrder = ['30天内', '31-90天', '91-180天', '181-365天', '1年以上']

function orderedAgingStats(stats: any[] = []) {
  return agingOrder
    .map((bucket) => stats.find((item: any) => item.bucket === bucket) || { bucket, amount_cents: 0, record_count: 0 })
}

function waitText(days?: number) {
  const value = Number(days || 0)
  if (value <= 0) return '今天'
  return `${value} 天`
}

function remarkText(row: any) {
  return row?.remark || row?.gift_description || '未填写'
}

function occasionTypesText(item: any) {
  const values = Array.isArray(item?.occasion_types) ? item.occasion_types : []
  if (!values.length) return '未指定'
  return values.map((value: string) => props.dictionaryLabel('occasion_type', value)).join('、')
}

function reloadFromFirstPage(query: any) {
  query.page = 1
  emit('load')
}
</script>

<template>
  <section class="view-stack receivable-view">
    <n-card title="时间范围" size="small">
      <div class="filter-bar">
        <div class="filter-fields time-filter">
          <n-select
            :value="query.range"
            :options="[
              { label: '全部时间', value: 'all' },
              { label: '本月', value: 'month' },
              { label: '本年', value: 'year' },
              { label: '近30天', value: 'last30' },
              { label: '自定义', value: 'custom' }
            ]"
            placeholder="选择时间维度"
            @update:value="(value: string) => emit('setRange', value)"
          />
          <n-date-picker v-model:formatted-value="query.start_date" value-format="yyyy-MM-dd" type="date" placeholder="开始日期" @update:formatted-value="query.range = 'custom'" />
          <n-date-picker v-model:formatted-value="query.end_date" value-format="yyyy-MM-dd" type="date" placeholder="结束日期" @update:formatted-value="query.range = 'custom'" />
        </div>
        <div class="filter-actions">
          <n-button type="primary" class="query-button" @click="emit('query')"><template #icon><Search :size="16" /></template>查询</n-button>
          <n-button secondary class="filter-action-button" @click="emit('reset')"><template #icon><RotateCcw :size="16" /></template>重置</n-button>
        </div>
      </div>
    </n-card>

    <div class="metrics">
      <article><span>{{ amountLabel }}</span><strong>{{ money(receivables?.cards?.amount_cents) }}</strong></article>
      <article><span>{{ contactLabel }}</span><strong>{{ receivables?.cards?.contact_count || 0 }}</strong></article>
      <article><span>{{ recordLabel }}</span><strong>{{ receivables?.cards?.record_count || 0 }}</strong></article>
      <article><span>最久等待</span><strong>{{ waitText(receivables?.cards?.max_wait_days) }}</strong></article>
    </div>

    <div class="analysis-grid">
      <n-card :title="topTitle" size="small">
        <div v-if="receivables?.contact_top?.length" class="rank-list">
          <article v-for="(item, index) in receivables.contact_top" :key="item.contact_id">
            <b>{{ index + 1 }}</b>
            <div>
              <strong>{{ item.name }}</strong>
              <span>{{ item.owner_name }} · {{ dictionaryLabel('relation_type', item.relation_type) }} · {{ occasionTypesText(item) }} · 等待 {{ waitText(item.wait_days) }}</span>
            </div>
            <em>{{ money(item.amount_cents) }}</em>
          </article>
        </div>
        <n-empty v-else :description="topEmptyText" />
      </n-card>

      <n-card title="账龄分布" size="small">
        <div v-if="receivables?.aging_stats?.length" class="metric-bars">
          <article v-for="item in orderedAgingStats(receivables.aging_stats)" :key="item.bucket">
            <span>{{ item.bucket }}</span>
            <strong>{{ money(item.amount_cents) }}</strong>
            <em>{{ item.record_count }} 笔</em>
          </article>
        </div>
        <n-empty v-else description="暂无账龄数据" />
      </n-card>

      <n-card :title="ownerTitle" size="small">
        <div v-if="receivables?.owner_stats?.length" class="summary-list">
          <article v-for="item in receivables.owner_stats" :key="item.name">
            <span>{{ item.name }}</span>
            <strong>{{ money(item.amount_cents) }}</strong>
            <em>{{ item.record_count }} 笔</em>
          </article>
        </div>
        <n-empty v-else description="暂无成员数据" />
      </n-card>

      <n-card :title="relationTitle" size="small">
        <div v-if="receivables?.relation_stats?.length" class="summary-list">
          <article v-for="item in receivables.relation_stats" :key="item.name">
            <span>{{ dictionaryLabel('relation_type', item.name) }}</span>
            <strong>{{ money(item.amount_cents) }}</strong>
            <em>{{ item.record_count }} 笔</em>
          </article>
        </div>
        <n-empty v-else description="暂无关系数据" />
      </n-card>
    </div>

    <n-card :title="detailTitle" size="small">
      <n-spin :show="loading">
      <div class="desktop-table">
        <n-table :bordered="false" :single-line="true" size="small">
          <thead>
            <tr>
              <th>发生日期</th>
              <th>对象</th>
              <th>归属成员</th>
              <th>关系</th>
              <th>事项</th>
              <th>金额</th>
              <th>等待</th>
              <th>备注</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in receivables?.records?.list || []" :key="row.id">
              <td>{{ formatDate(row.occurred_on) }}</td>
              <td>{{ row.contact_name }}</td>
              <td>{{ row.owner_name }}</td>
              <td>{{ dictionaryLabel('relation_type', row.relation_type) }}</td>
              <td>{{ dictionaryLabel('occasion_type', row.occasion_type) }}</td>
              <td>{{ money(row.amount_cents) }}</td>
              <td>{{ waitText(row.wait_days) }}</td>
              <td class="cell-muted">{{ remarkText(row) }}</td>
            </tr>
          </tbody>
        </n-table>
        <n-empty v-if="!receivables?.records?.list?.length" :description="detailEmptyText" />
      </div>
      <div class="mobile-list">
        <article v-for="row in receivables?.records?.list || []" :key="row.id" class="mobile-item">
          <div>
            <strong>{{ row.contact_name }}</strong>
            <span>{{ dictionaryLabel('occasion_type', row.occasion_type) }} · {{ formatDate(row.occurred_on) }}</span>
          </div>
          <p>{{ row.owner_name }} · {{ dictionaryLabel('relation_type', row.relation_type) }} · 等待 {{ waitText(row.wait_days) }}</p>
          <p>{{ remarkText(row) }}</p>
          <b>{{ money(row.amount_cents) }}</b>
        </article>
        <n-empty v-if="!receivables?.records?.list?.length" :description="detailEmptyText" />
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
      </n-spin>
    </n-card>
  </section>
</template>
