<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'

const props = defineProps<{
  stats: any
  active: boolean
  dictionaryLabel: (typeCode: string, value: string) => string
  statusText: (value: string | boolean) => string
  money: (cents?: number) => string
  moneyCompact: (cents?: number) => string
}>()

const trendChart = ref<HTMLElement | null>(null)
const occasionChart = ref<HTMLElement | null>(null)
const contactChart = ref<HTMLElement | null>(null)
const relationChart = ref<HTMLElement | null>(null)
const reciprocityChart = ref<HTMLElement | null>(null)
const balanceChart = ref<HTMLElement | null>(null)
const occasionTrendChart = ref<HTMLElement | null>(null)
const occasionMatrixChart = ref<HTMLElement | null>(null)
const relationMatrixChart = ref<HTMLElement | null>(null)
const periodCompareChart = ref<HTMLElement | null>(null)
const handlerChart = ref<HTMLElement | null>(null)
const ownerChart = ref<HTMLElement | null>(null)
let chartInstances: echarts.ECharts[] = []
let pendingRender = 0

const trendChartTitle = computed(() => props.stats?.trend_granularity === 'day' ? '每日收送趋势' : '月度收送趋势')
const periodCompareRangeText = computed(() => {
  const current = props.stats?.period_compare?.current_range
  const previous = props.stats?.period_compare?.previous_range
  if (!current?.start_date || !current?.end_date || !previous?.start_date || !previous?.end_date) return '当前周期与上一等长周期'
  return `当前 ${current.start_date} 至 ${current.end_date}，对比 ${previous.start_date} 至 ${previous.end_date}`
})
const ownerContactRows = computed(() =>
  (props.stats?.owner_contacts || [])
    .map((item: any) => ({ name: item.name || '未归属', count: Number(item.count || 0) }))
    .filter((item: any) => item.count > 0)
    .sort((a: any, b: any) => b.count - a.count)
)
const ownerMemberRows = computed(() => {
  const statsRows = props.stats?.owner_stats || []
  return ownerContactRows.value.map((contactRow: any) => {
    const statsRow = statsRows.find((item: any) => item.name === contactRow.name) || {}
    const received = Number(statsRow.received_cents || 0)
    const given = Number(statsRow.given_cents || 0)
    return {
      ...contactRow,
      record_count: Number(statsRow.record_count || 0),
      received_cents: received,
      given_cents: given,
      net_cents: received - given
    }
  })
})

