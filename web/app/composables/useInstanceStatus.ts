import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import type { Instance } from '~/services/api'

// Maps a raw systemd state (plus the InstanceStateError "degraded" marker
// from core.Instance) to what an operator should see. Single source of
// truth so status color/label stay consistent across every component that
// renders an instance's state.
export function useInstanceStatus(instance: MaybeRefOrGetter<Instance>) {
  const state = computed(() => toValue(instance).state)

  const isRunning = computed(() => state.value === 'active')
  const isTransitioning = computed(() => state.value === 'activating' || state.value === 'deactivating')

  const type = computed(() => {
    switch (state.value) {
      case 'active': return 'success'
      case 'activating':
      case 'deactivating': return 'warning'
      case 'failed':
      case 'error': return 'error'
      default: return 'default'
    }
  })

  const label = computed(() => {
    switch (state.value) {
      case 'active': return 'running'
      case 'inactive': return 'stopped'
      case 'activating': return 'starting'
      case 'deactivating': return 'stopping'
      default: return state.value
    }
  })

  return { label, type, isRunning, isTransitioning }
}
