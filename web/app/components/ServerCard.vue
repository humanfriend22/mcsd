<script setup lang="ts">
import { computed, ref } from 'vue'
import { NCard, NTag, NButton, NIcon, NTooltip } from 'naive-ui'
import {
  ServerOutline, GitNetworkOutline, LayersOutline,
  GlobeOutline, CopyOutline, CheckmarkOutline, TimeOutline
} from '@vicons/ionicons5'
import { formatUptimeSince, formatMemoryMB } from '~/utils/format'
import type { Instance } from '~/services/api'

const { instance } = defineProps<{ instance: Instance }>()
const emit = defineEmits<{ (e: 'open', id: string): void }>()

const statusType = computed(() => {
  switch (instance.state) {
    case 'active': return 'success'
    case 'activating':
    case 'deactivating': return 'warning'
    case 'failed': return 'error'
    case 'error': return 'error'
    default: return 'default'
  }
})

const statusLabel = computed(() => instance.state)

// Tooltip copy for a degraded instance. Renders the backend's message
// verbatim; falls back to a generic string when it's absent, empty, or
// whitespace-only so the tooltip never opens blank. Null (no tooltip)
// for every other state.
const errorMessage = computed(() => {
  if (instance.state !== 'error') return null
  const message = instance.error?.trim()
  return message ? message : 'Unknown error'
})

const gamePort = computed(() => instance.ports?.game ?? null)

const joinUrl = computed(() => {
  if (!gamePort.value) return null
  return `${window.location.hostname}:${gamePort.value}`
})

const memLabel = computed(() => {
  const used = instance.memory_used
  const alloc = instance.memory
  if (used != null && used > 0) return `${formatMemoryMB(used)} / ${formatMemoryMB(alloc)}`
  return `${formatMemoryMB(alloc)} allocated`
})

const uptimeLabel = computed(() => {
  if (!instance.active_since || instance.state !== 'active') return null
  return formatUptimeSince(instance.active_since)
})

const copied = ref(false)
async function copyJoin(e: Event) {
  e.stopPropagation()
  if (!joinUrl.value) return
  try {
    await navigator.clipboard.writeText(joinUrl.value)
    copied.value = true
    setTimeout(() => (copied.value = false), 1500)
  } catch { /* clipboard unavailable */ }
}
</script>

<template>
  <NCard :bordered="true" header-style="padding: 20px 20px 0 20px;" content-style="padding: 8px 20px 20px 20px;"
    role="button" tabindex="0"
    class="group cursor-pointer transition-colors hover:border-[#63e2b7] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400/50"
    @click="emit('open', instance.id)" @keydown.enter="emit('open', instance.id)">
    <template #header>
      <span class="text-lg font-medium text-neutral-100">{{ instance.name }}</span>
    </template>
    <template #header-extra>
      <NTooltip v-if="errorMessage" trigger="hover" style="max-width: 320px;">
        <template #trigger>
          <n-tag :type="statusType" size="small">{{ statusLabel }}</n-tag>
        </template>
        <span class="text-[13px]">{{ errorMessage }}</span>
      </NTooltip>
      <n-tag v-else :type="statusType" size="small">{{ statusLabel }}</n-tag>
    </template>

    <!-- Config strip -->
    <div class="flex items-center gap-3 text-[13px] text-neutral-400">
      <span class="inline-flex items-center gap-1.5">
        <NIcon :component="ServerOutline" :size="15" />
        {{ instance.vendor }}{{ instance.version ? ` ${instance.version}` : '' }}
      </span>
      <span v-if="gamePort" class="text-neutral-600">·</span>
      <span v-if="gamePort" class="inline-flex items-center gap-1.5">
        <NIcon :component="GitNetworkOutline" :size="15" />
        port {{ gamePort }}
      </span>
    </div>

    <!-- Memory + uptime row -->
    <div class="mt-5 flex items-center justify-between text-[13px] text-neutral-400">
      <span class="inline-flex items-center gap-1.5">
        <NIcon :component="LayersOutline" :size="16" />
        {{ memLabel }}
      </span>
      <span v-if="uptimeLabel" class="inline-flex items-center gap-1.5">
        <NIcon :component="TimeOutline" :size="14" />
        {{ uptimeLabel }}
      </span>
    </div>

    <!-- Divider -->
    <div class="my-4 border-t border-white/[0.08]" />

    <!-- Footer: join address + copy -->
    <div class="flex items-center justify-between gap-3">
      <span class="inline-flex items-center gap-1.5 font-mono text-[13px] text-neutral-400 truncate">
        <NIcon :component="GlobeOutline" :size="15" />
        {{ joinUrl ?? '—' }}
      </span>
      <NTooltip v-if="joinUrl" trigger="hover">
        <template #trigger>
          <NButton quaternary circle size="small" :aria-label="copied ? 'Copied' : 'Copy join address'"
            @click="copyJoin">
            <template #icon>
              <NIcon :component="copied ? CheckmarkOutline : CopyOutline" />
            </template>
          </NButton>
        </template>
        {{ copied ? 'Copied' : 'Copy address' }}
      </NTooltip>
    </div>
  </NCard>
</template>
