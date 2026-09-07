<script setup lang="ts">
import { useMessage } from "naive-ui";
import {
    deleteInstance,
    patchInstance,
    useInit,
    useVitals,
    upgradeInstance,
    useCurrentInstance,
} from "~/services/api";
import type { Instance } from "~/services/api";

const emit = defineEmits<{
    instanceUpdated: [config: Instance];
    deleted: [];
}>();

const instance = useCurrentInstance();
const message = useMessage();

const initData = useInit();
const vendors = computed(() => initData.value?.vendors ?? []);
const javaBinaries = computed(() => initData.value?.java_binaries ?? []);

// Java binaries
const selectedJavaPath = ref<string>("");
const customJavaPath = ref("");
const isCustomJava = ref(false);

// Settings form
const nameInput = ref("");
const binaryInput = ref("");
const memoryInput = ref<number | null>(null);
const javaArgsInput = ref("");
const serverArgsInput = ref("");
const gamePortInput = ref<number | null>(null);
const rconPortInput = ref<number | null>(null);
const saving = ref(false);
const formInitialized = ref(false);

watch(
    instance,
    (inst) => {
        if (formInitialized.value) return;
        formInitialized.value = true;
        nameInput.value = inst.name;
        binaryInput.value = inst.binary ?? "";
        const match = javaBinaries.value.find((j) => j.path === inst.binary);
        if (match) {
            selectedJavaPath.value = inst.binary;
            isCustomJava.value = false;
        } else if (inst.binary) {
            selectedJavaPath.value = "__custom__";
            customJavaPath.value = inst.binary;
            isCustomJava.value = true;
        }
        memoryInput.value = inst.memory;
        javaArgsInput.value = (inst.java_args ?? []).join(" ");
        serverArgsInput.value = (inst.server_args ?? []).join(" ");
        gamePortInput.value = inst.ports?.game ?? 25565;
        rconPortInput.value = inst.ports?.rcon ?? 25575;
    },
    { immediate: true },
);

const isJava = computed(() => instance.value.vendor !== "Bedrock");

watch(selectedJavaPath, (val) => {
    isCustomJava.value = val === "__custom__";
    if (isCustomJava.value) {
        binaryInput.value = customJavaPath.value;
    } else if (val) {
        binaryInput.value = val;
        customJavaPath.value = "";
    }
});

watch(customJavaPath, (val) => {
    if (isCustomJava.value) binaryInput.value = val;
});

const javaBinaryOptions = computed(() => {
    const options = javaBinaries.value.map((j) => ({
        label: `${j.description} — ${j.path}`,
        value: j.path,
    }));
    options.push({ label: "Custom path...", value: "__custom__" });
    return options;
});

async function saveSettings() {
    const trimmedName = nameInput.value.trim();
    if (!trimmedName) return;
    if (
        gamePortInput.value !== null &&
        gamePortInput.value === rconPortInput.value
    ) {
        message.error("Game port and RCON port must be different");
        return;
    }
    saving.value = true;
    const patch: Record<string, unknown> = {};
    if (trimmedName !== instance.value.name) patch.name = trimmedName;
    const trimmedBinary = binaryInput.value.trim();
    if (trimmedBinary !== (instance.value.binary ?? ""))
        patch.binary = trimmedBinary;
    if (
        memoryInput.value !== null &&
        memoryInput.value !== instance.value.memory
    )
        patch.memory = memoryInput.value;
    patch.java_args = javaArgsInput.value.trim().split(/\s+/).filter(Boolean);
    patch.server_args = serverArgsInput.value
        .trim()
        .split(/\s+/)
        .filter(Boolean);
    if (
        gamePortInput.value !== null &&
        gamePortInput.value !== (instance.value.ports?.game ?? 25565)
    )
        patch.game_port = gamePortInput.value;
    if (
        rconPortInput.value !== null &&
        rconPortInput.value !== (instance.value.ports?.rcon ?? 25575)
    )
        patch.rcon_port = rconPortInput.value;
    const updated = await patchInstance(instance.value.id, patch);
    if (!updated) {
        saving.value = false;
        return;
    }
    emit("instanceUpdated", updated);
    message.success("Settings saved");
    saving.value = false;
}

// Upgrade
const upgradeVersion = ref<string | null>(null);
const upgrading = ref(false);

const vitals = useVitals();

const { isRunning } = useInstanceStatus(instance);
const maxMemory = computed(() => vitals.value?.budget.total ?? undefined);

const upgradeVersionOptions = computed(() => {
    const vendor = vendors.value.find((v) => v.name === instance.value.vendor);
    const all = vendor?.versions ?? [];
    const idx = all.indexOf(instance.value.version ?? "");
    const newer = idx > 0 ? all.slice(0, idx) : all;
    return newer
        .filter((v) => v !== instance.value.version)
        .map((v) => ({ label: v, value: v }));
});

async function handleUpgrade() {
    if (!instance.value.vendor || !upgradeVersion.value) return;
    upgrading.value = true;
    const response = await upgradeInstance(
        instance.value.id,
        instance.value.vendor,
        upgradeVersion.value,
    );
    if (!response) {
        upgrading.value = false;
        return;
    }
    emit("instanceUpdated", response);
    message.success(
        `Upgraded to ${instance.value.vendor} ${upgradeVersion.value}`,
    );
    upgradeVersion.value = null;
    upgrading.value = false;
}

