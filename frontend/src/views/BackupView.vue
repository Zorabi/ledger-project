<script setup lang="ts">
import { Download, Settings } from 'lucide-vue-next'
import MobilePager from '../components/MobilePager.vue'

defineProps<{
  form: any
  query: any
  jobs: any[]
  columns: any[]
  loading: boolean
  statusText: (value: string | boolean) => string
  formatDateTime: (value?: string) => string
}>()

const emit = defineEmits<{
  load: []
  edit: []
  download: []
  sendEmail: []
}>()

function reloadFromFirstPage(query: any) {
  query.page = 1
  emit('load')
}
</script>

<template>
  <section class="view-stack">
    <div class="summary-grid">
      <article><span>备份状态</span><strong>{{ form.enabled ? '已启用' : '未启用' }}</strong></article>
      <article><span>备份频率</span><strong>{{ statusText(form.frequency) }}</strong></article>
      <article><span>ZIP 加密</span><strong>{{ form.encrypt_enabled ? '已开启' : '未开启' }}</strong></article>
      <article><span>收件人</span><strong>{{ form.recipient_emails_text.split('\n').filter(Boolean).length }} 个</strong></article>
    </div>
    <n-card title="备份操作" size="small">
      <div class="toolbar">
        <n-button type="primary" @click="emit('edit')"><template #icon><Settings :size="17" /></template>编辑备份配置</n-button>
        <n-button @click="emit('download')"><template #icon><Download :size="17" /></template>立即导出</n-button>
        <n-button @click="emit('sendEmail')">发送备份邮件</n-button>
      </div>
    </n-card>
    <n-card title="备份日志" size="small">
      <n-data-table class="desktop-table" :data="jobs" :columns="columns" :loading="loading" />
      <div class="mobile-list">
        <article v-for="job in jobs" :key="job.id" class="mobile-item">
          <div class="item-main"><strong>{{ job.file_name || '未生成文件' }}</strong><n-tag size="small">{{ statusText(job.status) }}</n-tag></div>
          <p>{{ statusText(job.trigger_type) }} · {{ formatDateTime(job.started_at) }}</p>
          <small>{{ job.error_message || '无错误信息' }}</small>
        </article>
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
    </n-card>
  </section>
</template>
