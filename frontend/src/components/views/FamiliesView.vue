<script setup lang="ts">
import { Plus } from 'lucide-vue-next'

defineProps<{
  families: any[]
  columns: any[]
  statusText: (value: string | boolean) => string
  formatDateTime: (value?: string) => string
}>()

const emit = defineEmits<{
  create: []
  edit: [family: any]
  updateStatus: [family: any]
}>()
</script>

<template>
  <section class="view-stack">
    <n-card title="家庭列表" size="small">
      <template #header-extra>
        <n-button type="primary" size="small" @click="emit('create')">
          <template #icon><Plus :size="16" /></template>
          新增家庭
        </n-button>
      </template>
      <n-data-table class="desktop-table" :data="families" :columns="columns" />
      <div class="mobile-list">
        <article v-for="family in families" :key="family.id" class="mobile-item">
          <div class="item-main"><strong>{{ family.name }}</strong><n-tag size="small">{{ statusText(family.status) }}</n-tag></div>
          <p>{{ family.contact_name || '未填写联系人' }} · {{ family.contact_email || '未填写邮箱' }}</p>
          <small>{{ formatDateTime(family.created_at) }}</small>
          <div class="item-actions">
            <n-button size="small" secondary @click="emit('edit', family)">编辑</n-button>
            <n-button size="small" secondary @click="emit('updateStatus', family)">{{ family.status === 'enabled' ? '禁用' : '启用' }}</n-button>
          </div>
        </article>
      </div>
    </n-card>
  </section>
</template>