function renderCharts() {
  chartInstances.forEach((chart) => chart.dispose())
  chartInstances = []
  if (!props.active || !props.stats) return

  const stats = props.stats
  const trendRows = stats.monthly_trend || []
  const trendPeriods = Array.from(new Set(trendRows.map((item: any) => item.period || item.month))).sort()
  const amountBy = (direction: string) =>
    trendPeriods.map((period) => (trendRows.find((item: any) => (item.period || item.month) === period && item.direction === direction)?.amount_cents || 0) / 100)
  const occasionData = (stats.occasion_pie || []).map((item: any) => ({ name: props.dictionaryLabel('occasion_type', item.name), value: item.amount_cents / 100 }))
  const contactData = stats.contact_top10 || []
  const relationData = (stats.relation_stats || []).map((item: any) => ({ ...item, label: props.dictionaryLabel('relation_type', item.name) }))
  const reciprocityData = (stats.reciprocity_stats || []).map((item: any) => ({ name: props.statusText(item.status), value: item.amount_cents / 100, count: item.count }))
  const balanceData = stats.balance_top10 || []
  const occasionTrendRows = stats.occasion_trend || []
  const occasionTrendPeriods = Array.from(new Set(occasionTrendRows.map((item: any) => item.period))).sort()
  const occasionTrendNames = Array.from(new Set(occasionTrendRows.map((item: any) => item.name))).slice(0, 6)
  const occasionMatrixData = (stats.occasion_matrix || []).map((item: any) => ({ ...item, label: props.dictionaryLabel('occasion_type', item.name) }))
  const relationMatrixData = (stats.relation_matrix || []).map((item: any) => ({ ...item, label: props.dictionaryLabel('relation_type', item.name) }))
  const compare = stats.period_compare || {}
  const handlerData = stats.handler_stats || []
  const ownerData = stats.owner_stats || []
  const axisStyle = { color: '#6a7880', fontSize: 11 }
  const splitLine = { lineStyle: { color: 'rgba(111, 130, 120, 0.12)', type: 'dashed' } }
  const isMobileChart = window.innerWidth <= 560
  const barGrid = { left: isMobileChart ? 74 : 84, right: isMobileChart ? 78 : 74, bottom: 28, top: 18 }
  const barValueLabel = { show: true, position: 'right', distance: 8, formatter: ({ value }: any) => `¥${Number(value || 0).toLocaleString('zh-CN', { maximumFractionDigits: 0 })}`, color: '#4a5960' }
  const richTooltip = { backgroundColor: 'rgba(15, 32, 39, 0.92)', borderWidth: 0, textStyle: { color: '#f7fbf9' }, extraCssText: 'box-shadow:0 12px 28px rgba(10,24,31,.18);border-radius:10px;' }
  const donutLabel = { formatter: '{b}\n{d}%', color: '#33424a', fontSize: 12, lineHeight: 16 }
  const emptyGraphic = (text: string) => ({
    type: 'text',
    left: 'center',
    top: 'middle',
    style: { text, fill: '#8a989f', fontSize: 14, fontWeight: 600 }
  })

  mountChart(trendChart.value, {
    color: ['#19a66b', '#f59f35'],
    tooltip: { ...richTooltip, trigger: 'axis', valueFormatter: (value: any) => `¥${Number(value || 0).toFixed(2)}` },
    legend: { top: 6, right: 12, itemWidth: 10, itemHeight: 10 },
    grid: { left: 54, right: 24, top: 54, bottom: 34 },
    xAxis: { type: 'category', boundaryGap: false, data: trendPeriods, axisLabel: { ...axisStyle, formatter: (value: string) => stats?.trend_granularity === 'day' ? value.slice(5) : value }, axisTick: { show: false }, axisLine: { lineStyle: { color: 'rgba(82, 104, 96, 0.22)' } } },
    yAxis: { type: 'value', axisLabel: { ...axisStyle, formatter: (value: number) => `¥${value}` }, splitLine },
    series: [
      { name: '我家收礼', type: 'line', smooth: true, symbolSize: 7, areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: 'rgba(25,166,107,0.28)' }, { offset: 1, color: 'rgba(25,166,107,0.02)' }]) }, data: amountBy('in') },
      { name: '我家送礼', type: 'line', smooth: true, symbolSize: 7, areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: 'rgba(245,159,53,0.24)' }, { offset: 1, color: 'rgba(245,159,53,0.02)' }]) }, data: amountBy('out') }
    ]
  })
  mountChart(ownerChart.value, {
    color: ['#19a66b', '#f59f35'],
    graphic: ownerData.length ? undefined : emptyGraphic('暂无归属成员统计'),
    tooltip: {
      ...richTooltip,
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: any[]) => {
        const row = ownerData[params?.[0]?.dataIndex]
        if (!row) return ''
        return `${row.name}<br/>收礼 ${props.money(row.received_cents)}<br/>送礼 ${props.money(row.given_cents)}<br/>净差额 ${props.money((row.received_cents || 0) - (row.given_cents || 0))}<br/>记录 ${row.record_count || 0} 笔`
      }
    },
    legend: { top: 4, right: 12, icon: 'circle' },
    grid: { left: 48, right: 24, top: 56, bottom: 42 },
    xAxis: { type: 'category', data: ownerData.map((item: any) => item.name), axisLabel: { ...axisStyle, interval: 0, width: 76, overflow: 'truncate' }, axisTick: { show: false }, axisLine: { show: false } },
    yAxis: { type: 'value', axisLabel: { ...axisStyle, formatter: (value: number) => `¥${value}` }, splitLine },
    series: [
      {
        name: '收礼',
        type: 'bar',
        barWidth: 14,
        barGap: '34%',
        data: ownerData.map((item: any) => item.received_cents / 100),
        itemStyle: { borderRadius: [8, 8, 0, 0] },
        label: { show: !isMobileChart, position: 'top', formatter: ({ value }: any) => `¥${Number(value || 0).toLocaleString('zh-CN', { maximumFractionDigits: 0 })}`, color: '#4a5960', fontSize: 11 }
      },
      {
        name: '送礼',
        type: 'bar',
        barWidth: 14,
        data: ownerData.map((item: any) => item.given_cents / 100),
        itemStyle: { borderRadius: [8, 8, 0, 0] },
        label: { show: !isMobileChart, position: 'top', formatter: ({ value }: any) => `¥${Number(value || 0).toLocaleString('zh-CN', { maximumFractionDigits: 0 })}`, color: '#4a5960', fontSize: 11 }
      }
    ]
  })
  mountChart(reciprocityChart.value, {
    color: ['#f2a83b', '#28a978', '#9aa5ad'],
    graphic: reciprocityData.length ? undefined : emptyGraphic('暂无回礼统计'),
    tooltip: { ...richTooltip, trigger: 'item', formatter: ({ name, value, data }: any) => `${name}<br/>${Number(data?.count || 0)} 笔 · ¥${Number(value || 0).toFixed(2)}` },
    legend: { bottom: 0, left: 10, right: 10, icon: 'circle' },
    series: [{
      name: '回礼状态',
      type: 'pie',
      radius: ['48%', '68%'],
      center: ['50%', '43%'],
      minAngle: 8,
      avoidLabelOverlap: true,
      itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 2 },
      label: { formatter: '{b}\n{d}%', color: '#33424a' },
      data: reciprocityData
    }]
  })
  mountChart(balanceChart.value, {
    color: ['#2f86c7'],
    graphic: balanceData.length ? undefined : emptyGraphic('暂无差额统计'),
    tooltip: { ...richTooltip, trigger: 'axis', axisPointer: { type: 'shadow' }, formatter: (params: any) => {
      const item = balanceData[params[0]?.dataIndex]
      if (!item) return ''
      return `${item.name}<br/>收礼 ${props.money(item.received_cents)}<br/>送礼 ${props.money(item.given_cents)}<br/>净差额 ${props.money(item.net_cents)}`
    } },
    grid: barGrid,
    xAxis: { type: 'value', axisLabel: { ...axisStyle, formatter: (value: number) => `¥${value}` }, splitLine },
    yAxis: { type: 'category', inverse: true, data: balanceData.map((item: any) => item.name), axisLabel: { ...axisStyle, width: 72, overflow: 'truncate' }, axisTick: { show: false }, axisLine: { show: false } },
    series: [{
      name: '净差额',
      type: 'bar',
      barWidth: 14,
      data: balanceData.map((item: any) => item.net_cents / 100),
      itemStyle: { borderRadius: [0, 8, 8, 0], color: ({ value }: any) => Number(value) >= 0 ? '#2f86c7' : '#d86b4a' },
      label: { ...barValueLabel, formatter: ({ value }: any) => `¥${Number(value || 0).toLocaleString('zh-CN', { maximumFractionDigits: 0 })}` }
    }]
  })
  mountChart(contactChart.value, {
    color: ['#19a66b'],
    graphic: contactData.length ? undefined : emptyGraphic('暂无对象统计'),
    tooltip: { ...richTooltip, trigger: 'axis', axisPointer: { type: 'shadow' }, valueFormatter: (value: any) => `¥${Number(value || 0).toFixed(2)}` },
    grid: barGrid,
    xAxis: { type: 'value', axisLabel: axisStyle, splitLine },
    yAxis: { type: 'category', inverse: true, data: contactData.map((item: any) => item.name), axisLabel: { ...axisStyle, width: 72, overflow: 'truncate' }, axisTick: { show: false }, axisLine: { show: false } },
    series: [{ name: '往来金额', type: 'bar', barWidth: 14, data: contactData.map((item: any) => item.amount_cents / 100), itemStyle: { borderRadius: [0, 8, 8, 0], color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [{ offset: 0, color: '#7ed8b0' }, { offset: 1, color: '#19a66b' }]) }, label: barValueLabel }]
  })
  mountChart(occasionTrendChart.value, {
    color: ['#1f9d72', '#e9a23b', '#4f7fd9', '#c75d8a', '#7a6ccf', '#1d9aa7'],
    graphic: occasionTrendRows.length ? undefined : emptyGraphic('暂无事项趋势'),
    tooltip: { ...richTooltip, trigger: 'axis', valueFormatter: (value: any) => `¥${Number(value || 0).toFixed(2)}` },
    legend: { type: 'scroll', top: 4, right: 12, icon: 'circle' },
    grid: { left: 54, right: 24, top: 56, bottom: 34 },
    xAxis: { type: 'category', data: occasionTrendPeriods, axisLabel: { ...axisStyle, formatter: (value: string) => stats?.trend_granularity === 'day' ? value.slice(5) : value }, axisTick: { show: false } },
    yAxis: { type: 'value', axisLabel: { ...axisStyle, formatter: (value: number) => `¥${value}` }, splitLine },
    series: occasionTrendNames.map((name) => ({
      name: props.dictionaryLabel('occasion_type', String(name)),
      type: 'line',
      smooth: true,
      symbolSize: 5,
      lineStyle: { width: 2.5 },
      areaStyle: { opacity: 0.08 },
      emphasis: { focus: 'series' },
      data: occasionTrendPeriods.map((period) => (occasionTrendRows.find((item: any) => item.period === period && item.name === name)?.amount_cents || 0) / 100)
    }))
  })
  mountChart(occasionMatrixChart.value, {
    color: ['#2f86c7', '#e69a32'],
    graphic: occasionMatrixData.length ? undefined : emptyGraphic('暂无事项收送统计'),
    tooltip: { ...richTooltip, trigger: 'axis', axisPointer: { type: 'shadow' }, valueFormatter: (value: any) => `¥${Number(value || 0).toFixed(2)}` },
    legend: { top: 4, right: 12, icon: 'circle' },
    grid: { ...barGrid, top: 48, right: isMobileChart ? 66 : 70 },
    xAxis: { type: 'value', axisLabel: axisStyle, splitLine },
    yAxis: { type: 'category', inverse: true, data: occasionMatrixData.map((item: any) => item.label), axisLabel: { ...axisStyle, width: 72, overflow: 'truncate' }, axisTick: { show: false }, axisLine: { show: false } },
    series: [
      { name: '收礼', type: 'bar', barWidth: 12, barGap: '36%', data: occasionMatrixData.map((item: any) => item.received_cents / 100), itemStyle: { borderRadius: [0, 8, 8, 0] }, label: { ...barValueLabel, show: !isMobileChart } },
      { name: '送礼', type: 'bar', barWidth: 12, data: occasionMatrixData.map((item: any) => item.given_cents / 100), itemStyle: { borderRadius: [0, 8, 8, 0] }, label: { ...barValueLabel, show: !isMobileChart } }
    ]
  })
  mountChart(occasionChart.value, {
    color: ['#19a66b', '#f59f35', '#4c8df6', '#d95d8b', '#7d68d8', '#16a0a6'],
    tooltip: { ...richTooltip, trigger: 'item', formatter: ({ name, value, percent }: any) => `${name}<br/>¥${Number(value || 0).toFixed(2)} · ${percent}%` },
    legend: { type: 'scroll', bottom: 0, left: 10, right: 10, icon: 'circle' },
    series: [{
      name: '事项占比',
      type: 'pie',
      radius: ['48%', '68%'],
      center: ['50%', '43%'],
      minAngle: 8,
      avoidLabelOverlap: true,
      itemStyle: { borderRadius: 7, borderColor: '#fff', borderWidth: 2 },
      label: { formatter: '{b}\n{d}%', color: '#33424a' },
      data: occasionData
    }]
  })
  mountChart(relationMatrixChart.value, {
    color: ['#36a97d', '#f0a43a'],
    graphic: relationMatrixData.length ? undefined : emptyGraphic('暂无关系收送统计'),
    tooltip: { ...richTooltip, trigger: 'axis', axisPointer: { type: 'shadow' }, valueFormatter: (value: any) => `¥${Number(value || 0).toFixed(2)}` },
    legend: { top: 4, right: 12, icon: 'circle' },
    grid: { ...barGrid, top: 48, right: isMobileChart ? 66 : 70 },
    xAxis: { type: 'value', axisLabel: axisStyle, splitLine },
    yAxis: { type: 'category', inverse: true, data: relationMatrixData.map((item: any) => item.label), axisLabel: axisStyle, axisTick: { show: false }, axisLine: { show: false } },
    series: [
      { name: '收礼', type: 'bar', barWidth: 12, barGap: '36%', data: relationMatrixData.map((item: any) => item.received_cents / 100), itemStyle: { borderRadius: [0, 8, 8, 0] }, label: { ...barValueLabel, show: !isMobileChart } },
      { name: '送礼', type: 'bar', barWidth: 12, data: relationMatrixData.map((item: any) => item.given_cents / 100), itemStyle: { borderRadius: [0, 8, 8, 0] }, label: { ...barValueLabel, show: !isMobileChart } }
    ]
  })
  mountChart(relationChart.value, {
    color: ['#4c8df6', '#19a66b', '#f59f35', '#d95d8b', '#7d68d8', '#16a0a6'],
    graphic: relationData.length ? undefined : emptyGraphic('暂无关系统计'),
    tooltip: { ...richTooltip, trigger: 'item', formatter: ({ name, value, percent }: any) => `${name}<br/>¥${Number(value || 0).toFixed(2)} · ${percent}%` },
    legend: { type: 'scroll', bottom: 0, left: 10, right: 10, icon: 'circle' },
    series: [{
      name: '关系金额',
      type: 'pie',
      radius: ['46%', '68%'],
      center: ['50%', '43%'],
      minAngle: 8,
      itemStyle: { borderRadius: 7, borderColor: '#fff', borderWidth: 2 },
      label: donutLabel,
      data: relationData.map((item: any) => ({ name: item.label, value: item.amount_cents / 100 }))
    }]
  })
  mountChart(periodCompareChart.value, {
    color: ['#2f86c7', '#aeb9c0'],
    tooltip: { ...richTooltip, trigger: 'axis', valueFormatter: (value: any) => `¥${Number(value || 0).toFixed(2)}` },
    legend: { top: 4, right: 12, icon: 'circle' },
    grid: { left: 54, right: 24, top: 56, bottom: 34 },
    xAxis: { type: 'category', data: ['收礼', '送礼', '净差额'], axisLabel: axisStyle, axisTick: { show: false } },
    yAxis: { type: 'value', axisLabel: { ...axisStyle, formatter: (value: number) => `¥${value}` }, splitLine },
    series: [
      { name: '当前周期', type: 'bar', barWidth: 18, data: [compare.current?.received_cents || 0, compare.current?.given_cents || 0, compare.current?.net_cents || 0].map((value) => value / 100), itemStyle: { borderRadius: [8, 8, 0, 0] } },
      { name: '上一周期', type: 'bar', barWidth: 18, data: [compare.previous?.received_cents || 0, compare.previous?.given_cents || 0, compare.previous?.net_cents || 0].map((value) => value / 100), itemStyle: { borderRadius: [8, 8, 0, 0] } }
    ]
  })
  mountChart(handlerChart.value, {
    color: ['#19a66b', '#f59f35'],
    graphic: handlerData.length ? undefined : emptyGraphic('暂无经办人统计'),
    tooltip: { ...richTooltip, trigger: 'axis', axisPointer: { type: 'shadow' }, valueFormatter: (value: any) => `¥${Number(value || 0).toFixed(2)}` },
    legend: { top: 4, right: 12, icon: 'circle' },
    grid: { left: 46, right: 20, top: 48, bottom: 42 },
    xAxis: { type: 'category', data: handlerData.map((item: any) => item.name), axisLabel: { ...axisStyle, interval: 0, width: 76, overflow: 'truncate' }, axisTick: { show: false }, axisLine: { show: false } },
    yAxis: { type: 'value', axisLabel: { ...axisStyle, formatter: (value: number) => `¥${value}` }, splitLine },
    series: [
      { name: '收礼', type: 'bar', barWidth: 14, barGap: '34%', data: handlerData.map((item: any) => item.received_cents / 100), itemStyle: { borderRadius: [8, 8, 0, 0] } },
      { name: '送礼', type: 'bar', barWidth: 14, data: handlerData.map((item: any) => item.given_cents / 100), itemStyle: { borderRadius: [8, 8, 0, 0] } }
    ]
  })
}

