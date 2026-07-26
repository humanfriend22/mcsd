<script setup lang="ts">
import { NIcon, NTag } from 'naive-ui'
import { TimeOutline, HardwareChipOutline } from '@vicons/ionicons5'
import { formatUptime, formatMemoryMB } from '~/utils/format'
import { useCurrentInstance } from '~/services/api'

const instance = useCurrentInstance();

const statusType = computed(() => {
  switch (instance.value.state) {
    case 'active': return 'success'
    case 'failed': return 'error'
    case 'activating':
    case 'deactivating': return 'warning'
    default: return 'default'
  }
})

const statusLabel = computed(() => {
  switch (instance.value.state) {
    case 'activating': return 'starting'
    case 'deactivating': return 'stopping'
    default: return instance.value.state
  }
})

const isRunning = computed(() => instance.value.state === 'active')
</script>

<template>
  <div class="flex flex-wrap items-center gap-3">
    <h1 class="text-xl font-bold">{{ instance.name }}</h1>
    <NTag :type="statusType" size="small">{{ statusLabel }}</NTag>
    <span v-if="instance.uptime_seconds && isRunning" class="inline-flex items-center gap-1 text-xs text-neutral-500">
      <NIcon :component="TimeOutline" :size="13" />
      {{ formatUptime(instance.uptime_seconds) }}
    </span>
    <span v-if="isRunning" class="inline-flex items-center gap-1 text-xs text-neutral-500">
      <NIcon :component="HardwareChipOutline" :size="13" />
      {{
        instance.memory_used
          ? `${formatMemoryMB(instance.memory_used)} / ${formatMemoryMB(instance.memory)}`
          : `${formatMemoryMB(instance.memory)} alloc`
      }}
    </span>
  </div>
</template>
