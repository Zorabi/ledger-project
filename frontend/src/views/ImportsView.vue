<script setup lang="ts">
import { ArchiveRestore, Download, Upload } from 'lucide-vue-next'
import MobilePager from '../components/MobilePager.vue'

withDefaults(defineProps<{
  query: any
  errors: any[]
  jobs: any[]
  errorColumns: any[]
  jobColumns: any[]
  loading: boolean
  statusText: (value: string | boolean) => string
  formatDateTime: (value?: string) => string
  isJobFailed: (job: any) => boolean
}>(), {
  isJobFailed: (job: any) => job?.status === 'dry_run_failed' || job?.status === 'failed' || job?.status === 'blocked' || job?.error_count > 0
})

const emit = defineEmits<{
  downloadTemplate: []
  openManual: []
  openBackup: []
  load: []
  executeImport: [jobId: number]
  showErrors: [job: any]
}>()

function reloadFromFirstPage(query: any) {
  query.page = 1
  emit('load')
}
</script>

<template>
  <section class="view-stack">
    <n-card title="导入恢复" size="small">
      <div class="toolbar import-toolbar">
        <n-button size="small" @click="emit('downloadTemplate')"><template #icon><Download :size="16" /></template>下载手工模板</n-button>
        <n-button type="primary" size="small" @click="emit('openManual')"><template #icon><Upload :size="16" /></template>手工导入</n-button>
        <n-button size="small" @click="emit('openBackup')"><template #icon><ArchiveRestore :size="16" /></template>备份恢复</n-button>
      </div>
      <n-data-table v-if="errors.length" class="desktop-table import-error-table" :data="errors" :columns="errorColumns" :loading="loading" />
      <div v-if="errors.length" class="mobile-list import-error-list">
        <article v-for="error in errors" :key="`${error.sheet_name}-${error.row_number}-${error.field_name}`" class="mobile-item">
          <div class="item-main"><strong>{{ error.sheet_name }}</strong><n-tag size="small" type="error">第 {{ error.row_number }} 行</n-tag></div>
          <p>{{ error.field_name }}</p>
          <small>{{ error.error_message }}</small>
        </article>
      </div>
      <n-empty v-else description="暂无导入错误" />
    </n-card>
    <n-card title="导入批次" size="small">
      <n-data-table class="desktop-table" :data="jobs" :columns="jobColumns" :loading="loading" />
      <div class="mobile-list">
        <article v-for="job in jobs" :key="job.id" class="mobile-item">
          <div class="item-main"><strong>{{ job.source_file_name || '未命名文件' }}</strong><n-tag size="small">{{ statusText(job.status) }}</n-tag></div>
          <p>新增 {{ job.created_count }} · 更新 {{ job.updated_count }} · 错误 {{ job.error_count }}</p>
          <small>{{ formatDateTime(job.started_at) }}</small>
          <div v-if="job.status === 'dry_run_ready'" class="item-actions"><n-button size="small" type="primary" secondary @click="emit('executeImport', job.id)">执行导入</n-button></div>
          <div v-else-if="isJobFailed(job)" class="item-actions"><n-button size="small" type="error" secondary @click="emit('showErrors', job)">查看详情</n-button></div>
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
