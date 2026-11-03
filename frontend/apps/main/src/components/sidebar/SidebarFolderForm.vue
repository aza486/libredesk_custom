<script setup>
import { computed, provide, ref, watch } from 'vue'
import api from '@main/api'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter
} from '@shared-ui/components/ui/dialog'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import FilterGroupBuilder from '@main/components/filter/FilterGroupBuilder.vue'
import { useConversationFilters } from '@main/composables/useConversationFilters'
import {
  createRoot,
  normalizeToTwoLevel,
  deserializeFilterTree,
  serializeFilterTree,
  collectLeaves,
  isPartialLeaf
} from '@main/components/filter/filterTree'
import { handleHTTPError } from '@shared-ui/utils/http'
const props = defineProps({
  scope: String,
  scopeId: Number,
  scopeName: String,
  folder: Object,
  folderCount: Number
})
const open = defineModel('open', { default: false })
const emit = defineEmits(['saved', 'deleted'])
const { conversationsListFilters } = useConversationFilters()
const fields = computed(() =>
  Object.entries(conversationsListFilters.value).map(([field, config]) => ({
    ...config,
    field,
    model: config.model || 'conversations',
    options: config.options || []
  }))
)
const name = ref('')
const filters = ref(createRoot())
const showCount = ref(true)
const position = ref(1)
const busy = ref(false)
const error = ref('')
const confirmingDelete = ref(false)
const validateTick = ref(0)
provide('filterValidateTick', validateTick)
watch(open, (value) => {
  if (!value) return
  name.value = props.folder?.name || ''
  filters.value = props.folder
    ? deserializeFilterTree(normalizeToTwoLevel(props.folder.filters), fields.value)
    : createRoot()
  showCount.value = props.folder?.show_count ?? true
  position.value = props.folder ? props.folder.sort_order + 1 : props.folderCount + 1
  confirmingDelete.value = false
  error.value = ''
})
async function save() {
  const leaves = collectLeaves(filters.value)
  if (!leaves.length || leaves.some(isPartialLeaf)) {
    validateTick.value++
    error.value = 'Bitte alle Filterbedingungen vollständig ausfüllen.'
    return
  }
  if (
    name.value.trim().length < 2 ||
    name.value.trim().length > 140 ||
    !Number.isInteger(position.value) ||
    position.value < 1
  ) {
    error.value = 'Bitte einen Namen mit 2 bis 140 Zeichen und eine gültige Position eingeben.'
    return
  }
  busy.value = true
  error.value = ''
  try {
    const payload = {
      name: name.value.trim(),
      filters: serializeFilterTree(filters.value),
      show_count: showCount.value,
      sort_order: position.value - 1
    }
    if (props.folder)
      await api.updateSidebarFolder(props.scope, props.scopeId, props.folder.id, payload)
    else await api.createSidebarFolder(props.scope, props.scopeId, payload)
    open.value = false
    emit('saved')
  } catch (e) {
    error.value = handleHTTPError(e).message
  } finally {
    busy.value = false
  }
}
async function remove() {
  busy.value = true
  try {
    await api.deleteSidebarFolder(props.scope, props.scopeId, props.folder.id)
    open.value = false
    emit('deleted', props.folder.id)
  } catch (e) {
    error.value = handleHTTPError(e).message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent class="max-w-3xl max-h-[90vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>{{ folder ? 'Ordner bearbeiten' : 'Ordner / Filter hinzufügen' }}</DialogTitle>
        <DialogDescription
          >Passende Konversationen erscheinen automatisch im ersten passenden Ordner dieses
          Bereichs.</DialogDescription
        >
      </DialogHeader>
      <form class="space-y-4" @submit.prevent="save">
        <label class="block space-y-1 text-sm"
          >Ordnername<Input v-model="name" required minlength="2" maxlength="140" autofocus
        /></label>
        <label class="block space-y-1 text-sm"
          >Bereich<Input :model-value="scopeName" readonly
        /></label>
        <div class="space-y-1 text-sm">
          <span>Filter</span><FilterGroupBuilder v-model="filters" :fields="fields" />
        </div>
        <label class="flex items-center gap-2 text-sm"
          ><input v-model="showCount" type="checkbox" />Anzahl der Konversationen anzeigen</label
        >
        <label class="block space-y-1 text-sm"
          >Priorität / Position (1 = höchste Priorität)<Input
            v-model.number="position"
            type="number"
            min="1"
            :max="folderCount + (folder ? 0 : 1)"
            required
        /></label>
        <p v-if="error" role="alert" class="text-sm text-destructive">{{ error }}</p>
        <div v-if="confirmingDelete" role="alert" class="space-y-3 rounded border p-3 text-sm">
          <p>
            Möchten Sie diesen Ordner wirklich löschen? Die enthaltenen Konversationen werden nicht
            gelöscht und erscheinen anschließend automatisch in den passenden Ordnern.
          </p>
          <Button type="button" variant="destructive" :disabled="busy" @click="remove"
            >Löschen bestätigen</Button
          >
          <Button type="button" variant="ghost" :disabled="busy" @click="confirmingDelete = false"
            >Abbrechen</Button
          >
        </div>
        <DialogFooter>
          <Button
            v-if="folder"
            type="button"
            variant="destructive"
            :disabled="busy"
            @click="confirmingDelete = true"
            >Löschen</Button
          >
          <Button type="button" variant="outline" :disabled="busy" @click="open = false"
            >Abbrechen</Button
          >
          <Button type="submit" :disabled="busy">Speichern</Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>
