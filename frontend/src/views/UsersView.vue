<script setup lang="ts">
import { Plus } from 'lucide-vue-next'

defineProps<{
  users: any[]
  columns: any[]
  statusText: (value: string | boolean) => string
  formatDateTime: (value?: string) => string
}>()

const emit = defineEmits<{
  create: []
  edit: [user: any]
  resetPassword: [user: any]
  updateStatus: [user: any]
}>()
</script>

<template>
  <section class="view-stack">
    <n-card title="用户列表" size="small">
      <template #header-extra>
        <n-button type="primary" size="small" @click="emit('create')">
          <template #icon><Plus :size="16" /></template>
          新增用户
        </n-button>
      </template>
      <n-data-table class="desktop-table" :data="users" :columns="columns" />
      <div class="mobile-list">
        <article v-for="item in users" :key="item.id" class="mobile-item">
          <div class="item-main"><strong>{{ item.display_name }}</strong><n-tag size="small">{{ statusText(item.role) }}</n-tag></div>
          <p>{{ item.username }} · {{ item.email || '未填写邮箱' }}</p>
          <small>{{ statusText(item.status) }} · 最后登录 {{ formatDateTime(item.last_login_at) }}</small>
          <div class="item-actions">
            <n-button size="small" secondary @click="emit('edit', item)">编辑</n-button>
            <n-button size="small" secondary type="primary" @click="emit('resetPassword', item)">重置密码</n-button>
            <n-button size="small" secondary @click="emit('updateStatus', item)">{{ item.status === 'enabled' ? '禁用' : '启用' }}</n-button>
          </div>
        </article>
      </div>
    </n-card>
  </section>
</template>
