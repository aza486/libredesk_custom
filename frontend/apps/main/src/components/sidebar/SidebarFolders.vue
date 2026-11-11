<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { FolderPen, Plus } from 'lucide-vue-next'
import { SidebarMenuSubItem, SidebarMenuButton } from '@shared-ui/components/ui/sidebar'
import { Badge } from '@shared-ui/components/ui/badge'
import api from '@main/api'
import { useEmitter } from '@main/composables/useEmitter'
import { useConversationStore } from '@main/stores/conversation'
import { handleHTTPError } from '@shared-ui/utils/http'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import SidebarFolderForm from './SidebarFolderForm.vue'
import { moveFolder } from './sidebarFolderOrder'
const props = defineProps({
  scope: String,
  scopeId: { type: Number, default: 0 },
  scopeName: String,
  unfiledLabel: String,
  addLabel: {type:String,default:"Ordner / Filter hinzufügen"}
})
const route = useRoute()
const router = useRouter()
const emitter = useEmitter()
const store = useConversationStore()
const folders = ref([])
const counts = ref({})
const open = ref(false)
const selected = ref(null)
const dragging = ref(null)
const busy = ref(false)
const error = ref('')
let loading = false
let disposed = false
function notify(e) {
  error.value = handleHTTPError(e).message
}
async function load() {
  if (loading) return
  loading = true
  try {
    const [definitions, totals] = await Promise.all([
      api.getSidebarFolders(props.scope, props.scopeId),
      api.getSidebarFolderCounts(props.scope, props.scopeId)
    ])
    if (disposed) return
    folders.value = definitions.data.data
    counts.value = totals.data.data
    error.value = ''
    // Deleted by another employee: return to the area's standard list.
    if (isCurrentScope() && route.query.folder) {
      const active = folders.value.find((f) => String(f.id) === route.query.folder)
      if (!active && !(props.scope === 'service_mail' && route.query.folder === '-1')) navigate()
      else if (active && route.query.folderName !== active.name)
        router.replace({ query: { ...route.query, folderName: active.name } })
    }
  } catch (e) {
    if (disposed) return
    notify(e)
    if ([403, 404].includes(e?.response?.status)) {
      folders.value = []
      counts.value = {}
    }
  } finally {
    loading = false
  }
}
function isCurrentScope() {
  if (props.scope === 'personal') return Number(route.params.inboxID) === props.scopeId
  if (props.scope === 'team') return Number(route.params.teamID) === props.scopeId
  if (props.scope === 'redistributed') return ['customer-redistributed', 'customer-redistributed-unfiled'].includes(route.params.type)
  if (props.scope === 'service_mail') return route.params.type === 'service-mails'
  if (props.scope === 'my_inbox') return route.params.type === 'assigned'
  return route.params.type === 'customer'
}
function navigate(folder) {
  const query = folder ? { folder: String(folder.id), folderName: folder.name } : ['redistributed', 'service_mail'].includes(props.scope) ? {} : { status: 'Open' }
  const params =
    props.scope === 'personal'
      ? { inboxID: props.scopeId }
      : props.scope === 'team'
        ? { teamID: props.scopeId }
        : { type: props.scope === 'service_mail' ? 'service-mails' : props.scope === 'redistributed' ? 'customer-redistributed' : props.scope === 'my_inbox' ? 'assigned' : 'customer' }
  const name =
    props.scope === 'personal' ? 'personal-inbox' : props.scope === 'team' ? 'team-inbox' : 'inbox'
  router.push({ name, params, query })
}
function showUnfiled() {
 if (props.scope === 'service_mail') router.push({name:'inbox',params:{type:'service-mails'},query:{folder:'-1',folderName:props.unfiledLabel}})
 else router.push({name:'inbox',params:{type:'customer-redistributed-unfiled'},query:{}})
}
function edit(folder = null) {
  selected.value = folder
  open.value = true
}
async function changed(deletedID) {
  if (isCurrentScope() && String(deletedID) === route.query.folder) navigate()
  else if (isCurrentScope() && selected.value && String(selected.value.id) === route.query.folder) {
    // Fetch the updated name after saving; the list always uses server filters.
    await load()
    const folder = folders.value.find((f) => f.id === selected.value.id)
    if (folder) router.replace({ query: { ...route.query, folderName: folder.name } })
  }
  emitter.emit('sidebar-folders-refresh')
  emitter.emit(EMITTER_EVENTS.REFRESH_LIST, { model: 'sidebar-folder' })
  store.refreshConversationListForRealtimeEvent()
}
async function reorder(id, targetIndex) {
  if (busy.value) return
  const ids = moveFolder(
    folders.value.map((f) => f.id),
    id,
    targetIndex
  )
  busy.value = true
  try {
    const response = await api.reorderSidebarFolders(props.scope, props.scopeId, ids)
    folders.value = response.data.data
    await changed()
  } catch (e) {
    notify(e)
    await load()
  } finally {
    busy.value = false
  }
}
onMounted(() => {
  load()
  emitter.on('sidebar-folders-refresh', load)
})
onUnmounted(() => {
  disposed = true
  emitter.off('sidebar-folders-refresh', load)
})
</script>

