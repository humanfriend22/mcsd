<script setup lang="ts">
import { NIcon, NTag } from 'naive-ui'
import { TimeOutline, HardwareChipOutline } from '@vicons/ionicons5'
import { formatUptime, formatMemoryMB } from '~/utils/format'
import type { Instance } from '~/api'

const props = defineProps<{ instance: Instance | null; instanceId: string }>()

const statusType = computed(() => {
  switch (props.instance?.state) {
    case 'active': return 'success'
    case 'failed': return 'error'
    case 'activating':
    case 'deactivating': return 'warning'
    default: return 'default'
  }
})

const statusLabel = computed(() => {
  switch (props.instance?.state) {
    case 'activating': return 'starting'
    case 'deactivating': return 'stopping'
    default: return props.instance?.state ?? '—'
  }
})

const isRunning = computed(() => props.instance?.state === 'active')
</script>

<template>
  <div class="flex flex-wrap items-center gap-3">
    <h1 class="text-xl font-bold">{{ instance?.name ?? instanceId }}</h1>
    <NTag v-if="instance" :type="statusType" size="small">{{ statusLabel }}</NTag>
    <span
      v-if="instance?.uptime_seconds && isRunning"
      class="inline-flex items-center gap-1 text-xs text-neutral-500"
    >
      <NIcon :component="TimeOutline" :size="13" />
      {{ formatUptime(instance.uptime_seconds) }}
    </span>
    <span
      v-if="instance && isRunning"
      class="inline-flex items-center gap-1 text-xs text-neutral-500"
    >
      <NIcon :component="HardwareChipOutline" :size="13" />
      {{
        instance.memory_used
          ? `${formatMemoryMB(instance.memory_used)} / ${formatMemoryMB(instance.memory)}`
          : `${formatMemoryMB(instance.memory)} alloc`
      }}
    </span>
  </div>
</template>
