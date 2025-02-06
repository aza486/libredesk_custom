<template>
  <div>
    <SidebarProvider :style="{ '--sidebar-width': '20rem' }" :open="conversationSidebarOpen">
      <Sidebar side="right" collapsible="offcanvas" variant="sidebar">
        <SidebarContent>
          <SidebarGroup style="padding: 0">
            <SidebarMenu>
              <SidebarMenuItem>
                <ConversationSideBar v-if="conversationStore.current" />
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>

        <SidebarRail>
          <button
            v-if="!conversationSidebarOpen"
            @click="toggleSidebar"
            class="absolute right-full top-16 p-1 -mr-2 rounded-l-full bg-white text-primary border hover:bg-opacity-90 transition-all duration-200 shadow-lg group"
          >
            <ChevronLeft size="16" class="group-hover:scale-110 transition-transform" />
          </button>
        </SidebarRail>
      </Sidebar>
    </SidebarProvider>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted } from 'vue'
import { ChevronLeft } from 'lucide-vue-next'
import ConversationSideBar from './ConversationSideBar.vue'
import { useConversationStore } from '@/stores/conversation'
import { useEmitter } from '@/composables/useEmitter'
import { useStorage } from '@vueuse/core'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarMenu,
  SidebarMenuItem,
  SidebarProvider,
  SidebarRail
} from '@/components/ui/sidebar'

const conversationStore = useConversationStore()
const emitter = useEmitter()
const conversationSidebarOpen = useStorage('conversationSidebarOpen', true)

const toggleSidebar = () => {
  conversationSidebarOpen.value = !conversationSidebarOpen.value
}

onMounted(() => {
  emitter.on(EMITTER_EVENTS.CONVERSATION_SIDEBAR_TOGGLE, toggleSidebar)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.CONVERSATION_SIDEBAR_TOGGLE)
})
</script>
