<script setup lang="ts">
import { RotateCcw, Search } from 'lucide-vue-next'
import MobilePager from '../components/MobilePager.vue'

const props = defineProps<{
  todos: any
  query: any
  loading: boolean
  money: (cents?: number) => string
  dictionaryLabel: (typeCode: string, value: string) => string
  directionText: (value: string) => string
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
  return agingOrder.map((bucket) => stats.find((item: any) => item.bucket === bucket) || { bucket, amount_cents: 0, record_count: 0 })
}

function todoTypeText(value: string) {
  const labels: Record<string, string> = {
    pending_receivable: '待回收',
    pending_payable: '待送出',
    unlinked_done: '已回礼未关联'
  }
  return labels[value] || value || '-'
}

function waitText(days?: number) {
  const value = Number(days || 0)
  if (value <= 0) return '今天'
  return `${value} 天`
}

function noteText(row: any) {
  return row?.remark || row?.gift_description || '未填写'
}

function reloadFromFirstPage(query: any) {
  query.page = 1
  emit('load')
}
</script>

<template>
  <section class="view-stack todo-view">
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
      <article><span>待关注金额</span><strong>{{ money(todos?.cards?.amount_cents) }}</strong></article>
      <article><span>涉及对象</span><strong>{{ todos?.cards?.contact_count || 0 }}</strong></article>
      <article><span>待关注笔数</span><strong>{{ todos?.cards?.record_count || 0 }}</strong></article>
      <article><span>最久等待</span><strong>{{ waitText(todos?.cards?.max_wait_days) }}</strong></article>
    </div>

    <div class="analysis-grid">
      <n-card title="待办类型" size="small">
        <div v-if="todos?.type_stats?.length" class="summary-list">
          <article v-for="item in todos.type_stats" :key="item.type">
            <span>{{ todoTypeText(item.type) }}</span>
            <strong>{{ money(item.amount_cents) }}</strong>
            <em>{{ item.record_count }} 笔</em>
          </article>
        </div>
        <n-empty v-else description="暂无待办数据" />
      </n-card>

      <n-card title="账龄分布" size="small">
        <div v-if="todos?.aging_stats?.length" class="metric-bars">
          <article v-for="item in orderedAgingStats(todos.aging_stats)" :key="item.bucket">
            <span>{{ item.bucket }}</span>
            <strong>{{ money(item.amount_cents) }}</strong>
            <em>{{ item.record_count }} 笔</em>
          </article>
        </div>
        <n-empty v-else description="暂无账龄数据" />
      </n-card>

      <n-card title="大金额关注" size="small">
        <div v-if="todos?.large_records?.length" class="rank-list">
          <article v-for="(item, index) in todos.large_records" :key="item.id">
            <b>{{ index + 1 }}</b>
            <div>
              <strong>{{ item.contact_name }}</strong>
              <span>{{ todoTypeText(item.todo_type) }} · {{ dictionaryLabel('occasion_type', item.occasion_type) }} · 等待 {{ waitText(item.wait_days) }}</span>
            </div>
            <em>{{ money(item.amount_cents) }}</em>
          </article>
        </div>
        <n-empty v-else description="暂无大金额记录" />
      </n-card>

      <n-card title="归属成员" size="small">
        <div v-if="todos?.owner_stats?.length" class="summary-list">
          <article v-for="item in todos.owner_stats" :key="item.name">
            <span>{{ item.name }}</span>
            <strong>{{ money(item.amount_cents) }}</strong>
            <em>{{ item.record_count }} 笔</em>
          </article>
        </div>
        <n-empty v-else description="暂无成员数据" />
      </n-card>

      <n-card title="关系类型" size="small">
        <div v-if="todos?.relation_stats?.length" class="summary-list">
          <article v-for="item in todos.relation_stats" :key="item.name">
            <span>{{ dictionaryLabel('relation_type', item.name) }}</span>
            <strong>{{ money(item.amount_cents) }}</strong>
            <em>{{ item.record_count }} 笔</em>
          </article>
        </div>
        <n-empty v-else description="暂无关系数据" />
      </n-card>
    </div>

    <n-card title="待办明细" size="small">
      <n-spin :show="loading">
        <div class="desktop-table">
          <n-table :bordered="false" :single-line="true" size="small">
            <thead>
              <tr>
                <th>类型</th>
                <th>发生日期</th>
                <th>对象</th>
                <th>方向</th>
                <th>归属成员</th>
                <th>关系</th>
                <th>事项</th>
                <th>金额</th>
                <th>等待</th>
                <th>备注</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in todos?.records?.list || []" :key="row.id">
                <td>{{ todoTypeText(row.todo_type) }}</td>
                <td>{{ formatDate(row.occurred_on) }}</td>
                <td>{{ row.contact_name }}</td>
                <td>{{ directionText(row.direction) }}</td>
                <td>{{ row.owner_name }}</td>
                <td>{{ dictionaryLabel('relation_type', row.relation_type) }}</td>
                <td>{{ dictionaryLabel('occasion_type', row.occasion_type) }}</td>
                <td>{{ money(row.amount_cents) }}</td>
                <td>{{ waitText(row.wait_days) }}</td>
                <td class="cell-muted">{{ noteText(row) }}</td>
              </tr>
            </tbody>
          </n-table>
          <n-empty v-if="!todos?.records?.list?.length" description="暂无待办明细" />
        </div>

        <div class="mobile-list">
          <article v-for="row in todos?.records?.list || []" :key="row.id" class="mobile-item">
            <div class="item-main">
              <strong>{{ row.contact_name }}</strong>
              <n-tag size="small">{{ todoTypeText(row.todo_type) }}</n-tag>
            </div>
            <p>{{ dictionaryLabel('occasion_type', row.occasion_type) }} · {{ formatDate(row.occurred_on) }} · 等待 {{ waitText(row.wait_days) }}</p>
            <p>{{ directionText(row.direction) }} · {{ row.owner_name }} · {{ dictionaryLabel('relation_type', row.relation_type) }}</p>
            <p class="item-note">{{ noteText(row) }}</p>
            <b>{{ money(row.amount_cents) }}</b>
          </article>
          <n-empty v-if="!todos?.records?.list?.length" description="暂无待办明细" />
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
      </n-spin>
    </n-card>
  </section>
</template>
