<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Plus, Trash2, X } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription
} from '@shared-ui/components/ui/dialog'
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel
} from '@shared-ui/components/ui/alert-dialog'
import api from '@main/api'
import { useMessageTemplates, templateError } from './useMessageTemplates'
import MessageTemplateEditor from './MessageTemplateEditor.vue'
import TemplateAttachments from './TemplateAttachments.vue'
const props = defineProps({ open: Boolean, conversationUUID: String, identity: String })
const emit = defineEmits(['update:open', 'changed'])
const { templates, canSupport, loading, error, accessDenied, load, clear } = useMessageTemplates(
  () => props.conversationUUID,
  () => props.identity
)
const scope = ref('personal')
const search = ref('')
const selected = ref(null)
const name = ref('')
const content = ref('')
const attachments = ref([])
const attachmentEditor = ref(null)
const uploading = ref(false)
const baseline = ref('')
const saving = ref(false)
const fetching = ref(false)
const message = ref('')
const confirm = ref(null)
const formOpen = ref(false)
let pending = null
let generation = 0
const snapshot = () =>
  JSON.stringify([name.value, content.value, attachments.value.map((file) => file.uuid)])
const dirty = computed(() => formOpen.value && snapshot() !== baseline.value)
const busy = computed(() => saving.value || fetching.value || uploading.value)
const list = computed(() =>
  templates.value
    .filter(
      (t) =>
        t.scope === scope.value &&
        t.name.toLocaleLowerCase().includes(search.value.toLocaleLowerCase())
    )
    .sort((a, b) => a.name.localeCompare(b.name, 'de', { sensitivity: 'base' }))
)
const reset = (item = null, creating = false) => {
  attachmentEditor.value?.resetUploads()
  attachments.value = item?.attachments || []
  selected.value = item
  name.value = item?.name || ''
  content.value = item?.content || ''
  baseline.value = snapshot()
  formOpen.value = !!item || creating
  message.value = ''
}
const guard = (action) => {
  if (busy.value) return
  if (dirty.value) {
    pending = action
    confirm.value = 'discard'
  } else action()
}
const accept = () => {
  const action = pending
  pending = null
  confirm.value = null
  action?.()
}
const requestClose = () => guard(() => emit('update:open', false))
const choose = (item) =>
  guard(async () => {
    const request = ++generation
    fetching.value = true
    error.value = ''
    try {
      const { data } = await api.getMessageTemplate(item.uuid, props.conversationUUID)
      if (request === generation) reset(data.data)
    } catch (err) {
      if (request === generation) {
        reset()
        clear()
        error.value = templateError(err)
      }
    } finally {
      if (request === generation) fetching.value = false
    }
  })
const newTemplate = () => guard(() => reset(null, true))
const changeScope = (value) =>
  guard(() => {
    scope.value = value
    reset()
  })
