<script setup lang="ts">
import type { Component } from 'vue'
import {
  HomeOutline as HomeIcon,
  PersonOutline as PersonIcon,
  WineOutline as WineIcon
} from '@vicons/ionicons5'
import { NIcon } from 'naive-ui'
import { h } from 'vue'
import { darkTheme } from 'naive-ui'

function renderIcon(icon: Component) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

const menuOptions = [
  {
    label: 'Dashboard',
    key: 'dashboard',
    icon: renderIcon(HomeIcon)
  },
]

const selected = ref('dashboard');
</script>

<template>
  <n-config-provider :theme="darkTheme">
    <n-message-provider>
      <n-layout class="h-screen" content-class="flex flex-col">
        <!-- <n-layout-header class="p-6" bordered>
          <h1 class="text-lg font-bold my-auto">mcsd</h1>
        </n-layout-header> -->

        <n-layout has-sider>
          <n-layout-sider class="flex-grow" bordered collapse-mode="width" :collapsed-width="64" :width="240"
            :native-scrollbar="false">
            <div class="p-6">
              <h1 class="text-lg font-bold my-auto">mcsd</h1>
            </div>
            <n-menu :collapsed-width="64" :collapsed-icon-size="22" :options="menuOptions" :value="selected" />
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
