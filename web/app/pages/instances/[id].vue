<script setup lang="ts">
import { InstanceKey, useInstance, loadInstance } from '~/services/api'

const route = useRoute()
const router = useRouter()
const instanceId = route.params.id as string

const activeTab = ref('overview')
const instance = useInstance(instanceId)

provide(InstanceKey, instance)
</script>

<template>
  <div v-if="!instance" class="text-center py-12">
    <p class="text-neutral-400 mb-4">Instance not found.</p>
    <n-button @click="router.push('/')">Go to dashboard</n-button>
  </div>

  <div v-else>
    <InstanceHeader class="mb-6" />

    <n-tabs v-model:value="activeTab" type="line" animated>

      <n-tab-pane name="overview" tab="Overview">
        <InstanceOverviewTab />
      </n-tab-pane>

      <n-tab-pane name="files" tab="Files">
        <FileBrowser :instance-id="instanceId" />
      </n-tab-pane>

      <n-tab-pane name="mods" tab="Mods">
        <div class="py-16 text-center space-y-2">
          <p class="text-neutral-300 font-medium">Mod manager coming soon</p>
          <p class="text-neutral-500 text-sm">
            Will support browsing and installing mods from Modrinth and CurseForge.
          </p>
        </div>
      </n-tab-pane>

      <n-tab-pane name="settings" tab="Settings">
        <InstanceSettingsTab @instance-updated="(inst) => loadInstance(inst.id)" @deleted="router.push('/')" />
      </n-tab-pane>

    </n-tabs>
  </div>
</template>
