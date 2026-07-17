<script setup lang="ts">
import { useMessage } from 'naive-ui'
import { NIcon, NButton, NPopconfirm, NSpin } from 'naive-ui'
import {
  FolderOutline, DocumentOutline, TrashOutline,
  CloudUploadOutline, SaveOutline, CloseOutline
} from '@vicons/ionicons5'
import {
  deleteFile,
  listFiles,
  readFile,
  uploadFile,
  writeFile,
} from '~/api'
import type { FileEntry } from '~/api'

const props = defineProps<{ instanceId: string }>()

const message = useMessage()

const TEXT_EXTENSIONS = new Set([
  '.properties', '.json', '.yml', '.yaml', '.toml',
  '.txt', '.log', '.conf', '.cfg',
])

function isText(name: string): boolean {
  const dot = name.lastIndexOf('.')
  return dot !== -1 && TEXT_EXTENSIONS.has(name.slice(dot))
}

// Directory state
const currentPath = ref('')
const entries = ref<FileEntry[]>([])
const loadingDir = ref(false)

// File editor state
const selectedFile = ref<FileEntry | null>(null)
const editorContent = ref('')
const originalContent = ref('')
const loadingFile = ref(false)
const saving = ref(false)
const isDirty = computed(() => editorContent.value !== originalContent.value)

// Upload state
const isDragOver = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

async function navigate(path: string) {
  selectedFile.value = null
  currentPath.value = path
  loadingDir.value = true
  const data = await listFiles(props.instanceId, path)
  if (data) {
    entries.value = [...data].sort((a, b) => {
      if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1
      return a.name.localeCompare(b.name)
    })
  }
  loadingDir.value = false
}

onMounted(() => navigate(''))

// Breadcrumb segments for current directory
const breadcrumb = computed(() =>
  currentPath.value ? currentPath.value.split('/') : []
)

function pathUpTo(index: number) {
  return breadcrumb.value.slice(0, index + 1).join('/')
}

function entryPath(name: string) {
  return currentPath.value ? `${currentPath.value}/${name}` : name
}

async function openEntry(entry: FileEntry) {
  if (entry.is_dir) {
    navigate(entryPath(entry.name))
    return
  }
  selectedFile.value = entry
  if (!isText(entry.name)) return

  loadingFile.value = true
  const content = await readFile(props.instanceId, entryPath(entry.name))
  if (content) {
    editorContent.value = content
    originalContent.value = content
  }
  loadingFile.value = false
}

function closeEditor() {
  selectedFile.value = null
  editorContent.value = ''
  originalContent.value = ''
}

async function saveFile() {
  if (!selectedFile.value) return
  saving.value = true
  const ok = await writeFile(props.instanceId, entryPath(selectedFile.value.name), editorContent.value)
  if (ok) {
    originalContent.value = editorContent.value
    message.success('Saved')
  }
  saving.value = false
}

async function deleteEntry(entry: FileEntry) {
  const ok = await deleteFile(props.instanceId, entryPath(entry.name))
  if (ok !== null) {
    message.success(`Deleted ${entry.name}`)
    await navigate(currentPath.value)
  }
}

async function uploadFiles(files: FileList | File[]) {
  let n = 0
  for (const file of Array.from(files)) {
    const ok = await uploadFile(props.instanceId, currentPath.value, file)
    if (ok) n++
  }
  if (n > 0) {
    message.success(`Uploaded ${n} file${n > 1 ? 's' : ''}`)
    await navigate(currentPath.value)
  }
}

function onDragOver(e: DragEvent) { e.preventDefault(); isDragOver.value = true }
function onDragLeave(e: DragEvent) {
  if (!(e.currentTarget as Element).contains(e.relatedTarget as Node)) {
    isDragOver.value = false
  }
}
async function onDrop(e: DragEvent) {
  e.preventDefault()
  isDragOver.value = false
  if (e.dataTransfer?.files?.length) await uploadFiles(e.dataTransfer.files)
}

function triggerUpload() { fileInput.value?.click() }
async function onFileInput(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files?.length) await uploadFiles(input.files)
  input.value = ''
}

function fmtSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1048576).toFixed(1)} MB`
}

function fmtDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
}
</script>

<template>
  <div class="mt-2 max-w-6xl">
    <!-- Breadcrumb (always visible) -->
    <div class="flex items-center gap-1 mb-4 pl-1">
      <NButton text size="small" @click="navigate('')">root</NButton>
      <template v-for="(seg, i) in breadcrumb" :key="i">
        <span class="text-neutral-600">/</span>
        <NButton text size="small" @click="navigate(pathUpTo(i))">{{ seg }}</NButton>
      </template>

      <template v-if="selectedFile">
        <span class="text-neutral-600">/</span>
        <span class="text-neutral-100 text-sm font-medium">{{ selectedFile.name }}</span>
        <div class="ml-auto flex gap-2" v-if="isText(selectedFile.name)">
          <NButton size="small" :disabled="!isDirty" @click="closeEditor">
            <template #icon>
              <NIcon :component="CloseOutline" />
            </template>
            Discard
          </NButton>
          <NButton type="primary" size="small" :loading="saving" :disabled="!isDirty" @click="saveFile">
            <template #icon>
              <NIcon :component="SaveOutline" />
            </template>
            Save
          </NButton>
        </div>
      </template>
      <template v-else>
        <div class="ml-auto">
          <NButton size="small" @click="triggerUpload">
            <template #icon>
              <NIcon :component="CloudUploadOutline" />
            </template>
            Upload
          </NButton>
          <input ref="fileInput" type="file" multiple class="hidden" @change="onFileInput" />
        </div>
      </template>
    </div>

    <!-- File editor view -->
    <div v-if="selectedFile">
      <div v-if="loadingFile" class="flex justify-center py-12">
        <NSpin size="large" />
      </div>
      <div v-else-if="!isText(selectedFile.name)" class="py-12 text-center text-neutral-500 text-sm">
        Binary file — edit via the Files tab or download and re-upload.
      </div>
      <textarea v-else v-model="editorContent"
        class="w-full font-mono text-xs bg-black text-green-400 rounded p-3 outline-none resize-none"
        style="height: 500px; tab-size: 2;" spellcheck="false" />
    </div>

    <!-- Directory listing view -->
    <div v-else>

      <div v-if="loadingDir" class="flex justify-center py-12">
        <NSpin size="large" />
      </div>

      <div v-else class="rounded border border-white/[0.08] overflow-hidden relative"
        :class="isDragOver ? 'border-[#63e2b7]' : ''" @dragover="onDragOver" @dragleave="onDragLeave" @drop="onDrop">
        <!-- Drag overlay -->
        <div v-if="isDragOver"
          class="absolute inset-0 bg-[#63e2b7]/10 flex items-center justify-center z-10 pointer-events-none rounded">
          <span class="text-[#63e2b7] text-sm font-medium">Drop to upload</span>
        </div>

        <div v-if="entries.length === 0" class="py-12 text-center text-neutral-500 text-sm">
          Empty directory
        </div>

        <div v-for="entry in entries" :key="entry.name"
          class="flex items-center gap-3 px-4 py-2.5 border-b border-white/[0.05] last:border-0 hover:bg-white/[0.03] cursor-pointer group"
          @click="openEntry(entry)">
          <NIcon :component="entry.is_dir ? FolderOutline : DocumentOutline" :size="16"
            :class="entry.is_dir ? 'text-yellow-400' : 'text-neutral-400'" />
          <span class="flex-1 text-sm truncate"
            :class="entry.is_dir ? 'text-neutral-200 font-medium' : 'text-neutral-300'">
            {{ entry.name }}
          </span>
          <span v-if="!entry.is_dir" class="text-xs text-neutral-600 w-16 text-right">
            {{ fmtSize(entry.size) }}
          </span>
          <span class="text-xs text-neutral-600 w-28 text-right hidden sm:block">
            {{ fmtDate(entry.modified) }}
          </span>
          <NPopconfirm @positive-click.stop="deleteEntry(entry)" @click.stop>
            <template #trigger>
              <NButton quaternary circle size="tiny"
                class="opacity-0 group-hover:opacity-100 transition-opacity text-neutral-500 hover:text-red-400"
                @click.stop>
                <template #icon>
                  <NIcon :component="TrashOutline" />
                </template>
              </NButton>
            </template>
            Delete {{ entry.name }}{{ entry.is_dir ? ' and all its contents' : '' }}?
          </NPopconfirm>
        </div>
      </div>
    </div>
  </div>
</template>
