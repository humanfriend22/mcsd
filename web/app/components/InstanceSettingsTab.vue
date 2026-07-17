<script setup lang="ts">
import { useMessage } from 'naive-ui'
import {
  deleteInstance,
  patchInstance,
  useInit,
  upgradeInstance,
} from '~/api'
import type { Instance, InstanceConfig } from '~/api'

const props = defineProps<{ instance: Instance }>()
const emit = defineEmits<{
  instanceUpdated: [config: InstanceConfig]
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
const upgradeVersion = ref<string | null>(null)
const upgrading = ref(false)

const { vendors } = useInit()

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
  <div class="mt-2 space-y-10 max-w-5xl">

    <!-- Configuration -->
    <section class="space-y-5">
      <h2 class="text-base font-semibold text-neutral-100">Configuration</h2>
      <div class="grid grid-cols-3 gap-4">
        <div>
          <div class="text-sm text-neutral-400 mb-1.5">Name</div>
          <n-input v-model:value="nameInput" placeholder="Instance name" />
        </div>
        <div>
          <div class="text-sm text-neutral-400 mb-1.5">Memory (MB)</div>
          <n-input-number v-model:value="memoryInput" :min="256" :step="256" class="w-full" placeholder="e.g. 2048" />
        </div>
        <div>
          <div class="text-sm text-neutral-400 mb-1.5">Server args</div>
          <n-input v-model:value="serverArgsInput" placeholder="nogui" class="font-mono" />
        </div>
      </div>
      <div class="">
        <div>
          <div class="text-sm text-neutral-400 mb-1.5">Java args</div>
          <n-input v-model:value="javaArgsInput" type="textarea" placeholder="-Xms512M -Xmx2G -XX:+UseG1GC"
            class="font-mono" :autosize="true" />
        </div>
      </div>
      <n-button type="primary" :loading="saving" @click="saveSettings">Save settings</n-button>
    </section>

    <!-- Danger zone -->
    <section class="space-y-5 w-1/2 rounded-lg border border-dashed border-red-500/40 hover:border-solid hover:border-red-500/70 transition-colors p-5">
      <h2 class="text-sm font-semibold text-red-500 uppercase tracking-wider">Danger Zone</h2>
      <div v-if="upgradeVersionOptions.length > 0" class="flex gap-4">
        <div class="w-fit">
          <div class="text-sm text-neutral-400 mb-1.5">Vendor</div>
          <div class="text-sm text-neutral-300 mt-2">{{ instance.vendor }}</div>
        </div>

        <div class="min-w-80">
          <div class="text-sm text-neutral-400 mb-1.5">Upgrade version</div>
          <div class="flex gap-4">
            <n-select v-model:value="upgradeVersion" :options="upgradeVersionOptions" placeholder="Select version" />
            <n-button :loading="upgrading" :disabled="!upgradeVersion || isRunning" @click="handleUpgrade">
              {{ isRunning ? 'Stop to upgrade' : 'Upgrade' }}
            </n-button>
          </div>
        </div>
      </div>
      <n-popconfirm @positive-click="handleDelete">
        <template #trigger>
          <n-button type="error">Delete instance</n-button>
        </template>
        Delete "{{ instance.name }}"? This cannot be undone.
      </n-popconfirm>
    </section>

  </div>
</template>
