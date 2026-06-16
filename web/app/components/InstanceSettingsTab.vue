<script setup lang="ts">
import { useMessage } from 'naive-ui'
import type { Instance, Vendor } from '~/types/api'

const props = defineProps<{ instance: Instance }>()
const emit = defineEmits<{
  instanceUpdated: [instance: Instance]
  deleted: []
}>()

const message = useMessage()

// Settings form
const nameInput = ref('')
const memoryInput = ref<number | null>(null)
const javaArgsInput = ref('')
const serverArgsInput = ref('')
const saving = ref(false)

watch(() => props.instance, (inst) => {
  if (!inst) return
  if (!nameInput.value) nameInput.value = inst.name
  if (memoryInput.value === null) memoryInput.value = inst.memory
  if (!javaArgsInput.value) javaArgsInput.value = (inst.java_args ?? []).join(' ')
  if (!serverArgsInput.value) serverArgsInput.value = (inst.server_args ?? []).join(' ')
}, { immediate: true })

async function saveSettings() {
  const trimmedName = nameInput.value.trim()
  if (!trimmedName) return
  saving.value = true
  const patch: Record<string, unknown> = {}
  if (trimmedName !== props.instance.name) patch.name = trimmedName
  if (memoryInput.value !== null && memoryInput.value !== props.instance.memory)
    patch.memory = memoryInput.value
  patch.java_args = javaArgsInput.value.trim().split(/\s+/).filter(Boolean)
  patch.server_args = serverArgsInput.value.trim().split(/\s+/).filter(Boolean)
  const updated = await patchInstance(props.instance.id, patch)
  if (updated) {
    emit('instanceUpdated', updated)
    message.success('Settings saved')
  }
  saving.value = false
}

// Upgrade
const vendors = ref<Vendor[]>([])
const upgradeVersion = ref<string | null>(null)
const upgrading = ref(false)

onMounted(async () => {
  vendors.value = await listVendors() ?? []
})

const isRunning = computed(() => props.instance.state === 'active')

const upgradeVersionOptions = computed(() => {
  const vendor = vendors.value.find(v => v.name === props.instance.vendor)
  const all = vendor?.versions ?? []
  const idx = all.indexOf(props.instance.version ?? '')
  const newer = idx > 0 ? all.slice(0, idx) : all
  return newer
    .filter(v => v !== props.instance.version)
    .map(v => ({ label: v, value: v }))
})

async function handleUpgrade() {
  if (!props.instance.vendor || !upgradeVersion.value) return
  upgrading.value = true
  const response = await upgradeInstance(props.instance.id, props.instance.vendor, upgradeVersion.value)
  if (response) {
    emit('instanceUpdated', response)
    message.success(`Upgraded to ${props.instance.vendor} ${upgradeVersion.value}`)
    upgradeVersion.value = null
  }
  upgrading.value = false
}

// Delete
async function handleDelete() {
  const result = await deleteInstance(props.instance.id)
  if (result !== null) {
    message.success('Instance deleted')
    emit('deleted')
  }
}
</script>

<template>
  <div class="mt-2 space-y-10 max-w-3xl">

    <!-- Configuration -->
    <section class="space-y-5">
      <h2 class="text-base font-semibold text-neutral-100">Configuration</h2>
      <div class="grid grid-cols-2 gap-4">
        <div>
          <div class="text-sm text-neutral-400 mb-1.5">Name</div>
          <n-input v-model:value="nameInput" placeholder="Instance name" />
        </div>
        <div>
          <div class="text-sm text-neutral-400 mb-1.5">Memory (MB)</div>
          <n-input-number v-model:value="memoryInput" :min="256" :step="256" class="w-full"
            placeholder="e.g. 2048" />
        </div>
        <div>
          <div class="text-sm text-neutral-400 mb-1.5">Java args</div>
          <n-input v-model:value="javaArgsInput" type="textarea" :autosize="{ minRows: 3 }"
            placeholder="-Xms512M -Xmx2G -XX:+UseG1GC" class="font-mono" />
        </div>
        <div>
          <div class="text-sm text-neutral-400 mb-1.5">Server args</div>
          <n-input v-model:value="serverArgsInput" type="textarea" :autosize="{ minRows: 3 }"
            placeholder="nogui" class="font-mono" />
        </div>
      </div>
      <n-button type="primary" :loading="saving" @click="saveSettings">Save settings</n-button>
    </section>

    <!-- Ports -->
    <section v-if="instance.ports" class="space-y-3">
      <h2 class="text-base font-semibold text-neutral-100">
        Ports
        <span class="text-neutral-600 ml-2 text-xs font-normal">(edit in server.properties via Files)</span>
      </h2>
      <div class="flex gap-8 text-sm">
        <div class="flex gap-3">
          <span class="text-neutral-500">Game</span>
          <span class="font-mono text-neutral-200">{{ instance.ports.game }}</span>
        </div>
        <div class="flex gap-3">
          <span class="text-neutral-500">RCON</span>
          <span class="font-mono text-neutral-200">{{ instance.ports.rcon }}</span>
        </div>
      </div>
    </section>

    <!-- Upgrade -->
    <n-card v-if="upgradeVersionOptions.length > 0" size="small">
      <template #header>
        <span class="text-base font-semibold text-neutral-100">Upgrade server</span>
      </template>
      <div class="flex items-center gap-3">
        <span class="text-sm text-neutral-400 shrink-0">{{ instance.vendor }}</span>
        <n-select v-model:value="upgradeVersion" :options="upgradeVersionOptions"
          placeholder="Select version" class="flex-1" />
        <n-button type="primary" :loading="upgrading" :disabled="!upgradeVersion || isRunning"
          class="shrink-0" @click="handleUpgrade">
          {{ isRunning ? 'Stop to upgrade' : 'Upgrade' }}
        </n-button>
      </div>
    </n-card>

    <!-- Danger zone -->
    <section class="space-y-3">
      <h2 class="text-sm font-semibold text-red-500 uppercase tracking-wider">Danger Zone</h2>
      <n-popconfirm @positive-click="handleDelete">
        <template #trigger>
          <n-button type="error">Delete instance</n-button>
        </template>
        Delete "{{ instance.name }}"? This cannot be undone.
      </n-popconfirm>
    </section>

  </div>
</template>
