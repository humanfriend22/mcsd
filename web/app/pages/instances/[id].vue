<script setup lang="ts">
const route = useRoute()
const router = useRouter()
const instanceId = route.params.id as string

const activeTab = ref('overview')
const { instance, notFound, update, refresh } = useInstance(instanceId)
</script>

<template>
  <div v-if="notFound" class="text-center py-12">
    <p class="text-neutral-400 mb-4">Instance not found.</p>
    <n-button @click="router.push('/')">Go to dashboard</n-button>
  </div>

  <div v-else>
    <InstanceHeader :instance="instance" :instance-id="instanceId" class="mb-6" />

    <n-tabs v-model:value="activeTab" type="line" animated>

      <n-tab-pane name="overview" tab="Overview">
        <InstanceOverviewTab v-if="instance" :instance="instance" :instance-id="instanceId" :refresh="refresh" />
        <n-spin v-else size="large" class="flex justify-center py-12" />
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
        <InstanceSettingsTab v-if="instance" :instance="instance" @instance-updated="update"
          @deleted="router.push('/')" />
        <n-spin v-else size="large" class="flex justify-center py-12" />
      </n-tab-pane>

    </n-tabs>
  </div>
</template>
