<script setup lang="ts">
defineProps<{
  show: boolean
  detail: any
  loading: boolean
  money: (cents?: number) => string
  dictionaryLabel: (typeCode: string, value: string) => string
  directionText: (value: string) => string
  statusText: (value: string | boolean) => string
  formatDate: (value?: string) => string
}>()

const emit = defineEmits<{
  close: []
  edit: [contact: any]
  addRecord: [contact: any]
}>()

function noteText(row: any) {
  return row?.remark || row?.gift_description || '未填写'
}
</script>

<template>
  <n-drawer :show="show" width="720" placement="right" @update:show="(value: boolean) => { if (!value) emit('close') }">
    <n-drawer-content title="对象详情" closable>
      <n-spin :show="loading">
        <section v-if="detail?.contact" class="contact-detail">
          <header class="contact-detail-head">
            <div>
              <strong>{{ detail.contact.name }}</strong>
              <span>{{ dictionaryLabel('contact_type', detail.contact.contact_type) }} · {{ dictionaryLabel('relation_type', detail.contact.relation_type) }}</span>
            </div>
            <div class="contact-detail-actions">
              <n-button secondary @click="emit('edit', detail.contact)">编辑对象</n-button>
              <n-button type="primary" @click="emit('addRecord', detail.contact)">新增流水</n-button>
            </div>
          </header>

          <div class="detail-info-grid">
            <article><span>归属成员</span><strong>{{ detail.contact.owner_user?.display_name || detail.contact.owner_user?.username || '未指定' }}</strong></article>
            <article><span>手机号</span><strong>{{ detail.contact.phone || '未填写' }}</strong></article>
            <article><span>地址</span><strong>{{ detail.contact.address || '未填写' }}</strong></article>
            <article><span>状态</span><strong>{{ statusText(detail.contact.status) }}</strong></article>
          </div>

          <p v-if="detail.contact.remark" class="detail-note">{{ detail.contact.remark }}</p>

          <div class="metrics contact-detail-metrics">
            <article><span>收礼合计</span><strong>{{ money(detail.summary?.received_cents) }}</strong></article>
            <article><span>送礼合计</span><strong>{{ money(detail.summary?.given_cents) }}</strong></article>
            <article><span>净差额</span><strong>{{ money(detail.summary?.net_cents) }}</strong></article>
            <article><span>流水笔数</span><strong>{{ detail.summary?.record_count || 0 }}</strong></article>
          </div>

          <section class="detail-section">
            <h2>待处理记录</h2>
            <div v-if="detail.todo_records?.length" class="detail-record-list">
              <article v-for="record in detail.todo_records" :key="record.id">
                <div>
                  <strong>{{ dictionaryLabel('occasion_type', record.occasion_type) }}</strong>
                  <span>{{ formatDate(record.occurred_on) }} · {{ directionText(record.direction) }} · {{ statusText(record.reciprocity_status) }}</span>
                </div>
                <em>{{ money(record.amount_cents) }}</em>
              </article>
            </div>
            <n-empty v-else description="暂无待处理记录" />
          </section>

          <section class="detail-section">
            <h2>往来时间线</h2>
            <div v-if="detail.timeline?.length" class="timeline-list">
              <article v-for="record in detail.timeline" :key="record.id">
                <time>{{ formatDate(record.occurred_on) }}</time>
                <div>
                  <strong>{{ dictionaryLabel('occasion_type', record.occasion_type) }} · {{ directionText(record.direction) }}</strong>
                  <span>{{ statusText(record.reciprocity_status) }} · {{ noteText(record) }}</span>
                </div>
                <em>{{ money(record.amount_cents) }}</em>
              </article>
            </div>
            <n-empty v-else description="暂无往来流水" />
          </section>

          <section class="detail-section">
            <h2>已关联回礼</h2>
            <div v-if="detail.linked_records?.length" class="detail-record-list">
              <article v-for="record in detail.linked_records" :key="record.id">
                <div>
                  <strong>{{ dictionaryLabel('occasion_type', record.occasion_type) }}</strong>
                  <span>{{ formatDate(record.occurred_on) }} · {{ directionText(record.direction) }} · 关联 #{{ record.related_record_id }}</span>
                </div>
                <em>{{ money(record.amount_cents) }}</em>
              </article>
            </div>
            <n-empty v-else description="暂无已关联回礼记录" />
          </section>
        </section>
      </n-spin>
    </n-drawer-content>
  </n-drawer>
</template>