const save = async () => {
  if (busy.value) return
  const request = ++generation
  saving.value = true
  error.value = ''
  message.value = ''
  const input = {
    name: name.value,
    content: content.value,
    scope: scope.value,
    revision: selected.value?.revision || 0,
    attachment_ids: attachments.value.map((file) => file.id)
  }
  try {
    const response = selected.value
      ? await api.updateMessageTemplate(selected.value.uuid, input, props.conversationUUID)
      : await api.createMessageTemplate(input, props.conversationUUID)
    if (request !== generation) return
    attachmentEditor.value?.markSaved(response.data.data.attachments)
    reset(response.data.data)
    message.value = 'Vorlage gespeichert.'
    emit('changed')
    await load()
  } catch (err) {
    if (request !== generation) return
    error.value = templateError(err)
    if ([401, 403, 404].includes(err?.response?.status)) {
      reset()
      clear()
      error.value = templateError(err)
    }
  } finally {
    if (request === generation) saving.value = false
  }
}
const remove = () => {
  if (!selected.value || busy.value) return
  confirm.value = 'delete'
  pending = async () => {
    const request = ++generation
    saving.value = true
    error.value = ''
    try {
      await api.deleteMessageTemplate(
        selected.value.uuid,
        selected.value.revision,
        props.conversationUUID
      )
      if (request !== generation) return
      reset()
      message.value = 'Vorlage gelöscht.'
      emit('changed')
      await load()
    } catch (err) {
      if (request === generation) {
        if ([401, 403, 404].includes(err?.response?.status)) {
          reset()
          clear()
        }
        error.value = templateError(err)
      }
    } finally {
      if (request === generation) saving.value = false
    }
  }
}
const refresh = async () => {
  const loaded = await load()
  if (!loaded && !accessDenied.value) return
  // Revalidate any open document as well as the list after tab focus. Preserve
  // authorized local edits, but discard protected data when authorization ends.
  if (
    selected.value &&
    !loading.value &&
    !templates.value.some((t) => t.uuid === selected.value.uuid)
  )
    reset()
  if (scope.value === 'customer_support' && !canSupport.value) {
    scope.value = 'personal'
    reset()
  }
}
const beforeUnload = (event) => {
  if (!dirty.value) return
  event.preventDefault()
  event.returnValue = ''
}
const onVisibility = () => {
  if (document.visibilityState === 'visible' && !busy.value) refresh()
}
watch([() => props.identity, () => props.conversationUUID], () => {
  generation++
  reset()
  clear()
  emit('update:open', false)
})
watch(
  () => props.open,
  (value) => {
    if (!value) {
      generation++
      reset()
      clear()
    }
  }
)
onMounted(() => {
  refresh()
  window.addEventListener('beforeunload', beforeUnload)
  window.addEventListener('focus', onVisibility)
})
onBeforeUnmount(() => {
  generation++
  window.removeEventListener('beforeunload', beforeUnload)
  window.removeEventListener('focus', onVisibility)
})
</script>
<template>
  <Dialog
    :open="open"
    @update:open="
      (value) => {
        if (!value) requestClose()
      }
    "
  >
    <DialogContent
      class="p-0 gap-0 flex flex-col overflow-hidden top-0 left-0 translate-x-0 translate-y-0 w-full max-w-none h-[var(--visual-viewport-height,100dvh)] max-h-none rounded-none md:top-4 md:left-4 md:w-[calc(100%-2rem)] md:h-[calc(var(--visual-viewport-height,100dvh)-2rem)] md:rounded-lg"
      hide-close-button
      @escape-key-down.prevent.stop="requestClose"
      @interact-outside.prevent
      @close-auto-focus.prevent
    >
      <header class="flex flex-wrap items-center gap-3 border-b p-4 shrink-0">
        <DialogTitle class="text-xl font-semibold flex-1">Vorlagen verwalten</DialogTitle>
        <Button type="button" :disabled="busy" @click="newTemplate"
          ><Plus class="mr-2 size-4" />Neue Vorlage</Button
        >
        <Button
          type="button"
          variant="ghost"
          aria-label="Vorlagenverwaltung schließen"
          :disabled="busy"
          @click="requestClose"
          ><X class="size-5"
        /></Button>
        <DialogDescription class="sr-only"
          >Persönliche und gemeinsame Nachrichtenvorlagen bearbeiten.</DialogDescription
        >
      </header>
      <div class="min-h-0 flex-1 flex flex-col md:flex-row">
        <aside
          class="md:w-72 shrink-0 border-b md:border-b-0 md:border-r flex flex-col min-h-0 max-h-[35dvh] md:max-h-none"
        >
          <div class="p-3 space-y-3 shrink-0">
            <div class="flex gap-1" aria-label="Vorlagenbereich">
              <Button
                type="button"
                size="sm"
                :variant="scope === 'personal' ? 'secondary' : 'ghost'"
                :aria-pressed="scope === 'personal'"
                :disabled="busy"
                @click="changeScope('personal')"
                >Meine Vorlagen</Button
              >
              <Button
                v-if="canSupport"
                type="button"
                size="sm"
                :variant="scope === 'customer_support' ? 'secondary' : 'ghost'"
                :aria-pressed="scope === 'customer_support'"
                :disabled="busy"
                @click="changeScope('customer_support')"
                >Kundensupport</Button
              >
            </div>
            <Input
              v-model="search"
              placeholder="Vorlagennamen suchen …"
              aria-label="Vorlagennamen suchen"
            />
          </div>
          <nav class="overflow-y-auto min-h-0 p-2" aria-label="Vorlagenliste" :aria-busy="loading">
            <p v-if="loading" class="p-2 text-sm text-muted-foreground" role="status">
              Vorlagen werden geladen …
            </p>
            <p v-else-if="!list.length" class="p-2 text-sm text-muted-foreground">
              {{
                search
                  ? 'Keine passenden Vorlagen gefunden.'
                  : 'Noch keine Vorlagen. Erstellen Sie Ihre erste Vorlage.'
              }}
            </p>
            <button
              v-for="item in list"
              :key="item.uuid"
              type="button"
              class="block w-full rounded-md p-2 text-left text-sm hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              :class="{ 'bg-accent': selected?.uuid === item.uuid }"
              :aria-current="selected?.uuid === item.uuid ? 'true' : undefined"
              :title="item.name"
              :disabled="busy"
              @click="choose(item)"
            >
              <span class="block truncate">{{ item.name }}</span>
            </button>
          </nav>
        </aside>
        <section class="flex-1 min-w-0 min-h-0 flex flex-col p-4 gap-3 overflow-y-auto">
          <p v-if="error" role="alert" class="text-sm text-destructive">
            {{ error }}
            <Button type="button" size="sm" variant="outline" :disabled="busy" @click="refresh"
              >Liste neu laden</Button
            >
          </p>
          <p v-if="message" role="status" class="text-sm text-muted-foreground">{{ message }}</p>
          <p v-if="fetching" role="status">Vorlage wird geladen …</p>
          <form
            v-else-if="formOpen"
            class="flex flex-col gap-3 min-h-0 flex-1"
            @submit.prevent="save"
          >
            <label for="message-template-name" class="text-sm font-medium">Vorlagenname</label>
            <Input
              id="message-template-name"
              v-model="name"
              :disabled="busy"
              maxlength="160"
              required
            />
            <p class="text-xs text-muted-foreground">
              {{
                scope === 'personal'
                  ? 'Nur für Sie sichtbar, unabhängig vom Postfach.'
                  : 'Änderungen gelten für alle berechtigten Kundensupportmitarbeiter.'
              }}
              Platzhalter wie [Name] bleiben bearbeitbarer Text.
            </p>
            <MessageTemplateEditor v-model="content" :disabled="busy" class="flex-1 min-h-60" />
            <TemplateAttachments
              ref="attachmentEditor"
              v-model="attachments"
              :conversationUUID="conversationUUID"
              :disabled="saving || fetching"
              @busy="uploading = $event"
            />
            <div class="flex flex-wrap gap-2 shrink-0">
              <Button type="submit" :disabled="busy || !name.trim()" :is-loading="saving"
                >Speichern</Button
              >
              <Button type="button" variant="outline" :disabled="busy" @click="guard(() => reset())"
                >Abbrechen</Button
              >
              <Button
                v-if="selected"
                type="button"
                variant="destructive"
                :disabled="busy"
                @click="remove"
                ><Trash2 class="mr-2 size-4" />Löschen</Button
              >
            </div>
          </form>
          <p v-else class="m-auto text-sm text-muted-foreground">
            Wählen Sie eine Vorlage aus oder erstellen Sie eine neue Vorlage.
          </p>
        </section>
      </div>
    </DialogContent>
  </Dialog>
  <AlertDialog
    :open="!!confirm"
    @update:open="
      (value) => {
        if (!value) {
          confirm = null
          pending = null
        }
      }
    "
  >
    <AlertDialogContent @escape-key-down.stop>
      <AlertDialogHeader>
        <AlertDialogTitle>{{
          confirm === 'delete' ? 'Vorlage löschen?' : 'Ungespeicherte Änderungen verwerfen?'
        }}</AlertDialogTitle>
        <AlertDialogDescription>{{
          confirm === 'delete'
            ? 'Diese Vorlage wird dauerhaft gelöscht. Gemeinsame Vorlagen werden für alle Mitarbeiter entfernt.'
            : 'Ihre Änderungen an der Vorlage wurden noch nicht gespeichert.'
        }}</AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>Abbrechen</AlertDialogCancel>
        <Button type="button" variant="destructive" @click="accept">{{
          confirm === 'delete' ? 'Löschen' : 'Verwerfen'
        }}</Button>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>
