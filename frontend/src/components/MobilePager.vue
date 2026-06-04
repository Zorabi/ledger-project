<script setup lang="ts">
import { computed } from 'vue'
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'

const props = defineProps<{
  query: {
    page: number
    page_size: number
    total: number
  }
}>()

const emit = defineEmits<{
  load: []
}>()

const pageCount = computed(() => Math.max(1, Math.ceil(Number(props.query.total || 0) / Number(props.query.page_size || 10))))

function changePage(nextPage: number) {
  const page = Math.min(pageCount.value, Math.max(1, nextPage))
  if (page === props.query.page) return
  props.query.page = page
  emit('load')
}
</script>

<template>
  <div class="mobile-pager">
    <n-button secondary :disabled="query.page <= 1" @click="changePage(query.page - 1)">
      <template #icon><ChevronLeft :size="16" /></template>
      上一页
    </n-button>
    <span>第 {{ query.page }} / {{ pageCount }} 页</span>
    <n-button secondary :disabled="query.page >= pageCount" @click="changePage(query.page + 1)">
      下一页
      <template #icon><ChevronRight :size="16" /></template>
    </n-button>
  </div>
</template>
