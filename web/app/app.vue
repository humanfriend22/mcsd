<script setup lang="ts">
import type { Component } from 'vue'
import { h } from 'vue'
import { HomeOutline as HomeIcon, ServerOutline as ServersIcon, ChevronBackOutline } from '@vicons/ionicons5'
import { NIcon, darkTheme } from 'naive-ui'
import { useInstances, useInit } from '~/api'


function renderIcon(icon: Component) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

function renderDot(state: string) {
  const color = state === 'active' ? '#63e2b7' : state === 'failed' ? '#f2697a' : '#525252'
  return () => h('span', {
    style: {
      display: 'inline-block',
      width: '7px',
      height: '7px',
      borderRadius: '50%',
      background: color,
      flexShrink: '0',
    }
  })
}

const router = useRouter()
const route = useRoute()

const { instances } = useInstances()
useInit()

const menuOptions = computed(() => [
  {
    label: 'Dashboard',
    key: 'dashboard',
    icon: renderIcon(HomeIcon),
  },
  {
    label: 'Servers',
    key: 'servers',
    icon: siderCollapsed.value ? renderIcon(ChevronBackOutline) : renderIcon(ServersIcon),
    children: instances.value.map(inst => ({
      label: inst.name,
      key: `instance-${inst.id}`,
      icon: renderDot(inst.state),
    })),
  },
])

const selectedKey = computed(() =>
  route.path.startsWith('/instances/') ? `instance-${route.params.id}` : 'dashboard'
)

const siderCollapsed = ref(false)
const expandedKeys = ref(['servers'])

function onSelect(key: string) {
  if (key === 'dashboard') router.push('/')
  else if (key.startsWith('instance-')) router.push(`/instances/${key.slice('instance-'.length)}`)
}
</script>

<template>
  <n-config-provider :theme="darkTheme">
    <n-message-provider>
      <n-layout class="h-screen" content-class="flex flex-col">
        <n-layout has-sider>
          <n-layout-sider bordered collapse-mode="width" :collapsed-width="64" :width="240"
            :native-scrollbar="false" show-trigger v-model:collapsed="siderCollapsed">
            <div class="p-6">
              <h1 class="text-lg font-bold">mcsd</h1>
            </div>
            <n-menu
              :collapsed-width="64"
              :collapsed-icon-size="22"
              :options="menuOptions"
              :value="selectedKey"
              v-model:expanded-keys="expandedKeys"
              @update:value="onSelect"
            />
          </n-layout-sider>
          <n-layout-content content-class="w-full grow p-5">
            <NuxtPage />
          </n-layout-content>
        </n-layout>

        <n-layout-footer style="height: 64px; padding: 24px" bordered>
          城府路
        </n-layout-footer>
      </n-layout>
    </n-message-provider>
  </n-config-provider>
</template>
