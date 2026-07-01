<script setup lang="ts">
import { computed } from 'vue'
import { RotateCcw, Search } from 'lucide-vue-next'
import MobilePager from '../components/MobilePager.vue'

const props = defineProps<{
  data: any
  query: any
  loading: boolean
  money: (cents?: number) => string
  moneyCompact: (cents?: number) => string
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

const ratioText = computed(() => `${((Number(props.data?.cards?.amount_ratio) || 0) * 100).toFixed(1)}%`)
const trendLabel = computed(() => props.data?.trend_granularity === 'day' ? '按日' : '按月')

function noteText(row: any) {
  return row?.remark || row?.gift_description || '未填写'
}

function amountShare(item: any) {
  const total = Number(props.data?.cards?.amount_cents || 0)
  if (!total) return '0%'
  return `${((Number(item?.amount_cents || 0) / total) * 100).toFixed(1)}%`
}

function reloadFromFirstPage(query: any) {
  query.page = 1
  emit('load')
}
</script>

<template>
  <section class="view-stack nonreciprocal-view">
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
      <article><span>无需回礼金额</span><strong>{{ money(data?.cards?.amount_cents) }}</strong></article>
      <article><span>收礼无需回礼</span><strong>{{ money(data?.cards?.received_cents) }}</strong></article>
      <article><span>送礼无需回礼</span><strong>{{ money(data?.cards?.given_cents) }}</strong></article>
      <article><span>金额占比</span><strong>{{ ratioText }}</strong></article>
    </div>

    <div class="summary-grid">
      <article><span>涉及对象</span><strong>{{ data?.cards?.contact_count || 0 }}</strong></article>
      <article><span>流水笔数</span><strong>{{ data?.cards?.record_count || 0 }}</strong></article>
      <article><span>收送净额</span><strong>{{ money(data?.cards?.net_cents) }}</strong></article>
      <article><span>统计粒度</span><strong>{{ trendLabel }}</strong></article>
    </div>

    <div class="analysis-grid nonreciprocal-grid">
      <n-card title="对象 Top 10" size="small">
        <div v-if="data?.contact_top?.length" class="rank-list">
          <article v-for="(item, index) in data.contact_top" :key="item.contact_id">
            <b>{{ index + 1 }}</b>
            <div>
              <strong>{{ item.name }}</strong>
              <span>收 {{ moneyCompact(item.received_cents) }} · 送 {{ moneyCompact(item.given_cents) }} · {{ item.record_count }} 笔</span>
            </div>
            <em>{{ money(item.amount_cents) }}</em>
          </article>
        </div>
        <n-empty v-else description="暂无对象数据" />
      </n-card>

      <n-card title="时间趋势" size="small">
        <div v-if="data?.trend?.length" class="timeline-bars">
          <article v-for="item in data.trend.slice(0, 12)" :key="`${item.period}-${item.direction}`">
            <span>{{ item.period }} · {{ directionText(item.direction) }}</span>
            <strong>{{ money(item.amount_cents) }}</strong>
            <em>{{ item.record_count }} 笔</em>
          </article>
        </div>
        <n-empty v-else description="暂无趋势数据" />
      </n-card>

      <n-card title="事项类型" size="small">
        <div v-if="data?.occasion_stats?.length" class="summary-list">
          <article v-for="item in data.occasion_stats" :key="item.name">
            <span>{{ dictionaryLabel('occasion_type', item.name) }}</span>
            <strong>{{ money(item.amount_cents) }}</strong>
            <em>{{ amountShare(item) }}</em>
          </article>
        </div>
        <n-empty v-else description="暂无事项数据" />
      </n-card>

      <n-card title="关系类型" size="small">
        <div v-if="data?.relation_stats?.length" class="summary-list">
          <article v-for="item in data.relation_stats" :key="item.name">
            <span>{{ dictionaryLabel('relation_type', item.name) }}</span>
            <strong>{{ money(item.amount_cents) }}</strong>
            <em>{{ item.record_count }} 笔</em>
          </article>
        </div>
        <n-empty v-else description="暂无关系数据" />
      </n-card>

      <n-card title="归属成员" size="small">
        <div v-if="data?.owner_stats?.length" class="summary-list">
          <article v-for="item in data.owner_stats" :key="item.name">
            <span>{{ item.name }}</span>
            <strong>{{ money(item.amount_cents) }}</strong>
            <em>{{ item.record_count }} 笔</em>
          </article>
        </div>
        <n-empty v-else description="暂无成员数据" />
      </n-card>
    </div>

    <n-card title="无需回礼明细" size="small">
      <n-spin :show="loading">
        <div class="desktop-table">
          <n-table :bordered="false" :single-line="true" size="small">
            <thead>
              <tr>
                <th>发生日期</th>
                <th>对象</th>
                <th>方向</th>
                <th>归属成员</th>
                <th>关系</th>
                <th>事项</th>
                <th>金额</th>
                <th>备注</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in data?.records?.list || []" :key="row.id">
                <td>{{ formatDate(row.occurred_on) }}</td>
                <td>{{ row.contact_name }}</td>
                <td>{{ directionText(row.direction) }}</td>
                <td>{{ row.owner_name }}</td>
                <td>{{ dictionaryLabel('relation_type', row.relation_type) }}</td>
                <td>{{ dictionaryLabel('occasion_type', row.occasion_type) }}</td>
                <td>{{ money(row.amount_cents) }}</td>
                <td class="cell-muted">{{ noteText(row) }}</td>
              </tr>
            </tbody>
          </n-table>
          <n-empty v-if="!data?.records?.list?.length" description="暂无无需回礼明细" />
        </div>

        <div class="mobile-list">
          <article v-for="row in data?.records?.list || []" :key="row.id" class="mobile-item">
            <div class="item-main">
              <strong>{{ row.contact_name }}</strong>
              <n-tag size="small" :type="row.direction === 'in' ? 'success' : 'warning'">{{ directionText(row.direction) }}</n-tag>
            </div>
            <p>{{ dictionaryLabel('occasion_type', row.occasion_type) }} · {{ formatDate(row.occurred_on) }}</p>
            <p>{{ row.owner_name }} · {{ dictionaryLabel('relation_type', row.relation_type) }}</p>
            <p class="item-note">{{ noteText(row) }}</p>
            <b>{{ money(row.amount_cents) }}</b>
          </article>
          <n-empty v-if="!data?.records?.list?.length" description="暂无无需回礼明细" />
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
