<script setup lang="ts">
import { NIcon } from 'naive-ui'
import {
  HardwareChipOutline, ServerOutline, SaveOutline, LayersOutline
} from '@vicons/ionicons5'
import { formatMemoryMB } from '~/utils/format'
import { useInstances, useVitals } from '~/services/api'

const vitals = useVitals()
const instances = useInstances()

const hostStats = computed(() => {
  const h = vitals.value?.host
  if (!h) return []
  const ramPct = Math.round((h.memory_used / h.memory_total) * 100)
  const diskPct = Math.round((h.disk_used_gb / h.disk_total_gb) * 100)
  return [
    {
      icon: HardwareChipOutline,
      label: 'RAM',
      value: `${formatMemoryMB(h.memory_used)} / ${formatMemoryMB(h.memory_total)}`,
      pct: ramPct,
    },
    {
      icon: ServerOutline,
      label: 'Load avg',
      value: `${h.load_avg_1.toFixed(2)} / ${h.cpu_cores}`,
      pct: Math.min(Math.round((h.load_avg_1 / h.cpu_cores) * 100), 100),
    },
    {
      icon: SaveOutline,
      label: 'Disk',
      value: `${h.disk_used_gb} GB / ${h.disk_total_gb} GB`,
      pct: diskPct,
    },
  ]
})

const budgetStats = computed(() => {
  const b = vitals.value?.budget
  if (!b) return null
  return {
    pct: Math.min(Math.round((b.used / b.total) * 100), 100),
    value: `${formatMemoryMB(b.used)} / ${formatMemoryMB(b.total)}`,
  }
})

function thresholdColor(pct: number) {
  if (pct >= 90) return '#f2697a'
  if (pct >= 70) return '#f0a020'
  return '#63e2b7'
}
</script>

<template>
  <div>
    <h1 class="text-xl font-bold mb-6">Dashboard</h1>

    <!-- Host vitals -->
    <div v-if="vitals" class="mb-6 grid grid-cols-2 md:grid-cols-4 gap-4">
      <n-card v-for="stat in hostStats" :key="stat.label" size="small" :bordered="true">
        <div class="flex items-center gap-1.5 text-[13px] text-neutral-400 mb-2">
          <NIcon :component="stat.icon" :size="14" />
          {{ stat.label }}
        </div>
        <div class="text-lg font-medium text-neutral-100">{{ stat.value }}</div>
        <n-progress type="line" :percentage="stat.pct" :show-indicator="false" :height="3"
          :color="thresholdColor(stat.pct)" :rail-color="'rgba(255,255,255,0.08)'" class="mt-3" />
      </n-card>
      <n-card v-if="budgetStats" size="small" :bordered="true">
        <div class="flex items-center gap-1.5 text-[13px] text-neutral-400 mb-2">
          <NIcon :component="LayersOutline" :size="14" />
          Memory budget
        </div>
        <div class="text-lg font-medium text-neutral-100">{{ budgetStats.value }}</div>
        <n-progress type="line" :percentage="budgetStats.pct" :show-indicator="false" :height="3"
          :color="thresholdColor(budgetStats.pct)" :rail-color="'rgba(255,255,255,0.08)'" class="mt-3" />
      </n-card>
    </div>

    <div v-if="!instances" class="flex justify-center py-12">
      <n-spin size="large" />
    </div>

    <div v-else-if="instances.length === 0" class="text-center py-12 text-gray-400">
      No instances. Use <code>mcsd create</code> to add one.
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <ServerCard v-for="inst in instances" :key="inst.id" :instance="inst"
        @open="id => $router.push(`/instances/${id}`)" />
    </div>
  </div>
</template>
