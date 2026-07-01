<script setup lang="ts">
import { Plus, RotateCcw, Search } from 'lucide-vue-next'

defineProps<{
  stats: any
  query: any
  recordForm: any
  contacts: any[]
  records: any[]
  occasionTypeOptions: any[]
  handlerOptions: any[]
  relatedRecordOptions: any[]
  recordColumns: any[]
  recordsLoading: boolean
  saving: boolean
  quickRecordKeepDraft: boolean
  money: (cents?: number) => string
  dictionaryLabel: (typeCode: string, value: string) => string
  directionText: (value: string) => string
  statusText: (value: string | boolean) => string
  formatDate: (value?: string) => string
  reciprocityRoleText: (direction: string) => string
}>()

const emit = defineEmits<{
  query: []
  reset: []
  setRange: [range: string]
  setRecordDirection: [direction: string]
  setRecordReciprocityStatus: [status: string]
  updateQuickRecordKeepDraft: [value: boolean]
  saveRecord: []
  editRecord: [record: any]
  reciprocityRecord: [record: any]
  deleteRecord: [record: any]
}>()
</script>

<template>
  <section class="view-stack">
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
      <article><span>累计收礼</span><strong>{{ money(stats?.cards?.received_cents) }}</strong></article>
      <article><span>累计送礼</span><strong>{{ money(stats?.cards?.given_cents) }}</strong></article>
      <article><span>净差额</span><strong>{{ money(stats?.cards?.net_cents) }}</strong></article>
      <article><span>对象/记录</span><strong>{{ stats?.cards?.contact_count || 0 }} / {{ stats?.cards?.record_count || 0 }}</strong></article>
    </div>

    <section class="split">
      <n-card title="快速录入" size="small" class="quick-record-card">
        <template #header-extra>
          <div class="quick-draft-switch">
            <span>保留表单</span>
            <n-switch :value="quickRecordKeepDraft" size="small" @update:value="(value: boolean) => emit('updateQuickRecordKeepDraft', value)" />
          </div>
        </template>
        <n-form label-placement="top">
          <n-form-item label="往来对象"><n-select v-model:value="recordForm.contact_id" filterable :options="contacts.map((item) => ({ label: item.name, value: item.id }))" placeholder="请选择往来对象" /></n-form-item>
          <n-form-item label="事项类型"><n-select v-model:value="recordForm.occasion_type" filterable :options="occasionTypeOptions" placeholder="请选择事项类型" /></n-form-item>
          <n-form-item label="往来方向">
            <div class="direction-choice">
              <button type="button" :class="{ active: recordForm.direction === 'in' }" @click="emit('setRecordDirection', 'in')">
                <strong>我家收礼</strong>
                <span>收到对方礼金或礼品</span>
              </button>
              <button type="button" :class="{ active: recordForm.direction === 'out' }" @click="emit('setRecordDirection', 'out')">
                <strong>我家送礼</strong>
                <span>送出礼金或礼品</span>
              </button>
            </div>
          </n-form-item>
          <div class="form-row">
            <n-form-item label="金额"><n-input-number v-model:value="recordForm.amount_yuan" :min="0" :step="100" /></n-form-item>
            <n-form-item label="日期"><n-date-picker v-model:formatted-value="recordForm.occurred_on" value-format="yyyy-MM-dd" type="date" :clearable="false" /></n-form-item>
          </div>
          <n-form-item label="经办人"><n-select v-model:value="recordForm.handled_by" clearable filterable tag :options="handlerOptions" placeholder="请选择或输入经办人" /></n-form-item>
          <n-form-item label="回礼状态">
            <n-select
              :value="recordForm.reciprocity_status"
              :options="[
                { label: '无需回礼', value: 'none' },
                { label: '待回礼', value: 'pending' },
                { label: '已回礼', value: 'done' }
              ]"
              placeholder="请选择回礼状态"
              @update:value="(value: string) => emit('setRecordReciprocityStatus', value)"
            />
          </n-form-item>
          <n-form-item v-if="recordForm.reciprocity_status === 'done'" :label="`关联${reciprocityRoleText(recordForm.direction)}流水`">
            <n-select v-model:value="recordForm.related_record_id" clearable filterable :options="relatedRecordOptions" :placeholder="`请选择${reciprocityRoleText(recordForm.direction)}对应的反向流水`" />
          </n-form-item>
          <n-form-item label="礼品说明"><n-input v-model:value="recordForm.gift_description" type="textarea" placeholder="请输入礼品名称、数量或其他说明" /></n-form-item>
          <n-form-item label="备注"><n-input v-model:value="recordForm.remark" type="textarea" placeholder="请输入备注" /></n-form-item>
          <n-button type="primary" block :loading="saving" :disabled="saving" @click="emit('saveRecord')">
            <template #icon><Plus :size="18" /></template>
            保存流水
          </n-button>
        </n-form>
      </n-card>

      <n-card title="最近流水" size="small">
        <n-data-table class="desktop-table" :columns="recordColumns" :data="records.slice(0, 8)" :loading="recordsLoading" size="small" />
        <div class="mobile-list">
          <article v-for="record in records.slice(0, 8)" :key="record.id" class="mobile-item">
            <div class="item-main">
              <strong>{{ record.contact?.name || record.contact_id }}</strong>
              <n-tag size="small" :type="record.direction === 'in' ? 'success' : 'warning'">{{ directionText(record.direction) }}</n-tag>
            </div>
            <p>{{ dictionaryLabel('occasion_type', record.occasion_type) }} · {{ money(record.amount_cents) }}</p>
            <p v-if="record.remark || record.gift_description" class="item-note">{{ record.remark || record.gift_description }}</p>
            <small>{{ formatDate(record.occurred_on) }} · {{ statusText(record.reciprocity_status) }}<template v-if="record.reciprocity_status === 'done'">（{{ reciprocityRoleText(record.direction) }}）</template></small>
            <div class="item-actions">
              <n-button size="small" secondary @click="emit('editRecord', record)">编辑</n-button>
              <n-button v-if="record.reciprocity_status === 'pending'" size="small" secondary type="success" @click="emit('reciprocityRecord', record)">回礼</n-button>
              <n-popconfirm @positive-click="emit('deleteRecord', record)">
                <template #trigger><n-button size="small" secondary type="error">删除</n-button></template>
                确认删除这条流水？
              </n-popconfirm>
            </div>
          </article>
        </div>
      </n-card>
    </section>
  </section>
</template>
