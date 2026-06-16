<script setup lang="ts">
import { useMessage, NIcon, NButton, NTooltip } from 'naive-ui'
import { useEventSource, useElementBounding, useWindowSize } from '@vueuse/core'
import {
  TimeOutline, HardwareChipOutline, GlobeOutline,
  WifiOutline, CopyOutline, CheckmarkOutline,
} from '@vicons/ionicons5'
import { formatUptime, formatMemoryMB } from '~/utils/format'
import type { Instance } from '~/types/api'

const props = defineProps<{
  instance: Instance
  instanceId: string
  refresh: () => Promise<void>
}>()

const message = useMessage()
const actionLoading = ref<string | null>(null)
const isRunning = computed(() => props.instance.state === 'active')
const isTransitioning = computed(() => {
  const s = props.instance.state
  return s === 'activating' || s === 'deactivating'
})

async function doAction(actionKey: string, actionFn: () => Promise<unknown>, successMessage: string) {
  actionLoading.value = actionKey
  const result = await actionFn()
  if (result !== null) message.success(successMessage)
  actionLoading.value = null
  await props.refresh()
}

const start = () => doAction('start', () => startInstance(props.instanceId), 'Instance started')
const stop = () => doAction('stop', () => stopInstance(props.instanceId), 'Instance stopped')
const toggleAutostart = () => props.instance.enabled
  ? doAction('autostart', () => disableInstance(props.instanceId), 'Autostart disabled')
  : doAction('autostart', () => enableInstance(props.instanceId), 'Autostart enabled')

// Stats
const publicIp = ref<string | null>(null)
onMounted(async () => {
  const result = await getPublicIp()
  if (result) publicIp.value = result.ip
})
const gamePort = computed(() => props.instance.ports?.game ?? null)
const localJoinAddress = computed(() =>
  gamePort.value ? `${window.location.hostname}:${gamePort.value}` : null
)
const publicJoinAddress = computed(() =>
  publicIp.value && gamePort.value ? `${publicIp.value}:${gamePort.value}` : null
)
const memoryUsedPercent = computed(() => {
  const used = props.instance.memory_used
  const total = props.instance.memory
  if (!used || !total) return 0
  return Math.min(Math.round((used / total) * 100), 100)
})
function memoryColor(pct: number) {
  if (pct >= 90) return '#f2697a'
  if (pct >= 70) return '#f0a020'
  return '#63e2b7'
}
const copied = ref<'local' | 'public' | null>(null)
async function copyAddress(address: string, key: 'local' | 'public') {
  try {
    await navigator.clipboard.writeText(address)
    copied.value = key
    setTimeout(() => (copied.value = null), 1500)
  } catch { /* clipboard unavailable */ }
}

// Log viewer
const logLines = ref<string[]>([])
const logContainer = ref<HTMLElement | null>(null)
const rconInput = ref('')
const viewerRef = ref<HTMLElement | null>(null)
const { top: viewerTop } = useElementBounding(viewerRef)
const { height: windowHeight } = useWindowSize()
const BOTTOM_SAFE_AREA = 84
const availableHeight = computed(() =>
  Math.max(200, windowHeight.value - viewerTop.value - BOTTOM_SAFE_AREA)
)
const { data: sseData } = useEventSource(`/api/instances/${props.instanceId}/logs`, [], { autoReconnect: true })
watch(sseData, (line) => {
  if (line === null) return
  logLines.value.push(line)
  nextTick(() => {
    if (logContainer.value) logContainer.value.scrollTop = logContainer.value.scrollHeight
  })
})
async function sendRcon() {
  const command = rconInput.value.trim()
  if (!command) return
  logLines.value.push(`> ${command}`)
  rconInput.value = ''
  const result = await rconCommand(props.instanceId, command).catch(() => null)
  if (result?.response) {
    logLines.value.push(result.response)
    nextTick(() => {
      if (logContainer.value) logContainer.value.scrollTop = logContainer.value.scrollHeight
    })
  }
}
</script>

