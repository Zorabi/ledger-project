<script setup lang="ts">
import { Plus } from 'lucide-vue-next'

defineProps<{
  contacts: any[]
  columns: any[]
  dictionaryLabel: (typeCode: string, value: string) => string
}>()

const emit = defineEmits<{
  create: []
  edit: [contact: any]
  delete: [contact: any]
}>()
</script>

<template>
  <section class="view-stack">
    <n-card title="对象列表" size="small">
      <template #header-extra>
        <n-button type="primary" size="small" @click="emit('create')">
          <template #icon><Plus :size="16" /></template>
          新增对象
        </n-button>
      </template>
      <n-data-table class="desktop-table" :data="contacts" :columns="columns" />
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
    </n-card>
  </section>
</template>
