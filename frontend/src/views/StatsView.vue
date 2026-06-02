<script setup lang="ts">
import { Search } from 'lucide-vue-next'
import StatsCharts from '../components/StatsCharts.vue'

defineProps<{
  active: boolean
  stats: any
  query: any
  money: (cents?: number) => string
  moneyCompact: (cents?: number) => string
  dictionaryLabel: (typeCode: string, value: string) => string
  statusText: (value: string | boolean) => string
}>()

const emit = defineEmits<{
  query: []
  setRange: [range: string]
}>()
</script>

<template>
  <section class="view-stack">
    <n-card title="时间范围" size="small">
      <div class="filter-grid time-filter">
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
        <n-button type="primary" class="query-button" @click="emit('query')">
          <template #icon><Search :size="16" /></template>
          查询
        </n-button>
      </div>
    </n-card>

    <div class="metrics">
      <article><span>累计收礼</span><strong>{{ money(stats?.cards?.received_cents) }}</strong></article>
      <article><span>累计送礼</span><strong>{{ money(stats?.cards?.given_cents) }}</strong></article>
      <article><span>净差额</span><strong>{{ money(stats?.cards?.net_cents) }}</strong></article>
      <article><span>记录总数</span><strong>{{ stats?.cards?.record_count || 0 }}</strong></article>
    </div>

    <StatsCharts
      :stats="stats"
      :active="active"
      :dictionary-label="dictionaryLabel"
      :status-text="statusText"
      :money="money"
      :money-compact="moneyCompact"
    />
  </section>
</template>
