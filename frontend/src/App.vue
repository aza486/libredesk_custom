<template>
  <div class="flex">
    <!-- Main sidebar -->
    <Sidebar
      class="w-full"
      :userTeams="userStore.teams"
      :userViews="userViews"
      @create-view="openCreateViewForm = true"
      @edit-view="editView"
      @delete-view="deleteView"
    >
      <div class="h-screen border-l">
        <PageHeader />
        <RouterView />
      </div>
      <ViewForm v-model:openDialog="openCreateViewForm" v-model:view="view" />
    </Sidebar>
  </div>
  <!-- Command box -->
  <Command />
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { RouterView } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { initWS } from '@/websocket.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { useConversationStore } from './stores/conversation'
import { useInboxStore } from '@/stores/inbox'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import { useSlaStore } from '@/stores/sla'
import { useMacroStore } from '@/stores/macro'
import { useTagStore } from '@/stores/tag'
import PageHeader from './components/layout/PageHeader.vue'
import ViewForm from '@/features/view/ViewForm.vue'
import api from '@/api'
import { toast as sooner } from 'vue-sonner'
import Sidebar from '@/components/sidebar/Sidebar.vue'
import Command from '@/features/command/CommandBox.vue'

const emitter = useEmitter()
const userStore = useUserStore()
const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamStore = useTeamStore()
const inboxStore = useInboxStore()
const slaStore = useSlaStore()
const macroStore = useMacroStore()
const tagStore = useTagStore()
const userViews = ref([])
const view = ref({})
const openCreateViewForm = ref(false)

initWS()
onMounted(() => {
  initToaster()
  listenViewRefresh()
  initStores()
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.SHOW_TOAST, toast)
  emitter.off(EMITTER_EVENTS.REFRESH_LIST, refreshViews)
})

// initialize data stores
const initStores = async () => {
  await Promise.allSettled([
    userStore.getCurrentUser(),
    getUserViews(),
    conversationStore.fetchStatuses(),
    conversationStore.fetchPriorities(),
    usersStore.fetchUsers(),
    teamStore.fetchTeams(),
    inboxStore.fetchInboxes(),
    slaStore.fetchSlas(),
    macroStore.loadMacros(),
    tagStore.fetchTags()
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

const initToaster = () => {
  emitter.on(EMITTER_EVENTS.SHOW_TOAST, (message) => {
    if (message.variant === 'destructive') {
      sooner.error(message.description)
    } else {
      sooner.success(message.description)
    }
  })
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
