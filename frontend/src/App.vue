<template>
  <Toaster />
  <Sidebar
    :isLoading="false"
    :open="sidebarOpen"
    :userTeams="userStore.teams"
    :userViews="userViews"
    @update:open="sidebarOpen = $event"
    @create-view="openCreateViewForm = true"
    @edit-view="editView"
    @delete-view="deleteView"
  >
    <ResizablePanelGroup direction="horizontal" auto-save-id="app.vue.resizable.panel">
      <ResizableHandle id="resize-handle-1" />
      <ResizablePanel id="resize-panel-2">
        <div class="w-full h-screen">
          <PageHeader />
          <RouterView />
        </div>
      </ResizablePanel>
      <ViewForm v-model:openDialog="openCreateViewForm" v-model:view="view" />
    </ResizablePanelGroup>
  </Sidebar>
  <div class="font-jakarta">
    <Command />
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { RouterView, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { initWS } from '@/websocket.js'
import { Toaster } from '@/components/ui/sonner'
import { useToast } from '@/components/ui/toast/use-toast'
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from '@/components/ui/resizable'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { useConversationStore } from './stores/conversation'
import { useInboxStore } from '@/stores/inbox'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import { useSlaStore } from '@/stores/sla'
import PageHeader from './components/common/PageHeader.vue'
import ViewForm from '@/components/ViewForm.vue'
import api from '@/api'
import Sidebar from '@/components/sidebar/Sidebar.vue'
import Command from '@/components/command/command.vue'

const { toast } = useToast()
const emitter = useEmitter()
const sidebarOpen = ref(true)
const userStore = useUserStore()
const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamStore = useTeamStore()
const inboxStore = useInboxStore()
const slaStore = useSlaStore()
const router = useRouter()
const userViews = ref([])
const view = ref({})
const openCreateViewForm = ref(false)

initWS()
onMounted(() => {
  initToaster()
  listenViewRefresh()
  getCurrentUser()
  getUserViews()
  initStores()
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.SHOW_TOAST, toast)
  emitter.off(EMITTER_EVENTS.REFRESH_LIST, refreshViews)
})

const initStores = async () => {
  await Promise.all([
    conversationStore.fetchStatuses(),
    conversationStore.fetchPriorities(),
    usersStore.fetchUsers(),
    teamStore.fetchTeams(),
    inboxStore.fetchInboxes(),
    slaStore.fetchSlas()
  ])
}

const editView = (v) => {
  view.value = { ...v }
  openCreateViewForm.value = true
}

const deleteView = async (view) => {
  try {
    await api.deleteView(view.id)
    emitter.emit(EMITTER_EVENTS.REFRESH_LIST, { model: 'view' })
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Success',
      variant: 'success',
      description: 'View deleted successfully'
    })
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  }
}

const getUserViews = async () => {
  try {
    const response = await api.getCurrentUserViews()
    userViews.value = response.data.data
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  }
}

const getCurrentUser = () => {
  userStore.getCurrentUser().catch((err) => {
    if (err.response && err.response.status === 401) {
      router.push('/')
    }
  })
}

const initToaster = () => {
  emitter.on(EMITTER_EVENTS.SHOW_TOAST, toast)
}

const listenViewRefresh = () => {
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, refreshViews)
}

const refreshViews = (data) => {
  openCreateViewForm.value = false
  // TODO: move model to constants.
  if (data?.model === 'view') {
    getUserViews()
  }
}
</script>
