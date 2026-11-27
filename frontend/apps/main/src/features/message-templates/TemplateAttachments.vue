<script setup>
import { onBeforeUnmount, ref } from 'vue'
import { File, FileImage, FileText, FileSpreadsheet, Download, X, Plus } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { formatBytes } from '@shared-ui/utils/file'
import api from '@main/api'
import { templateError } from './useMessageTemplates'
const files = defineModel({ default: () => [] })
const props = defineProps({ conversationUUID: String, disabled: Boolean })
const emit = defineEmits(['busy'])
const picker = ref(null)
const uploads = ref([])
const error = ref('')
const dragging = ref(false)
const staged = new Map()
const controllers = new Set()
let generation = 0
const icon = (file) =>
  file.content_type?.startsWith('image/')
    ? FileImage
    : /pdf|word|text/.test(file.content_type)
      ? FileText
      : /excel|sheet/.test(file.content_type)
        ? FileSpreadsheet
        : File
const discard = (file) => {
  api.discardMessageTemplateAttachment(file.uuid, props.conversationUUID).catch(() => {})
}
const markSaved = (saved) => {
  for (const file of saved || []) staged.delete(file.uuid)
}
const cancelPending = () => {
  generation++
  for (const controller of controllers) controller.abort()
  controllers.clear()
  uploads.value = []
  emit('busy', false)
}
const resetUploads = () => {
  cancelPending()
  for (const file of staged.values()) discard(file)
  staged.clear()
}
const add = async (selected) => {
  if (props.disabled || uploads.value.length) return
  const chosen = [...(selected || [])]
  if (!chosen.length) return
  const request = generation
  uploads.value = chosen.map((file, id) => ({ file, id, progress: 0 }))
  const queue = [...uploads.value]
  let cursor = 0
  error.value = ''
  emit('busy', true)
  const worker = async () => {
    while (cursor < queue.length && request === generation) {
      const entry = queue[cursor++]
      const controller = new AbortController()
      controllers.add(controller)
      try {
        const { data } = await api.uploadMessageTemplateAttachment(
          entry.file,
          props.conversationUUID,
          {
            signal: controller.signal,
            onUploadProgress: (event) => {
              entry.progress = event.total ? Math.round((event.loaded / event.total) * 100) : 0
            }
          }
        )
        if (request !== generation) {
          discard(data.data)
          continue
        }
        const file = data.data
        staged.set(file.uuid, file)
        const merged = new Map(files.value.map((item) => [item.uuid, item]))
        for (const stagedFile of staged.values()) merged.set(stagedFile.uuid, stagedFile)
        files.value = [...merged.values()]
      } catch (err) {
        if (request === generation && err.code !== 'ERR_CANCELED') {
          error.value = [error.value, `${entry.file.name}: ${templateError(err)}`]
            .filter(Boolean)
            .join('\n')
        }
      } finally {
        controllers.delete(controller)
        if (request === generation)
          uploads.value = uploads.value.filter((item) => item.id !== entry.id)
      }
    }
  }
  await Promise.all(Array.from({ length: Math.min(2, queue.length) }, worker))
  if (request === generation) emit('busy', false)
}
const onDrop = (event) => {
  dragging.value = false
  add(event.dataTransfer.files)
}
const onPick = (event) => {
  add(event.target.files)
  event.target.value = ''
}
const remove = (file) => {
  files.value = files.value.filter((item) => item.uuid !== file.uuid)
  if (staged.has(file.uuid)) {
    staged.delete(file.uuid)
    discard(file)
  }
}
onBeforeUnmount(resetUploads)
defineExpose({ markSaved, resetUploads })
</script>
<template>
  <section
    class="rounded-md border p-3 shrink-0 space-y-2"
    :class="{ 'border-primary bg-accent': dragging }"
    aria-label="Vorlagenanhänge"
    @dragover.prevent="dragging = true"
    @dragleave.prevent="dragging = false"
    @drop.prevent.stop="onDrop"
  >
    <div class="flex items-center justify-between gap-2">
      <h3 class="text-sm font-medium">Anhänge</h3>
      <input
        ref="picker"
        type="file"
        multiple
        class="hidden"
        :disabled="disabled || !!uploads.length"
        @change="onPick"
      />
      <Button
        type="button"
        size="sm"
        variant="outline"
        :disabled="disabled || !!uploads.length"
        @click="picker.click()"
        ><Plus class="mr-1 size-4" />Dateien hinzufügen</Button
      >
    </div>
    <p v-if="!files.length && !uploads.length" class="text-xs text-muted-foreground">
      Dateien auswählen oder hierher ziehen. Änderungen werden mit der Vorlage gespeichert.
    </p>
    <p v-if="error" role="alert" class="text-sm text-destructive whitespace-pre-line">
      {{ error }}
    </p>
    <ul class="max-h-40 overflow-y-auto space-y-1">
      <li
        v-for="file in files"
        :key="file.uuid"
        class="flex items-center gap-2 rounded border bg-background px-2 py-1 text-sm"
      >
        <component :is="icon(file)" class="size-4 shrink-0 text-muted-foreground" />
        <span class="min-w-0 flex-1 truncate" :title="file.filename">{{ file.filename }}</span>
        <span class="text-xs text-muted-foreground shrink-0">{{ formatBytes(file.size) }}</span>
        <a
          :href="file.url"
          target="_blank"
          rel="noopener noreferrer"
          :aria-label="`${file.filename} öffnen oder herunterladen`"
          class="rounded p-1 hover:bg-accent"
          ><Download class="size-4"
        /></a>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          :aria-label="`${file.filename} entfernen`"
          :disabled="disabled"
          @click="remove(file)"
          ><X class="size-4"
        /></Button>
      </li>
      <li v-for="entry in uploads" :key="entry.id" class="text-sm flex items-center gap-2">
        <span class="truncate flex-1">{{ entry.file.name }}</span>
        <progress
          :value="entry.progress"
          max="100"
          class="w-24"
          :aria-label="`Upload ${entry.file.name}`"
        />
        <span class="text-xs">{{ entry.progress }} %</span>
      </li>
    </ul>
    <Button v-if="uploads.length" type="button" size="sm" variant="outline" @click="cancelPending"
      >Uploads abbrechen</Button
    >
  </section>
</template>
