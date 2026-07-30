<script setup lang="ts">
import { useMessage, NIcon, NButton, NTooltip } from 'naive-ui'
import {
  TimeOutline, HardwareChipOutline, GlobeOutline,
  WifiOutline, CopyOutline, CheckmarkOutline,
} from '@vicons/ionicons5'
import { formatUptimeSince, formatMemoryMB } from '~/utils/format'
import {
  startInstance,
  stopInstance,
  disableInstance,
  enableInstance,
  useInit,
  useCurrentInstance,
} from '~/services/api'

const instance = useCurrentInstance()

const message = useMessage()
const actionLoading = ref<string | null>(null)
const isRunning = computed(() => instance.value.state === 'active')
const isTransitioning = computed(() => {
  const s = instance.value.state
  return s === 'activating' || s === 'deactivating'
})
const initData = useInit()
const publicIp = computed(() => initData.value?.public_ip ?? null)

async function doAction(actionKey: string, actionFn: () => Promise<unknown>, successMessage: string) {
  actionLoading.value = actionKey
  const result = await actionFn()
  if (result !== null) message.success(successMessage)
  actionLoading.value = null
}

const start = () => doAction('start', () => startInstance(instance.value.id), 'Instance started')
const stop = () => doAction('stop', () => stopInstance(instance.value.id), 'Instance stopped')
const toggleAutostart = () => instance.value.enabled
  ? doAction('autostart', () => disableInstance(instance.value.id), 'Autostart disabled')
  : doAction('autostart', () => enableInstance(instance.value.id), 'Autostart enabled')

const gamePort = computed(() => instance.value.ports?.game ?? null)
const localJoinAddress = computed(() =>
  gamePort.value ? `${window.location.hostname}:${gamePort.value}` : null
)
const publicJoinAddress = computed(() =>
  publicIp.value && gamePort.value ? `${publicIp.value}:${gamePort.value}` : null
)
const memoryUsedPercent = computed(() => {
  const used = instance.value.memory_used
  const total = instance.value.memory
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
        :value="isRunning && instance.active_since ? formatUptimeSince(instance.active_since) : '—'" />

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
    <LogViewer :instance-id="instance.id" :is-running="isRunning" />

  </div>
</template>