async function scheduleChartRender() {
  if (!props.active) return
  pendingRender += 1
  const currentRender = pendingRender
  await nextTick()
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      if (currentRender !== pendingRender) return
      renderCharts()
      chartInstances.forEach((chart) => chart.resize())
    })
  })
}

function mountChart(el: HTMLElement | null, option: echarts.EChartsOption) {
  if (!el || el.clientWidth === 0 || el.clientHeight === 0) return
  const chart = echarts.init(el)
  chart.setOption(option)
  chart.resize()
  chartInstances.push(chart)
}

function resizeCharts() {
  chartInstances.forEach((chart) => chart.resize())
}

watch(() => props.active, (active) => {
  if (active) scheduleChartRender()
})
watch(() => props.stats, () => scheduleChartRender(), { deep: true })
onMounted(() => {
  window.addEventListener('resize', resizeCharts)
  scheduleChartRender()
})
onBeforeUnmount(() => {
  window.removeEventListener('resize', resizeCharts)
  chartInstances.forEach((chart) => chart.dispose())
})
</script>

<template>
  <div class="chart-grid">
    <n-card :title="trendChartTitle" size="small" class="chart-card chart-card-priority"><div ref="trendChart" class="chart"></div></n-card>
    <n-card title="归属成员统计" size="small" class="chart-card chart-card-priority"><div ref="ownerChart" class="chart"></div></n-card>
    <n-card title="待回礼对象 Top 10" size="small" class="chart-card chart-card-priority">
      <div class="ranking-list ranking-list-priority">
        <article v-for="item in stats?.pending_top10 || []" :key="item.name">
          <span>{{ item.name }}</span>
          <small>{{ item.count }} 笔</small>
          <strong>{{ moneyCompact(item.amount_cents) }}</strong>
        </article>
        <n-empty v-if="!(stats?.pending_top10 || []).length" description="暂无待回礼对象" />
      </div>
    </n-card>
    <n-card title="回礼状态" size="small" class="chart-card chart-card-priority"><div ref="reciprocityChart" class="chart"></div></n-card>
    <n-card title="收送差额排行" size="small" class="chart-card chart-card-priority"><div ref="balanceChart" class="chart"></div></n-card>
    <n-card title="往来对象 Top 10" size="small" class="chart-card"><div ref="contactChart" class="chart"></div></n-card>
    <n-card title="成员对象分布" size="small" class="chart-card">
      <div v-if="ownerMemberRows.length" class="owner-member-board">
        <article v-for="item in ownerMemberRows" :key="item.name" class="owner-member-card">
          <div class="owner-member-head">
            <span>{{ item.name }}</span>
            <strong>{{ item.count }} 个对象</strong>
          </div>
          <div class="owner-member-metrics">
            <span>流水 <b>{{ item.record_count }}</b> 笔</span>
            <span>收礼 <b>{{ money(item.received_cents) }}</b></span>
            <span>送礼 <b>{{ money(item.given_cents) }}</b></span>
            <span>净差额 <b :class="{ negative: item.net_cents < 0 }">{{ money(item.net_cents) }}</b></span>
          </div>
        </article>
      </div>
      <n-empty v-else description="暂无成员对象分布" />
    </n-card>
    <n-card title="事项类型趋势" size="small" class="chart-card"><div ref="occasionTrendChart" class="chart"></div></n-card>
    <n-card title="事项收送对比" size="small" class="chart-card"><div ref="occasionMatrixChart" class="chart"></div></n-card>
    <n-card title="事项占比" size="small" class="chart-card"><div ref="occasionChart" class="chart"></div></n-card>
    <n-card title="关系收送矩阵" size="small" class="chart-card"><div ref="relationMatrixChart" class="chart"></div></n-card>
    <n-card title="关系统计" size="small" class="chart-card"><div ref="relationChart" class="chart"></div></n-card>
    <n-card title="周期对比" size="small" class="chart-card">
      <p class="chart-subtitle">{{ periodCompareRangeText }}</p>
      <div ref="periodCompareChart" class="chart chart-with-subtitle"></div>
    </n-card>
    <n-card title="经办人统计" size="small" class="chart-card"><div ref="handlerChart" class="chart"></div></n-card>
  </div>
</template>
