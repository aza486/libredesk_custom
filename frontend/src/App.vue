<template>
  <Toaster />
  <TooltipProvider :delay-duration="200">
    <div>
      <div v-if="$route.path !== '/'">
        <ResizablePanelGroup direction="horizontal" auto-save-id="app.vue.resizable.panel">
          <ResizablePanel class="shadow shadow-gray-300" id="resize-panel-1" collapsible :default-size="10" :collapsed-size="1" :min-size="7"
            :max-size="20" :class="cn(isCollapsed && 'min-w-[50px] transition-all duration-200 ease-in-out')"
            @expand="toggleNav(false)" @collapse="toggleNav(true)">
            <NavBar :is-collapsed="isCollapsed" :links="navLinks" :bottom-links="bottomLinks" />
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
import { ref, onMounted, computed, onUnmounted } from 'vue'
import { RouterView, useRouter } from 'vue-router'
import { cn } from '@/lib/utils'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { initWS } from '@/websocket.js'
import { useEmitter } from '@/composables/useEmitter'
import { Toaster } from '@/components/ui/toast'
import NavBar from '@/components/NavBar.vue'
import { useToast } from '@/components/ui/toast/use-toast'
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from '@/components/ui/resizable'
import { TooltipProvider } from '@/components/ui/tooltip'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'

const { t } = useI18n()
const { toast } = useToast()
const emitter = useEmitter()
const isCollapsed = ref(false)
const allNavLinks = ref([
  {
    title: t('navbar.dashboard'),
    to: '/dashboard',
    label: '',
    icon: 'lucide:layout-dashboard',
    permission: 'dashboard_global:read',
  },
  {
    title: t('navbar.conversations'),
    to: '/conversations',
    label: '',
    icon: 'lucide:message-circle-more'
  },
  {
    title: t('navbar.account'),
    to: '/account/profile',
    label: '',
    icon: 'lucide:circle-user-round'
  },
  {
    title: t('navbar.admin'),
    to: '/admin/general',
    label: '',
    icon: 'lucide:settings',
    permission: 'admin:read'
  }
])

const bottomLinks = ref([
  {
    to: '/logout',
    icon: 'lucide:log-out',
    title: 'Logout'
  }
])
const userStore = useUserStore()
const router = useRouter()

function toggleNav (v) {
  isCollapsed.value = v
}

onMounted(() => {
  initToaster()
  getCurrentUser()
  initWS()
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.SHOW_TOAST, toast)
})

const getCurrentUser = () => {
  userStore.getCurrentUser().catch((err) => {
    if (err.response && err.response.status === 401) {
      router.push('/login')
    }
  })
}

const initToaster = () => {
  emitter.on(EMITTER_EVENTS.SHOW_TOAST, toast)
}

const navLinks = computed(() =>
  allNavLinks.value.filter((link) =>
    link.permission ? userStore.hasPermission(link.permission) : true
  )
)
</script>
