import { loadInstanceData, loadInstanceStates, setInstanceConfig, useInstances } from '~/api'
import type { InstanceConfig } from '~/api'

export function useInstance(id: string) {
  const { instances, loaded } = useInstances()

  const instance = computed(() => instances.value.find(i => i.id === id) ?? null)
  const notFound = computed(() => loaded.value && !instance.value)

  function update(config?: InstanceConfig) {
    if (config) setInstanceConfig(id, config)
    else loadInstanceData()
  }

  return { instance, notFound, update, refresh: loadInstanceStates }
}
