<script setup lang="ts">
import { NIcon, NTag } from 'naive-ui'
import { TimeOutline, HardwareChipOutline } from '@vicons/ionicons5'
import { formatUptimeSince, formatMemoryMB } from '~/utils/format'
import { useCurrentInstance } from '~/services/api'

const instance = useCurrentInstance();

const { label: statusLabel, type: statusType, isRunning } = useInstanceStatus(instance)
</script>

<template>
  <div class="flex flex-wrap items-center gap-3">
    <h1 class="text-xl font-bold">{{ instance.name }}</h1>
    <NTag :type="statusType" size="small">{{ statusLabel }}</NTag>
    <span v-if="instance.active_since && isRunning" class="inline-flex items-center gap-1 text-xs text-neutral-500">
      <NIcon :component="TimeOutline" :size="13" />
      {{ formatUptimeSince(instance.active_since) }}
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