// Delete
async function handleDelete() {
    const ok = await deleteInstance(instance.value.id);
    if (!ok) return;
    message.success("Instance deleted");
    emit("deleted");
}
</script>

<template>
    <div class="mt-2 space-y-10 max-w-5xl">
        <div
            v-if="isRunning"
            class="text-sm text-amber-400 bg-amber-400/10 border border-amber-400/30 rounded px-3 py-2"
        >
            Stop the server to edit settings.
        </div>

        <!-- Configuration -->
        <section class="space-y-5">
            <h2 class="text-base font-semibold text-neutral-100">
                Configuration
            </h2>
            <div class="grid grid-cols-2 gap-4">
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <div class="text-sm text-neutral-400 mb-1.5">Name</div>
                        <n-input
                            v-model:value="nameInput"
                            placeholder="Instance name"
                            :disabled="isRunning"
                        />
                    </div>
                        <div>
                            <div class="text-sm text-neutral-400 mb-1.5">
                                Memory (MB)
                            </div>
                            <n-input-number
                                v-model:value="memoryInput"
                                :min="512"
                                :max="maxMemory"
                                :step="256"
                                class="w-full"
                                placeholder="e.g. 2048"
                                :disabled="isRunning"
                            />
                        </div>
                </div>
                <div>
                    <div v-if="isJava">
                        <div class="text-sm text-neutral-400 mb-1.5">Java binary</div>
                        <n-select
                            v-if="javaBinaries.length > 0"
                            v-model:value="selectedJavaPath"
                            :options="javaBinaryOptions"
                            placeholder="Select Java runtime"
                            :disabled="isRunning"
                        />
                        <n-input
                            v-if="isCustomJava"
                            v-model:value="customJavaPath"
                            placeholder="/usr/bin/java"
                            class="font-mono mt-2"
                            :disabled="isRunning"
                        />
                    </div>
                    <div v-else>
                        <div class="text-sm text-neutral-400 mb-1.5">Binary path</div>
                        <n-input
                            v-model:value="binaryInput"
                            placeholder="server"
                            class="font-mono"
                            :disabled="isRunning"
                        />
                    </div>
                </div>
            </div>
            <div class="space-y-4">
                <div>
                    <div class="text-sm text-neutral-400 mb-1.5">
                        Server args
                    </div>
                    <n-input
                        v-model:value="serverArgsInput"
                        placeholder="nogui"
                        class="font-mono"
                        :disabled="isRunning"
                    />
                </div>
                <div>
                    <div class="text-sm text-neutral-400 mb-1.5">Java args</div>
                    <n-input
                        v-model:value="javaArgsInput"
                        type="textarea"
                        placeholder="-Xms512M -Xmx2G -XX:+UseG1GC"
                        class="font-mono"
                        :autosize="true"
                        :disabled="isRunning"
                    />
                </div>
            </div>
            <n-button
                type="primary"
                :loading="saving"
                :disabled="isRunning"
                @click="saveSettings"
                >Save settings</n-button
            >
        </section>

        <!-- Network -->
        <section class="space-y-5">
            <h2 class="text-base font-semibold text-neutral-100">Network</h2>
            <div class="grid grid-cols-2 gap-4 w-1/2">
                <div>
                    <div class="text-sm text-neutral-400 mb-1.5">Game port</div>
                    <n-input-number
                        v-model:value="gamePortInput"
                        :min="1"
                        :max="65535"
                        :disabled="isRunning"
                        class="w-full"
                        placeholder="25565"
                    />
                </div>
                <div>
                    <div class="text-sm text-neutral-400 mb-1.5">RCON port</div>
                    <n-input-number
                        v-model:value="rconPortInput"
                        :min="1"
                        :max="65535"
                        :disabled="isRunning"
                        class="w-full"
                        placeholder="25575"
                    />
                </div>
            </div>
            <n-button
                type="primary"
                :loading="saving"
                :disabled="isRunning"
                @click="saveSettings"
                >Save settings</n-button
            >
        </section>

        <!-- Danger zone -->
        <section
            class="space-y-5 w-1/2 rounded-lg border border-dashed border-red-500/40 hover:border-solid hover:border-red-500/70 transition-colors p-5"
        >
            <h2
                class="text-sm font-semibold text-red-500 uppercase tracking-wider"
            >
                Danger Zone
            </h2>
            <div v-if="upgradeVersionOptions.length > 0" class="flex gap-4">
                <div class="w-fit">
                    <div class="text-sm text-neutral-400 mb-1.5">Vendor</div>
                    <div class="text-sm text-neutral-300 mt-2">
                        {{ instance.vendor }}
                    </div>
                </div>

                <div class="min-w-80">
                    <div class="text-sm text-neutral-400 mb-1.5">
                        Upgrade version
                    </div>
                    <div class="flex gap-4">
                        <n-select
                            v-model:value="upgradeVersion"
                            :options="upgradeVersionOptions"
                            placeholder="Select version"
                        />
                        <n-button
                            :loading="upgrading"
                            :disabled="!upgradeVersion || isRunning"
                            @click="handleUpgrade"
                        >
                            {{ isRunning ? "Stop to upgrade" : "Upgrade" }}
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
