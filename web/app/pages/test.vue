<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { NConfigProvider, NGlobalStyle, darkTheme, type GlobalThemeOverrides } from 'naive-ui'

// Mirror of the interface inside ServerCard.vue. Consider exporting it from
// there (`export interface ServerInstance`) and importing it instead of
// re-declaring, so the two never drift apart.
type ServerStatus = 'active' | 'starting' | 'stopping' | 'stopped' | 'crashed'
interface ServerInstance {
    id: string
    name: string
    status: ServerStatus
    loader: string
    version: string
    port: number
    joinUrl: string
    players: { online: number; max: number } | null
    cpu: number | null
    memory: { usedMb: number; allocatedMb: number } | null
    uptimeSeconds: number | null
    cpuHistory?: number[]
    connected: boolean
    lastUpdated: number | null
}

// Nudge NaiveUI's dark theme toward the green accent from your UI.
const themeOverrides: GlobalThemeOverrides = {
    common: {
        primaryColor: '#4ade80',
        primaryColorHover: '#5eead4',
        successColor: '#4ade80',
    },
}

const now = Date.now()
const servers = ref<ServerInstance[]>([
    {
        id: 'srv_test',
        name: 'Test',
        status: 'active',
        loader: 'fabric',
        version: '26.1.2',
        port: 25565,
        joinUrl: 'localhost:25565',
        players: { online: 12, max: 20 },
        cpu: 34,
        memory: { usedMb: 2150, allocatedMb: 3000 },
        uptimeSeconds: 367200, // 4d 6h
        cpuHistory: [28, 31, 24, 40, 33, 45, 30, 34],
        connected: true,
        lastUpdated: now,
    },
    {
        id: 'srv_skyblock',
        name: 'SkyBlock',
        status: 'active',
        loader: 'paper',
        version: '1.21.4',
        port: 25567,
        joinUrl: 'play.example.net:25567',
        players: { online: 38, max: 40 },
        cpu: 88, // high load → cpu + memory go amber/red
        memory: { usedMb: 7400, allocatedMb: 8000 },
        uptimeSeconds: 1900,
        cpuHistory: [62, 71, 80, 76, 90, 85, 92, 88],
        connected: false, // feed dropped → dot greys, footer reads "reconnecting…"
        lastUpdated: now - 9000,
    },
    {
        id: 'srv_modded',
        name: 'Modded SMP',
        status: 'starting',
        loader: 'forge',
        version: '1.20.1',
        port: 25568,
        joinUrl: 'localhost:25568',
        players: { online: 0, max: 10 },
        cpu: 12,
        memory: { usedMb: 900, allocatedMb: 6000 },
        uptimeSeconds: 15,
        cpuHistory: [4, 8, 6, 14, 10, 18, 12, 12],
        connected: true,
        lastUpdated: now,
    },
    {
        id: 'srv_creative',
        name: 'Creative',
        status: 'stopped',
        loader: 'paper',
        version: '1.21.4',
        port: 25566,
        joinUrl: 'localhost:25566',
        players: null,
        cpu: null,
        memory: null,
        uptimeSeconds: null,
        connected: false,
        lastUpdated: null,
    },
])

// ── Fake live feed: jitter the running servers every 2s ───────────────
let feed: ReturnType<typeof setInterval> | undefined
const clamp = (n: number, lo = 0, hi = 100) => Math.min(hi, Math.max(lo, n))

onMounted(() => {
    feed = setInterval(() => {
        const t = Date.now()
        servers.value = servers.value.map((s) => {
            const running = s.status === 'active' || s.status === 'starting'
            if (!running) return s
            const cpu = clamp(Math.round((s.cpu ?? 0) + (Math.random() * 16 - 8)))
            const history = [...(s.cpuHistory ?? []).slice(-7), cpu]
            const used = s.memory
                ? clamp(
                    Math.round(s.memory.usedMb + (Math.random() * 120 - 60)),
                    200,
                    s.memory.allocatedMb,
                )
                : null
            return {
                ...s,
                cpu,
                cpuHistory: history,
                memory: s.memory ? { ...s.memory, usedMb: used! } : null,
                uptimeSeconds: (s.uptimeSeconds ?? 0) + 2,
                // The stale-feed demo (SkyBlock) stays disconnected on purpose.
                lastUpdated: s.connected ? t : s.lastUpdated,
            }
        })
    }, 2000)
})
onBeforeUnmount(() => clearInterval(feed))

// In your app this is where you'd route to the detail page.
function openServer(id: string) {
    console.log('open server →', id)
    // router.push({ name: 'server', params: { id } })
}
</script>

<template>
    <NConfigProvider :theme="darkTheme" :theme-overrides="themeOverrides">
        <NGlobalStyle />
        <div class="page">
            <h2 class="page-title">Instances</h2>
            <div class="grid">
                <ServerCard v-for="server in servers" :key="server.id" :server="server" @open="openServer" />
            </div>
        </div>
    </NConfigProvider>
</template>

<style scoped>
.page {
    min-height: 100vh;
    background: #0b0e14;
    padding: 2rem 2.5rem;
}

.page-title {
    font-size: 22px;
    font-weight: 600;
    color: #f3f4f6;
    margin: 0 0 1.25rem;
}

.grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(440px, 1fr));
    gap: 1rem;
    max-width: 1100px;
}
</style>