<template>
  <div class="space-y-6 mt-2">

    <!-- Controls -->
    <div class="flex flex-wrap gap-2">
      <n-button v-if="!isRunning" type="primary" :loading="actionLoading === 'start'"
        :disabled="isTransitioning || actionLoading !== null" @click="start">
        Start
      </n-button>
      <n-button v-if="isRunning" type="warning" :loading="actionLoading === 'stop'" :disabled="actionLoading !== null"
        @click="stop">
        Stop
      </n-button>
      <n-button :loading="actionLoading === 'autostart'" :disabled="actionLoading !== null" @click="toggleAutostart">
        {{ instance.enabled ? 'Disable autostart' : 'Enable autostart' }}
      </n-button>
    </div>

    <!-- Stats -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
      <InstanceStatCard label="Memory" :icon="HardwareChipOutline"
        :progress="isRunning && instance.memory_used != null ? memoryUsedPercent : undefined"
        :progress-color="memoryColor(memoryUsedPercent)">
        <template v-if="isRunning && instance.memory_used != null">
          {{ formatMemoryMB(instance.memory_used) }} / {{ formatMemoryMB(instance.memory) }}
        </template>
        <template v-else>{{ formatMemoryMB(instance.memory) }} alloc</template>
      </InstanceStatCard>

      <InstanceStatCard label="Uptime" :icon="TimeOutline"
        :value="isRunning && instance.uptime_seconds ? formatUptime(instance.uptime_seconds) : '—'" />

      <InstanceStatCard label="Local address" :icon="WifiOutline">
        <div class="flex items-center justify-between gap-2">
          <span class="font-mono text-xs truncate">{{ localJoinAddress ?? '—' }}</span>
          <NTooltip v-if="localJoinAddress" trigger="hover">
            <template #trigger>
              <NButton quaternary circle size="tiny" @click="copyAddress(localJoinAddress!, 'local')">
                <template #icon>
                  <NIcon :component="copied === 'local' ? CheckmarkOutline : CopyOutline" :size="13" />
                </template>
              </NButton>
            </template>
            {{ copied === 'local' ? 'Copied' : 'Copy' }}
          </NTooltip>
        </div>
      </InstanceStatCard>

      <InstanceStatCard label="Public address" :icon="GlobeOutline">
        <div class="flex items-center justify-between gap-2">
          <span class="font-mono text-xs truncate">{{ publicJoinAddress ?? '—' }}</span>
          <NTooltip v-if="publicJoinAddress" trigger="hover">
            <template #trigger>
              <NButton quaternary circle size="tiny" @click="copyAddress(publicJoinAddress!, 'public')">
                <template #icon>
                  <NIcon :component="copied === 'public' ? CheckmarkOutline : CopyOutline" :size="13" />
                </template>
              </NButton>
            </template>
            {{ copied === 'public' ? 'Copied' : 'Copy' }}
          </NTooltip>
        </div>
      </InstanceStatCard>
    </div>


    <!-- Log viewer -->
    <div ref="viewerRef" class="flex flex-col" :style="{ height: `${availableHeight}px` }">
      <div class="text-sm text-neutral-400 mb-2 shrink-0">Live logs</div>
      <div ref="logContainer"
        class="flex-1 min-h-0 bg-black rounded-t font-mono text-xs text-green-400 p-3 overflow-y-auto whitespace-pre-wrap break-all">
        <div v-for="(line, i) in logLines" :key="i">{{ line }}</div>
        <div v-if="logLines.length === 0" class="text-neutral-600">Waiting for logs…</div>
      </div>
      <div class="shrink-0 flex rounded-b border border-t-0 border-white/[0.08] overflow-hidden">
        <span class="bg-black text-green-400 font-mono text-xs px-3 flex items-center select-none">$</span>
        <input v-model="rconInput" class="flex-1 bg-black text-green-400 font-mono text-xs px-2 py-2 outline-none"
          placeholder="Enter command…" :disabled="!isRunning" @keydown.enter="sendRcon" />
        <n-button size="small" :disabled="!isRunning || !rconInput.trim()" class="rounded-none" @click="sendRcon">
          Send
        </n-button>
      </div>
    </div>

  </div>
</template>
