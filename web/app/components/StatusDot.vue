<script setup lang="ts">
const { state } = defineProps<{ state: string }>()

const color = computed(() =>
  state === 'active' ? '#63e2b7'
    : state === 'failed' ? '#f2697a'
    : '#525252'
)
</script>

<template>
  <span :style="{
    display: 'inline-block',
    width: '7px',
    height: '7px',
    borderRadius: '50%',
    background: color,
    flexShrink: '0',
  }" />
</template>
