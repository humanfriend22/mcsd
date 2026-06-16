import { useIntervalFn } from '@vueuse/core'
import type { Instance } from '~/types/api'

export function useInstance(id: string) {
  const instance = ref<Instance | null>(null)
  const notFound = ref(false)

  async function refresh() {
    const data = await getInstance(id)
    if (data) {
      instance.value = data
    } else if (!instance.value) {
      notFound.value = true
    }
  }

  onMounted(refresh)
  useIntervalFn(refresh, 1000)

  return { instance, notFound, refresh }
}
