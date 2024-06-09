<template>
  <TooltipProvider :delay-duration=200>
    <div class="bg-background text-foreground">
      <div v-if="$route.path !== '/login'">
        <ResizablePanelGroup direction="horizontal" auto-save-id="app.vue.resizable.panel">
          <ResizablePanel id="resize-panel-1" collapsible :default-size="10" :collapsed-size="1" :min-size="3"
            :max-size="20" :class="cn(isCollapsed && 'min-w-[50px] transition-all duration-200 ease-in-out')"
            @expand="toggleNav(false)" @collapse="toggleNav(true)">
            <NavBar :is-collapsed="isCollapsed" :links="navLinks" />
          </ResizablePanel>
          <ResizableHandle id="resize-handle-1" />
          <ResizablePanel id="resize-panel-2">
            <div class="w-full h-screen">
              <RouterView />
            </div>
          </ResizablePanel>
        </ResizablePanelGroup>
      </div>
      <div v-else>
        <RouterView />
      </div>
    </div>
  </TooltipProvider>
</template>

<script setup>
import { ref, onMounted } from "vue"
import { RouterView, useRouter } from 'vue-router'
import { cn } from '@/lib/utils'
import { useUserStore } from '@/stores/user'
import { initWS } from "./websocket.js"
import api from '@/api';

import NavBar from './components/NavBar.vue'
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from '@/components/ui/resizable'
import {
  TooltipProvider,
} from '@/components/ui/tooltip'


// State
const isCollapsed = ref(false)
const navLinks = [
  {
    title: 'Dashboard',
    component: 'dashboard',
    label: '',
    icon: 'lucide:layout-dashboard',
  },
  {
    title: 'Conversations',
    component: 'conversations',
    label: '',
    icon: 'lucide:message-circle-more',
  },
  {
    title: 'Account',
    component: 'account',
    label: '',
    icon: 'lucide:circle-user-round',
  },
]
const userStore = useUserStore()
const router = useRouter()

// Functions, methods
function toggleNav (v) {
  isCollapsed.value = v
}

onMounted(() => {
  getCurrentUser()
  initWS()
})

const getCurrentUser = () => {
  api.getCurrentUser().then((resp) => {
    if (resp.data.data) {
      userStore.$patch((state) => {
        state.userAvatar = resp.data.data.avatar_url
        state.userFirstName = resp.data.data.first_name
        state.userLastName = resp.data.data.last_name
      })
    }
  }).catch((error) => {
    if (error.response.status === 401) {
      router.push("/login")
    }
  })
}
</script>
