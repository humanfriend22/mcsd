<script setup lang="ts">
import { NButton } from 'naive-ui'
import { useElementBounding, useWindowSize } from '@vueuse/core'
import { rconCommand, useInstanceLogs } from '~/services/api'

const { instanceId, isRunning } = defineProps<{
  instanceId: string
  isRunning: boolean
}>()

const viewerRef = ref<HTMLElement | null>(null)
const logContainer = ref<HTMLElement | null>(null)
const rconInput = ref('')
const { top: viewerTop } = useElementBounding(viewerRef)
const { height: windowHeight } = useWindowSize()
const BOTTOM_SAFE_AREA = 84
const availableHeight = computed(() =>
  Math.max(200, windowHeight.value - viewerTop.value - BOTTOM_SAFE_AREA)
)

const { logs, error, open, close } = useInstanceLogs(instanceId)

onMounted(() => open())
onUnmounted(() => close())

let scrolled = false
watch(logs, () => {
  if (!scrolled) { scrolled = true; return }
  nextTick(() => {
    if (logContainer.value) logContainer.value.scrollTop = logContainer.value.scrollHeight
  })
}, { deep: true })

async function sendRcon() {
  const command = rconInput.value.trim()
  if (!command) return
  // logs.value.push(`> ${command}`)
  rconInput.value = ''
  const result = await rconCommand(instanceId, command)
  if (!result) return
  logs.value.push(result.response)
  nextTick(() => {
    if (logContainer.value) logContainer.value.scrollTop = logContainer.value.scrollHeight
  })
}
</script>

<template>
  <div ref="viewerRef" class="flex flex-col" :style="{ height: `${availableHeight}px` }">
    <div class="flex items-center justify-between mb-2 shrink-0">
      <span class="text-sm text-neutral-400">Live logs</span>
    </div>
    <div ref="logContainer"
      class="flex-1 min-h-0 bg-black rounded-t font-mono text-xs text-green-400 p-3 overflow-y-auto whitespace-pre-wrap break-all">
      <div v-for="(line, i) in logs" :key="i" :class="error && i === 0 ? 'text-neutral-500 italic' : ''">{{ line }}
      </div>
      <div v-if="logs.length === 0 && !error" class="text-neutral-600">Waiting for logs…</div>
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
</template>