<template>
  <SidebarMenuSubItem v-if="['redistributed', 'service_mail'].includes(scope)">
    <SidebarMenuButton size="sm" :is-active="scope === 'service_mail' ? isCurrentScope() && route.query.folder === '-1' : route.params.type === 'customer-redistributed-unfiled'" @click="showUnfiled">
      <span class="min-w-0 truncate">{{ unfiledLabel }}</span>
      <Badge variant="outline" class="ml-auto shrink-0 tabular-nums bg-background px-1.5 py-0 font-medium">{{ counts[-1] || 0 }}</Badge>
    </SidebarMenuButton>
  </SidebarMenuSubItem>
  <SidebarMenuSubItem
    v-for="(folder, index) in folders"
    :key="folder.id"
    class="group/folder"
    draggable="true"
    @dragstart="dragging = folder.id"
    @dragend="dragging = null"
    @dragover.prevent
    @drop.prevent="dragging && reorder(dragging, index)"
  >
    <div
      class="group/folder flex items-center rounded-md hover:bg-sidebar-accent focus-within:bg-sidebar-accent"
      :class="{ 'bg-sidebar-accent': isCurrentScope() && route.query.folder === String(folder.id) }"
      @keydown.alt.up.prevent="reorder(folder.id, index - 1)"
      @keydown.alt.down.prevent="reorder(folder.id, index + 1)"
    >
      <SidebarMenuButton
        size="sm"
        class="min-w-0 flex-1"
        :is-active="isCurrentScope() && route.query.folder === String(folder.id)"
        :title="`${folder.name} – Alt+Pfeil zum Umsortieren`"
        @click="navigate(folder)"
      >
        <span class="truncate">{{ folder.name }}</span>
      </SidebarMenuButton>
      <button
        type="button"
        class="shrink-0 rounded p-1 text-muted-foreground hover:text-foreground focus-visible:outline focus-visible:outline-2"
        :aria-label="`${folder.name} bearbeiten`"
        @click="edit(folder)"
      >
        <FolderPen class="size-3.5" />
      </button>
      <Badge
        v-if="folder.show_count"
        variant="outline"
        class="mr-1 shrink-0 tabular-nums bg-background px-1.5 py-0 font-medium"
        >{{ counts[folder.id] || 0 }}</Badge
      >
    </div>
  </SidebarMenuSubItem>
  <SidebarMenuSubItem>
    <SidebarMenuButton size="sm" class="text-muted-foreground" @click="edit()"
      ><Plus class="size-3.5" /><span>{{ addLabel }}</span></SidebarMenuButton
    >
    <p v-if="error" role="alert" class="px-2 text-xs text-destructive">{{ error }}</p>
  </SidebarMenuSubItem>
  <SidebarFolderForm
    v-model:open="open"
    :scope="scope"
    :scope-id="scopeId"
    :scope-name="scopeName"
    :folder="selected"
    :folder-count="folders.length"
    @saved="changed()"
    @deleted="changed"
  />
</template>
