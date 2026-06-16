<script setup lang="ts">
import type { Component } from 'vue'
import { NIcon } from 'naive-ui'

defineProps<{
  label: string
  icon: Component
  value?: string
  progress?: number
  progressColor?: string
}>()
</script>

<template>
  <n-card size="small" :bordered="true">
    <div class="flex items-center gap-1.5 text-xs text-neutral-400 mb-2">
      <NIcon :component="icon" :size="13" />
      {{ label }}
    </div>
    <div class="text-sm font-medium text-neutral-100">
      <slot>{{ value }}</slot>
    </div>
    <n-progress
      v-if="progress !== undefined"
      type="line"
      :percentage="progress"
      :show-indicator="false"
      :height="3"
      :color="progressColor"
      :rail-color="'rgba(255,255,255,0.08)'"
      class="mt-2"
    />
  </n-card>
</template>
