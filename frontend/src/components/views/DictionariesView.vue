<script setup lang="ts">
import { Plus } from 'lucide-vue-next'

defineProps<{
  isSuperAdmin: boolean
  dictionaryTypes: any[]
  dictionaryGroups: any[]
  typeColumns: any[]
  itemColumns: any[]
  writableDictionaryTypes: any[]
  statusText: (value: string | boolean) => string
  boolText: (value: boolean) => string
  formatDateTime: (value?: string) => string
  canEditDictionaryItem: (item: any) => boolean
}>()

const emit = defineEmits<{
  createType: []
  createItem: []
  editType: [type: any]
  editItem: [item: any]
  deleteItem: [item: any]
}>()
</script>

<template>
  <section class="view-stack">
    <n-card title="数据字典" size="small">
      <template #header-extra>
        <n-space>
          <n-button v-if="isSuperAdmin" secondary size="small" @click="emit('createType')"><template #icon><Plus :size="16" /></template>新增字典类型</n-button>
          <n-button type="primary" size="small" :disabled="!writableDictionaryTypes.length" @click="emit('createItem')"><template #icon><Plus :size="16" /></template>新增字典项</n-button>
        </n-space>
      </template>
      <n-data-table
        v-if="isSuperAdmin"
        class="dict-type-table desktop-table"
        :data="dictionaryTypes"
        :columns="typeColumns"
        size="small"
      />
      <div v-if="isSuperAdmin" class="mobile-list dict-type-mobile">
        <article v-for="type in dictionaryTypes" :key="type.id" class="mobile-item">
          <div class="item-main"><strong>{{ type.name }}</strong><n-tag size="small">{{ statusText(type.enabled) }}</n-tag></div>
          <p>{{ type.code }}</p>
          <small>允许家庭自定义：{{ boolText(type.allow_family_custom) }} · 排序 {{ type.sort_order }}</small>
          <div class="item-actions"><n-button size="small" secondary @click="emit('editType', type)">编辑</n-button></div>
        </article>
      </div>
      <n-collapse accordion>
        <n-collapse-item v-for="group in dictionaryGroups" :key="group.type.id" :title="`${group.type.name}（${group.items.length}）`" :name="group.type.code">
          <n-data-table class="desktop-table" :data="group.items" :columns="itemColumns" size="small" />
          <div class="mobile-list">
            <article v-for="item in group.items" :key="item.id" class="mobile-item">
              <div class="item-main"><strong>{{ item.label }}</strong><n-tag size="small">{{ item.is_system ? '系统预置' : '家庭自定义' }}</n-tag></div>
              <p>值：{{ item.value }}</p>
              <small>{{ statusText(item.enabled) }} · 排序 {{ item.sort_order }} · {{ formatDateTime(item.updated_at) }}</small>
              <div v-if="canEditDictionaryItem(item)" class="item-actions">
                <n-button size="small" secondary @click="emit('editItem', item)">编辑</n-button>
                <n-popconfirm @positive-click="emit('deleteItem', item)">
                  <template #trigger><n-button size="small" secondary type="error">删除</n-button></template>
                  确认删除该字典项？
                </n-popconfirm>
              </div>
              <small v-else-if="!item.is_system">该类型已关闭家庭自定义，当前项只读</small>
            </article>
          </div>
        </n-collapse-item>
      </n-collapse>
    </n-card>
  </section>
</template>
